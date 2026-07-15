import { Plus, X } from 'lucide-react'
import { Input } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'

export interface EditableOption {
  value: string
  label: string
}

/** Редактор вариантов ответа для SELECT/RADIO. value генерируется из label. */
export function OptionsEditor({
  options,
  onChange,
}: {
  options: EditableOption[]
  onChange: (next: EditableOption[]) => void
}) {
  function update(i: number, label: string) {
    const next = options.slice()
    next[i] = { label, value: slugValue(label) || String(i + 1) }
    onChange(next)
  }
  function add() {
    onChange([...options, { value: '', label: '' }])
  }
  function remove(i: number) {
    onChange(options.filter((_, idx) => idx !== i))
  }

  return (
    <div className="flex flex-col gap-2">
      <span className="text-[14px] font-medium text-ink">Варианты ответа</span>
      {options.map((o, i) => (
        <div key={i} className="flex items-center gap-2">
          <Input
            value={o.label}
            onChange={(e) => update(i, e.target.value)}
            placeholder={`Вариант ${i + 1}`}
          />
          <button
            type="button"
            onClick={() => remove(i)}
            className="shrink-0 rounded-md p-2 text-muted hover:bg-danger/10 hover:text-danger"
            aria-label="Удалить вариант"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      ))}
      <Button type="button" variant="ghost" size="sm" onClick={add} className="self-start">
        <Plus className="h-4 w-4" /> Добавить вариант
      </Button>
    </div>
  )
}

function slugValue(s: string): string {
  return s
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9а-яё]+/gi, '_')
    .replace(/^_+|_+$/g, '')
}
