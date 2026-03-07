import * as bare from "libapps";

export class Hterm {
    elem: HTMLElement;

    term: bare.hterm.Terminal;
    io: bare.hterm.IO;

    columns: number = 80;
    rows: number = 24;

    // to "show" the current message when removeMessage() is called
    message: string = "";

    constructor(elem: HTMLElement) {
        this.elem = elem;
        bare.hterm.defaultStorage = new bare.lib.Storage.Memory();
        this.term = new bare.hterm.Terminal();
        this.term.getPrefs().set("send-encoding", "raw");
        this.term.decorate(this.elem);

        this.io = this.term.io.push();
        this.term.installKeyboard();
    }

    info(): { columns: number; rows: number } {
        return { columns: this.columns, rows: this.rows };
    }

    output(data: string): void {
        if (this.term.io != null) {
            this.term.io.writeUTF8(data);
        }
    }

    showMessage(message: string, timeout: number): void {
        this.message = message;
        if (timeout > 0) {
            this.term.io.showOverlay(message, timeout);
        } else {
            this.term.io.showOverlay(message, null);
        }
    }

    removeMessage(): void {
        // there is no hideOverlay(), so show the same message with 0 sec
        this.term.io.showOverlay(this.message, 0);
    }

    setWindowTitle(title: string): void {
        this.term.setWindowTitle(title);
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

    setPreferences(value: Record<string, unknown>): void {
        Object.keys(value).forEach((key) => {
            this.term.getPrefs().set(key, value[key] as string);
        });
    }

    onInput(callback: (input: string) => void): void {
        this.io.onVTKeystroke = (data) => {
            callback(data);
        };
        this.io.sendString = (data) => {
            callback(data);
        };
    }

    onResize(callback: (columns: number, rows: number) => void): void {
        this.io.onTerminalResize = (columns: number, rows: number) => {
            this.columns = columns;
            this.rows = rows;
            callback(columns, rows);
        };
    }

    deactivate(): void {
        this.io.onVTKeystroke = function () {};
        this.io.sendString = function () {};
        this.io.onTerminalResize = function () {};
        this.term.uninstallKeyboard();
    }

    reset(): void {
        this.removeMessage();
        this.term.installKeyboard();
    }

    close(): void {
        this.term.uninstallKeyboard();
    }
}
