import { ArrowRight, CheckCircle2, Clock3, RotateCcw, Target } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import { TaskIcon } from '@/entities/event-task/icon'
import { useParticipantTasks } from '@/entities/event-task/queries'
import type { TaskSubmissionStatus } from '@/entities/event-task/types'
import { formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Card, CardBody } from '@/shared/ui/card'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'

const submissionMeta: Record<
  TaskSubmissionStatus,
  { label: string; tone: 'warning' | 'success' | 'danger'; icon: typeof Clock3 }
> = {
  PENDING: { label: 'На проверке', tone: 'warning', icon: Clock3 },
  APPROVED: { label: 'Выполнено', tone: 'success', icon: CheckCircle2 },
  REJECTED: { label: 'Нужна доработка', tone: 'danger', icon: RotateCcw },
}

export function ParticipantTasksPage() {
  const { session } = useParticipantAuth()
  const tasks = useParticipantTasks(session?.event.id, session?.participant.id)
  if (!session) return null

  return (
    <div>
      <header className="mb-6">
        <p className="text-[13px] font-medium text-brand">{session.event.name}</p>
        <h1 className="mt-1 flex items-center gap-2 text-[28px] font-bold text-ink">
          <Target className="h-7 w-7 text-brand" /> Задания
        </h1>
        <p className="mt-1 text-[14px] text-muted">
          Выполняйте задания, отправляйте подтверждения и получайте баллы.
        </p>
      </header>
      {tasks.isLoading && (
        <div className="grid gap-3 sm:grid-cols-2">
          {[0, 1].map((item) => (
            <Skeleton key={item} className="h-44 w-full" />
          ))}
        </div>
      )}
      {tasks.isError && <ErrorState onRetry={() => tasks.refetch()} />}
      {tasks.data?.length === 0 && (
        <EmptyState
          title="Активных заданий пока нет"
          description="Здесь появятся задания от организаторов мероприятия."
        />
      )}
      {!!tasks.data?.length && (
        <div className="grid gap-3 sm:grid-cols-2">
          {tasks.data.map((task) => {
            const state = task.submission ? submissionMeta[task.submission.status] : null
            const StateIcon = state?.icon
            return (
              <Link
                key={task.id}
                to={`/event/${encodeURIComponent(session.event.slug)}/tasks/${task.id}`}
                className="group"
              >
                <Card className="h-full overflow-hidden transition-colors group-hover:border-brand">
                  {task.image_url && (
                    <img src={task.image_url} alt="" className="h-32 w-full object-cover" />
                  )}
                  <CardBody className="flex h-full flex-col p-5">
                    <div className="flex items-start justify-between gap-3">
                      <TaskIcon url={task.icon_url} />
                      <Badge tone="brand">+{task.points} баллов</Badge>
                    </div>
                    <h2 className="mt-3 text-[17px] font-semibold text-ink">{task.title}</h2>
                    <p className="mt-1 line-clamp-2 text-[13px] leading-relaxed text-muted">
                      {task.description}
                    </p>
                    <div className="mt-4 flex items-center justify-between gap-3">
                      {state && StateIcon ? (
                        <Badge tone={state.tone}>
                          <StateIcon className="h-3 w-3" /> {state.label}
                        </Badge>
                      ) : (
                        <span className="text-[12px] text-muted">
                          {task.ends_at ? `до ${formatDateTime(task.ends_at)}` : 'Без срока'}
                        </span>
                      )}
                      <ArrowRight className="h-4 w-4 text-brand transition-transform group-hover:translate-x-1" />
                    </div>
                  </CardBody>
                </Card>
              </Link>
            )
          })}
        </div>
      )}
    </div>
  )
}
