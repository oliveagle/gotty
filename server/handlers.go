package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/oliveagle/gotty/webtty"
)

func (server *Server) generateHandleWS(ctx context.Context, cancel context.CancelFunc, counter *counter) http.HandlerFunc {
	once := new(int64)

	go func() {
		select {
		case <-counter.timer().C:
			cancel()
		case <-ctx.Done():
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		if server.options.Once {
			success := atomic.CompareAndSwapInt64(once, 0, 1)
			if !success {
				http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
				return
			}
		}

		num := counter.add(1)
		closeReason := "unknown reason"

		defer func() {
			num := counter.done()
			log.Printf(
				"Connection closed by %s: %s, connections: %d/%d",
				closeReason, r.RemoteAddr, num, server.options.MaxConnection,
			)

			if server.options.Once {
				cancel()
			}
		}()

		if int64(server.options.MaxConnection) != 0 {
			if num > server.options.MaxConnection {
				closeReason = "exceeding max number of connections"
				return
			}
		}

		log.Printf("New client connected: %s, connections: %d/%d", r.RemoteAddr, num, server.options.MaxConnection)

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		conn, err := server.upgrader.Upgrade(w, r, nil)
		if err != nil {
			closeReason = err.Error()
			return
		}
		defer conn.Close()

		// Get session_id from query params
		sessionID := r.URL.Query().Get("session_id")

		err = server.processWSConn(ctx, conn, sessionID)

		switch err {
		case ctx.Err():
			closeReason = "cancelation"
		case webtty.ErrSlaveClosed:
			closeReason = server.factory.Name()
		case webtty.ErrMasterClosed:
			closeReason = "client"
		default:
			closeReason = fmt.Sprintf("an error: %s", err)
		}
	}
}

func (server *Server) processWSConn(ctx context.Context, conn *websocket.Conn, sessionID string) error {
	typ, initLine, err := conn.ReadMessage()
	if err != nil {
		return errors.Wrapf(err, "failed to authenticate websocket connection")
	}
	if typ != websocket.TextMessage {
		return errors.New("failed to authenticate websocket connection: invalid message type")
	}

	var init InitMessage
	err = json.Unmarshal(initLine, &init)
	if err != nil {
		return errors.Wrapf(err, "failed to authenticate websocket connection")
	}

	// Authenticate based on auth type
	if server.options.EnableAuth || server.options.EnableBasicAuth {
		if server.options.AuthType == "basic" || server.options.AuthType == "" {
			// Basic authentication: verify AuthToken matches Credential
			if init.AuthToken != server.options.Credential {
				log.Printf("Authentication failed for %s", conn.RemoteAddr())
				return errors.New("failed to authenticate websocket connection: invalid credential")
			}
			log.Printf("Authentication succeeded for %s", conn.RemoteAddr())
		} else if server.options.AuthType == "bitwarden" {
			// Bitwarden authentication: verify the token
			// The token should be the derived key identifier from the client
			// For now, we accept any non-empty token (real validation will be implemented later)
			if init.AuthToken == "" {
				log.Printf("Bitwarden authentication failed for %s", conn.RemoteAddr())
				return errors.New("failed to authenticate websocket connection: empty bitwarden token")
			}
			log.Printf("Bitwarden authentication succeeded for %s", conn.RemoteAddr())
		} else {
			log.Printf("Unsupported auth type: %s", server.options.AuthType)
			return errors.New("unsupported authentication type")
		}
	}

	// Determine slave (either existing session or new)
	var slave Slave
	var isNewSession bool
	if sessionID != "" {
		// Try to join existing session
		session, ok := server.sessionManager.Get(sessionID)
		if !ok {
			return errors.New("session not found")
		}
		slave = session.Slave
		isNewSession = false
		log.Printf("Client joined existing session: %s", sessionID)
	} else {
		// Create new session
		queryPath := "?"
		if server.options.PermitArguments && init.Arguments != "" {
			queryPath = init.Arguments
		}

		query, err := url.Parse(queryPath)
		if err != nil {
			return errors.Wrapf(err, "failed to parse arguments")
		}
		params := query.Query()
		slave, err = server.factory.New(params)
		if err != nil {
			return errors.Wrapf(err, "failed to create backend")
		}
		// Create session in manager
		server.sessionManager.Create(server.factory.Name(), slave)
		isNewSession = true
		log.Printf("Client created new session")
	}

	// Only close slave for new sessions (not for joined sessions)
	if isNewSession {
		defer slave.Close()
	}

	titleVars := server.titleVariables(
		[]string{"server", "master", "slave"},
		map[string]map[string]interface{}{
			"server": server.options.TitleVariables,
			"master": map[string]interface{}{
				"remote_addr": conn.RemoteAddr(),
			},
			"slave": slave.WindowTitleVariables(),
		},
	)

	titleBuf := new(bytes.Buffer)
	err = server.titleTemplate.Execute(titleBuf, titleVars)
	if err != nil {
		return errors.Wrapf(err, "failed to fill window title template")
	}

	opts := []webtty.Option{
		webtty.WithWindowTitle(titleBuf.Bytes()),
	}
	if server.options.PermitWrite {
		opts = append(opts, webtty.WithPermitWrite())
	}
	if server.options.EnableReconnect {
		opts = append(opts, webtty.WithReconnect(server.options.ReconnectTime))
	}
	if server.options.Width > 0 {
		opts = append(opts, webtty.WithFixedColumns(server.options.Width))
	}
	if server.options.Height > 0 {
		opts = append(opts, webtty.WithFixedRows(server.options.Height))
	}
	if server.options.Preferences != nil {
		opts = append(opts, webtty.WithMasterPreferences(server.options.Preferences))
	}

	tty, err := webtty.New(&wsWrapper{conn}, slave, opts...)
	if err != nil {
		return errors.Wrapf(err, "failed to create webtty")
	}

	err = tty.Run(ctx)

	return err
}

func (server *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	titleVars := server.titleVariables(
		[]string{"server", "master"},
		map[string]map[string]interface{}{
			"server": server.options.TitleVariables,
			"master": map[string]interface{}{
				"remote_addr": r.RemoteAddr,
			},
		},
	)

	titleBuf := new(bytes.Buffer)
	err := server.titleTemplate.Execute(titleBuf, titleVars)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	indexVars := map[string]interface{}{
		"title": titleBuf.String(),
	}

	indexBuf := new(bytes.Buffer)
	err = server.indexTemplate.Execute(indexBuf, indexVars)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	w.Write(indexBuf.Bytes())
}

func (server *Server) handleAuthToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")

	if server.options.EnableAuth && server.options.AuthType == "bitwarden" {
		// For bitwarden auth, tell the client to use bitwarden authentication
		// The client will derive the key from master password and send it
		w.Write([]byte("var gotty_auth_type = 'bitwarden';"))
		w.Write([]byte("var gotty_auth_token = '';"))
	} else if server.options.EnableAuth && (server.options.AuthType == "basic" || server.options.AuthType == "") {
		// Basic auth - return credential
		w.Write([]byte("var gotty_auth_type = 'basic';"))
		w.Write([]byte("var gotty_auth_token = '" + server.options.Credential + "';"))
	} else {
		// No auth
		w.Write([]byte("var gotty_auth_type = 'none';"))
		w.Write([]byte("var gotty_auth_token = '';"))
	}
}

