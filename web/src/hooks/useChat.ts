import { useState, useCallback, useEffect, useRef } from 'react'
import { Envelope } from '../services/ws'

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: number
}

interface DiffData {
  requestId: string
  description: string
  diffs: Array<{ path: string; diff: string }>
}

interface UseChatOptions {
  send: (env: Envelope) => void
  subscribe: (listener: (env: Envelope) => void) => () => void
  providerId: string
}

export function useChat({ send, subscribe, providerId }: UseChatOptions) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isStreaming, setIsStreaming] = useState(false)
  const [pendingDiff, setPendingDiff] = useState<DiffData | null>(null)
  const streamBufferRef = useRef('')

  useEffect(() => {
    const unsubscribe = subscribe((env: Envelope) => {
      const payload = env.payload as Record<string, unknown>

      switch (env.type) {
        case 'chat.chunk': {
          const content = (payload?.content as string) || ''
          const done = payload?.done as boolean

          if (content) {
            streamBufferRef.current += content
            setMessages(prev => {
              const last = prev[prev.length - 1]
              if (last?.role === 'assistant') {
                return [
                  ...prev.slice(0, -1),
                  { ...last, content: streamBufferRef.current },
                ]
              }
              return [
                ...prev,
                {
                  id: crypto.randomUUID(),
                  role: 'assistant',
                  content: streamBufferRef.current,
                  timestamp: Date.now(),
                },
              ]
            })
          }

          if (done) {
            setIsStreaming(false)
            streamBufferRef.current = ''
          }
          break
        }
        case 'selfmod.diff': {
          const raw = payload as Record<string, unknown>
          setPendingDiff({
            requestId: (raw.request_id as string) || '',
            description: (raw.description as string) || '',
            diffs: (raw.diffs as Array<{ path: string; diff: string }>) || [],
          })
          break
        }
        case 'selfmod.status': {
          const status = (payload?.status as string) || ''
          const message = (payload?.message as string) || ''
          const prUrl = (payload?.pr_url as string) || ''
          let statusText = message
          if (prUrl) {
            statusText += `\n[PR: ${prUrl}](${prUrl})`
          }
          setMessages(prev => [
            ...prev,
            {
              id: crypto.randomUUID(),
              role: 'assistant',
              content: `[${status}] ${statusText}`,
              timestamp: Date.now(),
            },
          ])
          break
        }
        case 'error': {
          const error = (payload?.error as string) || 'Unknown error'
          setMessages(prev => [
            ...prev,
            {
              id: crypto.randomUUID(),
              role: 'assistant',
              content: `Error: ${error}`,
              timestamp: Date.now(),
            },
          ])
          setIsStreaming(false)
          break
        }
      }
    })

    return unsubscribe
  }, [subscribe])

  const sendMessage = useCallback(
    (text: string) => {
      const userMessage: ChatMessage = {
        id: crypto.randomUUID(),
        role: 'user',
        content: text,
        timestamp: Date.now(),
      }

      setMessages(prev => [...prev, userMessage])
      setIsStreaming(true)
      streamBufferRef.current = ''

      // Send full message history to the backend
      const allMessages = [...messages, userMessage].map(m => ({
        role: m.role,
        content: m.content,
      }))

      send({
        type: 'chat.send',
        payload: {
          messages: allMessages,
          provider_id: providerId,
        },
      })
    },
    [send, messages, providerId],
  )

  const cancelStream = useCallback(() => {
    send({ type: 'chat.cancel' })
    setIsStreaming(false)
  }, [send])

  const approveDiff = useCallback(
    (requestId: string) => {
      send({
        type: 'selfmod.approve',
        payload: { request_id: requestId },
      })
      setPendingDiff(null)
    },
    [send],
  )

  const rejectDiff = useCallback(
    (requestId: string) => {
      send({
        type: 'selfmod.reject',
        payload: { request_id: requestId },
      })
      setPendingDiff(null)
    },
    [send],
  )

  return {
    messages,
    isStreaming,
    pendingDiff,
    sendMessage,
    cancelStream,
    approveDiff,
    rejectDiff,
  }
}
