import { Check, X } from 'lucide-react'
import { useLanguage } from '../../contexts/LanguageContext'

interface FileDiff {
  path: string
  diff: string
}

interface Props {
  requestId: string
  description: string
  diffs: FileDiff[]
  onApprove: (id: string) => void
  onReject: (id: string) => void
}

export function DiffViewer({ requestId, description, diffs, onApprove, onReject }: Props) {
  const { t } = useLanguage()
  return (
    <div className="mx-4 my-3 rounded-xl border border-gray-700 bg-gray-900 overflow-hidden">
      <div className="flex items-center justify-between border-b border-gray-700 px-4 py-3">
        <div>
          <h3 className="text-sm font-medium text-gray-200">{t('diff.title')}</h3>
          <p className="text-xs text-gray-400 mt-0.5">{description}</p>
        </div>
        <div className="flex gap-2">
          <button
            className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-green-700"
            onClick={() => onApprove(requestId)}
          >
            <Check className="h-3.5 w-3.5" />
            {t('diff.approve')}
          </button>
          <button
            className="flex items-center gap-1 rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-red-700"
            onClick={() => onReject(requestId)}
          >
            <X className="h-3.5 w-3.5" />
            {t('diff.reject')}
          </button>
        </div>
      </div>
      <div className="max-h-96 overflow-y-auto">
        {diffs.map((fileDiff, i) => (
          <div key={i} className="border-b border-gray-800 last:border-b-0">
            <div className="bg-gray-800/50 px-4 py-2 text-xs font-mono text-gray-300">
              {fileDiff.path}
            </div>
            <pre className="px-4 py-3 text-xs font-mono text-gray-400 overflow-x-auto whitespace-pre-wrap">
              {fileDiff.diff || t('diff.newFile')}
            </pre>
          </div>
        ))}
      </div>
    </div>
  )
}
