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
            // Explicitly enable selection
            screenReaderMode: false,
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

        // Resize handling: disconnect observer during transition, reconnect after
        // This prevents multiple intermediate fit() calls during CSS animation
        let lastWidth = Math.round(elem.clientWidth);
        let lastHeight = Math.round(elem.clientHeight);
        let resizeTimer: ReturnType<typeof setTimeout> | null = null;

        const doFit = () => {
            const currentWidth = Math.round(elem.clientWidth);
            const currentHeight = Math.round(elem.clientHeight);

            // Only fit if size actually changed
            if (currentWidth !== lastWidth || currentHeight !== lastHeight) {
                lastWidth = currentWidth;
                lastHeight = currentHeight;
                this.fitAddon.fit();
                this.term.scrollToBottom();
            }
        };

        this.resizeObserver = new ResizeObserver(() => {
            // When resize detected, disconnect immediately to prevent cascading events
            this.resizeObserver.disconnect();

            // Clear any pending timer
            if (resizeTimer) {
                clearTimeout(resizeTimer);
            }

            // Wait for CSS transition (200ms) + buffer, then fit once and reconnect
            resizeTimer = setTimeout(() => {
                resizeTimer = null;
                doFit();
                // Reconnect observer for future resize events
                this.resizeObserver.observe(elem);
            }, 280);
        });

        this.resizeObserver.observe(elem);

        // Fix IME composition view position to follow cursor
        this.setupCompositionViewFix();

        // Setup auto-copy to browser clipboard on selection
        this.setupClipboardOnSelection();
    }

    private setupClipboardOnSelection(): void {
        console.log("[gotty] Setting up clipboard sync (server -> browser)...");

        // Find xterm screen element
        const xtermScreen = this.elem.querySelector('.xterm-screen') as HTMLElement;
        console.log("[gotty] xterm-screen element found:", !!xtermScreen);

        // On mouseup, wait for zellij to copy to system clipboard, then sync to browser
        xtermScreen?.addEventListener('mouseup', (e) => {
            console.log("[gotty] mouseup event triggered", e);
            // Wait for zellij to complete its copy operation
            setTimeout(() => {
                console.log("[gotty] calling syncServerClipboardToBrowser after timeout");
                this.syncServerClipboardToBrowser();
            }, 200);
        }, true);

        // Also sync on double-click
        xtermScreen?.addEventListener('dblclick', (e) => {
            console.log("[gotty] dblclick event triggered", e);
            setTimeout(() => {
                this.syncServerClipboardToBrowser();
            }, 200);
        }, true);

        // Keyboard: Ctrl+V to paste from browser clipboard
        this.term.attachCustomKeyEventHandler((event: KeyboardEvent) => {
            if (event.ctrlKey && !event.metaKey && !event.altKey && event.key.toLowerCase() === 'v') {
                console.log("[gotty] Ctrl+V detected, pasting...");
                this.pasteFromClipboard();
                return false;
            }
            return true;
        });
    }

    private async syncServerClipboardToBrowser(): Promise<void> {
        console.log("[gotty] syncServerClipboardToBrowser called");
        try {
            // Fetch clipboard content from server
            console.log("[gotty] fetching /api/clipboard...");
            const response = await fetch('/api/clipboard');
            console.log("[gotty] response status:", response.status);
            if (!response.ok) {
                console.log("[gotty] response not ok, skipping");
                return;
            }

            const data = await response.json();
            console.log("[gotty] clipboard data:", data);
            const text = data.content || '';

            if (text && text.trim()) {
                console.log("[gotty] text found, length:", text.length);
                // Copy to browser clipboard
                if (navigator.clipboard && navigator.clipboard.writeText) {
                    console.log("[gotty] using navigator.clipboard.writeText");
                    await navigator.clipboard.writeText(text);
                    console.log("[gotty] Synced from server:", text.substring(0, 30));
                    this.showMessage("📋 Copied", 1500);
                } else {
                    console.log("[gotty] using fallback copy");
                    this.copyToClipboardFallback(text);
                    this.showMessage("📋 Copied", 1500);
                }
            } else {
                console.log("[gotty] no text in clipboard or empty");
            }
        } catch (error) {
            console.error("[gotty] syncServerClipboardToBrowser error:", error);
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
