import { Languages } from 'lucide-react'
import { useLanguage } from '../../contexts/LanguageContext'

export function LanguageSwitch() {
  const { language, setLanguage, t } = useLanguage()

  return (
    <div className="relative group">
      <button
        className="flex items-center gap-2 rounded-lg bg-gray-800 px-3 py-1.5 text-sm text-gray-300 hover:bg-gray-700 hover:text-gray-100 transition-colors"
        title={t('language.switch')}
      >
        <Languages className="h-4 w-4" />
        <span className="text-xs">{language === 'en' ? 'EN' : 'JP'}</span>
      </button>
      
      <div className="absolute right-0 top-full mt-1 w-24 rounded-lg border border-gray-700 bg-gray-800 py-1 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200">
        <button
          className={`w-full px-3 py-1.5 text-left text-xs transition-colors ${
            language === 'en'
              ? 'bg-blue-600 text-white'
              : 'text-gray-300 hover:bg-gray-700 hover:text-gray-100'
          }`}
          onClick={() => setLanguage('en')}
        >
          {t('language.en')}
        </button>
        <button
          className={`w-full px-3 py-1.5 text-left text-xs transition-colors ${
            language === 'ja'
              ? 'bg-blue-600 text-white'
              : 'text-gray-300 hover:bg-gray-700 hover:text-gray-100'
          }`}
          onClick={() => setLanguage('ja')}
        >
          {t('language.ja')}
        </button>
      </div>
    </div>
  )
}