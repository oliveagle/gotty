import { Hterm } from "./hterm";
import { Xterm } from "./xterm";
import { Terminal, WebTTY, protocols } from "./webtty";
import { ConnectionFactory } from "./websocket";
import { WebAuthnAuth, isWebAuthnAuthRequired, initWebAuthnAuth } from "./webauthn-auth";

// Export classes to global scope for use in inline scripts
(window as any).Hterm = Hterm;
(window as any).Xterm = Xterm;
(window as any).WebTTY = WebTTY;
(window as any).ConnectionFactory = ConnectionFactory;
(window as any).protocols = protocols;
(window as any).WebAuthnAuth = WebAuthnAuth;
(window as any).isWebAuthnAuthRequired = isWebAuthnAuthRequired;

// Global variables from server
declare var gotty_auth_token: string;
declare var gotty_auth_type: string;
declare var gotty_term: string;

// Initialize the terminal
function startTerminal(authToken: string) {
    const elem = document.getElementById("terminal");
    if (elem === null) return;

    let term: Terminal;
    if (gotty_term === "hterm") {
        term = new Hterm(elem);
    } else {
        term = new Xterm(elem);
    }
    const httpsEnabled = window.location.protocol === "https:";
    const url = (httpsEnabled ? 'wss://' : 'ws://') + window.location.host + window.location.pathname + 'ws';
    const args = window.location.search;
    const factory = new ConnectionFactory(url, protocols);
    const wt = new WebTTY(term, factory, args, authToken);
    const closer = wt.open();

    window.addEventListener("unload", () => {
        closer();
        term.close();
    });
}

// Main initialization
const elem = document.getElementById("terminal");

// Check if session management is enabled (look for session-list element)
const sessionListElem = document.getElementById("session-list");
const sessionMode = sessionListElem !== null;

if (elem !== null) {
    if (sessionMode) {
        // Session management mode - SessionManager in index.html handles everything
        const authType = (typeof gotty_auth_type !== "undefined") ? gotty_auth_type : "none";

        if (authType === "webauthn") {
            // Initialize WebAuthn for session mode and show the dialog
            initWebAuthnAuth((authToken: string) => {
                (window as any).gotty_auth_token = authToken;
                console.log("WebAuthn authentication completed");
            }).then((webauthnAuth) => {
                if (webauthnAuth) {
                    webauthnAuth.show();
                }
            });
        } else {
            // Hide auth UI if not needed
            const authContainer = document.getElementById("webauthn-auth");
            if (authContainer) {
                authContainer.style.display = "none";
            }
        }
        console.log("Session management mode enabled");
    } else {
        // Legacy mode - auto-start terminal
        const authType = (typeof gotty_auth_type !== "undefined") ? gotty_auth_type : "none";

        if (authType === "webauthn") {
            // WebAuthn authentication - show dialog and start terminal after auth
            initWebAuthnAuth((authToken: string) => {
                startTerminal(authToken);
            }).then((webauthnAuth) => {
                if (webauthnAuth) {
                    webauthnAuth.show();
                }
            });
        } else {
            // No auth required
            startTerminal("");

            // Hide auth UI
            const authContainer = document.getElementById("webauthn-auth");
            if (authContainer) {
                authContainer.style.display = "none";
            }
        }
    }
}
