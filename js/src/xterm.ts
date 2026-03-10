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

    // Resize tracking
    lastWidth: number = 0;
    lastHeight: number = 0;
    fitTimer: ReturnType<typeof setTimeout> | null = null;
    fitDebounceTimer: ReturnType<typeof setTimeout> | null = null;

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

        // Handle OSC 52 clipboard sequences from zellij/tmux
        // OSC 52 format: ESC ] 52 ; <target> ; <base64-data> ST
        this.term.parser.registerOscHandler(52, (data: string) => {
            // data format: "c;base64data" or just "base64data"
            const parts = data.split(';');
            let base64Data = parts.length > 1 ? parts[1] : parts[0];
            if (base64Data === '?') {
                // Query - we don't support this
                return true;
            }
            try {
                // Decode base64 to UTF-8 text
                const binaryString = atob(base64Data);
                const bytes = new Uint8Array(binaryString.length);
                for (let i = 0; i < binaryString.length; i++) {
                    bytes[i] = binaryString.charCodeAt(i);
                }
                const text = new TextDecoder('utf-8').decode(bytes);
                // Copy to browser clipboard
                const success = this.copyToClipboardFallback(text);
                if (success) {
                    this.showMessage("📋 Copied", 1500);
                }
            } catch (e) {
                // Silently ignore decode errors
            }
            return true;
        });

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

        // Force cursor blink after all addons are loaded
        // WebGL may override the initial cursorBlink setting
        this.term.options.cursorBlink = true;

        // Fit after DOM layout is complete
        // Delay slightly to ensure CSS and flexbox layout are settled
        setTimeout(() => {
            this.lastWidth = Math.round(this.elem.clientWidth);
            this.lastHeight = Math.round(this.elem.clientHeight);
            // Resize debug log disabled
            this.fitAddon.fit();
            // Final check: ensure cursor blink is enabled after fit
            this.term.options.cursorBlink = true;
        }, 100);

        // Create ResizeObserver but keep it disconnected
        // We'll only use it for window resize events
        this.resizeObserver = new ResizeObserver(() => {
            // Intentionally empty - we don't auto-react to size changes
            // to avoid infinite loops when fit() changes internal dimensions
        });

        // Listen for window resize instead of element resize
        window.addEventListener('resize', () => {
            this.scheduleFit('window-resize');
        });

        // Fix IME composition view position to follow cursor
        this.setupCompositionViewFix();

        // Setup auto-copy to browser clipboard on selection
        this.setupClipboardOnSelection();
    }

    private setupClipboardOnSelection(): void {
        // Save selection on mouseup (before right-click can clear it)
        let savedSelection: string | null = null;

        this.elem.addEventListener('mouseup', () => {
            const sel = this.term.getSelection();
            if (sel && sel.trim()) {
                savedSelection = sel;
            }
        });

        // Right-click to copy saved selection
        this.elem.addEventListener('contextmenu', (e) => {
            if (savedSelection && savedSelection.trim()) {
                e.preventDefault();
                this.copySelectionToClipboard(savedSelection);
            }
            // If no selection, let default context menu show
        });

        // Keyboard: Ctrl+V to paste from browser clipboard
        this.term.attachCustomKeyEventHandler((event: KeyboardEvent) => {
            if (event.ctrlKey && !event.metaKey && !event.altKey && event.key.toLowerCase() === 'v') {
                this.pasteFromClipboard();
                return false;
            }
            return true;
        });
    }

    private async copySelectionToClipboard(selection: string): Promise<void> {
        // Use fallback method - works on HTTP (navigator.clipboard requires HTTPS)
        const success = this.copyToClipboardFallback(selection);
        if (success) {
            this.showMessage("📋 Copied", 1500);
        } else {
            this.showMessage("Copy failed", 1500);
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
        this.term.onResize((data: { cols: number; rows: number }) => {
            // Resize debug log disabled
            callback(data.cols, data.rows);
        });
    }

    deactivate(): void {
        this.term.blur();
    }

    reset(): void {
        this.removeMessage();
        this.term.clear();
    }

    /**
     * Schedule a fit() call after a delay.
     * Multiple calls within the delay period will be coalesced into one.
     */
    scheduleFit(reason: string, delay: number = 250): void {
        if (this.fitTimer) {
            clearTimeout(this.fitTimer);
        }
        this.fitTimer = setTimeout(() => {
            this.fitTimer = null;
            this.doFit(reason);
        }, delay);
    }

    /**
     * Perform fit if size has changed.
     * Called externally when sidebar transitions complete.
     * Debounced to prevent multiple rapid calls from transitionend + setTimeout.
     */
    fit(): void {
        // Debounce: only fit once within 100ms
        if (this.fitDebounceTimer) {
            clearTimeout(this.fitDebounceTimer);
        }
        this.fitDebounceTimer = setTimeout(() => {
            this.fitDebounceTimer = null;
            this.doFit('external');
        }, 100);
    }

    /**
     * Fit terminal after sidebar/right panel toggle.
     * Waits for CSS transitions to complete before fitting.
     *
     * @param sidebarCollapsed - Whether sidebar is collapsed (hidden)
     * @param rightPanelVisible - Whether right panel is visible (for extended delay)
     */
    fitWithSidebarState(sidebarCollapsed: boolean, rightPanelVisible: boolean = false): void {
        // Resize debug log disabled

        // Use longer delay if right panel is visible (300ms transition + buffer)
        const delay = rightPanelVisible ? 350 : 250;

        // Wait for CSS transitions to complete
        setTimeout(() => {
            const width = Math.round(this.elem.clientWidth);
            const height = Math.round(this.elem.clientHeight);
            // Resize debug log disabled

            this.lastWidth = width;
            this.lastHeight = height;
            this.fitAddon.fit();
            // Only scroll if terminal is fully initialized
            // Delay scrollToBottom to ensure terminal is fully rendered
            setTimeout(() => {
                if (this.term.rows > 0 && this.term.cols > 0) {
                    try {
                        this.term.scrollToBottom();
                    } catch (e) {
                        // Ignore scrollToBottom errors during initialization
                    }
                }
            }, 50);
            // Resize debug log disabled
        }, delay);
    }

    private doFit(reason: string): void {
        // Use getBoundingClientRect for more accurate measurements
        const rect = this.elem.getBoundingClientRect();
        const currentWidth = Math.round(rect.width);
        const currentHeight = Math.round(rect.height);

        if (currentWidth !== this.lastWidth || currentHeight !== this.lastHeight) {
            // Resize debug log disabled
            this.lastWidth = currentWidth;
            this.lastHeight = currentHeight;

            // Force container dimensions before fit
            this.elem.style.width = `${currentWidth}px`;
            this.elem.style.height = `${currentHeight}px`;

            this.fitAddon.fit();

            // Force all xterm layers to match container dimensions exactly
            const viewport = this.elem.querySelector('.xterm-viewport') as HTMLElement;
            if (viewport) {
                viewport.style.width = '100%';
                viewport.style.right = '0';
                viewport.style.bottom = '0';
            }

            // Force screen and all layers to exact container size
            const screen = this.elem.querySelector('.xterm-screen') as HTMLElement;
            if (screen) {
                screen.style.width = `${currentWidth}px`;
                screen.style.height = `${currentHeight}px`;
            }

            // Force all layer canvases to match screen size
            const layers = this.elem.querySelectorAll('.xterm-screen canvas, .xterm-link-layer, .xterm-text-layer, .xterm-cursor-layer, .xterm-selection-layer');
            layers.forEach((layer) => {
                const el = layer as HTMLElement;
                el.style.width = '100%';
                el.style.height = '100%';
                // Also update canvas attributes for proper scaling
                if (el instanceof HTMLCanvasElement) {
                    const dpr = window.devicePixelRatio || 1;
                    el.width = currentWidth * dpr;
                    el.height = currentHeight * dpr;
                }
            });

            // Only scroll if terminal is fully initialized
            // Delay scrollToBottom to ensure terminal is fully rendered
            setTimeout(() => {
                if (this.term.rows > 0 && this.term.cols > 0) {
                    try {
                        this.term.scrollToBottom();
                    } catch (e) {
                        // Ignore scrollToBottom errors during initialization
                    }
                }
            }, 50);
            // Resize debug log disabled
        }
    }

    close(): void {
        this.resizeObserver.disconnect();
        this.webglAddon?.dispose();
        this.term.dispose();
    }
}
