import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebglAddon } from "@xterm/addon-webgl";
import { lib } from "libapps";
export declare class Xterm {
    elem: HTMLElement;
    term: Terminal;
    fitAddon: FitAddon;
    webglAddon: WebglAddon | null;
    resizeObserver: ResizeObserver;
    decoder: lib.UTF8Decoder;
    message: HTMLElement;
    messageTimeout: number;
    messageTimer: ReturnType<typeof setTimeout> | null;
    lastWidth: number;
    lastHeight: number;
    fitTimer: ReturnType<typeof setTimeout> | null;
    fitDebounceTimer: ReturnType<typeof setTimeout> | null;
    constructor(elem: HTMLElement);
    private setupClipboardOnSelection;
    private copySelectionToClipboard;
    private pasteFromClipboard;
    private copyToClipboardFallback;
    private setupCompositionViewFix;
    info(): {
        columns: number;
        rows: number;
    };
    output(data: string): void;
    showMessage(message: string, timeout: number): void;
    removeMessage(): void;
    setWindowTitle(title: string): void;
    setSubtitle(subtitle: string): void;
    setPreferences(_value: object): void;
    onInput(callback: (input: string) => void): void;
    onResize(callback: (columns: number, rows: number) => void): void;
    deactivate(): void;
    reset(): void;
    /**
     * Schedule a fit() call after a delay.
     * Multiple calls within the delay period will be coalesced into one.
     */
    scheduleFit(reason: string, delay?: number): void;
    /**
     * Perform fit if size has changed.
     * Called externally when sidebar transitions complete.
     * Debounced to prevent multiple rapid calls from transitionend + setTimeout.
     */
    fit(): void;
    /**
     * Fit terminal after sidebar toggle.
     * Waits for CSS transition (200ms) to complete before fitting.
     *
     * @param sidebarCollapsed - Whether sidebar is collapsed (hidden)
     */
    fitWithSidebarState(sidebarCollapsed: boolean): void;
    private doFit;
    close(): void;
}
//# sourceMappingURL=xterm.d.ts.map