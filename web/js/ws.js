class WSClient {
  constructor(url) {
    this.url = url;
    this.listeners = {};
    this.reconnectTimer = null;
    this.connect();
  }

  connect() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = this.url || `${protocol}//${location.host}/ws`;
    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      document.getElementById('status-connected').className = 'status-dot green';
      document.getElementById('status-text').textContent = 'Connected';
      this.emit('connected');
    };

    this.ws.onclose = () => {
      document.getElementById('status-connected').className = 'status-dot red';
      document.getElementById('status-text').textContent = 'Disconnected';
      this.emit('disconnected');
      this.reconnectTimer = setTimeout(() => this.connect(), 3000);
    };

    this.ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data);
        if (msg.type) {
          this.emit(msg.type, msg.data || msg);
        }
      } catch (err) {
        console.warn('ws parse error:', err);
      }
    };
  }

  on(type, fn) {
    if (!this.listeners[type]) this.listeners[type] = [];
    this.listeners[type].push(fn);
  }

  emit(type, data) {
    (this.listeners[type] || []).forEach(fn => fn(data));
  }

  send(cmd, id, extra) {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(Object.assign({ type: 'command', cmd, id }, extra)));
    }
  }

  disconnect() {
    clearTimeout(this.reconnectTimer);
    this.ws.close();
  }
}
