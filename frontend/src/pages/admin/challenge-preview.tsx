import { FieldRenderer } from '@/features/submit-form/field-renderer'
import type { AdminField } from '@/entities/challenge/admin-types'
import type { FormField, FieldOption } from '@/entities/challenge/types'

/** Маппинг админ-поля в контестант-FormField для read-only preview. */
function toFormField(f: AdminField): FormField {
  const s = f.settings ?? {}
  const options = Array.isArray(s.options) ? (s.options as FieldOption[]) : undefined
  return {
    id: f.id,
    key: f.key,
    type: f.type,
    label: f.label,
    description: f.description ?? undefined,
    help_text: f.help_text ?? undefined,
    placeholder: f.placeholder ?? undefined,
    required: f.required,
    sort_order: f.sort_order,
    options,
    settings: {
      multiple: s.multiple as boolean | undefined,
      allowed_extensions: s.allowed_extensions as string[] | undefined,
      max_file_size_mb: s.max_file_size_mb as number | undefined,
    },
  }
}

/** Превью формы глазами конкурсанта (всё отключено). */
export function ChallengePreview({ fields }: { fields: AdminField[] }) {
  if (fields.length === 0) {
    return <p className="text-[14px] text-muted">Добавьте поля, чтобы увидеть превью формы.</p>
  }
  return (
    <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
      {fields.map((f) => (
        <FieldRenderer
          key={f.id}
          field={toFormField(f)}
          value={undefined}
          files={[]}
          disabled
          onChange={() => {}}
          onAddFiles={() => {}}
          onRemoveFile={() => {}}
        />
      ))}
    </div>
  )
}
