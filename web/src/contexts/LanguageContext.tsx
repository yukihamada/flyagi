import { createContext, useContext, useState, ReactNode } from 'react'

type Language = 'en' | 'ja'

interface LanguageContextType {
  language: Language
  setLanguage: (lang: Language) => void
  t: (key: string) => string
}

const LanguageContext = createContext<LanguageContextType | undefined>(undefined)

const translations = {
  en: {
    // Header
    'header.title': 'FlyAGI',
    
    // Chat
    'chat.placeholder': 'Send a message...',
    'chat.recording.start': 'Start recording',
    'chat.recording.stop': 'Stop recording',
    'chat.generating.stop': 'Stop generating',
    'chat.send': 'Send message',
    
    // Diff Viewer
    'diff.title': 'Code Change Request',
    'diff.approve': 'Approve',
    'diff.reject': 'Reject',
    'diff.newFile': '(new file)',
    
    // Providers
    'provider.llm': 'LLM',
    'provider.tts': 'TTS',
    'provider.stt': 'STT',
    
    // Language
    'language.switch': 'Language',
    'language.en': 'English',
    'language.ja': '日本語',
  },
  ja: {
    // Header
    'header.title': 'FlyAGI',
    
    // Chat
    'chat.placeholder': 'メッセージを送信...',
    'chat.recording.start': '録音開始',
    'chat.recording.stop': '録音停止',
    'chat.generating.stop': '生成を停止',
    'chat.send': 'メッセージ送信',
    
    // Diff Viewer
    'diff.title': 'コード変更要求',
    'diff.approve': '承認',
    'diff.reject': '却下',
    'diff.newFile': '(新規ファイル)',
    
    // Providers
    'provider.llm': 'LLM',
    'provider.tts': 'TTS',
    'provider.stt': 'STT',
    
    // Language
    'language.switch': '言語',
    'language.en': 'English',
    'language.ja': '日本語',
  },
}

interface LanguageProviderProps {
  children: ReactNode
}

export function LanguageProvider({ children }: LanguageProviderProps) {
  const [language, setLanguage] = useState<Language>('en')

  const t = (key: string): string => {
    return translations[language][key as keyof typeof translations[typeof language]] || key
  }

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  )
}

export function useLanguage() {
  const context = useContext(LanguageContext)
  if (context === undefined) {
    throw new Error('useLanguage must be used within a LanguageProvider')
  }
  return context
}