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
        console.log("[gotty] Setting up clipboard on selection...");

        // Track last known selection before it gets cleared
        let lastSelection = '';
        let pollInterval: ReturnType<typeof setInterval> | null = null;

        // On mousedown, start polling for selection
        this.elem.addEventListener('mousedown', (e: MouseEvent) => {
            // Only start on left click
            if (e.button !== 0) return;
            console.log("[gotty] mousedown detected, starting poll...");
            lastSelection = '';

            // Poll for selection changes while mouse is down
            pollInterval = setInterval(() => {
                // Try xterm's selection API
                const xtermSelection = this.term.getSelection();
                const xtermHasSelection = this.term.hasSelection();

                // Try browser's native selection API
                const browserSelection = window.getSelection();
                const browserText = browserSelection ? browserSelection.toString() : '';

                console.log("[gotty] Poll - xterm:", xtermHasSelection, `"${xtermSelection?.substring(0, 20)}"`,
                           "| browser:", `"${browserText.substring(0, 20)}"`);

                // Use whichever has content
                const text = xtermSelection || browserText;
                if (text && text.trim()) {
                    lastSelection = text;
                }
            }, 100);
        });

        // On mouseup, stop polling and copy
        this.elem.addEventListener('mouseup', () => {
            console.log("[gotty] mouseup detected");
            if (pollInterval) {
                clearInterval(pollInterval);
                pollInterval = null;
            }

            // Final check with both APIs
            const xtermSelection = this.term.getSelection();
            const browserSelection = window.getSelection();
            const browserText = browserSelection ? browserSelection.toString() : '';

            console.log("[gotty] Final - xterm:", this.term.hasSelection(), `"${xtermSelection?.substring(0, 20)}"`);
            console.log("[gotty] Final - browser:", `"${browserText.substring(0, 20)}"`);
            console.log("[gotty] lastSelection:", lastSelection ? `"${lastSelection.substring(0, 30)}..."` : "empty");

            const textToCopy = xtermSelection || browserText || lastSelection;
            if (textToCopy && textToCopy.trim()) {
                console.log("[gotty] Copying to clipboard:", textToCopy.substring(0, 30));
                this.copyToClipboardSilent(textToCopy);
            }
        });

        // Keyboard copy/paste with Ctrl key
        this.term.attachCustomKeyEventHandler((event: KeyboardEvent) => {
            // Ctrl+C - Copy to browser clipboard (when there's a selection)
            if (event.ctrlKey && !event.metaKey && !event.altKey && event.key.toLowerCase() === 'c') {
                console.log("[gotty] Ctrl+C detected");
                const xtermSelection = this.term.getSelection();
                const browserSelection = window.getSelection();
                const browserText = browserSelection ? browserSelection.toString() : '';
                const selection = xtermSelection || browserText || lastSelection;
                if (selection) {
                    this.copyToClipboard(selection);
                    return false;
                }
            }
            // Ctrl+V - Paste from browser clipboard
            if (event.ctrlKey && !event.metaKey && !event.altKey && event.key.toLowerCase() === 'v') {
                console.log("[gotty] Ctrl+V detected");
                this.pasteFromClipboard();
                return false;
            }
            return true;
        });
    }

    private copyToClipboardSilent(text: string): void {
        if (!text || text.trim() === '') {
            return;
        }

        // Always try execCommand first (works in more contexts)
        const fallbackSuccess = this.copyToClipboardFallback(text);
        if (fallbackSuccess) {
            console.log("[gotty] Copied to clipboard via execCommand:", text.substring(0, 30));
            return;
        }

        // Then try modern Clipboard API
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(() => {
                console.log("[gotty] Copied to clipboard via Clipboard API:", text.substring(0, 30));
            }).catch((err) => {
                console.warn("[gotty] Clipboard copy failed:", err);
            });
        }
    }

    private pasteFromClipboard(): void {
        if (navigator.clipboard && navigator.clipboard.readText) {
            navigator.clipboard.readText().then((text) => {
                this.term.paste(text);
                this.showMessage("📋 Pasted", 1000);
            }).catch((err) => {
                console.error("Failed to paste:", err);
                this.showMessage("Paste failed (check permissions)", 2000);
            });
        } else {
            this.showMessage("Paste not supported in this browser", 2000);
        }
    }

    private copyToClipboard(text: string): void {
        // Don't copy empty text
        if (!text || text.trim() === '') {
            return;
        }

        // Track if we've already shown a message to avoid duplicates
        let messageShown = false;

        const showSuccess = () => {
            if (!messageShown) {
                messageShown = true;
                const preview = text.length > 20 ? text.substring(0, 20) + '...' : text;
                this.showMessage(`📋 Copied: ${preview}`, 2000);
            }
        };

        const showError = (err: any) => {
            console.error("Copy failed:", err);
            if (!messageShown) {
                messageShown = true;
                this.showMessage("❌ Copy failed - try Ctrl+Shift+C", 3000);
            }
        };

        // Try modern Clipboard API first (requires HTTPS or localhost)
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(() => {
                showSuccess();
            }).catch((err) => {
                console.warn("Clipboard API failed:", err);
                // Try fallback
                if (this.copyToClipboardFallback(text)) {
                    showSuccess();
                } else {
                    showError(err);
                }
            });
        } else {
            // Fallback for non-HTTPS or older browsers
            if (this.copyToClipboardFallback(text)) {
                showSuccess();
            } else {
                showError(new Error("Clipboard API not available"));
            }
        }
    }

    private copyToClipboardFallback(text: string): boolean {
        // Create a temporary textarea to copy from
        const textarea = document.createElement('textarea');
        textarea.value = text;
        // Make it invisible but still functional
        textarea.style.position = 'fixed';
        textarea.style.left = '0';
        textarea.style.top = '0';
        textarea.style.opacity = '0';
        textarea.style.pointerEvents = 'none';
        textarea.style.width = '2em';
        textarea.style.height = '2em';
        document.body.appendChild(textarea);

        let success = false;
        try {
            textarea.focus();
            textarea.select();
            textarea.setSelectionRange(0, text.length);
            success = document.execCommand('copy');
        } catch (err) {
            console.error("Fallback copy failed:", err);
            success = false;
        }
        document.body.removeChild(textarea);
        return success;
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
