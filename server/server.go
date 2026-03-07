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
	noesctmpl "text/template"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/oliveagle/gotty/pkg/homedir"
	"github.com/oliveagle/gotty/pkg/randomstring"
	"github.com/oliveagle/gotty/summary"
	"github.com/oliveagle/gotty/webtty"
)

// Server provides a webtty HTTP endpoint.
type Server struct {
	factory        Factory
	options        *Options
	sessionManager *SessionManager

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
	}

	sm := NewSessionManager()
	sm.SetFactory(factory)
	sm.RestoreSessions() // Restore sessions from persistent backends like zellij

	if options.EnableSummary {
		log.Printf("Session summarization enabled with model: %s at %s", options.SummaryModel, options.SummaryEndpoint)
	}

	return &Server{
		factory:        factory,
		options:        options,
		sessionManager: sm,

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

	return svc
}

// Run starts the main process of the Server.
// The cancelation of ctx will shutdown the server immediately with aborting
// existing connections. Use WithGracefullContext() to support gracefull shutdown.
func (server *Server) Run(ctx context.Context, options ...RunOption) error {
	cctx, cancel := context.WithCancel(ctx)
	opts := &RunOptions{gracefullCtx: context.Background()}
	for _, opt := range options {
		opt(opts)
	}

	counter := newCounter(time.Duration(server.options.Timeout) * time.Second)

	path := "/"
	if server.options.EnableRandomUrl {
		path = "/" + randomstring.Generate(server.options.RandomUrlLength) + "/"
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

	// Session management API
	siteMux.HandleFunc(pathPrefix+"api/sessions", server.handleSessions)
	siteMux.HandleFunc(pathPrefix+"api/sessions/", server.handleSession)

	siteHandler := http.Handler(siteMux)

	if server.options.EnableAuth || server.options.EnableBasicAuth {
		if server.options.AuthType == "basic" || server.options.AuthType == "" {
			log.Printf("Using Basic Authentication")
			siteHandler = server.wrapBasicAuth(siteHandler, server.options.Credential)
		} else if server.options.AuthType == "bitwarden" {
			log.Printf("Using Bitwarden E2E Encryption Authentication")
			// Bitwarden authentication is handled in WebSocket connection, no HTTP auth required
		} else {
			log.Printf("Unsupported authentication type: %s", server.options.AuthType)
		}
	}

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
