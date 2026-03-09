/**
 * WebAuthn/Passkeys Authentication Module
 *
 * Supports KeePassXC Passkeys and other WebAuthn authenticators
 */

export class WebAuthnAuth {
    private container: HTMLElement;
    private statusDiv: HTMLElement;
    private registerBtn: HTMLButtonElement;
    private loginBtn: HTMLButtonElement;
    private errorDiv: HTMLElement;
    private onAuthenticated: (authToken: string) => void;
    private hasAuth: boolean = false;
    private canRegister: boolean = false;
    private requiresToken: boolean = false;
    private isRegistering: boolean = false;
    private isLoggingIn: boolean = false;

    constructor(onAuthenticated: (authToken: string) => void) {
        this.onAuthenticated = onAuthenticated;

        this.container = document.getElementById('webauthn-auth')!;
        this.statusDiv = document.getElementById('webauthn-status')!;
        this.registerBtn = document.getElementById('webauthn-register') as HTMLButtonElement;
        this.loginBtn = document.getElementById('webauthn-login') as HTMLButtonElement;
        this.errorDiv = document.getElementById('webauthn-error')!;

        // Get WebAuthn status from server
        this.hasAuth = (window as any).gotty_webauthn_has_auth || false;

        this.setupListeners();
        this.updateUI();
    }

    private setupListeners(): void {
        this.registerBtn.addEventListener('click', () => this.register());
        this.loginBtn.addEventListener('click', () => this.login());
    }

    private updateUI(): void {
        if (this.hasAuth) {
            this.statusDiv.textContent = 'Passkey registered. Click to authenticate.';
            this.registerBtn.style.display = 'none';
            this.loginBtn.style.display = 'block';
        } else if (this.canRegister) {
            if (this.requiresToken) {
                this.statusDiv.textContent = 'Passkey registered. Registration requires token.';
                this.registerBtn.style.display = 'block';
                this.registerBtn.textContent = 'Register with Token';
            } else {
                this.statusDiv.textContent = 'No Passkey registered. Click to register.';
                this.registerBtn.style.display = 'block';
                this.registerBtn.textContent = 'Register Passkey';
            }
            this.loginBtn.style.display = 'none';
        } else {
            // Registration disabled
            this.statusDiv.textContent = 'Passkey already registered. Registration disabled.';
            this.registerBtn.style.display = 'none';
            this.loginBtn.style.display = 'block';
        }
    }

    private async register(): Promise<void> {
        // Prevent duplicate requests
        if (this.isRegistering) {
            console.log('[WebAuthn] Registration already in progress');
            return;
        }
        this.isRegistering = true;

        this.clearError();
        this.registerBtn.disabled = true;
        this.registerBtn.textContent = 'Registering...';

        try {
            // If token might be required, ask user
            let token = '';
            if (this.requiresToken || this.hasAuth) {
                token = prompt('Enter registration token (or leave empty if not required):') || '';
            }

            // Step 1: Begin registration
            const beginUrl = token
                ? `./api/webauthn/register/begin?token=${encodeURIComponent(token)}`
                : './api/webauthn/register/begin';
            const beginResp = await fetch(beginUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });

            if (!beginResp.ok) {
                const errorData = await beginResp.json().catch(() => ({ message: 'Failed to begin registration' }));
                throw new Error(errorData.message || 'Failed to begin registration');
            }

            const beginData = await beginResp.json();
            const sessionId = beginData.session_id;
            const options = beginData.options;

            // Step 2: Create credentials using WebAuthn API
            // Convert base64 strings to Uint8Array
            const credentialOptions: CredentialCreationOptions = {
                publicKey: {
                    ...options.publicKey,
                    challenge: this.base64ToUint8Array(options.publicKey.challenge),
                    user: {
                        ...options.publicKey.user,
                        id: this.base64ToUint8Array(options.publicKey.user.id)
                    },
                    excludeCredentials: options.publicKey.excludeCredentials?.map((cred: any) => ({
                        ...cred,
                        id: this.base64ToUint8Array(cred.id)
                    }))
                }
            };

            const credential = await navigator.credentials.create(credentialOptions) as PublicKeyCredential;
            if (!credential) {
                throw new Error('Failed to create credential');
            }

            // Step 3: Finish registration
            const finishData = {
                session_id: sessionId,
                response: {
                    id: credential.id,
                    rawId: this.uint8ArrayToBase64(new Uint8Array(credential.rawId)),
                    type: credential.type,
                    response: {
                        clientDataJSON: this.uint8ArrayToBase64(new Uint8Array((credential.response as AuthenticatorAttestationResponse).clientDataJSON)),
                        attestationObject: this.uint8ArrayToBase64(new Uint8Array((credential.response as AuthenticatorAttestationResponse).attestationObject))
                    }
                }
            };

            const finishResp = await fetch('./api/webauthn/register/finish', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(finishData)
            });

