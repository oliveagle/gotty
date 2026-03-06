interface SessionInfo {
    id: string;
    title: string;
    created_at: string;
}
interface SessionAPIResponse {
    sessions?: SessionInfo[];
    id?: string;
    title?: string;
    status?: string;
}
declare class SessionManager {
    private sessions;
    private activeSessionId;
    private onSessionChange;
    private onActiveSessionChange;
    constructor();
    private init();
    setOnSessionChange(callback: (sessions: SessionInfo[]) => void): void;
    setOnActiveSessionChange(callback: (sessionId: string | null) => void): void;
    fetchSessions(): Promise<void>;
    createSession(): Promise<string | null>;
    closeSession(id: string): Promise<void>;
    setActiveSession(id: string): void;
    getActiveSessionId(): string | null;
    getSessions(): SessionInfo[];
}
declare global  {
    interface Window {
        SessionManager: typeof SessionManager;
    }
}
