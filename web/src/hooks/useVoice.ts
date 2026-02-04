import { useState, useCallback, useRef } from 'react'
import { api } from '../services/api'

interface UseVoiceOptions {
  onTranscript: (text: string) => void
  sttProvider?: string
}

export function useVoice({ onTranscript, sttProvider }: UseVoiceOptions) {
  const [isRecording, setIsRecording] = useState(false)
  const [isTranscribing, setIsTranscribing] = useState(false)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const chunksRef = useRef<Blob[]>([])

  const startRecording = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mediaRecorder = new MediaRecorder(stream, {
        mimeType: 'audio/webm;codecs=opus',
      })

      chunksRef.current = []
      mediaRecorder.ondataavailable = (e) => {
        if (e.data.size > 0) {
          chunksRef.current.push(e.data)
        }
      }

      mediaRecorder.onstop = async () => {
        stream.getTracks().forEach(t => t.stop())
        const blob = new Blob(chunksRef.current, { type: 'audio/webm' })
        if (blob.size > 0) {
          setIsTranscribing(true)
          try {
            const text = await api.transcribe(blob, sttProvider)
            if (text) onTranscript(text)
          } catch (err) {
            console.error('Transcription failed:', err)
          } finally {
            setIsTranscribing(false)
          }
        }
      }

      mediaRecorderRef.current = mediaRecorder
      mediaRecorder.start()
      setIsRecording(true)
    } catch (err) {
      console.error('Failed to start recording:', err)
    }
  }, [onTranscript, sttProvider])

  const stopRecording = useCallback(() => {
    mediaRecorderRef.current?.stop()
    mediaRecorderRef.current = null
    setIsRecording(false)
  }, [])

  return { isRecording, isTranscribing, startRecording, stopRecording }
}
