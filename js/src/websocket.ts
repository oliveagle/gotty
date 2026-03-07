export class ConnectionFactory {
    url: string;
    protocols: string[];

    constructor(url: string, protocols: string[]) {
        this.url = url;
        this.protocols = protocols;
    }

    create(): Connection {
        return new Connection(this.url, this.protocols);
    }
}

export class Connection {
    bare: WebSocket;

    constructor(url: string, protocols: string[]) {
        this.bare = new WebSocket(url, protocols);
    }

    open(): void {
        // nothing todo for websocket
    }

    close(): void {
        this.bare.close();
    }

    send(data: string): void {
        this.bare.send(data);
    }

    isOpen(): boolean {
        if (this.bare.readyState === WebSocket.CONNECTING ||
            this.bare.readyState === WebSocket.OPEN) {
            return true;
        }
        return false;
    }

    onOpen(callback: () => void): void {
        this.bare.onopen = () => {
            callback();
        };
    }

    onReceive(callback: (data: string) => void): void {
        this.bare.onmessage = (event) => {
            callback(event.data);
        };
    }

    onClose(callback: () => void): void {
        this.bare.onclose = () => {
            callback();
        };
    }
}
