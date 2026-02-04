const BASE_URL = '/api'

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(`${BASE_URL}${path}`, init)
  if (!resp.ok) {
    const body = await resp.json().catch(() => ({ error: resp.statusText }))
    throw new Error(body.error || resp.statusText)
  }
  return resp.json()
}

export interface ProvidersResponse {
  llm: string[]
  tts: string[]
  stt: string[]
}

export interface CodeTreeResponse {
  files: string[]
}

export interface CodeFileResponse {
  path: string
  content: string
}

export const api = {
  getHealth: () => fetchJSON<{ status: string }>('/health'),

  getProviders: () => fetchJSON<ProvidersResponse>('/providers'),

  synthesize: async (text: string, provider?: string): Promise<Blob> => {
    const resp = await fetch(`${BASE_URL}/tts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text, provider }),
    })
    if (!resp.ok) throw new Error('TTS failed')
    return resp.blob()
  },

  transcribe: async (audio: Blob, provider?: string): Promise<string> => {
    const form = new FormData()
    form.append('audio', audio, 'audio.webm')
    if (provider) form.append('provider', provider)
    const resp = await fetchJSON<{ text: string }>('/stt', {
      method: 'POST',
      body: form,
    })
    return resp.text
  },

  getCodeTree: () => fetchJSON<CodeTreeResponse>('/code/tree'),

  getCodeFile: (path: string) =>
    fetchJSON<CodeFileResponse>(`/code/file?path=${encodeURIComponent(path)}`),
}
