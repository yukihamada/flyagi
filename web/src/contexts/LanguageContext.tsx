import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'

export type Language = 'en' | 'ja'

interface LanguageContextType {
  language: Language
  setLanguage: (language: Language) => void
  t: (key: string) => string
}

const LanguageContext = createContext<LanguageContextType | undefined>(undefined)

export function useLanguage() {
  const context = useContext(LanguageContext)
  if (!context) {
    throw new Error('useLanguage must be used within a LanguageProvider')
  }
  return context
}

interface Props {
  children: ReactNode
}

export function LanguageProvider({ children }: Props) {
  const [language, setLanguageState] = useState<Language>('en')

  useEffect(() => {
    const saved = localStorage.getItem('language') as Language | null
    if (saved && (saved === 'en' || saved === 'ja')) {
      setLanguageState(saved)
    }
  }, [])

  const setLanguage = (lang: Language) => {
    setLanguageState(lang)
    localStorage.setItem('language', lang)
  }

  const t = (key: string) => {
    return translations[language][key] || key
  }

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  )
}

const translations: Record<Language, Record<string, string>> = {
  en: {
    // App Header
    'app.title': 'FlyAGI',
    
    // Chat
    'chat.placeholder': 'Send a message...',
    'chat.recording.stop': 'Stop recording',
    'chat.recording.start': 'Start recording',
    'chat.stop.generating': 'Stop generating',
    'chat.send.message': 'Send message',
    
    // Diff Viewer
    'diff.title': 'Code Change Request',
    'diff.approve': 'Approve',
    'diff.reject': 'Reject',
    'diff.new.file': '(new file)',
    
    // Provider
    'provider.llm': 'LLM',
    'provider.tts': 'TTS',
    'provider.stt': 'STT',
    
    // Language
    'language.english': 'English',
    'language.japanese': '日本語',
    'language.switch': 'Language',
    
    // Connection Status
    'connection.connected': 'Connected',
    'connection.disconnected': 'Disconnected',
  },
  ja: {
    // App Header
    'app.title': 'FlyAGI',
    
    // Chat
    'chat.placeholder': 'メッセージを入力...',
    'chat.recording.stop': '録音停止',
    'chat.recording.start': '録音開始',
    'chat.stop.generating': '生成停止',
    'chat.send.message': 'メッセージ送信',
    
    // Diff Viewer
    'diff.title': 'コード変更リクエスト',
    'diff.approve': '承認',
    'diff.reject': '拒否',
    'diff.new.file': '（新規ファイル）',
    
    // Provider
    'provider.llm': 'LLM',
    'provider.tts': 'TTS',
    'provider.stt': 'STT',
    
    // Language
    'language.english': 'English',
    'language.japanese': '日本語',
    'language.switch': '言語',
    
    // Connection Status
    'connection.connected': '接続済み',
    'connection.disconnected': '未接続',
  },
}