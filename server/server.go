package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	noesctmpl "text/template"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/oliveagle/gotty/irc"
	"github.com/oliveagle/gotty/pkg/homedir"
	"github.com/oliveagle/gotty/pkg/randomstring"
	"github.com/oliveagle/gotty/summary"
	"github.com/oliveagle/gotty/webtty"
)

// Build info (set from main)
var (
	BuildVersion = "unknown"
	BuildCommit  = "unknown"
	BuildTime    = "unknown"
)

// Server provides a webtty HTTP endpoint.
type Server struct {
	factory           Factory
	options           *Options
	sessionManager    *SessionManager
	workspaceManager  *WorkspaceManager
	clipboardManager  *ClipboardManager
	webAuthnManager   *WebAuthnManager
	webAuthnSessions  *SessionDataManager
	authSessionMgr    *AuthSessionManager
	loginAttemptMgr   *LoginAttemptManager // SECURITY: Login attempt tracking
	ipFilter          *IPFilter            // SECURITY: IP-based access control

	ircHandler *irc.IRCHandler

	upgrader      *websocket.Upgrader
	indexTemplate *template.Template
	titleTemplate *noesctmpl.Template
}

// New creates a new instance of Server.
// Server will use the New() of the factory provided to handle each request.
func New(factory Factory, options *Options) (*Server, error) {
	indexData, err := Asset("resources/index.html")
	if err != nil {
		panic("index not found") // must be in bindata
	}
	if options.IndexFile != "" {
		path := homedir.Expand(options.IndexFile)
		indexData, err = os.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read custom index file at `%s`", path)
		}
	}
	indexTemplate, err := template.New("index").Parse(string(indexData))
	if err != nil {
		panic("index template parse failed") // must be valid
	}

	titleTemplate, err := noesctmpl.New("title").Parse(options.TitleFormat)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse window title format `%s`", options.TitleFormat)
	}

	var originChekcer func(r *http.Request) bool
	if options.WSOrigin != "" {
		matcher, err := regexp.Compile(options.WSOrigin)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compile regular expression of Websocket Origin: %s", options.WSOrigin)
		}
		originChekcer = func(r *http.Request) bool {
			return matcher.MatchString(r.Header.Get("Origin"))
		}
	} else {
		// SECURITY: Default to same-origin only when WSOrigin is not configured
		// This prevents Cross-Site WebSocket Hijacking (CSWSH) attacks
		originChekcer = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Allow requests without Origin header (e.g., CLI tools, same-origin browsers)
			if origin == "" {
				return true
			}
			// Parse origin and check if it matches the request host
			// Allow both http and https for localhost development
			host := r.Host
			// Same-origin check
			if strings.HasPrefix(origin, "http://"+host) || strings.HasPrefix(origin, "https://"+host) {
				return true
			}
			// Allow localhost for development (both http and https)
			if strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "https://localhost") ||
				strings.HasPrefix(origin, "http://127.0.0.1") ||
				strings.HasPrefix(origin, "https://127.0.0.1") {
				return true
			}
			// Reject all other origins
			log.Printf("[SECURITY] Rejected WebSocket connection from non-allowed origin: %s", origin)
			return false
		}
	}

	sm := NewSessionManager()
	sm.SetFactory(factory)
	sm.RestoreSessions() // Restore sessions from persistent backends like zellij

	// Initialize workspace manager
	wm := NewWorkspaceManager()

	// Initialize clipboard manager
	cm := NewClipboardManager()

	// Initialize WebAuthn manager if auth is enabled
	var webAuthnMgr *WebAuthnManager
	var webAuthnSessions *SessionDataManager
	var authSessionMgr *AuthSessionManager
	if options.EnableAuth {
		var err error
		webAuthnMgr, err = NewWebAuthnManager(
			options.WebAuthnDisplayName,
			options.HostName,
			options.WebAuthnDataDir,
			options.WebAuthnRegisterToken,
			options.WebAuthnAllowRegister,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to initialize WebAuthn")
		}
		webAuthnSessions = NewSessionDataManager()

		// Initialize auth session manager for long-lived sessions
		authSessionMgr = NewAuthSessionManager(homedir.Expand(options.WebAuthnDataDir), options.WebAuthnSessionTTL)
		log.Printf("WebAuthn/Passkeys enabled. Display name: %s, Session TTL: %d hours", options.WebAuthnDisplayName, options.WebAuthnSessionTTL)

		if webAuthnMgr.HasCredentials() {
			log.Printf("WebAuthn: Found existing credentials")
			if options.WebAuthnRegisterToken != "" {
				log.Printf("WebAuthn: Registration allowed with token")
			} else {
				log.Printf("WebAuthn: Registration blocked (no token configured)")
			}
		} else {
			log.Printf("WebAuthn: No credentials registered yet, registration required")
		}
	}

	// Migrate sessions without workspace to default workspace
	sm.MigrateToWorkspace(DefaultWorkspaceID)

	if options.EnableSummary {
		log.Printf("Session summarization enabled with model: %s at %s", options.SummaryModel, options.SummaryEndpoint)
	}

	// Initialize IRC handler if enabled
	var ircHandler *irc.IRCHandler
	if options.EnableIRC {
		ircConfig := irc.DefaultConfig()
		ircConfig.DefaultChannel = options.IRCDefaultChannel
		ircConfig.NetworkName = options.IRCNetworkName
		ircServer := irc.NewServer(ircConfig)
		ircHandler = irc.NewIRCHandler(ircServer)
		log.Printf("IRC chatroom enabled. Network: %s, Default channel: %s", ircConfig.NetworkName, ircConfig.DefaultChannel)
	}

	// SECURITY: Initialize login attempt manager (5 attempts, 15 minute block)
	loginAttemptMgr := NewLoginAttemptManager(5, 15*time.Minute)

	// SECURITY: Initialize IP filter
	ipFilter := NewIPFilter()

	return &Server{
		factory:          factory,
		options:          options,
		sessionManager:   sm,
		workspaceManager: wm,
		clipboardManager: cm,
		webAuthnManager:  webAuthnMgr,
		webAuthnSessions: webAuthnSessions,
		authSessionMgr:   authSessionMgr,
		loginAttemptMgr:  loginAttemptMgr,
		ipFilter:         ipFilter,
		ircHandler:       ircHandler,

		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols:    webtty.Protocols,
			CheckOrigin:     originChekcer,
		},
		indexTemplate: indexTemplate,
		titleTemplate: titleTemplate,
	}, nil
}

