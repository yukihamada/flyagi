import { Globe } from 'lucide-react'
import { useLanguage, type Language } from '../../contexts/LanguageContext'

export function LanguageSwitch() {
  const { language, setLanguage, t } = useLanguage()

  const handleLanguageChange = (lang: Language) => {
    setLanguage(lang)
  }

  return (
    <div className="flex items-center gap-2">
      <Globe className="h-4 w-4 text-gray-500" />
      <div className="flex rounded-lg border border-gray-700 bg-gray-800 overflow-hidden">
        <button
          className={`px-3 py-1 text-xs font-medium transition-colors ${
            language === 'en'
              ? 'bg-blue-600 text-white'
              : 'text-gray-300 hover:text-gray-100'
          }`}
          onClick={() => handleLanguageChange('en')}
          title={t('language.english')}
        >
          EN
        </button>
        <button
          className={`px-3 py-1 text-xs font-medium transition-colors ${
            language === 'ja'
              ? 'bg-blue-600 text-white'
              : 'text-gray-300 hover:text-gray-100'
          }`}
          onClick={() => handleLanguageChange('ja')}
          title={t('language.japanese')}
        >
          JA
        </button>
      </div>
    </div>
  )
}