export const protocols = ["webtty"];

export const msgInputUnknown = '0';
export const msgInput = '1';
export const msgPing = '2';
export const msgResizeTerminal = '3';

export const msgUnknownOutput = '0';
export const msgOutput = '1';
export const msgPong = '2';
export const msgSetWindowTitle = '3';
export const msgSetPreferences = '4';
export const msgSetReconnect = '5';
export const msgSetSubtitle = '6';


export interface Terminal {
    info(): { columns: number; rows: number };
    output(data: string): void;
    showMessage(message: string, timeout: number): void;
    removeMessage(): void;
    setWindowTitle(title: string): void;
    setSubtitle(subtitle: string): void;
    setPreferences(value: object): void;
    onInput(callback: (input: string) => void): void;
    onResize(callback: (columns: number, rows: number) => void): void;
    reset(): void;
    deactivate(): void;
    close(): void;
}

export interface Connection {
    open(): void;
    close(): void;
    send(data: string): void;
    isOpen(): boolean;
    onOpen(callback: () => void): void;
    onReceive(callback: (data: string) => void): void;
    onClose(callback: () => void): void;
}

export interface ConnectionFactory {
    create(): Connection;
}


export class WebTTY {
    term: Terminal;
    connectionFactory: ConnectionFactory;
    args: string;
    authToken: string;
    reconnect: number;

    // Chunk size for large inputs (characters)
    // Most terminals can handle ~4KB at a time, but we use smaller chunks for reliability
    private static readonly INPUT_CHUNK_SIZE = 1024;
    // Delay between chunks (ms) to allow PTY to process
    private static readonly CHUNK_DELAY = 10;

    constructor(term: Terminal, connectionFactory: ConnectionFactory, args: string, authToken: string) {
        this.term = term;
        this.connectionFactory = connectionFactory;
        this.args = args;
        this.authToken = authToken;
        this.reconnect = -1;
    }

    // Send input in chunks to handle large pastes
    private sendInputChunked(connection: Connection, input: string): void {
        if (input.length <= WebTTY.INPUT_CHUNK_SIZE) {
            connection.send(msgInput + input);
            return;
        }

        // Split into chunks and send with delays
        let offset = 0;
        const sendNextChunk = () => {
            if (offset >= input.length) return;

            const chunk = input.slice(offset, offset + WebTTY.INPUT_CHUNK_SIZE);
            connection.send(msgInput + chunk);
            offset += WebTTY.INPUT_CHUNK_SIZE;

            if (offset < input.length) {
                setTimeout(sendNextChunk, WebTTY.CHUNK_DELAY);
            }
        };

        sendNextChunk();
    }

    open() {
        let connection = this.connectionFactory.create();
        let pingTimer: ReturnType<typeof setInterval>;
        let reconnectTimeout: ReturnType<typeof setTimeout>;

        const setup = () => {
            connection.onOpen(() => {
                // Remove "Connecting..." message
                this.term.removeMessage();

                const termInfo = this.term.info();

                connection.send(JSON.stringify(
                    {
                        Arguments: this.args,
                        AuthToken: this.authToken,
                    }
                ));


                const resizeHandler = (columns: number, rows: number) => {
                    connection.send(
                        msgResizeTerminal + JSON.stringify(
                            {
                                columns: columns,
                                rows: rows
                            }
                        )
                    );
                };

                this.term.onResize(resizeHandler);
                resizeHandler(termInfo.columns, termInfo.rows);

                this.term.onInput(
                    (input: string) => {
                        this.sendInputChunked(connection, input);
                    }
                );

                pingTimer = setInterval(() => {
                    connection.send(msgPing);
                }, 30 * 1000);

            });

            connection.onReceive((data) => {
                const payload = data.slice(1);
                switch (data[0]) {
                    case msgOutput:
                        this.term.output(atob(payload));
                        break;
                    case msgPong:
                        break;
                    case msgSetWindowTitle:
                        this.term.setWindowTitle(payload);
                        break;
                    case msgSetPreferences:
                        const preferences = JSON.parse(payload);
                        this.term.setPreferences(preferences);
                        break;
                    case msgSetReconnect:
                        const autoReconnect = JSON.parse(payload);
                        console.log("Enabling reconnect: " + autoReconnect + " seconds");
                        this.reconnect = autoReconnect;
                        break;
                    case msgSetSubtitle:
                        this.term.setSubtitle(payload);
                        break;
                }
            });

            connection.onClose(() => {
                clearInterval(pingTimer);
                this.term.deactivate();
                // Show message for 3 seconds then auto-hide
                this.term.showMessage("Connection Closed", 3000);
                if (this.reconnect > 0) {
                    reconnectTimeout = setTimeout(() => {
                        connection = this.connectionFactory.create();
                        this.term.reset();
                        setup();
                    }, this.reconnect * 1000);
                }
            });

            connection.open();
        };

        setup();
        return () => {
            clearTimeout(reconnectTimeout);
            connection.close();
        };
    }
}
