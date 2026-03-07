package summary

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Config for the summary service
type Config struct {
	Enabled      bool          `json:"enabled"`
	Interval     time.Duration `json:"interval"`      // Summary generation interval
	BufferSize   int           `json:"buffer_size"`   // Max bytes to keep in buffer
	LLMProvider  string        `json:"llm_provider"`  // "ollama", "openai", "custom"
	LLMModel     string        `json:"llm_model"`     // Model name
	LLMEndpoint  string        `json:"llm_endpoint"`  // API endpoint
	LLMAPIKey    string        `json:"llm_api_key"`   // API key (optional)
	OutputFile   string        `json:"output_file"`   // File to write summaries (optional)
	MaxTokens    int           `json:"max_tokens"`    // Max tokens for summary
	SystemPrompt string        `json:"system_prompt"` // Custom system prompt
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		Interval:    30 * time.Second, // 每30秒更新
		BufferSize:  16384,            // 16KB 足够
		LLMProvider: "openai",         // llama.cpp uses OpenAI-compatible API
		LLMModel:    "qwen3-0.6b",
		LLMEndpoint: "http://localhost:8888",
		MaxTokens:   50, // 限制输出字数
		SystemPrompt: `/no_think
你是一个终端会话标题生成器。根据终端输出，生成一个简短的会话标题（不超过20个中文字或30个英文字符）。

规则：
1. 突出当前正在执行的主要任务
2. 如果有错误，标注 [错误]
3. 如果在等待输入，标注 [等待]
4. 只输出标题，不要其他内容

示例：
- "编辑 nginx 配置"
- "git commit [错误]"
- "python 脚本运行中"
- "htop 系统监控"`,
	}
}

// SessionSummary represents a generated summary
type SessionSummary struct {
	Timestamp   time.Time `json:"timestamp"`
	SessionID   string    `json:"session_id"`
	Summary     string    `json:"summary"`
	Commands    []string  `json:"commands,omitempty"`
	Errors      []string  `json:"errors,omitempty"`
	OutputBytes int       `json:"output_bytes"`
	InputBytes  int       `json:"input_bytes"`
}

// SummaryCallback is called when a summary is generated
type SummaryCallback func(sessionID string, summary string)

// Service handles terminal session summarization
type Service struct {
	config   Config
	buffer   *RingBuffer
	inputBuf *RingBuffer
	client   *http.Client
	timer    *time.Timer
	mu       sync.Mutex

	// Callbacks
	onSummary func(summary SessionSummary)

	// Stats
	outputBytes int
	inputBytes  int
	sessionID   string
}

// NewService creates a new summary service
func NewService(config Config) *Service {
	return &Service{
		config:    config,
		buffer:    NewRingBuffer(config.BufferSize),
		inputBuf:  NewRingBuffer(config.BufferSize / 4),
		client:    &http.Client{Timeout: 30 * time.Second},
		sessionID: fmt.Sprintf("session-%d", time.Now().Unix()),
	}
}

// OnSummary sets a callback for when a summary is generated
func (s *Service) OnSummary(callback func(summary SessionSummary)) {
	s.onSummary = callback
}

// CaptureOutput captures PTY output for summarization
func (s *Service) CaptureOutput(data []byte) {
	if !s.config.Enabled {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffer.Write(data)
	s.outputBytes += len(data)
}

// CaptureInput captures user input for context
func (s *Service) CaptureInput(data []byte) {
	if !s.config.Enabled {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputBuf.Write(data)
	s.inputBytes += len(data)
}

// SetSessionID sets the current session ID
func (s *Service) SetSessionID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID = id
}

// Start begins periodic summarization
func (s *Service) Start(ctx context.Context) {
	if !s.config.Enabled {
		return
	}

	log.Printf("[Summary] Service started, interval: %v, model: %s", s.config.Interval, s.config.LLMModel)
	s.timer = time.NewTimer(s.config.Interval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.timer.Stop()
				// Generate final summary on exit
				s.generateSummary()
				return
			case <-s.timer.C:
				s.generateSummary()
				s.timer.Reset(s.config.Interval)
			}
		}
	}()
}

// Stop stops the service and generates a final summary
func (s *Service) Stop() {
	if s.timer != nil {
		s.timer.Stop()
	}
	s.generateSummary()
}

