import { useEffect, useState } from 'react'
import { Check, ExternalLink, Image, Link2, X } from 'lucide-react'
import { toast } from 'sonner'
import { getAdminTaskAssetURL } from '@/entities/event-task/api'
import { useModerateTask, useTaskSubmission } from '@/entities/event-task/queries'
import type { TaskAsset, TaskSubmission } from '@/entities/event-task/types'
import { formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Textarea } from '@/shared/ui/input'
import { ErrorState, Skeleton } from '@/shared/ui/states'

async function openAsset(contestId: string, submissionId: string, asset: TaskAsset) {
  if (asset.type === 'LINK' && asset.url) {
    window.open(asset.url, '_blank', 'noopener,noreferrer')
    return
  }
  try {
    const { download_url } = await getAdminTaskAssetURL(contestId, submissionId, asset.id)
    window.open(download_url, '_blank', 'noopener,noreferrer')
  } catch {
    toast.error('Не удалось открыть изображение')
  }
}

export function TaskModerationDialog({
  contestId,
  submission,
  onClose,
}: {
  contestId: string
  submission: TaskSubmission | null
  onClose: () => void
}) {
  const detail = useTaskSubmission(contestId, submission?.id ?? null)
  const moderate = useModerateTask(contestId)
  const [comment, setComment] = useState('')

  useEffect(() => setComment(''), [submission?.id])

  async function run(action: 'approve' | 'reject') {
    if (!submission) return
    if (action === 'reject' && comment.trim().length < 3) {
      toast.error('Укажите причину отказа')
      return
    }
    try {
      const result = await moderate.mutateAsync({ submissionId: submission.id, action, comment })
      toast.success(
        result.replayed
          ? 'Решение уже было сохранено'
          : action === 'approve'
            ? `Подтверждено: +${result.submission.points} баллов`
            : 'Отправлено на доработку',
      )
      onClose()
    } catch {
      toast.error('Не удалось сохранить решение')
    }
  }

  const value = detail.data
  const current = value?.attempts?.find((item) => item.attempt_number === value.current_attempt)
  return (
    <Dialog open={!!submission} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        className="max-h-[90vh] max-w-2xl overflow-y-auto"
        title="Проверка подтверждения"
        description={
          submission ? `${submission.participant_name} · ${submission.task_title}` : undefined
        }
      >
        {detail.isLoading && <Skeleton className="h-48 w-full" />}
        {detail.isError && <ErrorState onRetry={() => detail.refetch()} />}
        {value && current && (
          <div className="flex flex-col gap-5">
            <div className="flex flex-wrap gap-2">
              <Badge tone="warning">Попытка №{current.attempt_number}</Badge>
              <Badge tone="brand">+{value.points} баллов</Badge>
              <span className="text-[13px] text-muted">{formatDateTime(current.submitted_at)}</span>
            </div>
            {current.participant_comment && (
              <div className="rounded-[12px] bg-surface-2 p-4">
                <p className="text-[12px] font-medium uppercase tracking-wide text-muted">
                  Комментарий участника
                </p>
                <p className="mt-1 whitespace-pre-wrap text-[14px] text-ink">
                  {current.participant_comment}
                </p>
              </div>
            )}
            <div>
              <p className="mb-2 text-[14px] font-medium text-ink">Материалы</p>
              <div className="grid gap-2 sm:grid-cols-2">
                {current.assets.map((asset) => (
                  <button
                    key={asset.id}
                    type="button"
                    onClick={() => openAsset(contestId, value.id, asset)}
                    className="flex items-center gap-3 rounded-[12px] border border-border p-3 text-left hover:border-brand"
                  >
                    {asset.type === 'IMAGE' ? (
                      <Image className="h-5 w-5 text-brand" />
                    ) : (
                      <Link2 className="h-5 w-5 text-brand" />
                    )}
                    <span className="min-w-0 flex-1 truncate text-[13px] text-ink">
                      {asset.original_name ?? asset.url}
                    </span>
                    <ExternalLink className="h-4 w-4 text-muted" />
                  </button>
                ))}
              </div>
            </div>
            {value.status === 'PENDING' ? (
              <>
                <div>
                  <label
                    htmlFor="moderation-comment"
                    className="mb-1.5 block text-[14px] font-medium text-ink"
                  >
                    Комментарий модератора
                  </label>
                  <Textarea
                    id="moderation-comment"
                    value={comment}
                    onChange={(e) => setComment(e.target.value)}
                    maxLength={2000}
                    placeholder="Для отказа причина обязательна"
                  />
                </div>
                <div className="flex flex-wrap justify-end gap-2">
                  <Button
                    variant="secondary"
                    loading={moderate.isPending}
                    onClick={() => run('reject')}
                  >
                    <X className="h-4 w-4 text-danger" /> На доработку
                  </Button>
                  <Button loading={moderate.isPending} onClick={() => run('approve')}>
                    <Check className="h-4 w-4" /> Подтвердить
                  </Button>
                </div>
              </>
            ) : (
              <div className="rounded-[12px] bg-surface-2 p-4 text-[14px] text-ink">
                Решение: {value.status === 'APPROVED' ? 'подтверждено' : 'отправлено на доработку'}.
                {value.moderator_comment && (
                  <p className="mt-1 text-muted">{value.moderator_comment}</p>
                )}
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
