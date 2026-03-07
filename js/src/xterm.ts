import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebglAddon } from "@xterm/addon-webgl";
import { lib } from "libapps";

export class Xterm {
    elem: HTMLElement;
    term: Terminal;
    fitAddon: FitAddon;
    webglAddon: WebglAddon | null = null;
    resizeListener: () => void;
    decoder: lib.UTF8Decoder;

    message: HTMLElement;
    messageTimeout: number;
    messageTimer: ReturnType<typeof setTimeout> | null = null;

    // IME handling
    imeInput: HTMLInputElement | null = null;
    isComposing: boolean = false;
    dataCallback: ((data: string) => void) | null = null;

    constructor(elem: HTMLElement) {
        this.elem = elem;
        this.term = new Terminal({
            allowTransparency: true,
            cursorBlink: true,
            fontFamily: '"DejaVu Sans Mono", "Everson Mono", FreeMono, Menlo, monospace',
            fontSize: 14,
            lineHeight: 1.2,
            theme: {
                background: '#000000',
            }
        });

        this.fitAddon = new FitAddon();
        this.decoder = new lib.UTF8Decoder();

        this.message = elem.ownerDocument.createElement("div");
        this.message.className = "xterm-overlay";
        this.messageTimeout = 2000;

        this.resizeListener = () => {
            this.fitAddon.fit();
            this.term.scrollToBottom();
            this.showMessage(`${this.term.cols}x${this.term.rows}`, this.messageTimeout);
        };

        // Open terminal FIRST (xterm v6 requirement)
        this.term.open(elem);

        // Load addons AFTER open
        this.term.loadAddon(this.fitAddon);

        // Try WebGL renderer for better performance
        try {
            this.webglAddon = new WebglAddon();
            this.webglAddon.onContextLoss(() => {
                this.webglAddon?.dispose();
                this.webglAddon = null;
            });
            this.term.loadAddon(this.webglAddon);
        } catch (e) {
            console.log("WebGL not available, using canvas:", e);
            this.webglAddon = null;
        }

        // Fit after everything is loaded
        this.fitAddon.fit();
        window.addEventListener("resize", this.resizeListener);

        // Create IME input overlay
        this.createImeOverlay();
    }

    private createImeOverlay(): void {
        // Create a transparent input that covers the terminal
        this.imeInput = document.createElement('input');
        this.imeInput.type = 'text';
        this.imeInput.className = 'xterm-ime-overlay';
        this.imeInput.style.cssText = `
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            opacity: 0.01;
            background: transparent;
            border: none;
            outline: none;
            color: transparent;
            caret-color: transparent;
            font-size: 16px;
            z-index: 100;
            cursor: text;
        `;
        console.log('[Xterm] Created IME overlay');

        // Handle composition events
        this.imeInput.addEventListener('compositionstart', () => {
            this.isComposing = true;
        });

        this.imeInput.addEventListener('compositionend', () => {
            this.isComposing = false;
            if (this.imeInput && this.imeInput.value && this.dataCallback) {
                this.dataCallback(this.imeInput.value);
                this.imeInput.value = '';
            }
        });

        // Handle regular input (non-IME)
        this.imeInput.addEventListener('input', () => {
            if (!this.isComposing && this.imeInput && this.dataCallback) {
                const value = this.imeInput.value;
                if (value) {
                    this.dataCallback(value);
                    this.imeInput.value = '';
                }
            }
        });

        // Forward keyboard events to xterm
        this.imeInput.addEventListener('keydown', (e) => {
            if (this.isComposing) return;

            // Special keys - forward to xterm
            const specialKeys = ['Enter', 'Backspace', 'Delete', 'Tab', 'Escape',
                                  'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight',
                                  'Home', 'End', 'PageUp', 'PageDown'];

            if (specialKeys.includes(e.key)) {
                // Let these go through to the terminal
                e.stopPropagation();
            }
        });

        // Click to focus
        this.elem.addEventListener('click', () => {
            if (this.imeInput) {
                this.imeInput.focus();
            }
        });

        this.elem.style.position = 'relative';
        this.elem.appendChild(this.imeInput);

        // Focus the input
        setTimeout(() => {
            if (this.imeInput) {
                this.imeInput.focus();
            }
        }, 100);
    }

    info(): { columns: number; rows: number } {
        return { columns: this.term.cols, rows: this.term.rows };
    }

    output(data: string): void {
        this.term.write(this.decoder.decode(data));
    }

    showMessage(message: string, timeout: number): void {
        this.message.textContent = message;
        this.elem.appendChild(this.message);

        if (this.messageTimer) {
            clearTimeout(this.messageTimer);
        }
        if (timeout > 0) {
            this.messageTimer = setTimeout(() => {
                if (this.message.parentNode === this.elem) {
                    this.elem.removeChild(this.message);
                }
            }, timeout);
        }
    }

    removeMessage(): void {
        if (this.message.parentNode === this.elem) {
            this.elem.removeChild(this.message);
        }
    }

    setWindowTitle(title: string): void {
        document.title = title;
    }

    setSubtitle(subtitle: string): void {
        // Update session object if sessionManager exists
        if ((window as any).sessionManager) {
            const sm = (window as any).sessionManager;
            if (sm.activeSessionId) {
                const session = sm.sessions.find((s: any) => s.id === sm.activeSessionId);
                if (session) {
                    session.subtitle = subtitle;
                }
            }
        }

        // Update or create subtitle element in the active session item
        const activeItem = document.querySelector('.session-item.active .session-info');
        if (activeItem) {
            let subtitleElem = activeItem.querySelector('.session-subtitle');
            if (!subtitleElem) {
                // Create subtitle element if it doesn't exist
                subtitleElem = document.createElement('div');
                subtitleElem.className = 'session-subtitle';
                const timeElem = activeItem.querySelector('.session-time');
                if (timeElem) {
                    activeItem.insertBefore(subtitleElem, timeElem);
                } else {
                    activeItem.appendChild(subtitleElem);
                }
            }
            subtitleElem.textContent = subtitle;
        }
    }

    setPreferences(_value: object): void {
        // Apply preferences if needed
    }

    onInput(callback: (input: string) => void): void {
        this.dataCallback = callback;
        // Also register with xterm for special keys that bypass our input
        this.term.onData(callback);
    }

    onResize(callback: (columns: number, rows: number) => void): void {
        this.term.onResize((data: { cols: number; rows: number }) => callback(data.cols, data.rows));
    }

    deactivate(): void {
        this.term.blur();
        if (this.imeInput) {
            this.imeInput.blur();
        }
    }

    reset(): void {
        this.removeMessage();
        this.term.clear();
    }

    close(): void {
        window.removeEventListener("resize", this.resizeListener);
        if (this.imeInput && this.imeInput.parentNode) {
            this.imeInput.parentNode.removeChild(this.imeInput);
        }
        this.webglAddon?.dispose();
        this.term.dispose();
    }
}
