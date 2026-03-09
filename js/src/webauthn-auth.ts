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
        } else {
            this.statusDiv.textContent = 'No Passkey registered. Click to register.';
            this.registerBtn.style.display = 'block';
            this.loginBtn.style.display = 'none';
        }
    }

    private async register(): Promise<void> {
        this.clearError();
        this.registerBtn.disabled = true;
        this.registerBtn.textContent = 'Registering...';

        try {
            // Step 1: Begin registration
            const beginResp = await fetch('./api/webauthn/register/begin', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });

            if (!beginResp.ok) {
                throw new Error('Failed to begin registration');
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
                const errorData = await finishResp.json();
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
        }
    }

    private async login(): Promise<void> {
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
                const errorData = await finishResp.json();
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
        }
    }

    private base64ToUint8Array(base64: string): Uint8Array {
        const binaryString = atob(base64);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes;
    }

    private uint8ArrayToBase64(bytes: Uint8Array): string {
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return btoa(binary);
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
    public show(): void {
        this.container.style.display = 'flex';
        this.container.classList.remove('hidden');
        this.clearError();
        this.registerBtn.disabled = false;
        this.loginBtn.disabled = false;
        this.registerBtn.textContent = 'Register Passkey';
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
export function initWebAuthnAuth(onAuthenticated: (authToken: string) => void): WebAuthnAuth | null {
    const container = document.getElementById('webauthn-auth');
    if (!container) {
        return null;
    }

    if (!isWebAuthnAuthRequired()) {
        container.style.display = 'none';
        return null;
    }

    return new WebAuthnAuth(onAuthenticated);
}
