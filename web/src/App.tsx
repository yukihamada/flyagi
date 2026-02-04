import { useState, useEffect, useCallback } from 'react'
import { Wifi, WifiOff } from 'lucide-react'
import { ChatContainer } from './components/Chat/ChatContainer'
import { DiffViewer } from './components/Diff/DiffViewer'
import { ProviderSelector } from './components/Provider/ProviderSelector'
import { LanguageSwitch } from './components/Language/LanguageSwitch'
import { useWebSocket } from './hooks/useWebSocket'
import { useChat } from './hooks/useChat'
import { useVoice } from './hooks/useVoice'
import { useLanguage } from './contexts/LanguageContext'
import { api, type ProvidersResponse } from './services/api'

function App() {
  const { t } = useLanguage()
  const { send, subscribe, connected } = useWebSocket()
  const [providers, setProviders] = useState<ProvidersResponse>({
    llm: [],
    tts: [],
    stt: [],
  })
  const [selectedLLM, setSelectedLLM] = useState('')
  const [selectedTTS, setSelectedTTS] = useState('')
  const [selectedSTT, setSelectedSTT] = useState('')

  useEffect(() => {
    api.getProviders().then(p => {
      setProviders(p)
      if (p.llm.length > 0 && !selectedLLM) setSelectedLLM(p.llm[0])
      if (p.tts.length > 0 && !selectedTTS) setSelectedTTS(p.tts[0])
      if (p.stt.length > 0 && !selectedSTT) setSelectedSTT(p.stt[0])
    }).catch(() => {
      // Providers not available yet
    })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const {
    messages,
    isStreaming,
    pendingDiff,
    sendMessage,
    cancelStream,
    approveDiff,
    rejectDiff,
  } = useChat({ send, subscribe, providerId: selectedLLM })

  const handleTranscript = useCallback(
    (text: string) => {
      sendMessage(text)
    },
    [sendMessage],
  )

  const { isRecording, isTranscribing, startRecording, stopRecording } =
    useVoice({ onTranscript: handleTranscript, sttProvider: selectedSTT })

  return (
    <div className="flex h-screen flex-col bg-gray-950 text-gray-100">
      <header className="flex items-center justify-between border-b border-gray-800 px-4 py-2.5">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-bold">{t('header.title')}</h1>
          <div className="flex items-center gap-1 text-xs">
            {connected ? (
              <Wifi className="h-3.5 w-3.5 text-green-500" />
            ) : (
              <WifiOff className="h-3.5 w-3.5 text-red-500" />
            )}
          </div>
        </div>
        <div className="flex items-center gap-4">
          {providers.llm.length > 0 && (
            <ProviderSelector
              label={t('provider.llm')}
              providers={providers.llm}
              selected={selectedLLM}
              onChange={setSelectedLLM}
            />
          )}
          {providers.tts.length > 0 && (
            <ProviderSelector
              label={t('provider.tts')}
              providers={providers.tts}
              selected={selectedTTS}
              onChange={setSelectedTTS}
            />
          )}
          {providers.stt.length > 0 && (
            <ProviderSelector
              label={t('provider.stt')}
              providers={providers.stt}
              selected={selectedSTT}
              onChange={setSelectedSTT}
            />
          )}
          <LanguageSwitch />
        </div>
      </header>

      <main className="flex-1 overflow-hidden">
        <ChatContainer
          messages={messages}
          isStreaming={isStreaming}
          onSend={sendMessage}
          onCancel={cancelStream}
          isRecording={isRecording}
          isTranscribing={isTranscribing}
          onStartRecording={startRecording}
          onStopRecording={stopRecording}
        />
      </main>

      {pendingDiff && (
        <DiffViewer
          requestId={pendingDiff.requestId}
          description={pendingDiff.description}
          diffs={pendingDiff.diffs}
          onApprove={approveDiff}
          onReject={rejectDiff}
        />
      )}
    </div>
  )
}

export default App
