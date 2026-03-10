// AG-UI Protocol Client - Based on CopilotKit's AG-UI protocol
// SSE (Server-Sent Events) based communication for Agent UI interaction

export type AGUIEvent =
  | { type: "message"; id: string; role: "user" | "assistant" | "system"; content: string; timestamp: number }
  | { type: "tool_call"; id: string; name: string; args: Record<string, any> }
  | { type: "tool_result"; id: string; name: string; result: any }
  | { type: "state_update"; key: string; value: any; operation: "set" | "delete" | "merge" }
  | { type: "ui_component"; id: string; component: string; props: Record<string, any> }
  | { type: "human_in_the_loop"; id: string; requestType: "confirm" | "input" | "select"; message: string; schema?: any }
  | { type: "human_response"; id: string; data: any }
  | { type: "error"; message: string; error?: any }
  | { type: "ping" }
  | { type: "done" };

export type AGUIConnectionStatus = "disconnected" | "connecting" | "connected" | "error";

export interface AGUIClientOptions {
  endpoint?: string;
  onEvent?: (event: AGUIEvent) => void;
  onStatusChange?: (status: AGUIConnectionStatus) => void;
  onError?: (error: Error) => void;
}

export class AGUIClient {
  private eventSource: EventSource | null = null;
  private status: AGUIConnectionStatus = "disconnected";
  private options: AGUIClientOptions;
  private messageQueue: AGUIEvent[] = [];
  private maxQueueSize: number = 100;

  constructor(options: AGUIClientOptions = {}) {
    this.options = {
      endpoint: "/api/agui",
      ...options,
    };
  }

  getStatus(): AGUIConnectionStatus {
    return this.status;
  }

  getMessages(): AGUIEvent[] {
    return [...this.messageQueue];
  }

  clearMessages(): void {
    this.messageQueue = [];
  }

  private setStatus(status: AGUIConnectionStatus): void {
    if (this.status !== status) {
      this.status = status;
      this.options.onStatusChange?.(status);
    }
  }

  private enqueueMessage(event: AGUIEvent): void {
    this.messageQueue.push(event);
    if (this.messageQueue.length > this.maxQueueSize) {
      this.messageQueue.shift();
    }
  }

  connect(endpoint?: string): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.eventSource) {
        this.disconnect();
      }

      const url = endpoint || this.options.endpoint;
      if (!url) {
        reject(new Error("AG-UI endpoint is required"));
        return;
      }

      this.setStatus("connecting");

      try {
        // Add auth token if available
        const token = (window as any).gotty_auth_token || localStorage.getItem("gotty_auth_token");
        const separator = url.includes("?") ? "&" : "?";
        const urlWithToken = token ? `${url}${separator}token=${encodeURIComponent(token)}` : url;

        this.eventSource = new EventSource(urlWithToken);

        this.eventSource.onopen = () => {
          this.setStatus("connected");
          resolve();
        };

        this.eventSource.onerror = (error) => {
          this.setStatus("error");
          const err = new Error("SSE connection error");
          this.options.onError?.(err);
          reject(err);
        };

        this.eventSource.onmessage = (event) => {
          this.handleSSEMessage(event);
        };

        // Listen to typed events
        this.eventSource.addEventListener("message", (e) => this.handleTypedEvent("message", e));
        this.eventSource.addEventListener("tool_call", (e) => this.handleTypedEvent("tool_call", e));
        this.eventSource.addEventListener("tool_result", (e) => this.handleTypedEvent("tool_result", e));
        this.eventSource.addEventListener("state_update", (e) => this.handleTypedEvent("state_update", e));
        this.eventSource.addEventListener("ui_component", (e) => this.handleTypedEvent("ui_component", e));
        this.eventSource.addEventListener("human_in_the_loop", (e) => this.handleTypedEvent("human_in_the_loop", e));
        this.eventSource.addEventListener("human_response", (e) => this.handleTypedEvent("human_response", e));
        this.eventSource.addEventListener("error", (e) => this.handleTypedEvent("error", e));
        this.eventSource.addEventListener("ping", (e) => this.handleTypedEvent("ping", e));
        this.eventSource.addEventListener("done", (e) => this.handleTypedEvent("done", e));
      } catch (error) {
        this.setStatus("error");
        const err = error instanceof Error ? error : new Error(String(error));
        this.options.onError?.(err);
        reject(err);
      }
    });
  }

  disconnect(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    this.setStatus("disconnected");
  }

  private handleSSEMessage(event: MessageEvent): void {
    try {
      const data = JSON.parse(event.data);
      if (data.type) {
        const aguiEvent = data as AGUIEvent;
        this.enqueueMessage(aguiEvent);
        this.options.onEvent?.(aguiEvent);
      }
    } catch (error) {
      console.warn("[AG-UI] Failed to parse SSE message:", event.data);
    }
  }

  private handleTypedEvent(type: string, event: MessageEvent): void {
    try {
      const data = event.data ? JSON.parse(event.data) : {};
      const aguiEvent = { type, ...data, timestamp: Date.now() } as AGUIEvent;
      this.enqueueMessage(aguiEvent);
      this.options.onEvent?.(aguiEvent);
    } catch (error) {
      console.warn(`[AG-UI] Failed to parse ${type} event:`, event.data);
    }
  }

  // Send message to agent (via REST API)
  async sendMessage(content: string): Promise<void> {
    const url = this.options.endpoint + "/chat";
    const token = (window as any).gotty_auth_token || localStorage.getItem("gotty_auth_token");

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    const body: Record<string, any> = {
      content,
      timestamp: Date.now(),
    };

    const separator = url.includes("?") ? "&" : "?";
    const urlWithToken = token ? `${url}${separator}token=${encodeURIComponent(token)}` : url;

    const response = await fetch(urlWithToken, {
      method: "POST",
      headers,
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      throw new Error(`Failed to send message: ${response.status}`);
    }
  }

  // Send tool response
  async sendToolResult(callId: string, result: any): Promise<void> {
    const url = this.options.endpoint + "/tool_result";
    const token = (window as any).gotty_auth_token || localStorage.getItem("gotty_auth_token");

    const separator = url.includes("?") ? "&" : "?";
    const urlWithToken = token ? `${url}${separator}token=${encodeURIComponent(token)}` : url;

    const response = await fetch(urlWithToken, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ callId, result }),
    });

    if (!response.ok) {
      throw new Error(`Failed to send tool result: ${response.status}`);
    }
  }

  // Send human-in-the-loop response
  async sendHumanResponse(requestId: string, data: any): Promise<void> {
    const url = this.options.endpoint + "/human_response";
    const token = (window as any).gotty_auth_token || localStorage.getItem("gotty_auth_token");

    const separator = url.includes("?") ? "&" : "?";
    const urlWithToken = token ? `${url}${separator}token=${encodeURIComponent(token)}` : url;

    const response = await fetch(urlWithToken, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ requestId, data }),
    });

    if (!response.ok) {
      throw new Error(`Failed to send human response: ${response.status}`);
    }
  }

  // Update shared state
  async setState(key: string, value: any): Promise<void> {
    const url = this.options.endpoint + "/state";
    const token = (window as any).gotty_auth_token || localStorage.getItem("gotty_auth_token");

    const separator = url.includes("?") ? "&" : "?";
    const urlWithToken = token ? `${url}${separator}token=${encodeURIComponent(token)}` : url;

    const response = await fetch(urlWithToken, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key, value, operation: "set" }),
    });

    if (!response.ok) {
      throw new Error(`Failed to update state: ${response.status}`);
    }
  }
}

// Export to global scope
(window as any).AGUIClient = AGUIClient;
