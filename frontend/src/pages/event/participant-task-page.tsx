import { useState, type FormEvent } from 'react'
import { ArrowLeft, CheckCircle2, ExternalLink, Image, Link2, Send, Upload } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import { getParticipantTaskAssetURL } from '@/entities/event-task/api'
import { useParticipantTask, useSubmitParticipantTask } from '@/entities/event-task/queries'
import type { TaskAsset } from '@/entities/event-task/types'
import { formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody, CardHeader, CardTitle } from '@/shared/ui/card'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { ErrorState, Skeleton } from '@/shared/ui/states'

async function openAsset(asset: TaskAsset) {
  if (asset.type === 'LINK' && asset.url) {
    window.open(asset.url, '_blank', 'noopener,noreferrer')
    return
  }
  try {
    const { download_url } = await getParticipantTaskAssetURL(asset.id)
    window.open(download_url, '_blank', 'noopener,noreferrer')
  } catch {
    toast.error('Не удалось открыть изображение')
  }
}

export function ParticipantTaskPage() {
  const { taskId } = useParams()
  const { session } = useParticipantAuth()
  const task = useParticipantTask(session?.event.id, session?.participant.id, taskId)
  const submit = useSubmitParticipantTask(
    session?.event.id ?? '',
    session?.participant.id ?? '',
    taskId ?? '',
  )
  const [comment, setComment] = useState('')
  const [links, setLinks] = useState('')
  const [images, setImages] = useState<File[]>([])

  if (!session) return null
  const back = `/event/${encodeURIComponent(session.event.slug)}/tasks`

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (!task.data) return
    const normalizedLinks = links
      .split(/\r?\n/)
      .map((value) => value.trim())
      .filter(Boolean)
    if (images.length + normalizedLinks.length === 0) {
      toast.error('Добавьте хотя бы одно изображение или ссылку')
      return
    }
    if (images.length > 10 || normalizedLinks.length > 10) {
      toast.error('Можно добавить не более 10 изображений и 10 ссылок')
      return
    }
    const form = new FormData()
    form.append('participant_comment', comment)
    normalizedLinks.forEach((value) => form.append('links', value))
    images.forEach((value) => form.append('images', value))
    try {
      await submit.mutateAsync(form)
      setComment('')
      setLinks('')
      setImages([])
      toast.success('Подтверждение отправлено на проверку')
    } catch {
      toast.error('Не удалось отправить. Проверьте ссылки, изображения и статус задания')
    }
  }

  if (task.isLoading) return <Skeleton className="h-80 w-full" />
  if (task.isError || !task.data) return <ErrorState onRetry={() => task.refetch()} />
  const value = task.data
  const submission = value.submission
  const canSubmit = value.available && (!submission || submission.status === 'REJECTED')

  return (
    <div className="flex flex-col gap-5">
      <Link
        to={back}
        className="inline-flex items-center gap-1 text-[14px] text-muted hover:text-ink"
      >
        <ArrowLeft className="h-4 w-4" /> Все задания
      </Link>
      <Card className="overflow-hidden">
        {value.image_url && (
          <img src={value.image_url} alt="" className="max-h-72 w-full object-cover" />
        )}
        <CardBody className="p-6 sm:p-8">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="flex items-center gap-3">
              <div className="flex h-12 w-12 items-center justify-center rounded-[14px] bg-brand-subtle text-2xl">
                {value.icon || '🎯'}
              </div>
              <div>
                <h1 className="text-[26px] font-bold text-ink">{value.title}</h1>
                <p className="mt-0.5 text-[13px] text-muted">
                  {value.ends_at
                    ? `Доступно до ${formatDateTime(value.ends_at)}`
                    : 'Без ограничения по сроку'}
                </p>
              </div>
            </div>
            <Badge tone="brand" className="px-3 py-1 text-[14px]">
              +{value.points} баллов
            </Badge>
          </div>
          <p className="mt-6 whitespace-pre-wrap text-[15px] leading-relaxed text-ink">
            {value.description}
          </p>
          <p className="mt-4 text-[12px] text-muted">
            Подтверждение:{' '}
            {value.allowed_submission_types
              .map((type) => (type === 'IMAGE' ? 'изображения' : 'ссылки'))
              .join(', ')}
          </p>
        </CardBody>
      </Card>

      {submission && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-3">
            <CardTitle>Статус выполнения</CardTitle>
            <Badge
              tone={
                submission.status === 'APPROVED'
                  ? 'success'
                  : submission.status === 'REJECTED'
                    ? 'danger'
                    : 'warning'
              }
            >
              {submission.status === 'APPROVED'
                ? 'Выполнено'
                : submission.status === 'REJECTED'
                  ? 'Нужна доработка'
                  : 'На проверке'}
            </Badge>
          </CardHeader>
          <CardBody className="flex flex-col gap-4">
            {submission.status === 'APPROVED' && (
              <div className="flex items-center gap-2 rounded-[12px] bg-success/[0.12] p-4 text-[14px] text-success-dark">
                <CheckCircle2 className="h-5 w-5" /> Начислено {submission.points} баллов
              </div>
            )}
            {submission.moderator_comment && (
              <div
                className={`rounded-[12px] p-4 text-[14px] ${submission.status === 'REJECTED' ? 'bg-danger/10 text-danger' : 'bg-surface-2 text-ink'}`}
              >
                <p className="text-[12px] font-medium uppercase tracking-wide opacity-70">
                  Комментарий модератора
                </p>
                <p className="mt-1 whitespace-pre-wrap">{submission.moderator_comment}</p>
              </div>
            )}
            {!!submission.attempts?.length && (
              <div>
                <p className="mb-2 text-[13px] font-medium text-muted">История отправок</p>
                <div className="flex flex-col gap-3">
                  {submission.attempts.map((attempt) => (
                    <div key={attempt.id} className="rounded-[12px] border border-border p-4">
                      <div className="flex items-center justify-between gap-3">
                        <p className="text-[14px] font-medium text-ink">
                          Попытка №{attempt.attempt_number}
                        </p>
                        <span className="text-[12px] text-muted">
                          {formatDateTime(attempt.submitted_at)}
                        </span>
                      </div>
                      {attempt.participant_comment && (
                        <p className="mt-2 whitespace-pre-wrap text-[13px] text-muted">
                          {attempt.participant_comment}
                        </p>
                      )}
                      <div className="mt-3 flex flex-wrap gap-2">
                        {attempt.assets.map((asset) => (
                          <button
                            key={asset.id}
                            type="button"
                            onClick={() => openAsset(asset)}
                            className="inline-flex max-w-full items-center gap-2 rounded-[9px] bg-surface-2 px-3 py-2 text-[12px] text-ink hover:text-brand"
                          >
                            {asset.type === 'IMAGE' ? (
                              <Image className="h-4 w-4" />
                            ) : (
                              <Link2 className="h-4 w-4" />
                            )}
                            <span className="truncate">{asset.original_name ?? asset.url}</span>
                            <ExternalLink className="h-3 w-3" />
                          </button>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      {canSubmit && (
        <Card>
          <CardHeader>
            <CardTitle>
              {submission?.status === 'REJECTED' ? 'Отправить повторно' : 'Подтвердить выполнение'}
            </CardTitle>
          </CardHeader>
          <CardBody>
            <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
              <Field label="Комментарий" description="Кратко расскажите, что было сделано.">
                {(props) => (
                  <Textarea
                    {...props}
                    value={comment}
                    onChange={(e) => setComment(e.target.value)}
                    maxLength={2000}
                  />
                )}
              </Field>
              {value.allowed_submission_types.includes('LINK') && (
                <Field label="Ссылки" description="По одной ссылке http(s) в строке.">
                  {(props) => (
                    <Textarea
                      {...props}
                      value={links}
                      onChange={(e) => setLinks(e.target.value)}
                      placeholder="https://..."
                    />
                  )}
                </Field>
              )}
              {value.allowed_submission_types.includes('IMAGE') && (
                <Field
                  label="Изображения"
                  description="До 10 файлов JPG, PNG, WEBP или GIF, каждый до 20 МБ."
                >
                  {(props) => (
                    <Input
                      {...props}
                      type="file"
                      multiple
                      accept="image/jpeg,image/png,image/webp,image/gif"
                      onChange={(e) => setImages(Array.from(e.target.files ?? []))}
                    />
                  )}
                </Field>
              )}
              <Button type="submit" loading={submit.isPending} className="self-start">
                <Upload className="h-4 w-4" /> <Send className="h-4 w-4" /> Отправить на проверку
              </Button>
            </form>
          </CardBody>
        </Card>
      )}
      {!value.available && !submission && (
        <div className="rounded-card border border-border bg-surface p-5 text-[14px] text-muted">
          Сейчас приём подтверждений закрыт.
        </div>
      )}
    </div>
  )
}
