package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
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

	// Authenticate using WebAuthn if enabled
	if server.options.EnableAuth {
		// WebAuthn/Passkeys authentication
		if init.AuthToken == "" {
			log.Printf("WebAuthn authentication failed for %s: empty token", conn.RemoteAddr())
			return errors.New("failed to authenticate websocket connection: empty token")
		}

		// Validate token using AuthSessionManager
		if !server.authSessionMgr.ValidateToken(init.AuthToken) {
			log.Printf("WebAuthn authentication failed for %s: invalid or expired token", conn.RemoteAddr())
			return errors.New("failed to authenticate websocket connection: invalid or expired token")
		}
		log.Printf("WebAuthn authentication succeeded for %s", conn.RemoteAddr())
	}

	// Determine slave (either existing session or new)
	var slave Slave
	var isNewSession bool
	var currentSessionID string

	if sessionID != "" {
		// Try to join existing session
		session, ok := server.sessionManager.Get(sessionID)
		if !ok {
			return errors.New("session not found")
		}
		currentSessionID = sessionID

		// For persistent backends (like zellij), create a new slave to attach
		// For non-persistent backends, reuse existing slave
		if server.factory.IsPersistent() {
			// Create new slave to attach to existing zellij session
			queryPath := "?"
			if server.options.PermitArguments && init.Arguments != "" {
				queryPath = init.Arguments
			}
			query, parseErr := url.Parse(queryPath)
			if parseErr != nil {
				return errors.Wrapf(parseErr, "failed to parse arguments")
			}
			params := query.Query()

			// Add target_tab if we have a saved active tab
			if session.ActiveTab != "" {
				params["_target_tab"] = []string{session.ActiveTab}
			}

			var attachErr error
			slave, attachErr = server.factory.NewWithID(sessionID, params)
			if attachErr != nil {
				return errors.Wrapf(attachErr, "failed to attach to session")
			}
			log.Printf("Client attached to persistent session: %s (tab: %s)", sessionID, session.ActiveTab)
		} else {
			slave = session.Slave
		}
		isNewSession = false
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

		// Use CreateWithID for backends that need session ID (like zellij)
		session, createErr := server.sessionManager.CreateWithID(server.factory.Name(), params)
		if createErr != nil {
			return errors.Wrapf(createErr, "failed to create backend")
		}
		if session != nil {
			slave = session.Slave
			currentSessionID = session.ID
			// Set workdir for new session
			if workDir, err := os.Getwd(); err == nil {
				server.sessionManager.UpdateWorkDir(currentSessionID, workDir)
			}
		} else {
			// Fallback to old method if CreateWithID returned nil (no factory set)
			slave, err = server.factory.New(params)
			if err != nil {
				return errors.Wrapf(err, "failed to create backend")
			}
			s := server.sessionManager.Create(server.factory.Name(), slave)
			currentSessionID = s.ID
			// Set workdir for new session
			if workDir, err := os.Getwd(); err == nil {
				server.sessionManager.UpdateWorkDir(currentSessionID, workDir)
			}
		}
		isNewSession = true
		log.Printf("Client created new session: %s", currentSessionID)
	}

	// Save current zellij tab if available
	if server.factory.Name() == "zellij" && currentSessionID != "" {
		activeTab := GetZellijActiveTab()
		if activeTab != "" {
			server.sessionManager.UpdateActiveTab(currentSessionID, activeTab)
		}
	}

	// Close slave when done
	// For persistent backends, this just closes the PTY connection, not the underlying session
	// For non-persistent backends with existing session, don't close (shared)
	if isNewSession || server.factory.IsPersistent() {
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

	// Add summary service if enabled
	if server.options.EnableSummary && currentSessionID != "" {
		// Get context from slave
		slaveVars := slave.WindowTitleVariables()
		command, _ := slaveVars["command"].(string)
		argv, _ := slaveVars["argv"].([]string)
		workDir, _ := os.Getwd()

		summarySvc := server.newSummaryService(currentSessionID, command, argv, workDir)
		opts = append(opts, webtty.WithSummaryService(summarySvc))
	}

	// Add activity callback for session activity tracking
	if currentSessionID != "" {
		opts = append(opts, webtty.WithActivityCallback(func() {
			server.sessionManager.UpdateActivity(currentSessionID)
		}))
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
		"title":    titleBuf.String(),
		"commit":   BuildCommit,
		"buildAt":  BuildTime,
		// Host name for display
		"hostName": server.options.HostName,
		// IRC chatroom config
		"enableIRC":      server.options.EnableIRC,
		"ircChannel":     server.options.IRCDefaultChannel,
		"ircNetworkName": server.options.IRCNetworkName,
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

	if server.options.EnableAuth {
		// WebAuthn/Passkeys authentication
		w.Write([]byte("var gotty_auth_type = 'webauthn';"))
		w.Write([]byte("var gotty_auth_token = '';"))
		// Include WebAuthn status
		hasAuth := "false"
		if server.webAuthnManager != nil && server.webAuthnManager.HasCredentials() {
			hasAuth = "true"
		}
		w.Write([]byte("var gotty_webauthn_has_auth = " + hasAuth + ";"))
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
		// Get workspace filter from query parameter
		workspaceID := r.URL.Query().Get("workspace_id")
		if workspaceID == "" {
			workspaceID = server.workspaceManager.GetActive()
		}

		// List sessions in the specified workspace
		sessions := server.sessionManager.ListByWorkspace(workspaceID)
		type sessionInfo struct {
			ID              string `json:"id"`
			Title           string `json:"title"`
			Subtitle        string `json:"subtitle,omitempty"`
			WorkDir         string `json:"workdir,omitempty"`
			ParentID        string `json:"parent_id,omitempty"`
			WorkspaceID     string `json:"workspace_id,omitempty"`
			IsFolder        bool   `json:"is_folder"`
			Order           int    `json:"order"`
			HasChildren     bool   `json:"has_children"`
			IsActive        bool   `json:"is_active"`
			LastActiveAgo   int64  `json:"last_active_ago"`
			CreatedAt       string `json:"created_at"`
		}
		result := make([]sessionInfo, 0, len(sessions))
		for _, s := range sessions {
			isActive, lastActiveAgo := server.sessionManager.GetActivity(s.ID)
			result = append(result, sessionInfo{
				ID:            s.ID,
				Title:         s.Title,
				Subtitle:      s.Subtitle,
				WorkDir:       s.WorkDir,
				ParentID:      s.ParentID,
				WorkspaceID:   s.WorkspaceID,
				IsFolder:      s.IsFolder,
				Order:         s.Order,
				HasChildren:   server.sessionManager.HasChildren(s.ID),
				IsActive:      isActive,
				LastActiveAgo: lastActiveAgo,
				CreatedAt:     s.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions":    result,
			"workspace_id": workspaceID,
		})
	case "POST":
		// Parse request body
		var req struct {
			ParentID string `json:"parent_id"`
			IsFolder bool   `json:"is_folder"`
			Title    string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// If no body, that's fine - create root session
			req.ParentID = ""
		}

		// Validate parent exists if parent_id is provided
		if req.ParentID != "" {
			parent, ok := server.sessionManager.Get(req.ParentID)
			if !ok {
				http.Error(w, "Parent not found", 404)
				return
			}
			// Parent must be a folder
			if !parent.IsFolder {
				http.Error(w, "Parent must be a folder", 400)
				return
			}
		}

		// Create folder or session
		if req.IsFolder {
			title := req.Title
			if title == "" {
				title = "New Folder"
			}
			session := server.sessionManager.CreateFolder(title, req.ParentID)
			// Set workspace for the folder
			activeWorkspace := server.workspaceManager.GetActive()
			server.sessionManager.SetWorkspaceID(session.ID, activeWorkspace)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           session.ID,
				"title":        session.Title,
				"is_folder":    true,
				"workspace_id": activeWorkspace,
			})
			return
		}

		// Create regular session
		params := make(map[string][]string)
		slave, err := server.factory.New(params)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var session *Session
		if req.ParentID != "" {
			session = server.sessionManager.CreateChild(server.factory.Name(), req.ParentID, slave)
		} else {
			session = server.sessionManager.Create(server.factory.Name(), slave)
		}
		// Set workspace for the session
		activeWorkspace := server.workspaceManager.GetActive()
		server.sessionManager.SetWorkspaceID(session.ID, activeWorkspace)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           session.ID,
			"title":        session.Title,
			"parent_id":    session.ParentID,
			"is_folder":    false,
			"workspace_id": activeWorkspace,
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
		response := map[string]interface{}{
			"id":         session.ID,
			"title":      session.Title,
			"created_at": session.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if session.ActiveTab != "" {
			response["active_tab"] = session.ActiveTab
		}
		json.NewEncoder(w).Encode(response)
	case "DELETE":
		// Check if this is a kill request (permanent deletion)
		kill := r.URL.Query().Get("kill") == "true"
		var err error
		if kill {
			// Permanently kill the session (including zellij backend)
			err = server.sessionManager.Kill(id)
		} else {
			// Just close the connection (session persists)
			err = server.sessionManager.Close(id)
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		status := "closed"
		if kill {
			status = "killed"
		}
		json.NewEncoder(w).Encode(map[string]string{"status": status})
	case "PATCH":
		// Update session fields
		var req struct {
			Title     string `json:"title"`
			ActiveTab string `json:"active_tab"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", 400)
			return
		}
		// Update title if provided
		if req.Title != "" {
			if !server.sessionManager.Rename(id, req.Title) {
				http.Error(w, "Session not found", 404)
				return
			}
		}
		// Update active tab if provided
		if req.ActiveTab != "" {
			server.sessionManager.UpdateActiveTab(id, req.ActiveTab)
		}
		response := map[string]string{"id": id}
		if req.Title != "" {
			response["title"] = req.Title
		}
		if req.ActiveTab != "" {
			response["active_tab"] = req.ActiveTab
		}
		json.NewEncoder(w).Encode(response)
	case "PUT":
		// Move session to folder or workspace
		var req struct {
			FolderID    string `json:"folder_id"`     // Empty string means move to root
			WorkspaceID string `json:"workspace_id"` // Move to different workspace
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", 400)
			return
		}
		if req.WorkspaceID != "" {
			// Move to workspace
			if !server.sessionManager.MoveToWorkspace(id, req.WorkspaceID) {
				http.Error(w, "Failed to move session to workspace", 400)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{
				"id":           id,
				"workspace_id": req.WorkspaceID,
			})
			return
		}
		// Move to folder
		if !server.sessionManager.MoveToFolder(id, req.FolderID) {
			http.Error(w, "Failed to move session", 400)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"id":        id,
			"parent_id": req.FolderID,
		})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// handleReorder handles session reorder requests
func (server *Server) handleReorder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
		ParentID  string `json:"parent_id"`  // Target parent (empty for root)
		AfterID   string `json:"after_id"`   // Insert after this session (empty for beginning)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", 400)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "session_id is required", 400)
		return
	}

	if !server.sessionManager.Reorder(req.SessionID, req.ParentID, req.AfterID) {
		http.Error(w, "Failed to reorder session", 400)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"id":        req.SessionID,
		"parent_id": req.ParentID,
		"after_id":  req.AfterID,
	})
}

// handleWorkspaces handles workspace list and creation
func (server *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// List all workspaces
		workspaces := server.workspaceManager.List()
		type workspaceInfo struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			ColorTheme string `json:"color_theme"`
			Icon       string `json:"icon"`
			Order      int    `json:"order"`
			IsActive   bool   `json:"is_active"`
			CreatedAt  string `json:"created_at"`
		}
		activeID := server.workspaceManager.GetActive()
		result := make([]workspaceInfo, 0, len(workspaces))
		for _, ws := range workspaces {
			result = append(result, workspaceInfo{
				ID:         ws.ID,
				Name:       ws.Name,
				ColorTheme: ws.ColorTheme,
				Icon:       ws.Icon,
				Order:      ws.Order,
				IsActive:   ws.ID == activeID,
				CreatedAt:  ws.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"workspaces": result,
			"active_id":  activeID,
		})
	case "POST":
		// Create new workspace
		var req struct {
			Name       string `json:"name"`
			ColorTheme string `json:"color_theme"`
			Icon       string `json:"icon"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", 400)
			return
		}
		if req.Name == "" {
			http.Error(w, "Name is required", 400)
			return
		}
		workspace := server.workspaceManager.Create(req.Name, req.ColorTheme, req.Icon)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          workspace.ID,
			"name":        workspace.Name,
			"color_theme": workspace.ColorTheme,
			"icon":        workspace.Icon,
		})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// handleWorkspace handles individual workspace operations
func (server *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if this is an activate request
	if strings.HasSuffix(r.URL.Path, "/activate") {
		server.handleWorkspaceActivate(w, r)
		return
	}

	// Extract workspace ID from URL path
	id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

	switch r.Method {
	case "GET":
		// Get workspace info
		workspace, ok := server.workspaceManager.Get(id)
		if !ok {
			http.Error(w, "Workspace not found", 404)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          workspace.ID,
			"name":        workspace.Name,
			"color_theme": workspace.ColorTheme,
			"icon":        workspace.Icon,
			"order":       workspace.Order,
			"created_at":  workspace.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	case "PATCH":
		// Update workspace
		var req struct {
			Name       string `json:"name"`
			ColorTheme string `json:"color_theme"`
			Icon       string `json:"icon"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", 400)
			return
		}
		if !server.workspaceManager.Update(id, req.Name, req.ColorTheme, req.Icon) {
			http.Error(w, "Workspace not found", 404)
			return
		}
		workspace, _ := server.workspaceManager.Get(id)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          id,
			"name":        workspace.Name,
			"color_theme": workspace.ColorTheme,
			"icon":        workspace.Icon,
		})
	case "DELETE":
		// Delete workspace (sessions will be moved to default)
		deletedID, ok := server.workspaceManager.Delete(id)
		if !ok {
			http.Error(w, "Cannot delete workspace", 400)
			return
		}
		// Move all sessions from deleted workspace to default
		sessions := server.sessionManager.ListByWorkspace(deletedID)
		for _, s := range sessions {
			server.sessionManager.MoveToWorkspace(s.ID, DefaultWorkspaceID)
		}
		json.NewEncoder(w).Encode(map[string]string{
			"id":      deletedID,
			"status":  "deleted",
			"message": "Sessions moved to default workspace",
		})
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// handleWorkspaceActivate handles activating a workspace
func (server *Server) handleWorkspaceActivate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	// Extract workspace ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	id := pathParts[len(pathParts)-2] // Second to last element is the ID

	if !server.workspaceManager.SetActive(id) {
		http.Error(w, "Workspace not found", 404)
		return
	}

	workspace, _ := server.workspaceManager.Get(id)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          id,
		"name":        workspace.Name,
		"color_theme": workspace.ColorTheme,
		"icon":        workspace.Icon,
		"active":      true,
	})
}

// handleWeather proxies weather API requests to avoid CORS issues
func (server *Server) handleWeather(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	// Get city code from query parameter
	cityCode := r.URL.Query().Get("city")
	if cityCode == "" {
		cityCode = "101020100" // Default to Shanghai
	}

	// Fetch weather data from sojson API
	weatherURL := fmt.Sprintf("http://t.weather.sojson.com/api/weather/city/%s", cityCode)

	resp, err := http.Get(weatherURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch weather: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response status and headers
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusBadGateway)
		return
	}
	w.Write(buf.Bytes())
}

// handleWeatherPreview serves the weather preview debug page
func (server *Server) handleWeatherPreview(w http.ResponseWriter, r *http.Request) {
	// Read the weather-preview.html from embedded assets
	data, err := Asset("resources/weather-preview.html")
	if err != nil {
		http.Error(w, "Weather preview page not found", 404)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