// newSummaryService creates a new summary service for a specific session
func (server *Server) newSummaryService(sessionID string, command string, argv []string, workDir string) *summary.Service {
	config := summary.Config{
		Enabled:     true,
		Interval:    time.Duration(server.options.SummaryInterval) * time.Second,
		BufferSize:  16384,
		LLMProvider: "openai", // llama.cpp uses OpenAI-compatible API
		LLMModel:    server.options.SummaryModel,
		LLMEndpoint: server.options.SummaryEndpoint,
		MaxTokens:   30,
		SystemPrompt: `/no_think
根据终端输出，简短描述用户正在做什么活动（不超过15字）。

关注：
1. 正在操作的文件或目录
2. 正在监控或查看的内容
3. 正在执行的任务

规则：
1. 突出具体操作对象，不只说程序名
2. 有错误标注 [错]
3. 只输出一句话

示例：
编辑 nginx.conf
查看系统监控
编译 Go 项目
git 提交代码
查看日志文件`,
	}

	svc := summary.NewService(config)
	svc.SetSessionID(sessionID)
	svc.SetContext(command, argv, workDir)

	// Set callback to update session subtitle
	svc.OnSummary(func(s summary.SessionSummary) {
		// Update session in manager
		server.sessionManager.UpdateSubtitle(sessionID, s.Summary)
		log.Printf("[Summary] Session %s: %s", sessionID, s.Summary)
	})

	// Set callback to capture output to session buffer
	svc.OnOutput(func(data []byte) {
		server.sessionManager.CaptureOutput(sessionID, data)
	})

	return svc
}

