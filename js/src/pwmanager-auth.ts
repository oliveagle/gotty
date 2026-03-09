/**
 * Password Manager Authentication Module
 *
 * Supports KeePassXC, Bitwarden, 1Password, LastPass and other password managers
 * through standard HTML form auto-fill detection.
 *
 * For KeePassXC: Uses Ed25519 challenge-response authentication
 */

export class PWManagerAuth {
    private container: HTMLElement;
    private form: HTMLFormElement;
    private passwordInput: HTMLInputElement;
    private submitBtn: HTMLButtonElement;
    private errorDiv: HTMLElement;
    private onAuthenticated: (password: string) => void;
    private autoSubmitTimeout: number | null = null;
    private authType: string;
    private publicKey: string | null = null;
    private sessionId: string | null = null;
    private challenge: string | null = null;

    constructor(onAuthenticated: (password: string) => void) {
        this.onAuthenticated = onAuthenticated;

        this.container = document.getElementById('pwmanager-auth')!;
        this.form = document.getElementById('pwmanager-form') as HTMLFormElement;
        this.passwordInput = document.getElementById('pwmanager-password') as HTMLInputElement;
        this.submitBtn = document.getElementById('pwmanager-submit') as HTMLButtonElement;
        this.errorDiv = document.getElementById('pwmanager-error')!;

        // Get auth type from server
        this.authType = (window as any).gotty_auth_type || 'none';

        // Get public key for KeePassXC auth
        if (this.authType === 'keepassxc') {
            this.publicKey = (window as any).gotty_public_key || null;
            this.setupKeePassXCAuth();
        }

        this.setupListeners();
        this.focusPassword();
    }

    /**
     * Setup KeePassXC challenge-response authentication
     */
    private async setupKeePassXCAuth(): Promise<void> {
        if (!this.publicKey) {
            this.showError('Server configuration error: public key not found');
            return;
        }

        // Generate session ID
        this.sessionId = this.generateSessionId();

        // Fetch challenge from server
        try {
            const response = await fetch('./api/challenge', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ session_id: this.sessionId })
            });

            if (!response.ok) {
                throw new Error('Failed to get challenge from server');
            }

            const data = await response.json();
            this.challenge = data.challenge;

            // Update UI to show challenge
            this.updateUIForKeePassXC();
        } catch (error) {
            console.error('Failed to setup KeePassXC auth:', error);
            this.showError('Failed to initialize authentication');
        }
    }

    /**
     * Generate a random session ID
     */
    private generateSessionId(): string {
        const array = new Uint8Array(16);
        crypto.getRandomValues(array);
        return Array.from(array, b => b.toString(16).padStart(2, '0')).join('');
    }

    /**
     * Update UI for KeePassXC challenge-response authentication
     */
    private updateUIForKeePassXC(): void {
        const subtitle = document.querySelector('.pwmanager-subtitle');
        if (subtitle) {
            subtitle.innerHTML = `
                <div style="text-align: left; margin-top: 10px;">
                    <p style="font-size: 12px; color: #888; margin-bottom: 8px;">
                        Challenge (sign this with your private key):
                    </p>
                    <code style="display: block; background: #1a1a1f; padding: 8px; border-radius: 4px;
                                font-size: 10px; word-break: break-all; color: #4ade80; margin-bottom: 12px;">
                        ${this.challenge}
                    </code>
                    <p style="font-size: 11px; color: #888;">
                        1. Copy the challenge above<br>
                        2. Sign it with your Ed25519 private key<br>
                        3. Paste the base64-encoded signature below
                    </p>
                </div>
            `;
        }

        // Update password field placeholder
        this.passwordInput.placeholder = 'Paste base64 signature here...';

        // Update button text
        this.submitBtn.textContent = 'Verify Signature';
    }

    private setupListeners(): void {
        // Form submit
        this.form.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleSubmit();
        });

        // Listen for password input changes (auto-fill detection)
        this.passwordInput.addEventListener('input', () => {
            this.clearError();

            // Auto-submit after a short delay when password is filled
            // This helps with password manager auto-fill
            if (this.passwordInput.value.length > 0 && this.authType !== 'keepassxc') {
                this.scheduleAutoSubmit();
            }
        });

        // Cancel auto-submit if user manually focuses
        this.passwordInput.addEventListener('focus', () => {
            this.cancelAutoSubmit();
        });

        // Cancel auto-submit if user types
        this.passwordInput.addEventListener('keydown', (e) => {
            // Don't cancel on Tab or Enter
            if (e.key !== 'Tab' && e.key !== 'Enter') {
                this.cancelAutoSubmit();
            }
        });
    }

    private focusPassword(): void {
        // Focus the password field after a short delay
        // This helps password managers detect the field
        setTimeout(() => {
            this.passwordInput.focus();
        }, 100);
    }

    private scheduleAutoSubmit(): void {
        // Cancel any existing timeout
        this.cancelAutoSubmit();

        // Auto-submit after 500ms of no changes
        // This gives password managers time to fill
        this.autoSubmitTimeout = window.setTimeout(() => {
            if (this.passwordInput.value.length > 0) {
                this.handleSubmit();
            }
        }, 500);
    }

    private cancelAutoSubmit(): void {
        if (this.autoSubmitTimeout !== null) {
            clearTimeout(this.autoSubmitTimeout);
            this.autoSubmitTimeout = null;
        }
    }

    private handleSubmit(): void {
        const inputValue = this.passwordInput.value;

        if (!inputValue) {
            this.showError(this.authType === 'keepassxc'
                ? 'Please enter your signature'
                : 'Please enter a password');
            this.passwordInput.focus();
            return;
        }

        // Disable button during authentication
        this.submitBtn.disabled = true;
        this.submitBtn.textContent = 'Authenticating...';

        // Hide the auth dialog
        this.hide();

        // For KeePassXC, send session_id:signature format
        // For other password managers, send the value directly
        const authToken = this.authType === 'keepassxc' && this.sessionId
            ? `${this.sessionId}:${inputValue}`
            : inputValue;

        // Call the authentication callback
        this.onAuthenticated(authToken);
    }

    private hide(): void {
        this.container.classList.add('hidden');

        // Remove from DOM after transition
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
        this.passwordInput.value = '';
        this.submitBtn.disabled = false;
        this.submitBtn.textContent = this.authType === 'keepassxc' ? 'Verify Signature' : 'Connect';
        this.clearError();
        this.focusPassword();
    }

    /**
     * Check if authentication is required
     * Returns true if auth dialog is visible
     */
    public isVisible(): boolean {
        return !this.container.classList.contains('hidden');
    }
}

/**
 * Check if password manager auth is needed
 * Based on gotty_auth_type from server
 */
export function isPWManagerAuthRequired(): boolean {
    // Check if server has set auth_type
    const authType = (window as any).gotty_auth_type;
    return authType === 'bitwarden' || authType === 'pwmanager' || authType === 'keepassxc';
}

/**
 * Initialize password manager authentication
 * Returns the auth instance or null if not needed
 */
export function initPWManagerAuth(onAuthenticated: (password: string) => void): PWManagerAuth | null {
    // Check if auth UI exists in DOM
    const container = document.getElementById('pwmanager-auth');
    if (!container) {
        return null;
    }

    // Check if auth is required
    if (!isPWManagerAuthRequired()) {
        // Hide auth UI if not needed
        container.style.display = 'none';
        return null;
    }

    // Create and return auth instance
    return new PWManagerAuth(onAuthenticated);
}
