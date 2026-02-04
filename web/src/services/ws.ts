export interface Envelope {
  type: string
  payload?: unknown
}

export type MessageListener = (env: Envelope) => void

export class WebSocketClient {
  private ws: WebSocket | null = null
  private listeners: Set<MessageListener> = new Set()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private url: string

  constructor(url?: string) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    this.url = url || `${protocol}//${window.location.host}/ws`
  }

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) return

    this.ws = new WebSocket(this.url)

    this.ws.onopen = () => {
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer)
        this.reconnectTimer = null
      }
    }

    this.ws.onmessage = (event) => {
      try {
        const env: Envelope = JSON.parse(event.data)
        this.listeners.forEach(listener => listener(env))
      } catch {
        console.error('Failed to parse WebSocket message')
      }
    }

    this.ws.onclose = () => {
      this.scheduleReconnect()
    }

    this.ws.onerror = () => {
      this.ws?.close()
    }
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.ws?.close()
    this.ws = null
  }

  send(env: Envelope) {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      console.warn('WebSocket not connected')
      return
    }
    this.ws.send(JSON.stringify(env))
  }

  subscribe(listener: MessageListener): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  get connected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, 3000)
  }
}
