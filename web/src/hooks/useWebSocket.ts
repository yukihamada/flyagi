import { useEffect, useRef, useCallback, useState } from 'react'
import { WebSocketClient, Envelope, MessageListener } from '../services/ws'

export function useWebSocket() {
  const clientRef = useRef<WebSocketClient | null>(null)
  const [connected, setConnected] = useState(false)

  useEffect(() => {
    const client = new WebSocketClient()
    clientRef.current = client
    client.connect()

    const checkConnection = setInterval(() => {
      setConnected(client.connected)
    }, 1000)

    return () => {
      clearInterval(checkConnection)
      client.disconnect()
    }
  }, [])

  const send = useCallback((env: Envelope) => {
    clientRef.current?.send(env)
  }, [])

  const subscribe = useCallback((listener: MessageListener) => {
    return clientRef.current?.subscribe(listener) ?? (() => {})
  }, [])

  return { send, subscribe, connected }
}
