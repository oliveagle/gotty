package server

type Options struct {
	Address             string           `hcl:"address" json:"address" flagName:"address" flagSName:"a" flagDescribe:"IP address to listen" default:"127.0.0.1"`
	Port                string           `hcl:"port" json:"port" flagName:"port" flagSName:"p" flagDescribe:"Port number to liten" default:"13782"`
	EnableAuth          bool             `hcl:"enable_auth" json:"enable_auth" flagName:"auth" flagSName:"A" flagDescribe:"Enable WebAuthn authentication" default:"false"`
	EnableRandomUrl     bool             `hcl:"enable_random_url" json:"enable_random_url" flagName:"random-url" flagSName:"r" flagDescribe:"Add a random string to the URL" default:"false"`
	RandomUrlLength     int              `hcl:"random_url_length" json:"random_url_length" flagName:"random-url-length" flagDescribe:"Random URL length (minimum 16 for security)" default:"16"`
	EnableTLS           bool             `hcl:"enable_tls" json:"enable_tls" flagName:"tls" flagSName:"t" flagDescribe:"Enable TLS/SSL" default:"false"`
	TLSCrtFile          string           `hcl:"tls_crt_file" json:"tls_crt_file" flagName:"tls-crt" flagDescribe:"TLS/SSL certificate file path" default:"~/.gotty.crt"`
	TLSKeyFile          string           `hcl:"tls_key_file" json:"tls_key_file" flagName:"tls-key" flagDescribe:"TLS/SSL key file path" default:"~/.gotty.key"`
	EnableTLSClientAuth bool             `hcl:"enable_tls_client_auth" json:"enable_tls_client_auth" default:"false"`
	TLSCACrtFile        string           `hcl:"tls_ca_crt_file" json:"tls_ca_crt_file" flagName:"tls-ca-crt" flagDescribe:"TLS/SSL CA certificate file for client certifications" default:"~/.gotty.ca.crt"`
	IndexFile           string           `hcl:"index_file" json:"index_file" flagName:"index" flagDescribe:"Custom index.html file" default:""`
	TitleFormat         string           `hcl:"title_format" json:"title_format" flagName:"title-format" flagSName:"" flagDescribe:"Title format of browser window" default:"{{ .command }}@{{ .hostname }}"`
	EnableReconnect     bool             `hcl:"enable_reconnect" json:"enable_reconnect" flagName:"reconnect" flagDescribe:"Enable reconnection" default:"false"`
	ReconnectTime       int              `hcl:"reconnect_time" json:"reconnect_time" flagName:"reconnect-time" flagDescribe:"Time to reconnect" default:"10"`
	MaxConnection       int              `hcl:"max_connection" json:"max_connection" flagName:"max-connection" flagDescribe:"Maximum connection to gotty" default:"0"`
	Once                bool             `hcl:"once" json:"once" flagName:"once" flagDescribe:"Accept only one client and exit on disconnection" default:"false"`
	Timeout             int              `hcl:"timeout" json:"timeout" flagName:"timeout" flagDescribe:"Timeout seconds for waiting a client(0 to disable)" default:"0"`
	PermitWrite         bool             `hcl:"permit_write" json:"permit_write" flagName:"permit-write" flagSName:"w" flagDescribe:"Permit clients to write to the TTY (BE CAREFUL)" default:"false"`
	PermitArguments     bool             `hcl:"permit_arguments" json:"permit_arguments" flagName:"permit-arguments" flagDescribe:"Permit clients to send command line arguments in URL (e.g. http://example.com:8080/?arg=AAA&arg=BBB)" default:"false"`
	Preferences         *HtermPrefernces `hcl:"preferences" json:"preferences"`
	Width               int              `hcl:"width" json:"width" flagName:"width" flagDescribe:"Static width of the screen, 0(default) means dynamically resize" default:"0"`
	Height              int              `hcl:"height" json:"height" flagName:"height" flagDescribe:"Static height of the screen, 0(default) means dynamically resize" default:"0"`
	WSOrigin            string           `hcl:"ws_origin" json:"ws_origin" flagName:"ws-origin" flagDescribe:"A regular expression that matches origin URLs to be accepted by WebSocket. No cross origin requests are acceptable by default" default:""`
	Term                string           `hcl:"term" json:"term" flagName:"term" flagDescribe:"Terminal name to use on the browser, one of xterm or hterm." default:"xterm"`

	// Summary options
	EnableSummary   bool   `hcl:"enable_summary" json:"enable_summary" flagName:"summary" flagDescribe:"Enable session summarization using LLM" default:"false"`
	SummaryInterval int    `hcl:"summary_interval" json:"summary_interval" flagName:"summary-interval" flagDescribe:"Summary generation interval in seconds" default:"30"`
	SummaryModel    string `hcl:"summary_model" json:"summary_model" flagName:"summary-model" flagDescribe:"LLM model for summarization" default:"Qwen3.5-4B-UD-Q4_K_XL"`
	SummaryEndpoint string `hcl:"summary_endpoint" json:"summary_endpoint" flagName:"summary-endpoint" flagDescribe:"LLM API endpoint (Ollama or OpenAI compatible)" default:"http://localhost:43669"`

	// IRC chatroom options
	EnableIRC         bool   `hcl:"enable_irc" json:"enable_irc" flagName:"irc" flagDescribe:"Enable IRC chatroom mode" default:"true"`
	IRCDefaultChannel string `hcl:"irc_default_channel" json:"irc_default_channel" flagName:"irc-channel" flagDescribe:"Default IRC channel" default:"#general"`
	IRCNetworkName    string `hcl:"irc_network_name" json:"irc_network_name" flagName:"irc-network" flagDescribe:"IRC network name" default:"GoTTY Network"`

	// Host display options
	HostName string `hcl:"host_name" json:"host_name" flagName:"host-name" flagDescribe:"Custom host name displayed in sidebar and browser tab (empty to use URL host)" default:""`

	// WebAuthn/Passkeys options
	WebAuthnDisplayName   string `hcl:"webauthn_display_name" json:"webauthn_display_name" flagName:"webauthn-display-name" flagDescribe:"Display name for WebAuthn relying party" default:"GoTTY"`
	WebAuthnDataDir       string `hcl:"webauthn_data_dir" json:"webauthn_data_dir" flagName:"webauthn-data-dir" flagDescribe:"Directory to store WebAuthn credentials" default:"~/.config/gotty/webauthn"`
	WebAuthnRegisterToken string `hcl:"webauthn_register_token" json:"webauthn_register_token" flagName:"webauthn-register-token" flagDescribe:"Token required for new passkey registration (empty = no registration allowed after first credential)" default:""`
	WebAuthnAllowRegister bool   `hcl:"webauthn_allow_register" json:"webauthn_allow_register" flagName:"webauthn-allow-register" flagDescribe:"Allow new passkey registration (only works if no credentials exist or register-token is set)" default:"false"`
	WebAuthnSessionTTL    int    `hcl:"webauthn_session_ttl" json:"webauthn_session_ttl" flagName:"webauthn-session-ttl" flagDescribe:"Session TTL in hours for WebAuthn authentication (0 = require auth every time)" default:"168"`

	// Backend options
	Backend string `hcl:"backend" json:"backend" flagName:"backend" flagDescribe:"Backend type: 'local' for direct command, 'zellij' for persistent sessions" default:"zellij"`

	TitleVariables map[string]interface{}

	// Internal flags for tracking explicit settings
	permitWriteExplicit bool
}