// Run starts the main process of the Server.
// The cancelation of ctx will shutdown the server immediately with aborting
// existing connections. Use WithGracefullContext() to support gracefull shutdown.
func (server *Server) Run(ctx context.Context, options ...RunOption) error {
	// SECURITY: Run security configuration checks at startup
	securityChecker := NewSecurityChecker(server.options)
	securityChecker.RunChecks()
	securityChecker.PrintReport()

	cctx, cancel := context.WithCancel(ctx)
	opts := &RunOptions{gracefullCtx: context.Background()}
	for _, opt := range options {
		opt(opts)
	}

	counter := newCounter(time.Duration(server.options.Timeout) * time.Second)

	path := "/"
	if server.options.EnableRandomUrl {
		// SECURITY: Enforce minimum random URL length to prevent brute-force attacks
		urlLength := server.options.RandomUrlLength
		if urlLength < 16 {
			log.Printf("[SECURITY] Warning: RandomUrlLength %d is too short, using minimum 16", urlLength)
			urlLength = 16
		}
		path = "/" + randomstring.Generate(urlLength) + "/"
	}

	handlers := server.setupHandlers(cctx, cancel, path, counter)
	srv, err := server.setupHTTPServer(handlers)
	if err != nil {
		return errors.Wrapf(err, "failed to setup an HTTP server")
	}

	if server.options.PermitWrite {
		log.Printf("Permitting clients to write input to the PTY.")
	}
	if server.options.Once {
		log.Printf("Once option is provided, accepting only one client")
	}

	// Start session poller for background subtitle updates
	if server.options.EnableSummary {
		pollerConfig := summary.Config{
			Enabled:     true,
			LLMProvider: "openai",
			LLMModel:    server.options.SummaryModel,
			LLMEndpoint: server.options.SummaryEndpoint,
			MaxTokens:   30,
			SystemPrompt: `/no_think
根据终端输出，简短描述用户正在做什么活动（不超过15字）。

关注：
1. 正在操作的文件或目录
2. 正在监控或查看的内容
3. 正在执行的任务

规则：
1. 突出具体操作对象，不只说程序名
2. 有错误标注 [错]
3. 只输出一句话

示例：
编辑 nginx.conf
查看系统监控
编译 Go 项目
git 提交代码
查看日志文件`,
		}
		poller := NewSessionPoller(server.sessionManager, pollerConfig, 120*time.Second)
		poller.Start(cctx)
	}

	if server.options.Port == "0" {
		log.Printf("Port number configured to `0`, choosing a random port")
	}
	hostPort := net.JoinHostPort(server.options.Address, server.options.Port)
	listener, err := net.Listen("tcp", hostPort)
	if err != nil {
		return errors.Wrapf(err, "failed to listen at `%s`", hostPort)
	}

	scheme := "http"
	if server.options.EnableTLS {
		scheme = "https"
	}
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	log.Printf("HTTP server is listening at: %s", scheme+"://"+host+":"+port+path)
	if server.options.Address == "0.0.0.0" {
		for _, address := range listAddresses() {
			log.Printf("Alternative URL: %s", scheme+"://"+address+":"+port+path)
		}
	}

	srvErr := make(chan error, 1)
	go func() {
		if server.options.EnableTLS {
			crtFile := homedir.Expand(server.options.TLSCrtFile)
			keyFile := homedir.Expand(server.options.TLSKeyFile)
			log.Printf("TLS crt file: " + crtFile)
			log.Printf("TLS key file: " + keyFile)

			err = srv.ServeTLS(listener, crtFile, keyFile)
		} else {
			err = srv.Serve(listener)
		}
		if err != nil {
			srvErr <- err
		}
	}()

	go func() {
		select {
		case <-opts.gracefullCtx.Done():
			srv.Shutdown(context.Background())
		case <-cctx.Done():
		}
	}()

	select {
	case err = <-srvErr:
		if err == http.ErrServerClosed { // by gracefull ctx
			err = nil
		} else {
			cancel()
		}
	case <-cctx.Done():
		srv.Close()
		err = cctx.Err()
	}

	conn := counter.count()
	if conn > 0 {
		log.Printf("Waiting for %d connections to be closed", conn)
	}
	counter.wait()

	return err
}

