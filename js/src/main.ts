import { Hterm } from "./hterm";
import { Xterm } from "./xterm";
import { Terminal, WebTTY, protocols } from "./webtty";
import { ConnectionFactory } from "./websocket";
import { PWManagerAuth, isPWManagerAuthRequired, initPWManagerAuth } from "./pwmanager-auth";

// Export classes to global scope for use in inline scripts
(window as any).Hterm = Hterm;
(window as any).Xterm = Xterm;
(window as any).WebTTY = WebTTY;
(window as any).ConnectionFactory = ConnectionFactory;
(window as any).protocols = protocols;
(window as any).PWManagerAuth = PWManagerAuth;
(window as any).isPWManagerAuthRequired = isPWManagerAuthRequired;

// Global variables from server
declare var gotty_auth_token: string;
declare var gotty_auth_type: string;
declare var gotty_term: string;

// Create basic auth UI
function createBasicAuthUI(onAuthenticated: (token: string) => void): HTMLElement {
    const container = document.createElement("div");
    container.id = "basic-auth";
    container.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.9);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    `;

    const form = document.createElement("div");
    form.style.cssText = `
        background: #1e1e1e;
        padding: 40px;
        border-radius: 8px;
        text-align: center;
        color: #fff;
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    `;

    const title = document.createElement("h2");
    title.textContent = "Authentication Required";
    title.style.marginBottom = "20px";

    const passwordInput = document.createElement("input");
    passwordInput.type = "password";
    passwordInput.placeholder = "Password";
    passwordInput.style.cssText = `
        width: 300px;
        padding: 12px;
        margin-bottom: 20px;
        border: 1px solid #444;
        border-radius: 4px;
        background: #2d2d2d;
        color: #fff;
        font-size: 14px;
    `;

    const button = document.createElement("button");
    button.textContent = "Connect";
    button.style.cssText = `
        width: 326px;
        padding: 12px;
        background: #175dcf;
        color: white;
        border: none;
        border-radius: 4px;
        font-size: 16px;
        cursor: pointer;
    `;

    button.addEventListener("click", () => {
        onAuthenticated(passwordInput.value);
    });

    form.appendChild(title);
    form.appendChild(passwordInput);
    form.appendChild(button);
    container.appendChild(form);

    return container;
}

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
        // But we still need to handle password manager auth if required
        const authType = (typeof gotty_auth_type !== "undefined") ? gotty_auth_type : "none";

        if (authType === "bitwarden" || authType === "pwmanager") {
            // Initialize password manager auth for session mode
            initPWManagerAuth((password: string) => {
                // Store the password for session manager to use
                (window as any).gotty_auth_token = password;
                console.log("Password manager authentication completed");
            });
        } else {
            // Hide auth UI if not needed
            const authContainer = document.getElementById("pwmanager-auth");
            if (authContainer) {
                authContainer.style.display = "none";
            }
        }
        console.log("Session management mode enabled");
    } else {
        // Legacy mode - auto-start terminal
        const authType = (typeof gotty_auth_type !== "undefined") ? gotty_auth_type : "none";
        const authToken = (typeof gotty_auth_token !== "undefined") ? gotty_auth_token : "";

        if (authType === "bitwarden" || authType === "pwmanager") {
            // Password manager authentication
            initPWManagerAuth((password: string) => {
                startTerminal(password);
            });
        } else if (authType === "basic" && !authToken) {
            const authUI = createBasicAuthUI((token) => {
                const ui = document.getElementById("basic-auth");
                if (ui) ui.remove();
                startTerminal(token);
            });
            document.body.appendChild(authUI);

            // Hide password manager auth UI
            const pwAuthContainer = document.getElementById("pwmanager-auth");
            if (pwAuthContainer) {
                pwAuthContainer.style.display = "none";
            }
        } else {
            // No auth or token already provided
            startTerminal(authToken);

            // Hide password manager auth UI
            const pwAuthContainer = document.getElementById("pwmanager-auth");
            if (pwAuthContainer) {
                pwAuthContainer.style.display = "none";
            }
        }
    }
}