// generateSummary calls the LLM to generate a summary
func (s *Service) generateSummary() {
	s.mu.Lock()
	outputData := s.buffer.Bytes()
	inputData := s.inputBuf.Bytes()
	outputBytes := s.outputBytes
	inputBytes := s.inputBytes
	s.buffer.Reset()
	s.inputBuf.Reset()
	s.mu.Unlock()

	// Skip if no data
	if len(outputData) == 0 {
		return
	}

	// Build prompt
	prompt := s.buildPrompt(outputData, inputData)

	// Call LLM
	summary, err := s.callLLM(prompt)
	if err != nil {
		log.Printf("[Summary] LLM error: %v", err)
		summary = fmt.Sprintf("[LLM Error: %v]", err)
	}

	sessionSummary := SessionSummary{
		Timestamp:   time.Now(),
		SessionID:   s.sessionID,
		Summary:     summary,
		OutputBytes: outputBytes,
		InputBytes:  inputBytes,
		Commands:    extractCommands(string(outputData)),
		Errors:      extractErrors(string(outputData)),
	}

	// Write to file if configured
	if s.config.OutputFile != "" {
		s.writeToFile(sessionSummary)
	}

	// Callback
	if s.onSummary != nil {
		s.onSummary(sessionSummary)
	}

	// Log the summary
	log.Printf("[Summary] %s | bytes: %d/%d | title: %s",
		sessionSummary.Timestamp.Format("15:04:05"),
		sessionSummary.OutputBytes,
		sessionSummary.InputBytes,
		sessionSummary.Summary,
	)
}

// buildPrompt constructs the prompt for the LLM
func (s *Service) buildPrompt(output, input []byte) string {
	var sb strings.Builder

	sb.WriteString(s.config.SystemPrompt)
	sb.WriteString("\n\n--- 终端输出 ---\n")

	// Truncate if too large
	maxOutput := 8000
	if len(output) > maxOutput {
		sb.WriteString("... (前 ")
		sb.WriteString(fmt.Sprintf("%d", len(output)-maxOutput))
		sb.WriteString(" 字节已省略)\n")
		sb.WriteString(string(output[len(output)-maxOutput:]))
	} else {
		sb.WriteString(string(output))
	}

	if len(input) > 0 {
		sb.WriteString("\n--- 用户输入 ---\n")
		maxInput := 2000
		if len(input) > maxInput {
			sb.WriteString(string(input[len(input)-maxInput:]))
		} else {
			sb.WriteString(string(input))
		}
	}

	return sb.String()
}

// callLLM makes an API call to the LLM provider
func (s *Service) callLLM(prompt string) (string, error) {
	switch s.config.LLMProvider {
	case "ollama":
		return s.callOllama(prompt)
	case "openai":
		return s.callOpenAI(prompt)
	default:
		return s.callOllama(prompt)
	}
}

// OllamaRequest represents an Ollama API request
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents an Ollama API response
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// callOllama calls the Ollama API
func (s *Service) callOllama(prompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:  s.config.LLMModel,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/api/generate", s.config.LLMEndpoint)
	resp, err := s.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama error: %s", string(respBody))
	}

	var result OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Response, nil
}

// OpenAIRequest represents an OpenAI-compatible API request
type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents an OpenAI API response
type OpenAIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// callOpenAI calls an OpenAI-compatible API
func (s *Service) callOpenAI(prompt string) (string, error) {
	reqBody := OpenAIRequest{
		Model: s.config.LLMModel,
		Messages: []Message{
			{Role: "system", Content: s.config.SystemPrompt},
			{Role: "user", Content: prompt},
		},
		MaxTokens: s.config.MaxTokens,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", s.config.LLMEndpoint+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.config.LLMAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.LLMAPIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai error: %s", string(respBody))
	}

	var result OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from LLM")
}

// writeToFile appends a summary to the output file
func (s *Service) writeToFile(summary SessionSummary) error {
	file, err := openAppendFile(s.config.OutputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

// openAppendFile opens a file for appending, creating if needed
func openAppendFile(path string) (io.WriteCloser, error) {
	return nil, fmt.Errorf("not implemented - use os.OpenFile")
}

// extractCommands tries to identify commands in the output
func extractCommands(output string) []string {
	var commands []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Simple heuristics for command detection
		if strings.HasPrefix(line, "$ ") ||
			strings.HasPrefix(line, "# ") ||
			strings.HasPrefix(line, "> ") {
			cmd := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "$ "), "# "), "> ")
			if len(cmd) > 0 && len(cmd) < 200 {
				commands = append(commands, cmd)
			}
		}
	}

	// Limit to 10 commands
	if len(commands) > 10 {
		commands = commands[len(commands)-10:]
	}

	return commands
}

// extractErrors tries to identify errors in the output
func extractErrors(output string) []string {
	var errors []string
	lines := strings.Split(output, "\n")

	errorKeywords := []string{"error", "Error", "ERROR", "failed", "Failed", "FAILED", "exception", "Exception"}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		for _, keyword := range errorKeywords {
			if strings.Contains(line, keyword) && len(line) < 500 {
				errors = append(errors, line)
				break
			}
		}
	}

	// Limit to 5 errors
	if len(errors) > 5 {
		errors = errors[len(errors)-5:]
	}

	return errors
}

// GetStats returns current statistics
func (s *Service) GetStats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"session_id":   s.sessionID,
		"output_bytes": s.outputBytes,
		"input_bytes":  s.inputBytes,
		"buffer_used":  s.buffer.Used(),
	}
}
