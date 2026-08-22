import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, Download, FileText, Pencil, Search, Trash2, Upload } from 'lucide-react'
import { toast } from 'sonner'
import {
  useAdminBriefing,
  useAdminChallenge,
  useClearBriefingOverride,
  useDeleteBriefingFile,
  useSaveBriefing,
  useSaveBriefingOverride,
  useUploadBriefingFile,
  useUploadOverrideFile,
} from '@/entities/challenge/admin-queries'
import type {
  AdminBriefingContestant,
  AdminBriefingFile,
  OverrideInput,
} from '@/entities/challenge/admin-types'
import { useAdminContest } from '@/entities/contest/queries'
import { canEditContest } from '@/entities/contest/types'
import { ApiRequestError } from '@/shared/api/client'
import { formatBytes, isoToLocalInput, localInputToIso } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { Switch } from '@/shared/ui/switch'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'

function errMsg(e: unknown, fallback: string) {
  return e instanceof ApiRequestError ? e.message : fallback
}

function briefingStatus(publishAt: string | null, visible: boolean, hidden?: boolean) {
  if (hidden) return { label: 'Скрыто', tone: 'neutral' as const }
  if (visible) return { label: 'Видно', tone: 'success' as const }
  if (publishAt && new Date(publishAt).getTime() > Date.now()) {
    return { label: 'Запланировано', tone: 'warning' as const }
  }
  return { label: 'Не опубликовано', tone: 'neutral' as const }
}

export function ChallengeBriefingPage() {
  const { challengeId } = useParams()
  const challengeQ = useAdminChallenge(challengeId)
  const briefingQ = useAdminBriefing(challengeId)
  const { data: contest } = useAdminContest(challengeQ.data?.contest_id)
  const canEdit = canEditContest(contest?.access_level)

  if (challengeQ.isLoading || briefingQ.isLoading) return <Skeleton className="h-64 w-full" />
  if (challengeQ.isError || briefingQ.isError) {
    return <ErrorState onRetry={() => { challengeQ.refetch(); briefingQ.refetch() }} />
  }
  const challenge = challengeQ.data
  const briefing = briefingQ.data
  if (!challenge || !briefing || !challengeId) {
    return <EmptyState title="Испытание не найдено" description="Нет доступа к материалам." />
  }

  return (
    <div className="flex flex-col gap-6">
      <Link
        to={`/admin/challenges/${challenge.id}`}
        className="inline-flex w-fit items-center gap-1 text-[14px] text-muted hover:text-ink"
      >
        <ArrowLeft className="h-4 w-4" /> К испытанию
      </Link>
      <div>
        <h1 className="text-[28px] font-bold tracking-tight text-ink">Материалы для конкурсантов</h1>
        <p className="mt-1 text-[14px] text-muted">{challenge.title}</p>
      </div>
      <DefaultBriefingCard challengeId={challengeId} canEdit={canEdit} />
      <ContestantsCard challengeId={challengeId} canEdit={canEdit} />
    </div>
  )
}

function DefaultBriefingCard({ challengeId, canEdit }: { challengeId: string; canEdit: boolean }) {
  const { data } = useAdminBriefing(challengeId)
  const save = useSaveBriefing(challengeId)
  const upload = useUploadBriefingFile(challengeId)
  const remove = useDeleteBriefingFile(challengeId)
  const [body, setBody] = useState(data?.body_text ?? '')
  const [publishAt, setPublishAt] = useState(isoToLocalInput(data?.publish_at ?? null))

  useEffect(() => {
    if (!data) return
    setBody(data.body_text)
    setPublishAt(isoToLocalInput(data.publish_at))
  }, [data])

  if (!data) return null
  const status = briefingStatus(data.publish_at, Boolean(data.publish_at && new Date(data.publish_at) <= new Date()))

  function persist(nextPublish: string | null, nextBody = body) {
    save.mutate(
      { body_text: nextBody, publish_at: nextPublish },
      {
        onSuccess: () => toast.success('Материалы сохранены'),
        onError: (e) => toast.error(errMsg(e, 'Не удалось сохранить')),
      },
    )
  }

  return (
    <Card>
      <CardBody className="flex flex-col gap-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 className="text-[18px] font-semibold text-ink">Общие материалы</h2>
            <p className="mt-1 text-[13px] text-muted">
              Текст и файлы, которые увидят все, если для человека нет своей выдачи.
            </p>
          </div>
          <Badge tone={status.tone}>{status.label}</Badge>
        </div>
        <Field label="Справочный текст">
          {(p) => (
            <Textarea
              {...p}
              rows={8}
              value={body}
              disabled={!canEdit}
              placeholder="Короткое объявление или длинное ТЗ. Переносы строк сохранятся."
              onChange={(e) => setBody(e.target.value)}
            />
          )}
        </Field>
        <Field label="Время публикации">
          {(p) => (
            <Input
              {...p}
              type="datetime-local"
              value={publishAt}
              disabled={!canEdit}
              onChange={(e) => setPublishAt(e.target.value)}
            />
          )}
        </Field>
        {canEdit && (
          <div className="flex flex-wrap gap-2">
            <Button
              loading={save.isPending}
              onClick={() => persist(localInputToIso(publishAt))}
            >
              Сохранить
            </Button>
            <Button
              variant="secondary"
              disabled={save.isPending}
              onClick={() => persist(new Date().toISOString())}
            >
              Опубликовать сейчас
            </Button>
            <Button
              variant="outline"
              disabled={save.isPending || !data.publish_at}
              onClick={() => persist(null)}
            >
              Снять публикацию
            </Button>
          </div>
        )}
        <FileList
          files={data.files}
          canEdit={canEdit}
          uploading={upload.isPending}
          onAdd={(list) => {
            Array.from(list).forEach((file) =>
              upload.mutate(file, {
                onError: (e) => toast.error(errMsg(e, 'Не удалось загрузить файл')),
              }),
            )
          }}
          onRemove={(id) =>
            remove.mutate(id, { onError: (e) => toast.error(errMsg(e, 'Не удалось удалить файл')) })
          }
        />
      </CardBody>
    </Card>
  )
}

