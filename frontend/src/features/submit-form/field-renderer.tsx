import { Info } from 'lucide-react'
import { Input, Textarea } from '@/shared/ui/input'
import { Field } from '@/shared/ui/field'
import { Checkbox, RadioGroup, RadioItem } from '@/shared/ui/choice'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'
import { FileUpload, type UploadedFile } from '@/shared/ui/file-upload'
import type { FormField } from '@/entities/challenge/types'
import type { AnswerValue } from '@/entities/submission/types'

interface Props {
  field: FormField
  value: AnswerValue
  files: UploadedFile[]
  error?: string
  disabled?: boolean
  onChange: (value: AnswerValue) => void
  onAddFiles: (files: FileList) => void
  onRemoveFile: (id: string) => void
}

// Динамический рендер поля по типу (SITE.md §11). Форма не захардкожена.
export function FieldRenderer({
  field,
  value,
  files,
  error,
  disabled,
  onChange,
  onAddFiles,
  onRemoveFile,
}: Props) {
  // Небизнесовые типы — оформление секций/инфоблоков.
  if (field.type === 'SECTION') {
    return (
      <div className="col-span-full mt-2 border-b border-border pb-2">
        <h3 className="text-[18px] font-semibold text-ink">{field.label}</h3>
      </div>
    )
  }
  if (field.type === 'INFO_BLOCK') {
    return (
      <div className="col-span-full flex gap-3 rounded-card bg-brand-subtle p-4">
        <Info className="mt-0.5 h-5 w-5 shrink-0 text-brand" />
        <div>
          <p className="text-[15px] font-medium text-ink">{field.label}</p>
          {field.description && (
            <p className="mt-0.5 text-[14px] text-muted">{field.description}</p>
          )}
        </div>
      </div>
    )
  }

  const wide = ['LONG_TEXT', 'FILE_GROUP', 'RADIO'].includes(field.type)

  return (
    <Field
      label={field.label}
      helpText={field.help_text}
      description={field.type === 'FILE_GROUP' ? field.description : undefined}
      required={field.required}
      error={error}
      className={wide ? 'col-span-full' : ''}
    >
      {(p) => (
        <FieldControl
          field={field}
          value={value}
          files={files}
          disabled={disabled}
          onChange={onChange}
          onAddFiles={onAddFiles}
          onRemoveFile={onRemoveFile}
          controlProps={p}
        />
      )}
    </Field>
  )
}

function FieldControl({
  field,
  value,
  files,
  disabled,
  onChange,
  onAddFiles,
  onRemoveFile,
  controlProps,
}: Omit<Props, 'error'> & { controlProps: { id: string; 'aria-invalid': boolean } }) {
  switch (field.type) {
    case 'LONG_TEXT':
      return (
        <Textarea
          {...controlProps}
          disabled={disabled}
          placeholder={field.placeholder}
          value={(value as string) ?? ''}
          onChange={(e) => onChange(e.target.value)}
        />
      )
    case 'NUMBER':
      return (
        <Input
          {...controlProps}
          type="number"
          disabled={disabled}
          placeholder={field.placeholder}
          value={(value as number | undefined) ?? ''}
          onChange={(e) => onChange(e.target.value === '' ? undefined : Number(e.target.value))}
        />
      )
    case 'CHECKBOX':
      return (
        <label className="flex items-center gap-2.5 pt-1">
          <Checkbox
            id={controlProps.id}
            disabled={disabled}
            checked={!!value}
            onCheckedChange={(c) => onChange(!!c)}
          />
          <span className="text-[15px] text-ink">Да</span>
        </label>
      )
    case 'RADIO':
      return (
        <RadioGroup
          disabled={disabled}
          value={(value as string) ?? ''}
          onValueChange={onChange}
          className="flex flex-col gap-2.5 pt-1"
        >
          {field.options?.map((o) => (
            <label key={o.value} className="flex items-center gap-2.5">
              <RadioItem value={o.value} id={`${field.id}-${o.value}`} />
              <span className="text-[15px] text-ink">{o.label}</span>
            </label>
          ))}
        </RadioGroup>
      )
    case 'SELECT':
      return (
        <Select disabled={disabled} value={(value as string) ?? ''} onValueChange={onChange}>
          <SelectTrigger id={controlProps.id} aria-invalid={controlProps['aria-invalid']}>
            <SelectValue placeholder="Выберите…" />
          </SelectTrigger>
          <SelectContent>
            {field.options?.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )
    case 'FILE_GROUP': {
      const exts = field.settings?.allowed_extensions
      const formatsHint =
        exts && exts.length > 0 ? `Разрешено: ${exts.join(', ')}` : 'Разрешены любые форматы'
      const maxMb = field.settings?.max_file_size_mb
      const sizeHint = maxMb ? `до ${maxMb} МБ` : undefined
      return (
        <FileUpload
          files={files}
          hint={[formatsHint, sizeHint].filter(Boolean).join(', ')}
          accept={exts?.map((e) => `.${e}`).join(',')}
          onAdd={onAddFiles}
          onRemove={onRemoveFile}
        />
      )
    }
    default:
      // SHORT_TEXT, URL, EMAIL, PHONE, DATE
      return (
        <Input
          {...controlProps}
          disabled={disabled}
          type={
            field.type === 'DATE'
              ? 'date'
              : field.type === 'EMAIL'
                ? 'email'
                : field.type === 'URL'
                  ? 'url'
                  : field.type === 'PHONE'
                    ? 'tel'
                    : 'text'
          }
          placeholder={field.placeholder}
          value={(value as string) ?? ''}
          onChange={(e) => onChange(e.target.value)}
        />
      )
  }
}