            if (!finishResp.ok) {
                const errorData = await finishResp.json().catch(() => ({ message: 'Registration failed' }));
                throw new Error(errorData.message || 'Failed to complete registration');
            }

            // Registration successful
            this.hasAuth = true;
            this.updateUI();
            this.statusDiv.textContent = 'Passkey registered successfully! Click to authenticate.';

        } catch (error) {
            console.error('Registration error:', error);
            this.showError(error instanceof Error ? error.message : 'Registration failed');
            this.registerBtn.disabled = false;
            this.registerBtn.textContent = 'Register Passkey';
        } finally {
            this.isRegistering = false;
        }
    }

    private async login(): Promise<void> {
        // Prevent duplicate requests
        if (this.isLoggingIn) {
            console.log('[WebAuthn] Login already in progress');
            return;
        }
        this.isLoggingIn = true;

        this.clearError();
        this.loginBtn.disabled = true;
        this.loginBtn.textContent = 'Authenticating...';

        try {
            // Step 1: Begin login
            const beginResp = await fetch('./api/webauthn/login/begin', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });

            if (!beginResp.ok) {
                throw new Error('Failed to begin login');
            }

            const beginData = await beginResp.json();
            const sessionId = beginData.session_id;
            const options = beginData.options;

            // Step 2: Get credentials using WebAuthn API
            const assertionOptions: CredentialRequestOptions = {
                publicKey: {
                    ...options.publicKey,
                    challenge: this.base64ToUint8Array(options.publicKey.challenge),
                    allowCredentials: options.publicKey.allowCredentials?.map((cred: any) => ({
                        ...cred,
                        id: this.base64ToUint8Array(cred.id)
                    }))
                }
            };

            const assertion = await navigator.credentials.get(assertionOptions) as PublicKeyCredential;
            if (!assertion) {
                throw new Error('Failed to get credential');
            }

            // Step 3: Finish login
            const finishData = {
                session_id: sessionId,
                response: {
                    id: assertion.id,
                    rawId: this.uint8ArrayToBase64(new Uint8Array(assertion.rawId)),
                    type: assertion.type,
                    response: {
                        clientDataJSON: this.uint8ArrayToBase64(new Uint8Array((assertion.response as AuthenticatorAssertionResponse).clientDataJSON)),
                        authenticatorData: this.uint8ArrayToBase64(new Uint8Array((assertion.response as AuthenticatorAssertionResponse).authenticatorData)),
                        signature: this.uint8ArrayToBase64(new Uint8Array((assertion.response as AuthenticatorAssertionResponse).signature)),
                        userHandle: (assertion.response as AuthenticatorAssertionResponse).userHandle
                            ? this.uint8ArrayToBase64(new Uint8Array((assertion.response as AuthenticatorAssertionResponse).userHandle!))
                            : null
                    }
                }
            };

            const finishResp = await fetch('./api/webauthn/login/finish', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(finishData)
            });

            if (!finishResp.ok) {
                const errorData = await finishResp.json().catch(() => ({ message: 'Authentication failed' }));
                throw new Error(errorData.message || 'Authentication failed');
            }

            const result = await finishResp.json();
            this.hide();

            // Call the authentication callback with the auth token
            this.onAuthenticated(result.auth_token);

        } catch (error) {
            console.error('Login error:', error);
            this.showError(error instanceof Error ? error.message : 'Authentication failed');
            this.loginBtn.disabled = false;
            this.loginBtn.textContent = 'Authenticate';
        } finally {
            this.isLoggingIn = false;
        }
    }

    /**
     * Convert base64url string to Uint8Array
     * WebAuthn uses base64url encoding (RFC 4648), not standard base64
     */
    private base64ToUint8Array(base64url: string): Uint8Array {
        // Convert base64url to standard base64
        // 1. Replace - with + and _ with /
        // 2. Add padding if needed
        let base64 = base64url
            .replace(/-/g, '+')
            .replace(/_/g, '/');

        // Add padding
        const padding = base64.length % 4;
        if (padding) {
            base64 += '='.repeat(4 - padding);
        }

        const binaryString = atob(base64);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes;
    }

    /**
     * Convert Uint8Array to base64url string (without padding)
     * WebAuthn expects base64url encoding (RFC 4648)
     */
    private uint8ArrayToBase64(bytes: Uint8Array): string {
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        const base64 = btoa(binary);
        // Convert to base64url: replace + with -, / with _, and remove padding
        return base64
            .replace(/\+/g, '-')
            .replace(/\//g, '_')
            .replace(/=+$/, '');
    }

    private hide(): void {
        this.container.classList.add('hidden');
        setTimeout(() => {
            this.container.style.display = 'none';
        }, 300);
    }

    private showError(message: string): void {
        this.errorDiv.textContent = message;
        this.errorDiv.classList.add('visible');
    }

    private clearError(): void {
        this.errorDiv.textContent = '';
        this.errorDiv.classList.remove('visible');
    }

    /**
     * Show the authentication dialog
     */
    public async show(): Promise<void> {
        this.container.style.display = 'flex';
        this.container.classList.remove('hidden');
        this.clearError();
        this.registerBtn.disabled = false;
        this.loginBtn.disabled = false;

        // Fetch current status from server
        try {
            const resp = await fetch('./api/webauthn/status');
            const status = await resp.json();
            this.hasAuth = status.has_auth;
            this.canRegister = status.can_register;
            this.requiresToken = status.requires_token;
        } catch (e) {
            // Fallback to defaults
            this.hasAuth = (window as any).gotty_webauthn_has_auth || false;
            this.canRegister = !this.hasAuth;
            this.requiresToken = false;
        }

        this.updateUI();
        this.loginBtn.textContent = 'Authenticate';
    }

    /**
     * Check if authentication is required
     */
    public isVisible(): boolean {
        return !this.container.classList.contains('hidden');
    }
}

