export class WebSocketClient {
    constructor(url, onMessage, onOpen, onClose) {
        this.url = url;
        this.onMessage = onMessage;
        this.onOpen = onOpen;
        this.onClose = onClose;
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10; // Limited to prevent battery drain (was 9999)
        this.isConnected = false;
        this.shouldReconnect = true;
        this.isKicked = false; // Permanent flag for kick
    }

    connect() {
        // Never reconnect if user was kicked
        if (this.isKicked) return;

        if (this.ws) {
            this.ws.close();
        }

        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
            this.isConnected = true;
            this.reconnectAttempts = 0;
            if (this.onOpen) this.onOpen();
        };

        this.ws.onclose = () => {
            this.isConnected = false;
            if (this.onClose) this.onClose();
            // Never reconnect if kicked
            if (this.shouldReconnect && !this.isKicked) {
                this.attemptReconnect();
            }
        };

        this.ws.onerror = (err) => {
            console.error('WebSocket error:', err);
        };

        this.ws.onmessage = (event) => {
            const rawMessages = event.data.split('\n');
            rawMessages.forEach(raw => {
                if (raw.trim()) {
                    try {
                        const msg = JSON.parse(raw);
                        if (this.onMessage) this.onMessage(msg);
                    } catch (e) {
                        console.error('Failed to parse message:', e);
                    }
                }
            });
        };
    }

    attemptReconnect() {
        // Never reconnect if kicked
        if (this.isKicked || !this.shouldReconnect) return;

        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            return;
        }

        const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
        this.reconnectAttempts++;

        setTimeout(() => {
            if (!this.isKicked) this.connect();
        }, delay);
    }

    send(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(data));
        }
    }

    disableReconnect() {
        this.shouldReconnect = false;
    }

    // Special method for kick - permanently disables reconnection
    kickedClose() {
        this.isKicked = true;
        this.shouldReconnect = false;
        if (this.ws) {
            this.ws.close(1000, 'Kicked');
        }
    }

    close() {
        if (this.ws) {
            this.ws.close(1000, 'Page unload');
        }
    }
}
