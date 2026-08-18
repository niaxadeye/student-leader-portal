import { useState } from 'react'
import { Archive, Check, ClipboardCheck, Pause, Pencil, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  useAdminTasks,
  useDeleteTask,
  useTaskModeration,
  useTransitionTask,
} from '@/entities/event-task/queries'
import type { EventTask, TaskSubmission, TaskSubmissionStatus } from '@/entities/event-task/types'
import { formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'
import { EventTaskDialog } from './event-task-dialog'
import { TaskModerationDialog } from './task-moderation-dialog'

const taskStatus = {
  DRAFT: { label: 'Черновик', tone: 'neutral' as const },
  ACTIVE: { label: 'Активно', tone: 'success' as const },
  DISABLED: { label: 'Приостановлено', tone: 'warning' as const },
  ARCHIVED: { label: 'В архиве', tone: 'neutral' as const },
}

const submissionStatus: Record<TaskSubmissionStatus, string> = {
  PENDING: 'Ожидают',
  APPROVED: 'Приняты',
  REJECTED: 'На доработке',
}

export function EventTasksSection({
  contestId,
  canManage = true,
  canModerate = true,
}: {
  contestId: string
  canManage?: boolean
  canModerate?: boolean
}) {
  const tasks = useAdminTasks(contestId)
  const transition = useTransitionTask(contestId)
  const remove = useDeleteTask(contestId)
  const [editing, setEditing] = useState<EventTask | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [filter, setFilter] = useState<TaskSubmissionStatus>('PENDING')
  const moderation = useTaskModeration(contestId, filter, canModerate)
  const [selected, setSelected] = useState<TaskSubmission | null>(null)

  function openTask(task: EventTask | null) {
    setEditing(task)
    setDialogOpen(true)
  }

  function change(task: EventTask, action: 'activate' | 'disable' | 'archive') {
    transition.mutate(
      { taskId: task.id, action },
      {
        onSuccess: () => toast.success('Статус задания обновлён'),
        onError: () => toast.error('Этот переход сейчас недоступен'),
      },
    )
  }

  function deleteTask(task: EventTask) {
    if (!window.confirm(`Удалить черновик «${task.title}»?`)) return
    remove.mutate(task.id, {
      onSuccess: () => toast.success('Задание удалено'),
      onError: () => toast.error('Удалить можно только черновик без отправок'),
    })
  }

  return (
    <section>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-[20px] font-semibold text-ink">Задания мероприятия</h2>
          <p className="mt-1 text-[13px] text-muted">
            Подтверждения выполнения, модерация и награды.
          </p>
        </div>
        {canManage && (
        <Button size="sm" onClick={() => openTask(null)}>
          <Plus className="h-4 w-4" /> Новое задание
        </Button>
        )}
      </div>

      {tasks.isLoading && <Skeleton className="h-32 w-full" />}
      {tasks.isError && <ErrorState onRetry={() => tasks.refetch()} />}
      {tasks.data?.length === 0 && (
        <EmptyState
          title="Заданий пока нет"
          description="Создайте первое задание с наградой в баллах."
        />
      )}
      {!!tasks.data?.length && (
        <div className="flex flex-col gap-2">
          {tasks.data.map((task) => (
            <Card key={task.id}>
              <CardBody className="flex flex-col gap-3 py-4 lg:flex-row lg:items-center">
                {task.image_url ? (
                  <img src={task.image_url} alt="" className="h-14 w-20 rounded-lg object-cover" />
                ) : (
                  <div className="flex h-12 w-12 items-center justify-center rounded-[12px] bg-brand-subtle text-2xl">
                    {task.icon || '🎯'}
                  </div>
                )}
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-medium text-ink">{task.title}</p>
                    <Badge tone={taskStatus[task.status].tone}>
                      {taskStatus[task.status].label}
                    </Badge>
                    <Badge tone="brand">+{task.points} баллов</Badge>
                  </div>
                  <p className="mt-1 line-clamp-1 text-[13px] text-muted">{task.description}</p>
                  {(task.starts_at || task.ends_at) && (
                    <p className="mt-1 text-[12px] text-muted">
                      {task.starts_at ? `с ${formatDateTime(task.starts_at)}` : 'сразу'}
                      {task.ends_at ? ` · до ${formatDateTime(task.ends_at)}` : ''}
                    </p>
                  )}
                </div>
                {canManage && (
                <div className="flex flex-wrap gap-1">
                  {task.status !== 'ARCHIVED' && (
                    <Button size="sm" variant="ghost" onClick={() => openTask(task)}>
                      <Pencil className="h-4 w-4" /> Изменить
                    </Button>
                  )}
                  {(task.status === 'DRAFT' || task.status === 'DISABLED') && (
                    <Button size="sm" variant="subtle" onClick={() => change(task, 'activate')}>
                      <Check className="h-4 w-4" /> Активировать
                    </Button>
                  )}
                  {task.status === 'ACTIVE' && (
                    <Button size="sm" variant="secondary" onClick={() => change(task, 'disable')}>
                      <Pause className="h-4 w-4" /> Пауза
                    </Button>
                  )}
                  {task.status !== 'ARCHIVED' && (
                    <Button size="sm" variant="ghost" onClick={() => change(task, 'archive')}>
                      <Archive className="h-4 w-4" />
                    </Button>
                  )}
                  {task.status === 'DRAFT' && (
                    <Button size="sm" variant="ghost" onClick={() => deleteTask(task)}>
                      <Trash2 className="h-4 w-4 text-danger" />
                    </Button>
                  )}
                </div>
                )}
              </CardBody>
            </Card>
          ))}
        </div>
      )}

      {canModerate && (
      <>
      <div className="mb-3 mt-7 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="flex items-center gap-2 text-[18px] font-semibold text-ink">
            <ClipboardCheck className="h-5 w-5 text-brand" /> Проверка подтверждений
          </h3>
          <p className="mt-1 text-[13px] text-muted">
            Награда начисляется атомарно при подтверждении.
          </p>
        </div>
        <div className="flex gap-1 rounded-[12px] bg-surface-2 p-1">
          {(Object.keys(submissionStatus) as TaskSubmissionStatus[]).map((status) => (
            <button
              key={status}
              type="button"
              onClick={() => setFilter(status)}
              className={`rounded-[9px] px-3 py-1.5 text-[13px] font-medium ${filter === status ? 'bg-surface text-brand shadow-sm' : 'text-muted'}`}
            >
              {submissionStatus[status]}
            </button>
          ))}
        </div>
      </div>
      {moderation.isLoading && <Skeleton className="h-24 w-full" />}
      {moderation.isError && <ErrorState onRetry={() => moderation.refetch()} />}
      {moderation.data?.length === 0 && (
        <div className="rounded-card border border-dashed border-border p-7 text-center text-[13px] text-muted">
          В этой очереди отправок нет.
        </div>
      )}
      {!!moderation.data?.length && (
        <div className="overflow-hidden rounded-card border border-border bg-surface">
          <div className="divide-y divide-border">
            {moderation.data.map((submission) => (
              <button
                key={submission.id}
                type="button"
                onClick={() => setSelected(submission)}
                className="flex w-full items-center justify-between gap-4 p-4 text-left hover:bg-surface-2"
              >
                <div>
                  <p className="text-[14px] font-medium text-ink">{submission.participant_name}</p>
                  <p className="mt-0.5 text-[12px] text-muted">
                    {submission.task_title} · попытка №{submission.current_attempt}
                  </p>
                </div>
                <div className="text-right">
                  <Badge
                    tone={
                      filter === 'APPROVED'
                        ? 'success'
                        : filter === 'REJECTED'
                          ? 'danger'
                          : 'warning'
                    }
                  >
                    {submissionStatus[filter]}
                  </Badge>
                  <p className="mt-1 text-[12px] text-muted">
                    {submission.submitted_at ? formatDateTime(submission.submitted_at) : ''}
                  </p>
                </div>
              </button>
            ))}
          </div>
        </div>
      )}
      </>
      )}

      <EventTaskDialog
        contestId={contestId}
        task={editing}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
      <TaskModerationDialog
        contestId={contestId}
        submission={selected}
        onClose={() => setSelected(null)}
      />
    </section>
  )
}
