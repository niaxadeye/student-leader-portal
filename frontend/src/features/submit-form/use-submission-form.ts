import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Challenge } from '@/entities/challenge/types'
import type { AnswerValue } from '@/entities/submission/types'
import type { SubmissionDTO } from '@/entities/submission/api'
import type { UploadedFile } from '@/shared/ui/file-upload'
import { useSaveDraft, useUploadFile, useDeleteFile } from '@/entities/submission/queries'

export type SaveState = 'idle' | 'saving' | 'saved'

// Группирует файлы работы по ключу поля для FieldRenderer.
function filesByKey(sub: SubmissionDTO | undefined): Record<string, UploadedFile[]> {
  const out: Record<string, UploadedFile[]> = {}
  for (const f of sub?.files ?? []) {
    const item: UploadedFile = {
      id: f.file_id,
      name: f.original_name,
      size: f.size_bytes ?? 0,
      status: 'READY',
      progress: 100,
    }
    ;(out[f.field_key] ??= []).push(item)
  }
  return out
}

// Состояние заполнения формы поверх реального submission: автосейв черновика + загрузка файлов.
export function useSubmissionForm(challenge: Challenge | undefined, sub: SubmissionDTO | undefined) {
  const challengeId = challenge?.id ?? ''
  const saveMut = useSaveDraft(challengeId)
  const uploadMut = useUploadFile(challengeId)
  const deleteMut = useDeleteFile(challengeId)

  const [answers, setAnswers] = useState<Record<string, AnswerValue>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [saveState, setSaveState] = useState<SaveState>('idle')
  // Локально загружаемые файлы (оптимистично, до ответа сервера), по ключу поля.
  const [pending, setPending] = useState<Record<string, UploadedFile[]>>({})
  const idByKey = useMemo(() => {
    const m: Record<string, string> = {}
    challenge?.fields.forEach((f) => (m[f.key] = f.id))
    return m
  }, [challenge])

  // Инициализация ответов из submission (при первой загрузке / смене работы).
  const loadedFor = useRef<string>()
  useEffect(() => {
    if (sub && loadedFor.current !== sub.id) {
      setAnswers(sub.answers ?? {})
      loadedFor.current = sub.id
    }
  }, [sub])

  const serverFiles = useMemo(() => filesByKey(sub), [sub])
  // Слияние серверных и ещё-загружающихся файлов.
  const files = useMemo(() => {
    const merged: Record<string, UploadedFile[]> = { ...serverFiles }
    for (const [k, arr] of Object.entries(pending)) {
      merged[k] = [...(merged[k] ?? []), ...arr]
    }
    return merged
  }, [serverFiles, pending])

  const setAnswer = useCallback((key: string, value: AnswerValue) => {
    setAnswers((prev) => ({ ...prev, [key]: value }))
    setErrors((prev) => (prev[key] ? { ...prev, [key]: '' } : prev))
    // Правка поля означает несохранённые изменения — сбрасываем индикатор «сохранено».
    setSaveState('idle')
  }, [])

  const saveNow = useCallback(() => {
    setSaveState('saving')
    return saveMut
      .mutateAsync(answers)
      .then(() => setSaveState('saved'))
      .catch((e) => {
        setSaveState('idle')
        throw e
      })
  }, [answers, saveMut])

  const addFiles = useCallback(
    (key: string, list: FileList) => {
      const fieldId = idByKey[key]
      if (!fieldId) return
      Array.from(list).forEach((file, i) => {
        const tmpId = `pending-${key}-${file.size}-${i}`
        const optimistic: UploadedFile = {
          id: tmpId,
          name: file.name,
          size: file.size,
          status: 'UPLOADING',
          progress: 50,
        }
        setPending((p) => ({ ...p, [key]: [...(p[key] ?? []), optimistic] }))
        uploadMut.mutate(
          { fieldId, file },
          {
            onSettled: () =>
              setPending((p) => ({
                ...p,
                [key]: (p[key] ?? []).filter((f) => f.id !== tmpId),
              })),
          },
        )
      })
    },
    [idByKey, uploadMut],
  )

  const removeFile = useCallback(
    (_key: string, id: string) => {
      if (id.startsWith('pending-')) return
      deleteMut.mutate(id)
    },
    [deleteMut],
  )

  const hasUploading = useMemo(
    () => Object.values(pending).some((arr) => arr.length > 0),
    [pending],
  )

  const validate = useCallback((): boolean => {
    if (!challenge) return false
    const next: Record<string, string> = {}
    for (const field of challenge.fields) {
      if (!field.required) continue
      if (field.type === 'FILE_GROUP') {
        if (!(files[field.key]?.some((f) => f.status === 'READY')))
          next[field.key] = 'Загрузите хотя бы один файл'
      } else if (field.type === 'CHECKBOX') {
        if (!answers[field.key]) next[field.key] = 'Обязательное поле'
      } else {
        const v = answers[field.key]
        if (v === undefined || v === '') next[field.key] = 'Заполните поле'
      }
    }
    setErrors(next)
    return Object.keys(next).length === 0
  }, [challenge, answers, files])

  return {
    answers,
    files,
    errors,
    saveState,
    hasUploading,
    setAnswer,
    addFiles,
    removeFile,
    validate,
    saveNow,
    setErrors,
  }
}
