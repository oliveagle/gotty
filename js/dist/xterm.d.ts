import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebglAddon } from "@xterm/addon-webgl";
import { lib } from "libapps";
export declare class Xterm {
    elem: HTMLElement;
    term: Terminal;
    fitAddon: FitAddon;
    webglAddon: WebglAddon | null;
    resizeListener: () => void;
    decoder: lib.UTF8Decoder;
    message: HTMLElement;
    messageTimeout: number;
    messageTimer: ReturnType<typeof setTimeout> | null;
    constructor(elem: HTMLElement);
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
    close(): void;
}
//# sourceMappingURL=xterm.d.ts.map