func (server *Server) setupHandlers(ctx context.Context, cancel context.CancelFunc, pathPrefix string, counter *counter) http.Handler {
	staticFileHandler := http.FileServer(
		&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "resources"},
	)

	var siteMux = http.NewServeMux()
	siteMux.HandleFunc(pathPrefix, server.handleIndex)
	siteMux.Handle(pathPrefix+"js/", http.StripPrefix(pathPrefix, staticFileHandler))
	siteMux.Handle(pathPrefix+"favicon.png", http.StripPrefix(pathPrefix, staticFileHandler))
	siteMux.Handle(pathPrefix+"css/", http.StripPrefix(pathPrefix, staticFileHandler))

	siteMux.HandleFunc(pathPrefix+"auth_token.js", server.handleAuthToken)
	siteMux.HandleFunc(pathPrefix+"config.js", server.handleConfig)

	// Create auth middleware
	authMiddleware := NewAuthMiddleware(server)

	// Session management API (protected)
	siteMux.HandleFunc(pathPrefix+"api/sessions", authMiddleware.Wrap(server.handleSessions))
	siteMux.HandleFunc(pathPrefix+"api/sessions/reorder", authMiddleware.Wrap(server.handleReorder))
	siteMux.HandleFunc(pathPrefix+"api/sessions/", authMiddleware.Wrap(server.handleSession))

	// Workspace management API (protected)
	siteMux.HandleFunc(pathPrefix+"api/workspaces", authMiddleware.Wrap(server.handleWorkspaces))
	siteMux.HandleFunc(pathPrefix+"api/workspaces/", authMiddleware.Wrap(server.handleWorkspace))

	// Clipboard API (protected)
	siteMux.HandleFunc(pathPrefix+"api/clipboard", authMiddleware.Wrap(server.handleClipboard))

	// Weather API proxy (protected)
	siteMux.HandleFunc(pathPrefix+"api/weather", authMiddleware.Wrap(server.handleWeather))

	// Build info API (protected)
	siteMux.HandleFunc(pathPrefix+"api/build-info", authMiddleware.Wrap(server.handleBuildInfo))

	// Weather preview debug page (protected)
	siteMux.HandleFunc(pathPrefix+"weather-preview.html", authMiddleware.Wrap(server.handleWeatherPreview))

	// Settings page (protected)
	siteMux.HandleFunc(pathPrefix+"settings.html", authMiddleware.Wrap(server.handleSettings))

	// WebAuthn/Passkeys API (public - used for authentication flow)
	siteMux.HandleFunc(pathPrefix+"api/webauthn/status", server.handleWebAuthnStatus)
	siteMux.HandleFunc(pathPrefix+"api/webauthn/register/begin", server.handleWebAuthnRegisterBegin)
	siteMux.HandleFunc(pathPrefix+"api/webauthn/register/finish", server.handleWebAuthnRegisterFinish)
	siteMux.HandleFunc(pathPrefix+"api/webauthn/login/begin", server.handleWebAuthnLoginBegin)
	siteMux.HandleFunc(pathPrefix+"api/webauthn/login/finish", server.handleWebAuthnLoginFinish)
	siteMux.HandleFunc(pathPrefix+"api/webauthn/validate", server.handleWebAuthnValidateToken)

	// IRC chatroom routes
	if server.options.EnableIRC && server.ircHandler != nil {
		ircData := struct {
			DefaultChannel string
			NetworkName    string
		}{
			DefaultChannel: server.options.IRCDefaultChannel,
			NetworkName:    server.options.IRCNetworkName,
		}
		// IRC index page (protected)
		siteMux.HandleFunc("/irc/", authMiddleware.Wrap(server.handleIRCIndex(ircData)))
		// IRC WebSocket is protected with token in query parameter
		siteMux.HandleFunc("/irc/ws", authMiddleware.WrapWS(server.ircHandler.HandleWS))
	}

	siteHandler := http.Handler(siteMux)

	if server.options.EnableAuth {
		log.Printf("Using WebAuthn/Passkeys Authentication")
		// WebAuthn authentication is handled via API endpoints, no HTTP auth wrapper needed
	}

	// Apply rate limiting middleware for security
	siteHandler = server.RateLimitMiddleware(siteHandler)

	withGz := gziphandler.GzipHandler(server.wrapHeaders(siteHandler))
	siteHandler = server.wrapLogger(withGz)

	wsMux := http.NewServeMux()
	wsMux.Handle("/", siteHandler)
	wsMux.HandleFunc(pathPrefix+"ws", server.generateHandleWS(ctx, cancel, counter))
	siteHandler = http.Handler(wsMux)

	return siteHandler
}

func (server *Server) setupHTTPServer(handler http.Handler) (*http.Server, error) {
	srv := &http.Server{
		Handler: handler,
	}

	if server.options.EnableTLSClientAuth {
		tlsConfig, err := server.tlsConfig()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to setup TLS configuration")
		}
		srv.TLSConfig = tlsConfig
	}

	return srv, nil
}

func (server *Server) tlsConfig() (*tls.Config, error) {
	caFile := homedir.Expand(server.options.TLSCACrtFile)
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, errors.New("could not open CA crt file " + caFile)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("could not parse CA crt file data in " + caFile)
	}
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	return tlsConfig, nil
}

// handleIRCIndex handles the IRC chatroom index page
func (server *Server) handleIRCIndex(ircCfg struct {
	DefaultChannel string
	NetworkName    string
}) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse the IRC mode template
		ircData, err := Asset("resources/irc_mode.html")
		if err != nil {
			http.Error(w, "IRC template not found", 404)
			return
		}

		// Simple template replacement
		content := string(ircData)
		content = strings.ReplaceAll(content, "{{.DefaultChannel}}", ircCfg.DefaultChannel)
		content = strings.ReplaceAll(content, "{{.NetworkName}}", ircCfg.NetworkName)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}
}
