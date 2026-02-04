import { useRef, useEffect } from 'react'
import { MessageBubble } from './MessageBubble'
import { ChatInput } from './ChatInput'
import type { ChatMessage } from '../../hooks/useChat'

interface Props {
  messages: ChatMessage[]
  isStreaming: boolean
  onSend: (text: string) => void
  onCancel: () => void
  isRecording: boolean
  isTranscribing: boolean
  onStartRecording: () => void
  onStopRecording: () => void
}

export function ChatContainer({
  messages,
  isStreaming,
  onSend,
  onCancel,
  isRecording,
  isTranscribing,
  onStartRecording,
  onStopRecording,
}: Props) {
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    scrollRef.current?.scrollTo({
      top: scrollRef.current.scrollHeight,
      behavior: 'smooth',
    })
  }, [messages])

  return (
    <div className="flex h-full flex-col">
      <div ref={scrollRef} className="flex-1 overflow-y-auto p-4">
        {messages.length === 0 ? (
          <div className="flex h-full items-center justify-center">
            <p className="text-gray-500">Start a conversation...</p>
          </div>
        ) : (
          messages.map(msg => <MessageBubble key={msg.id} message={msg} />)
        )}
        {isStreaming && (
          <div className="flex justify-start mb-3">
            <div className="flex gap-1 rounded-2xl bg-gray-800 px-4 py-3">
              <span className="h-2 w-2 rounded-full bg-gray-500 animate-bounce [animation-delay:0ms]" />
              <span className="h-2 w-2 rounded-full bg-gray-500 animate-bounce [animation-delay:150ms]" />
              <span className="h-2 w-2 rounded-full bg-gray-500 animate-bounce [animation-delay:300ms]" />
            </div>
          </div>
        )}
      </div>
      <ChatInput
        onSend={onSend}
        onCancel={onCancel}
        isStreaming={isStreaming}
        isRecording={isRecording}
        isTranscribing={isTranscribing}
        onStartRecording={onStartRecording}
        onStopRecording={onStopRecording}
      />
    </div>
  )
}
