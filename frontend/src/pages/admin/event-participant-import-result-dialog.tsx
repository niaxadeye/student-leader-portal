import type {
  ParticipantImportResult,
  ParticipantImportRowStatus,
} from '@/entities/event-participant/admin-types'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent } from '@/shared/ui/dialog'

const rowStatusMeta: Record<
  ParticipantImportRowStatus,
  { label: string; tone: 'success' | 'brand' | 'danger' | 'warning' }
> = {
  added: { label: 'Добавлен', tone: 'success' },
  updated: { label: 'Обновлён', tone: 'brand' },
  error: { label: 'Ошибка', tone: 'danger' },
  duplicate: { label: 'Дубликат', tone: 'warning' },
}

export function EventParticipantImportResultDialog({
  result,
  onOpenChange,
}: {
  result: ParticipantImportResult | null
  onOpenChange: (open: boolean) => void
}) {
  return (
    <Dialog open={result !== null} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-3xl"
        title="Результат импорта"
        description="Колонка «Направление» необязательна: пустая ячейка не снимает уже назначенное направление, новое название попадает в каталог."
      >
        {result && (
          <div className="flex flex-col gap-5">
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              <Summary value={result.added} label="Добавлено" tone="text-success" />
              <Summary value={result.updated} label="Обновлено" tone="text-brand" />
              <Summary value={result.errors} label="Ошибок" tone="text-danger" />
              <Summary value={result.duplicates} label="Дубликатов" tone="text-amber-700" />
            </div>

            <div className="max-h-[50vh] overflow-auto rounded-[12px] border border-border">
              <table className="w-full min-w-[600px] text-left text-[13px]">
                <thead className="sticky top-0 bg-surface-2 text-[11px] uppercase tracking-wide text-muted-2">
                  <tr>
                    <th className="px-3 py-2 font-medium">Строка</th>
                    <th className="px-3 py-2 font-medium">ФИО</th>
                    <th className="px-3 py-2 font-medium">Направление</th>
                    <th className="px-3 py-2 font-medium">Результат</th>
                    <th className="px-3 py-2 font-medium">Комментарий</th>
                  </tr>
                </thead>
                <tbody>
                  {result.rows.map((row) => {
                    const status = rowStatusMeta[row.status]
                    return (
                      <tr
                        key={`${row.line}-${row.participant_id ?? row.full_name}`}
                        className="border-t border-border"
                      >
                        <td className="whitespace-nowrap px-3 py-2 text-muted">{row.line}</td>
                        <td className="px-3 py-2 font-medium text-ink">{row.full_name || '—'}</td>
                        <td className="px-3 py-2 text-muted">{row.direction || '—'}</td>
                        <td className="px-3 py-2">
                          <Badge tone={status.tone}>{status.label}</Badge>
                        </td>
                        <td className="px-3 py-2 text-muted">{row.message || '—'}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>

            <div className="flex justify-end">
              <Button onClick={() => onOpenChange(false)}>Готово</Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function Summary({ value, label, tone }: { value: number; label: string; tone: string }) {
  return (
    <div className="rounded-[12px] bg-surface-2 p-3 text-center">
      <p className={`text-[24px] font-bold ${tone}`}>{value}</p>
      <p className="text-[12px] text-muted">{label}</p>
    </div>
  )
}
