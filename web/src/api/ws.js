
class WsClient {
    constructor() {
        this.socket = null;
        this.handlers = new Set();
        this.reconnectTimer = null;
    }

    connect(token) {
        if (!token) return;

        if (this.socket) {
            this.socket.close();
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        const wsUrl = `${protocol}//${host}/ws`;

        console.log('Connecting WS');

        this.socket = new WebSocket(wsUrl, ['auth', token]);

        this.socket.onopen = () => {
            console.log('WS Connected');
        };

        this.socket.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data); // backend sends JSON
                this.handlers.forEach(handler => handler(data));
            } catch (e) {
                console.error('WS Parse Error:', e);
            }
        };

        this.socket.onclose = () => {
            console.log('WS Closed, retrying in 3s...');
            this.reconnectTimer = setTimeout(() => this.connect(token), 3000);
        };

        this.socket.onerror = (err) => {
            console.error('WS Error:', err);
        };
    }

    subscribe(handler) {
        this.handlers.add(handler);
        return () => this.handlers.delete(handler);
    }

    disconnect() {
        if (this.socket) {
            this.socket.onclose = null; // prevent reconnect loop
            this.socket.close();
            this.socket = null;
        }
        if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    }
}

export const wsClient = new WsClient();
