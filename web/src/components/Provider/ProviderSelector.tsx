import { ChevronDown } from 'lucide-react'

interface Props {
  label: string
  providers: string[]
  selected: string
  onChange: (value: string) => void
}

export function ProviderSelector({ label, providers, selected, onChange }: Props) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-gray-500">{label}:</span>
      <div className="relative">
        <select
          className="appearance-none rounded-lg border border-gray-700 bg-gray-800 px-3 py-1 pr-7 text-xs text-gray-300 focus:border-blue-500 focus:outline-none"
          value={selected}
          onChange={e => onChange(e.target.value)}
        >
          {providers.map(p => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-3 w-3 -translate-y-1/2 text-gray-500" />
      </div>
    </div>
  )
}