/**
 * Check if WebAuthn auth is needed
 */
export function isWebAuthnAuthRequired(): boolean {
    const authType = (window as any).gotty_auth_type;
    return authType === 'webauthn';
}

/**
 * Initialize WebAuthn authentication
 */
export async function initWebAuthnAuth(onAuthenticated: (authToken: string) => void): Promise<WebAuthnAuth | null> {
    const container = document.getElementById('webauthn-auth');
    if (!container) {
        return null;
    }

    if (!isWebAuthnAuthRequired()) {
        container.style.display = 'none';
        return null;
    }

    // Check if there's a cached token in localStorage
    const cachedToken = localStorage.getItem('gotty_auth_token');
    if (cachedToken) {
        // Validate the token with server
        try {
            const resp = await fetch(`./api/webauthn/validate?token=${encodeURIComponent(cachedToken)}`);
            const result = await resp.json();
            if (result.valid) {
                console.log('[WebAuthn] Using cached auth token');
                onAuthenticated(cachedToken);
                container.style.display = 'none';
                return null;
            } else {
                console.log('[WebAuthn] Cached token invalid or expired');
                localStorage.removeItem('gotty_auth_token');
            }
        } catch (e) {
            console.log('[WebAuthn] Failed to validate cached token:', e);
            localStorage.removeItem('gotty_auth_token');
        }
    }

    // No valid cached token, show WebAuthn dialog
    const auth = new WebAuthnAuth((authToken: string) => {
        // Cache the token in localStorage
        localStorage.setItem('gotty_auth_token', authToken);
        onAuthenticated(authToken);
    });
    return auth;
}
