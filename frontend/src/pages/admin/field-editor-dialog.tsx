import { useEffect, useState } from 'react'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { Checkbox } from '@/shared/ui/choice'
import { Button } from '@/shared/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'
import { toast } from 'sonner'
import { ApiRequestError } from '@/shared/api/client'
import { useAddField, useUpdateField } from '@/entities/challenge/admin-queries'
import type { AdminField, FieldInput } from '@/entities/challenge/admin-types'
import type { FieldType } from '@/entities/challenge/types'
import { fieldTypeLabels } from './challenge-status'
import { OptionsEditor, type EditableOption } from './options-editor'

const TYPES_WITH_OPTIONS: FieldType[] = ['SELECT', 'RADIO']
const TYPES_WITHOUT_INPUT: FieldType[] = ['SECTION', 'INFO_BLOCK']

interface Props {
  challengeId: string
  field: AdminField | null // null → создание
  open: boolean
  onOpenChange: (v: boolean) => void
}

export function FieldEditorDialog({ challengeId, field, open, onOpenChange }: Props) {
  const [key, setKey] = useState('')
  const [type, setType] = useState<FieldType>('SHORT_TEXT')
  const [label, setLabel] = useState('')
  const [helpText, setHelpText] = useState('')
  const [placeholder, setPlaceholder] = useState('')
  const [required, setRequired] = useState(false)
  const [options, setOptions] = useState<EditableOption[]>([])
  const [multiple, setMultiple] = useState(false)
  const [extensions, setExtensions] = useState('')
  const [maxFileSizeMb, setMaxFileSizeMb] = useState('')
  const [error, setError] = useState<string>()

  const add = useAddField(challengeId)
  const update = useUpdateField(challengeId)
  const pending = add.isPending || update.isPending

  // Заполняем форму при открытии (редактирование) или сбрасываем (создание).
  useEffect(() => {
    if (!open) return
    setError(undefined)
    if (field) {
      setKey(field.key)
      setType(field.type)
      setLabel(field.label)
      setHelpText(field.help_text ?? '')
      setPlaceholder(field.placeholder ?? '')
      setRequired(field.required)
      const s = field.settings ?? {}
      setOptions(Array.isArray(s.options) ? (s.options as EditableOption[]) : [])
      setMultiple(!!s.multiple)
      setExtensions(Array.isArray(s.allowed_extensions) ? s.allowed_extensions.join(', ') : '')
      setMaxFileSizeMb(typeof s.max_file_size_mb === 'number' ? String(s.max_file_size_mb) : '')
    } else {
      setKey('')
      setType('SHORT_TEXT')
      setLabel('')
      setHelpText('')
      setPlaceholder('')
      setRequired(false)
      setOptions([])
      setMultiple(false)
      setExtensions('')
      setMaxFileSizeMb('')
    }
  }, [open, field])

  function buildSettings(): Record<string, unknown> {
    if (TYPES_WITH_OPTIONS.includes(type)) {
      return { options: options.filter((o) => o.label.trim()) }
    }
    if (type === 'FILE_GROUP') {
      const exts = extensions
        .split(',')
        .map((e) => e.trim().replace(/^\./, '').toLowerCase())
        .filter(Boolean)
      const maxMb = Number(maxFileSizeMb)
      return {
        multiple,
        allowed_extensions: exts,
        max_file_size_mb: maxFileSizeMb.trim() && maxMb > 0 ? maxMb : undefined,
      }
    }
    return {}
  }

  function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(undefined)
    if (!label.trim()) {
      setError('Укажите подпись поля.')
      return
    }
    if (!key.trim()) {
      setError('Укажите ключ поля (латиницей, без пробелов).')
      return
    }
    if (TYPES_WITH_OPTIONS.includes(type) && options.filter((o) => o.label.trim()).length === 0) {
      setError('Добавьте хотя бы один вариант ответа.')
      return
    }
    const input: FieldInput = {
      key: key.trim(),
      type,
      label: label.trim(),
      help_text: helpText.trim() || null,
      placeholder: placeholder.trim() || null,
      required,
      settings: buildSettings(),
    }
    const onError = (err: unknown) => {
      if (err instanceof ApiRequestError && err.code === 'FIELD_KEY_TAKEN') {
        setError('Поле с таким ключом уже есть в форме.')
      } else {
        setError('Не удалось сохранить поле.')
      }
    }
    const onSuccess = () => {
      toast.success(field ? 'Поле обновлено' : 'Поле добавлено')
      onOpenChange(false)
    }
    if (field) {
      update.mutate({ fieldId: field.id, input }, { onSuccess, onError })
    } else {
      add.mutate(input, { onSuccess, onError })
    }
  }

  const showOptions = TYPES_WITH_OPTIONS.includes(type)
  const showFile = type === 'FILE_GROUP'
  const showPlaceholder = !TYPES_WITHOUT_INPUT.includes(type) && !showOptions && !showFile

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        title={field ? 'Редактировать поле' : 'Новое поле'}
        description="Настройте поле формы. Тип определяет, как оно отображается конкурсанту."
      >
        <form onSubmit={submit} className="flex flex-col gap-4">
          <Field label="Тип поля">
            {() => (
              <Select value={type} onValueChange={(v) => setType(v as FieldType)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(fieldTypeLabels).map(([value, lbl]) => (
                    <SelectItem key={value} value={value}>
                      {lbl}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </Field>

          <Field label="Подпись" required>
            {(p) => (
              <Input
                {...p}
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                placeholder="Название проекта"
                autoFocus
              />
            )}
          </Field>

          <Field
            label="Ключ"
            required
            helpText="Технический идентификатор ответа. Латиница, цифры, подчёркивание."
            error={error}
          >
            {(p) => (
              <Input
                {...p}
                value={key}
                onChange={(e) => setKey(e.target.value)}
                placeholder="project_name"
              />
            )}
          </Field>

          <Field label="Справка (подсказка)">
            {(p) => (
              <Input
                {...p}
                value={helpText}
                onChange={(e) => setHelpText(e.target.value)}
                placeholder="Короткое пояснение к полю"
              />
            )}
          </Field>

          {showPlaceholder && (
            <Field label="Плейсхолдер">
              {(p) => (
                <Input
                  {...p}
                  value={placeholder}
                  onChange={(e) => setPlaceholder(e.target.value)}
                  placeholder="Текст-подсказка внутри поля"
                />
              )}
            </Field>
          )}

          {showOptions && <OptionsEditor options={options} onChange={setOptions} />}

          {showFile && (
            <>
              <Field
                label="Разрешённые расширения"
                helpText="Через запятую, например: pdf, docx, mp4. Оставьте пустым — разрешены любые форматы"
              >
                {(p) => (
                  <Textarea
                    {...p}
                    value={extensions}
                    onChange={(e) => setExtensions(e.target.value)}
                    placeholder="pdf, pptx, mp4, png, zip"
                  />
                )}
              </Field>
              <Field
                label="Максимальный размер файла, МБ"
                helpText="Оставьте пустым — без ограничения размера"
              >
                {(p) => (
                  <Input
                    {...p}
                    type="number"
                    min={1}
                    max={1048576}
                    value={maxFileSizeMb}
                    onChange={(e) => setMaxFileSizeMb(e.target.value)}
                    placeholder="1024"
                  />
                )}
              </Field>
              <label className="flex items-center gap-2 text-[14px] text-ink">
                <Checkbox checked={multiple} onCheckedChange={(v) => setMultiple(!!v)} />
                Разрешить несколько файлов
              </label>
            </>
          )}

          {!TYPES_WITHOUT_INPUT.includes(type) && (
            <label className="flex items-center gap-2 text-[14px] text-ink">
              <Checkbox checked={required} onCheckedChange={(v) => setRequired(!!v)} />
              Обязательное поле
            </label>
          )}

          {error && <p className="text-[13px] text-danger">{error}</p>}

          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" loading={pending}>
              {field ? 'Сохранить' : 'Добавить'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