// SetPermitWriteExplicit marks that PermitWrite was explicitly set by user
func (options *Options) SetPermitWriteExplicit() {
	options.permitWriteExplicit = true
}

func (options *Options) Validate() error {
	// WebAuthn is the only auth type, no validation needed
	// Credentials are managed at runtime

	// When auth is enabled, default PermitWrite to true unless explicitly set to false
	if options.EnableAuth && !options.permitWriteExplicit {
		options.PermitWrite = true
	}

	return nil
}

type HtermPrefernces struct {
	AltGrMode                     *string                      `hcl:"alt_gr_mode" json:"alt-gr-mode,omitempty"`
	AltBackspaceIsMetaBackspace   bool                         `hcl:"alt_backspace_is_meta_backspace" json:"alt-backspace-is-meta-backspace,omitempty"`
	AltIsMeta                     bool                         `hcl:"alt_is_meta" json:"alt-is-meta,omitempty"`
	AltSendsWhat                  string                       `hcl:"alt_sends_what" json:"alt-sends-what,omitempty"`
	AudibleBellSound              string                       `hcl:"audible_bell_sound" json:"audible-bell-sound,omitempty"`
	DesktopNotificationBell       bool                         `hcl:"desktop_notification_bell" json:"desktop-notification-bell,omitempty"`
	BackgroundColor               string                       `hcl:"background_color" json:"background-color,omitempty"`
	BackgroundImage               string                       `hcl:"background_image" json:"background-image,omitempty"`
	BackgroundSize                string                       `hcl:"background_size" json:"background-size,omitempty"`
	BackgroundPosition            string                       `hcl:"background_position" json:"background-position,omitempty"`
	BackspaceSendsBackspace       bool                         `hcl:"backspace_sends_backspace" json:"backspace-sends-backspace,omitempty"`
	CharacterMapOverrides         map[string]map[string]string `hcl:"character_map_overrides" json:"character-map-overrides,omitempty"`
	CloseOnExit                   bool                         `hcl:"close_on_exit" json:"close-on-exit,omitempty"`
	CursorBlink                   bool                         `hcl:"cursor_blink" json:"cursor-blink,omitempty"`
	CursorBlinkCycle              [2]int                       `hcl:"cursor_blink_cycle" json:"cursor-blink-cycle,omitempty"`
	CursorColor                   string                       `hcl:"cursor_color" json:"cursor-color,omitempty"`
	ColorPaletteOverrides         []*string                    `hcl:"color_palette_overrides" json:"color-palette-overrides,omitempty"`
	CopyOnSelect                  bool                         `hcl:"copy_on_select" json:"copy-on-select,omitempty"`
	UseDefaultWindowCopy          bool                         `hcl:"use_default_window_copy" json:"use-default-window-copy,omitempty"`
	ClearSelectionAfterCopy       bool                         `hcl:"clear_selection_after_copy" json:"clear-selection-after-copy,omitempty"`
	CtrlPlusMinusZeroZoom         bool                         `hcl:"ctrl_plus_minus_zero_zoom" json:"ctrl-plus-minus-zero-zoom,omitempty"`
	CtrlCCopy                     bool                         `hcl:"ctrl_c_copy" json:"ctrl-c-copy,omitempty"`
	CtrlVPaste                    bool                         `hcl:"ctrl_v_paste" json:"ctrl-v-paste,omitempty"`
	EastAsianAmbiguousAsTwoColumn bool                         `hcl:"east_asian_ambiguous_as_two_column" json:"east-asian-ambiguous-as-two-column,omitempty"`
	Enable8BitControl             *bool                        `hcl:"enable_8_bit_control" json:"enable-8-bit-control,omitempty"`
	EnableBold                    *bool                        `hcl:"enable_bold" json:"enable-bold,omitempty"`
	EnableBoldAsBright            bool                         `hcl:"enable_bold_as_bright" json:"enable-bold-as-bright,omitempty"`
	EnableClipboardNotice         bool                         `hcl:"enable_clipboard_notice" json:"enable-clipboard-notice,omitempty"`
	EnableClipboardWrite          bool                         `hcl:"enable_clipboard_write" json:"enable-clipboard-write,omitempty"`
	EnableDec12                   bool                         `hcl:"enable_dec12" json:"enable-dec12,omitempty"`
	Environment                   map[string]string            `hcl:"environment" json:"environment,omitempty"`
	FontFamily                    string                       `hcl:"font_family" json:"font-family,omitempty"`
	FontSize                      int                          `hcl:"font_size" json:"font-size,omitempty"`
	FontSmoothing                 string                       `hcl:"font_smoothing" json:"font-smoothing,omitempty"`
	ForegroundColor               string                       `hcl:"foreground_color" json:"foreground-color,omitempty"`
	HomeKeysScroll                bool                         `hcl:"home_keys_scroll" json:"home-keys-scroll,omitempty"`
	Keybindings                   map[string]string            `hcl:"keybindings" json:"keybindings,omitempty"`
	MaxStringSequence             int                          `hcl:"max_string_sequence" json:"max-string-sequence,omitempty"`
	MediaKeysAreFkeys             bool                         `hcl:"media_keys_are_fkeys" json:"media-keys-are-fkeys,omitempty"`
	MetaSendsEscape               bool                         `hcl:"meta_sends_escape" json:"meta-sends-escape,omitempty"`
	MousePasteButton              *int                         `hcl:"mouse_paste_button" json:"mouse-paste-button,omitempty"`
	PageKeysScroll                bool                         `hcl:"page_keys_scroll" json:"page-keys-scroll,omitempty"`
	PassAltNumber                 *bool                        `hcl:"pass_alt_number" json:"pass-alt-number,omitempty"`
	PassCtrlNumber                *bool                        `hcl:"pass_ctrl_number" json:"pass-ctrl-number,omitempty"`
	PassMetaNumber                *bool                        `hcl:"pass_meta_number" json:"pass-meta-number,omitempty"`
	PassMetaV                     bool                         `hcl:"pass_meta_v" json:"pass-meta-v,omitempty"`
	ReceiveEncoding               string                       `hcl:"receive_encoding" json:"receive-encoding,omitempty"`
	ScrollOnKeystroke             bool                         `hcl:"scroll_on_keystroke" json:"scroll-on-keystroke,omitempty"`
	ScrollOnOutput                bool                         `hcl:"scroll_on_output" json:"scroll-on-output,omitempty"`
	ScrollbarVisible              bool                         `hcl:"scrollbar_visible" json:"scrollbar-visible,omitempty"`
	ScrollWheelMoveMultiplier     int                          `hcl:"scroll_wheel_move_multiplier" json:"scroll-wheel-move-multiplier,omitempty"`
	SendEncoding                  string                       `hcl:"send_encoding" json:"send-encoding,omitempty"`
	ShiftInsertPaste              bool                         `hcl:"shift_insert_paste" json:"shift-insert-paste,omitempty"`
	UserCss                       string                       `hcl:"user_css" json:"user-css,omitempty"`
}

// GetAuthKeysList returns a formatted list of authorized keys for logging
func (options *Options) GetAuthKeysList() []string {
	if options.EnableAuth {
		return []string{"WebAuthn/Passkeys"}
	}
	return []string{"No authentication"}
}
