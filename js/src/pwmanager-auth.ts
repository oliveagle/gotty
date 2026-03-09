/**
 * Password Manager Authentication Module
 *
 * Supports KeePassXC, Bitwarden, 1Password, LastPass and other password managers
 * through standard HTML form auto-fill detection.
 */

export class PWManagerAuth {
    private container: HTMLElement;
    private form: HTMLFormElement;
    private passwordInput: HTMLInputElement;
    private submitBtn: HTMLButtonElement;
    private errorDiv: HTMLElement;
    private onAuthenticated: (password: string) => void;
    private autoSubmitTimeout: number | null = null;

    constructor(onAuthenticated: (password: string) => void) {
        this.onAuthenticated = onAuthenticated;

        this.container = document.getElementById('pwmanager-auth')!;
        this.form = document.getElementById('pwmanager-form') as HTMLFormElement;
        this.passwordInput = document.getElementById('pwmanager-password') as HTMLInputElement;
        this.submitBtn = document.getElementById('pwmanager-submit') as HTMLButtonElement;
        this.errorDiv = document.getElementById('pwmanager-error')!;

        this.setupListeners();
        this.focusPassword();
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
            if (this.passwordInput.value.length > 0) {
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
        const password = this.passwordInput.value;

        if (!password) {
            this.showError('Please enter a password');
            this.passwordInput.focus();
            return;
        }

        // Disable button during authentication
        this.submitBtn.disabled = true;
        this.submitBtn.textContent = 'Authenticating...';

        // Hide the auth dialog
        this.hide();

        // Call the authentication callback
        this.onAuthenticated(password);
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
        this.submitBtn.textContent = 'Connect';
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
    return authType === 'bitwarden' || authType === 'pwmanager';
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
