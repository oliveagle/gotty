import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebglAddon } from "@xterm/addon-webgl";
import { lib } from "libapps";

export class Xterm {
    elem: HTMLElement;
    term: Terminal;
    fitAddon: FitAddon;
    webglAddon: WebglAddon | null = null;
    resizeObserver: ResizeObserver;
    decoder: lib.UTF8Decoder;

    message: HTMLElement;
    messageTimeout: number;
    messageTimer: ReturnType<typeof setTimeout> | null = null;

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
            },
            // Enable selection features
            allowProposedApi: true,
        });

        this.fitAddon = new FitAddon();
        this.decoder = new lib.UTF8Decoder();

        this.message = elem.ownerDocument.createElement("div");
        this.message.className = "xterm-overlay";
        this.messageTimeout = 2000;

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

        // Use ResizeObserver to detect container size changes (handles sidebar toggle, etc.)
        // Debounce to avoid excessive calls during CSS transitions (200ms matches sidebar transition)
        let resizeTimeout: ReturnType<typeof setTimeout> | null = null;
        this.resizeObserver = new ResizeObserver(() => {
            if (resizeTimeout) {
                clearTimeout(resizeTimeout);
            }
            // Wait for CSS transition to complete before fitting
            resizeTimeout = setTimeout(() => {
                this.fitAddon.fit();
                this.term.scrollToBottom();
                resizeTimeout = null;
            }, 250);
        });
        this.resizeObserver.observe(elem);

        // Fix IME composition view position to follow cursor
        this.setupCompositionViewFix();

        // Setup auto-copy to browser clipboard on selection
        this.setupClipboardOnSelection();
    }

    private setupClipboardOnSelection(): void {
        // Track shift key state
        let shiftPressed = false;

        // Track shift key
        document.addEventListener('keydown', (e: KeyboardEvent) => {
            if (e.key === 'Shift') shiftPressed = true;
        });
        document.addEventListener('keyup', (e: KeyboardEvent) => {
            if (e.key === 'Shift') shiftPressed = false;
        });

        // Auto-copy when Shift is released after selection (mouseup)
        this.elem.addEventListener('mouseup', () => {
            if (shiftPressed) {
                const selection = this.term.getSelection();
                if (selection) {
                    this.copyToClipboard(selection);
                }
            }
        });

        // Also auto-copy on selection change when shift is held
        this.term.onSelectionChange(() => {
            const selection = this.term.getSelection();
            // Auto-copy when there's a selection (works with or without shift)
            if (selection) {
                this.copyToClipboard(selection);
            }
        });
    }

    private copyToClipboard(text: string): void {
        // Try modern Clipboard API first
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(() => {
                this.showMessage("📋 Copied", 1500);
            }).catch((err) => {
                console.error("Clipboard API failed, trying fallback:", err);
                this.copyToClipboardFallback(text);
            });
        } else {
            // Fallback for non-HTTPS
            this.copyToClipboardFallback(text);
        }
    }

    private copyToClipboardFallback(text: string): void {
        // Create a temporary textarea to copy from
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.left = '-9999px';
        textarea.style.top = '-9999px';
        document.body.appendChild(textarea);
        textarea.select();
        textarea.setSelectionRange(0, text.length);
        try {
            document.execCommand('copy');
            this.showMessage("📋 Copied", 1500);
        } catch (err) {
            console.error("Fallback copy failed:", err);
            this.showMessage("Copy failed", 1500);
        }
        document.body.removeChild(textarea);
    }

    private setupCompositionViewFix(): void {
        const forcePosition = () => {
            const compositionView = this.elem.querySelector('.composition-view') as HTMLElement;

            if (!compositionView || !compositionView.classList.contains('active')) return false;

            // Position at bottom of screen
            compositionView.style.position = 'fixed';
            compositionView.style.left = '50%';
            compositionView.style.top = 'auto';
            compositionView.style.bottom = '20px';
            compositionView.style.transform = 'translateX(-50%)';

            return true;
        };

        // Use MutationObserver to watch for style changes
        const observer = new MutationObserver(() => {
            forcePosition();
        });

        const observeComposition = () => {
            const compositionView = this.elem.querySelector('.composition-view') as HTMLElement;
            if (compositionView) {
                observer.observe(compositionView, {
                    attributes: true,
                    attributeFilter: ['style', 'class']
                });
            }
        };

        setTimeout(observeComposition, 200);

        // Update on composition events
        this.elem.addEventListener('compositionstart', () => {
            requestAnimationFrame(forcePosition);
        });
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
        this.term.onData(callback);
    }

    onResize(callback: (columns: number, rows: number) => void): void {
        this.term.onResize((data: { cols: number; rows: number }) => callback(data.cols, data.rows));
    }

    deactivate(): void {
        this.term.blur();
    }

    reset(): void {
        this.removeMessage();
        this.term.clear();
    }

    close(): void {
        this.resizeObserver.disconnect();
        this.webglAddon?.dispose();
        this.term.dispose();
    }
}