func (server *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte("var gotty_term = '" + server.options.Term + "';"))
}

// titleVariables merges maps in a specified order.
// varUnits are name-keyed maps, whose names will be iterated using order.
func (server *Server) titleVariables(order []string, varUnits map[string]map[string]interface{}) map[string]interface{} {
	titleVars := map[string]interface{}{}

	for _, name := range order {
		vars, ok := varUnits[name]
		if !ok {
			panic("title variable name error")
		}
		for key, val := range vars {
			titleVars[key] = val
		}
	}

	// safe net for conflicted keys
	for _, name := range order {
		titleVars[name] = varUnits[name]
	}

	return titleVars
}

// handleSessions handles session list and creation
func (server *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// List all sessions
		sessions := server.sessionManager.List()
		type sessionInfo struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			CreatedAt string `json:"created_at"`
		}
		result := make([]sessionInfo, 0, len(sessions))
		for _, s := range sessions {
			result = append(result, sessionInfo{
				ID:        s.ID,
				Title:     s.Title,
				CreatedAt: s.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": result,
		})
	case "POST":
		// Create new session
		params := make(map[string][]string)
		slave, err := server.factory.New(params)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		session := server.sessionManager.Create(server.factory.Name(), slave)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    session.ID,
			"title": session.Title,
		})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// handleSession handles individual session operations
func (server *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract session ID from URL path
	id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

	switch r.Method {
	case "GET":
		// Get session info
		session, ok := server.sessionManager.Get(id)
		if !ok {
			http.Error(w, "Session not found", 404)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        session.ID,
			"title":     session.Title,
			"created_at": session.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	case "DELETE":
		// Close session
		err := server.sessionManager.Close(id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "closed"})
	case "PATCH":
		// Rename session
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", 400)
			return
		}
		if req.Title == "" {
			http.Error(w, "Title cannot be empty", 400)
			return
		}
		if !server.sessionManager.Rename(id, req.Title) {
			http.Error(w, "Session not found", 404)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"id":    id,
			"title": req.Title,
		})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}
