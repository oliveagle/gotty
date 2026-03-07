package server

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/oliveagle/gotty/summary"
)

// SessionPoller periodically polls all sessions and updates their subtitles
type SessionPoller struct {
	sessionManager *SessionManager
	summaryConfig  summary.Config
	interval       time.Duration

	// Track last output length for each session to detect changes
	lastLengths map[string]int
	mu          sync.Mutex
}

// NewSessionPoller creates a new session poller
func NewSessionPoller(sm *SessionManager, config summary.Config, interval time.Duration) *SessionPoller {
	return &SessionPoller{
		sessionManager: sm,
		summaryConfig:  config,
		interval:       interval,
		lastLengths:    make(map[string]int),
	}
}

// Start begins the polling loop
func (p *SessionPoller) Start(ctx context.Context) {
	if !p.summaryConfig.Enabled {
		return
	}

	log.Printf("[Poller] Starting session poller, interval: %v", p.interval)

	ticker := time.NewTicker(p.interval)
	go func() {
		defer ticker.Stop()

		// Initial poll after 5 seconds
		select {
		case <-time.After(5 * time.Second):
			p.pollAll(ctx)
		case <-ctx.Done():
			return
		}

		for {
			select {
			case <-ctx.Done():
				log.Printf("[Poller] Stopped")
				return
			case <-ticker.C:
				p.pollAll(ctx)
			}
		}
	}()
}

// pollAll iterates through all sessions and updates subtitles if needed
func (p *SessionPoller) pollAll(ctx context.Context) {
	sessions := p.sessionManager.List()
	if len(sessions) == 0 {
		return
	}

	log.Printf("[Poller] Polling %d sessions...", len(sessions))

	for _, session := range sessions {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Get output buffer
		output := p.sessionManager.GetOutputBuffer(session.ID)
		if len(output) == 0 {
			continue
		}

		// Check if output changed (by length as quick check)
		p.mu.Lock()
		lastLen, exists := p.lastLengths[session.ID]
		changed := !exists || lastLen != len(output)
		if changed {
			p.lastLengths[session.ID] = len(output)
		}
		p.mu.Unlock()

		if !changed {
			continue
		}

		// Check if subtitle already matches (no need to regenerate)
		if session.Subtitle != "" && !changed {
			continue
		}

		// Generate subtitle
		subtitle, err := p.generateSubtitle(ctx, output)
		if err != nil {
			log.Printf("[Poller] Failed to generate subtitle for %s: %v", session.ID, err)
			continue
		}

		if subtitle == "" {
			continue
		}

		// Update session subtitle
		p.sessionManager.UpdateSubtitle(session.ID, subtitle)
		log.Printf("[Poller] Session %s: %s", session.ID, subtitle)

		// Small delay between sessions to avoid overwhelming LLM
		time.Sleep(500 * time.Millisecond)
	}
}

// generateSubtitle calls the LLM to generate a subtitle
func (p *SessionPoller) generateSubtitle(ctx context.Context, output []byte) (string, error) {
	svc := summary.NewService(p.summaryConfig)
	return svc.GenerateFromOutput(ctx, output)
}
