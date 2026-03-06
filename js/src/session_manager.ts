// Session Manager for frontend
// Handles session list, creation, and selection

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

class SessionManager {
    private sessions: SessionInfo[] = [];
    private activeSessionId: string | null = null;
    private onSessionChange: ((sessions: SessionInfo[]) => void) | null = null;
    private onActiveSessionChange: ((sessionId: string | null) => void) | null = null;

    constructor() {
        this.init();
    }

    private init() {
        // Listen for click on new session button
        const newSessionBtn = document.getElementById('new-session-btn');
        if (newSessionBtn) {
            newSessionBtn.addEventListener('click', () => {
                this.createSession();
            });
        }
    }

    setOnSessionChange(callback: (sessions: SessionInfo[]) => void) {
        this.onSessionChange = callback;
    }

    setOnActiveSessionChange(callback: (sessionId: string | null) => void) {
        this.onActiveSessionChange = callback;
    }

    async fetchSessions(): Promise<void> {
        try {
            const response = await fetch('/api/sessions');
            const data: SessionAPIResponse = await response.json();
            this.sessions = data.sessions || [];
            if (this.onSessionChange) {
                this.onSessionChange(this.sessions);
            }
        } catch (error) {
            console.error('Failed to fetch sessions:', error);
        }
    }

    async createSession(): Promise<string | null> {
        try {
            const response = await fetch('/api/sessions', {
                method: 'POST',
            });
            const data: SessionAPIResponse = await response.json();

            if (data.id) {
                await this.fetchSessions();
                this.setActiveSession(data.id);
                return data.id;
            }
        } catch (error) {
            console.error('Failed to create session:', error);
        }
        return null;
    }

    async closeSession(id: string): Promise<void> {
        try {
            await fetch(`/api/sessions/${id}`, {
                method: 'DELETE',
            });

            if (this.activeSessionId === id) {
                this.activeSessionId = null;
                if (this.onActiveSessionChange) {
                    this.onActiveSessionChange(null);
                }
            }

            await this.fetchSessions();
        } catch (error) {
            console.error('Failed to close session:', error);
        }
    }

    setActiveSession(id: string) {
        this.activeSessionId = id;
        if (this.onActiveSessionChange) {
            this.onActiveSessionChange(id);
        }

        // Update UI
        const sessionItems = document.querySelectorAll('.session-item');
        sessionItems.forEach(item => {
            item.classList.remove('active');
            if ((item as HTMLElement).dataset.sessionId === id) {
                item.classList.add('active');
            }
        });
    }

    getActiveSessionId(): string | null {
        return this.activeSessionId;
    }

    getSessions(): SessionInfo[] {
        return this.sessions;
    }
}

// Export for use in main.ts
declare global {
    interface Window {
        SessionManager: typeof SessionManager;
    }
}

window.SessionManager = SessionManager;
