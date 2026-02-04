import { useState, useCallback, type KeyboardEvent } from 'react'
import { Send, Square, Mic, MicOff, Loader2 } from 'lucide-react'
import { useLanguage } from '../../contexts/LanguageContext'

interface Props {
  onSend: (text: string) => void
  onCancel: () => void
  isStreaming: boolean
  isRecording: boolean
  isTranscribing: boolean
  onStartRecording: () => void
  onStopRecording: () => void
}

export function ChatInput({
  onSend,
  onCancel,
  isStreaming,
  isRecording,
  isTranscribing,
  onStartRecording,
  onStopRecording,
}: Props) {
  const { t } = useLanguage()
  const [input, setInput] = useState('')

  const handleSend = useCallback(() => {
    const text = input.trim()
    if (!text || isStreaming) return
    onSend(text)
    setInput('')
  }, [input, isStreaming, onSend])

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSend()
      }
    },
    [handleSend],
  )

  return (
    <div className="border-t border-gray-800 p-4">
      <div className="flex items-end gap-2">
        <textarea
          className="flex-1 resize-none rounded-xl border border-gray-700 bg-gray-900 px-4 py-2.5 text-sm text-gray-100 placeholder-gray-500 focus:border-blue-500 focus:outline-none"
          placeholder={t('chat.placeholder')}
          rows={1}
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={isStreaming}
        />

        <button
          className={`rounded-xl p-2.5 transition-colors ${
            isRecording
              ? 'bg-red-600 text-white'
              : isTranscribing
                ? 'bg-gray-700 text-gray-400'
                : 'bg-gray-800 text-gray-400 hover:bg-gray-700 hover:text-gray-200'
          }`}
          onClick={isRecording ? onStopRecording : onStartRecording}
          disabled={isTranscribing || isStreaming}
          title={isRecording ? t('chat.recording.stop') : t('chat.recording.start')}
        >
          {isTranscribing ? (
            <Loader2 className="h-5 w-5 animate-spin" />
          ) : isRecording ? (
            <MicOff className="h-5 w-5" />
          ) : (
            <Mic className="h-5 w-5" />
          )}
        </button>

        {isStreaming ? (
          <button
            className="rounded-xl bg-red-600 p-2.5 text-white transition-colors hover:bg-red-700"
            onClick={onCancel}
            title={t('chat.stop.generating')}
          >
            <Square className="h-5 w-5" />
          </button>
        ) : (
          <button
            className="rounded-xl bg-blue-600 p-2.5 text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
            onClick={handleSend}
            disabled={!input.trim()}
            title={t('chat.send.message')}
          >
            <Send className="h-5 w-5" />
          </button>
        )}
      </div>
    </div>
  )
}
