import { Hterm } from "./hterm";
import { Xterm } from "./xterm";
import { Terminal, WebTTY, protocols } from "./webtty";
import { ConnectionFactory } from "./websocket";
import { initCrypto, deriveKeyFromPassword, isBitwardenExtensionAvailable } from "./bitwarden";

// Export classes to global scope for use in inline scripts
// Use eval('window') to prevent UglifyJS from optimizing this away
// and ensure we get the actual global window object
const globalWindow = eval('window') as any;
globalWindow.Hterm = Hterm;
globalWindow.Xterm = Xterm;
globalWindow.WebTTY = WebTTY;
globalWindow.ConnectionFactory = ConnectionFactory;
globalWindow.protocols = protocols;

// @TODO remove these
declare var gotty_auth_token: string;
declare var gotty_auth_type: string;
declare var gotty_term: string;

// Global state for encryption key
let encryptionKey: string | null = null;

// Create authentication UI for bitwarden
function createBitwardenAuthUI(onAuthenticated: (token: string) => void): HTMLElement {
    const container = document.createElement("div");
    container.id = "bitwarden-auth";
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

    // Check for secure context upfront
    const isSecure = window.isSecureContext || location.protocol === 'https:' ||
                     location.hostname === 'localhost' || location.hostname === '127.0.0.1';

    if (!isSecure) {
        const warningDiv = document.createElement("div");
        warningDiv.style.cssText = `
            background: #2d2d2d;
            padding: 40px;
            border-radius: 8px;
            text-align: center;
            color: #fff;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 500px;
        `;

        const warningTitle = document.createElement("h2");
        warningTitle.textContent = "HTTPS Required";
        warningTitle.style.color = "#ff6b6b";
        warningTitle.style.marginBottom = "20px";

        const warningText = document.createElement("p");
        warningText.innerHTML = `Bitwarden authentication requires a <strong>secure context</strong> (HTTPS or localhost).<br><br>
            Please access gotty via:<br>
            <code style="background:#444;padding:4px 8px;border-radius:4px;">https://your-host:port</code><br><br>
            Or use localhost if running locally.`;
        warningText.style.color = "#aaa";
        warningText.style.lineHeight = "1.6";

        warningDiv.appendChild(warningTitle);
        warningDiv.appendChild(warningText);
        container.appendChild(warningDiv);
        return container;
    }

    const form = document.createElement("div");
    form.style.cssText = `
        background: #1e1e1e;
        padding: 40px;
        border-radius: 8px;
        text-align: center;
        color: #fff;
        font-family: -apple-system, Blink "Segoe UI", Roboto,MacSystemFont, sans-serif;
    `;

    const title = document.createElement("h2");
    title.textContent = "Bitwarden Authentication";
    title.style.marginBottom = "20px";

    const subtitle = document.createElement("p");
    subtitle.textContent = "Enter your Bitwarden master password to decrypt";
    subtitle.style.color = "#aaa";
    subtitle.style.marginBottom = "30px";

    const emailInput = document.createElement("input");
    emailInput.type = "email";
    emailInput.placeholder = " keyEmail (for derivation)";
    emailInput.style.cssText = `
        width: 300px;
        padding: 12px;
        margin-bottom: 15px;
        border: 1px solid #444;
        border-radius: 4px;
        background: #2d2d2d;
        color: #fff;
        font-size: 14px;
    `;

    const passwordInput = document.createElement("input");
    passwordInput.type = "password";
    passwordInput.placeholder = "Master Password";
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
    button.textContent = "Unlock";
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

    const errorMsg = document.createElement("div");
    errorMsg.style.color = "#ff4444";
    errorMsg.style.marginTop = "15px";
    errorMsg.style.display = "none";

    const extStatus = document.createElement("div");
    extStatus.style.cssText = `
        margin-top: 20px;
        font-size: 12px;
        color: #888;
    `;

    if (isBitwardenExtensionAvailable()) {
        extStatus.textContent = "Bitwarden browser extension detected";
        extStatus.style.color = "#4caf50";
    }

    button.addEventListener("click", async () => {
        const email = emailInput.value;
        const password = passwordInput.value;

        if (!email || !password) {
            errorMsg.textContent = "Please enter both email and password";
            errorMsg.style.display = "block";
            return;
        }

        button.textContent = "Deriving key...";
        button.disabled = true;

        try {
            await initCrypto();

            // Derive encryption key from master password
            const derivedKey = await deriveKeyFromPassword(password, email);
            encryptionKey = derivedKey;

            // Use the derived key as the auth token
            onAuthenticated(derivedKey);
        } catch (error) {
            console.error("Authentication failed:", error);
            // Show the actual error message (e.g., HTTPS requirement)
            errorMsg.textContent = error instanceof Error ? error.message : "Failed to derive key. Please check your password.";
            errorMsg.style.display = "block";
            button.textContent = "Unlock";
            button.disabled = false;
        }
    });

    form.appendChild(title);
    form.appendChild(subtitle);
    form.appendChild(emailInput);
    form.appendChild(passwordInput);
    form.appendChild(button);
    form.appendChild(errorMsg);
    form.appendChild(extStatus);
    container.appendChild(form);

    return container;
}

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
    const elem = document.getElementById("terminal")
    if (elem === null) return;

    var term: Terminal;
    if (gotty_term == "hterm") {
        term = new Hterm(elem);
    } else {
        term = new Xterm(elem);
    }
    const httpsEnabled = window.location.protocol == "https:";
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
const elem = document.getElementById("terminal")

if (elem !== null) {
    // Check authentication type
    const authType = (typeof gotty_auth_type !== "undefined") ? gotty_auth_type : "none";
    const authToken = (typeof gotty_auth_token !== "undefined") ? gotty_auth_token : "";

    if (authType === "bitwarden" && !authToken) {
        // Show Bitwarden authentication UI
        const authUI = createBitwardenAuthUI((token) => {
            // Remove auth UI
            const ui = document.getElementById("bitwarden-auth");
            if (ui) ui.remove();
            // Start terminal with the token
            startTerminal(token);
        });
        document.body.appendChild(authUI);
    } else if (authType === "basic" && !authToken) {
        // Show basic authentication UI
        const authUI = createBasicAuthUI((token) => {
            // Remove auth UI
            const ui = document.getElementById("basic-auth");
            if (ui) ui.remove();
            // Start terminal with the token
            startTerminal(token);
        });
        document.body.appendChild(authUI);
    } else {
        // No auth required or token already provided
        startTerminal(authToken);
    }
};