function ContestantsCard({ challengeId, canEdit }: { challengeId: string; canEdit: boolean }) {
  const { data } = useAdminBriefing(challengeId)
  const [q, setQ] = useState('')
  const [editing, setEditing] = useState<AdminBriefingContestant | null>(null)
  const people = useMemo(() => {
    const needle = q.trim().toLowerCase()
    const list = data?.contestants ?? []
    if (!needle) return list
    return list.filter(
      (p) =>
        p.full_name.toLowerCase().includes(needle) ||
        p.login.toLowerCase().includes(needle) ||
        (p.organization ?? '').toLowerCase().includes(needle),
    )
  }, [data, q])

  return (
    <Card>
      <CardBody className="flex flex-col gap-4">
        <div>
          <h2 className="text-[18px] font-semibold text-ink">По конкурсантам</h2>
          <p className="mt-1 text-[13px] text-muted">
            Можно открыть материалы раньше, заменить текст или скрыть выдачу. Так же сообщают жеребьёвку заранее.
          </p>
        </div>
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-2" />
          <Input className="pl-9" placeholder="Поиск" value={q} onChange={(e) => setQ(e.target.value)} />
        </div>
        {people.length === 0 ? (
          <EmptyState title="Нет конкурсантов" description="Сначала добавьте участников в конкурс." />
        ) : (
          <ul className="divide-y divide-border rounded-[12px] border border-border">
            {people.map((p) => {
              const st = briefingStatus(p.publish_at, p.visible, p.override?.hidden)
              return (
                <li key={p.user_id} className="flex items-center justify-between gap-3 px-3 py-2.5">
                  <div className="min-w-0">
                    <p className="truncate text-[14px] font-medium text-ink">{p.full_name}</p>
                    <p className="truncate text-[12px] text-muted">
                      {p.login}
                      {p.organization ? ` · ${p.organization}` : ''}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    {p.personalized && <Badge tone="brand">Своя выдача</Badge>}
                    <Badge tone={st.tone}>{st.label}</Badge>
                    {canEdit && (
                      <Button size="sm" variant="secondary" onClick={() => setEditing(p)}>
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </CardBody>
      {editing && (
        <OverrideDialog
          challengeId={challengeId}
          personId={editing.user_id}
          onClose={() => setEditing(null)}
        />
      )}
    </Card>
  )
}

function OverrideDialog({
  challengeId,
  personId,
  onClose,
}: {
  challengeId: string
  personId: string
  onClose: () => void
}) {
  const { data } = useAdminBriefing(challengeId)
  const person = data?.contestants.find((p) => p.user_id === personId)
  const save = useSaveBriefingOverride(challengeId)
  const clear = useClearBriefingOverride(challengeId)
  const upload = useUploadOverrideFile(challengeId)
  const removeFile = useDeleteBriefingFile(challengeId)
  const o = person?.override
  const [hidden, setHidden] = useState(o?.hidden ?? false)
  const [customText, setCustomText] = useState(o?.custom_text ?? false)
  const [body, setBody] = useState(o?.custom_text ? o.body_text : (data?.body_text ?? ''))
  const [customPublish, setCustomPublish] = useState(o?.custom_publish ?? false)
  const [publishAt, setPublishAt] = useState(
    isoToLocalInput(o?.custom_publish ? o.publish_at : (person?.publish_at ?? data?.publish_at ?? null)),
  )
  const [replaceFiles, setReplaceFiles] = useState(o?.replace_files ?? false)

  if (!person) return null

  function payload(): OverrideInput {
    return {
      hidden,
      custom_text: customText,
      body_text: body,
      custom_publish: customPublish,
      publish_at: customPublish ? localInputToIso(publishAt) : null,
      replace_files: replaceFiles,
    }
  }

  return (
    <Dialog open onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent
        className="max-w-lg"
        title={person.full_name}
        description="Персональная выдача материалов. Пустые настройки можно сбросить к общим."
      >
        <div className="flex max-h-[70vh] flex-col gap-4 overflow-y-auto pr-1">
          <label className="flex items-center justify-between gap-3 text-[14px] text-ink">
            Скрыть материалы
            <Switch checked={hidden} onCheckedChange={setHidden} />
          </label>
          <label className="flex items-center justify-between gap-3 text-[14px] text-ink">
            Свой текст
            <Switch
              checked={customText}
              onCheckedChange={(v) => {
                setCustomText(v)
                if (v && !body) setBody(data?.body_text ?? '')
              }}
            />
          </label>
          {customText && (
            <Textarea rows={6} value={body} onChange={(e) => setBody(e.target.value)} />
          )}
          <label className="flex items-center justify-between gap-3 text-[14px] text-ink">
            Своё время публикации
            <Switch checked={customPublish} onCheckedChange={setCustomPublish} />
          </label>
          {customPublish && (
            <Input type="datetime-local" value={publishAt} onChange={(e) => setPublishAt(e.target.value)} />
          )}
          <label className="flex items-center justify-between gap-3 text-[14px] text-ink">
            Свои файлы вместо общих
            <Switch checked={replaceFiles} onCheckedChange={setReplaceFiles} />
          </label>
          {replaceFiles && (
            <FileList
              files={o?.files ?? []}
              canEdit
              uploading={upload.isPending}
              onAdd={(list) => {
                Array.from(list).forEach((file) =>
                  upload.mutate(
                    { userId: person.user_id, file },
                    { onError: (e) => toast.error(errMsg(e, 'Не удалось загрузить файл')) },
                  ),
                )
              }}
              onRemove={(id) =>
                removeFile.mutate(id, { onError: (e) => toast.error(errMsg(e, 'Не удалось удалить файл')) })
              }
            />
          )}
          <div className="flex flex-wrap justify-end gap-2">
            {person.override && (
              <Button
                variant="outline"
                loading={clear.isPending}
                onClick={() =>
                  clear.mutate(person.user_id, {
                    onSuccess: () => {
                      toast.success('Вернули общую выдачу')
                      onClose()
                    },
                    onError: (e) => toast.error(errMsg(e, 'Не удалось сбросить')),
                  })
                }
              >
                Как у всех
              </Button>
            )}
            <Button
              loading={save.isPending}
              onClick={() =>
                save.mutate(
                  { userId: person.user_id, input: payload() },
                  {
                    onSuccess: () => {
                      toast.success('Выдача обновлена')
                      onClose()
                    },
                    onError: (e) => toast.error(errMsg(e, 'Не удалось сохранить')),
                  },
                )
              }
            >
              Сохранить
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function FileList({
  files,
  canEdit,
  uploading,
  onAdd,
  onRemove,
}: {
  files: AdminBriefingFile[]
  canEdit: boolean
  uploading?: boolean
  onAdd: (files: FileList) => void
  onRemove: (id: string) => void
}) {
  return (
    <div className="flex flex-col gap-2">
      <p className="text-[13px] font-medium text-ink">Файлы</p>
      {files.length === 0 && <p className="text-[13px] text-muted">Пока нет файлов.</p>}
      <ul className="flex flex-col gap-2">
        {files.map((f) => (
          <li
            key={f.file_id}
            className="flex items-center gap-2 rounded-[10px] border border-border px-3 py-2"
          >
            <FileText className="h-4 w-4 shrink-0 text-muted-2" />
            <div className="min-w-0 flex-1">
              <p className="truncate text-[14px] text-ink">{f.original_name}</p>
              {f.size_bytes != null && (
                <p className="text-[12px] text-muted">{formatBytes(f.size_bytes)}</p>
              )}
            </div>
            {f.download_url && (
              <a href={f.download_url} className="text-brand hover:text-brand-dark" target="_blank" rel="noreferrer">
                <Download className="h-4 w-4" />
              </a>
            )}
            {canEdit && (
              <button type="button" className="text-muted-2 hover:text-danger" onClick={() => onRemove(f.file_id)}>
                <Trash2 className="h-4 w-4" />
              </button>
            )}
          </li>
        ))}
      </ul>
      {canEdit && (
        <label className="inline-flex w-fit cursor-pointer items-center gap-2 text-[14px] text-brand hover:text-brand-dark">
          <Upload className="h-4 w-4" />
          {uploading ? 'Загрузка…' : 'Добавить файл'}
          <input
            type="file"
            className="sr-only"
            multiple
            disabled={uploading}
            onChange={(e) => {
              if (e.target.files) onAdd(e.target.files)
              e.target.value = ''
            }}
          />
        </label>
      )}
    </div>
  )
}
