import { useState } from 'react'
import { Link } from 'react-router-dom'
import { CalendarClock, Check, Flag, Pencil, Plus, QrCode, Tags, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import type { Lecture } from '@/entities/lecture/types'
import { lecturePeopleLine } from '@/entities/lecture/types'
import {
  useAdminLectures,
  useDeleteLecture,
  useTransitionLecture,
} from '@/entities/lecture/queries'
import { formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'
import { EventDirectionsDialog } from './event-directions-dialog'
import { LectureDialog } from './lecture-dialog'

const statusMeta = {
  DRAFT: { label: 'Черновик', tone: 'neutral' as const },
  ACTIVE: { label: 'Активна', tone: 'success' as const },
  FINISHED: { label: 'Завершена', tone: 'brand' as const },
}

export function LecturesSection({
  contestId,
  canManage = true,
  canEditDirections = false,
}: {
  contestId: string
  canManage?: boolean
  canEditDirections?: boolean
}) {
  const lectures = useAdminLectures(contestId)
  const transition = useTransitionLecture(contestId)
  const remove = useDeleteLecture(contestId)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [directionsOpen, setDirectionsOpen] = useState(false)
  const [editing, setEditing] = useState<Lecture | null>(null)

  function openCreate() {
    setEditing(null)
    setDialogOpen(true)
  }

  function openEdit(lecture: Lecture) {
    setEditing(lecture)
    setDialogOpen(true)
  }

  function changeStatus(lecture: Lecture, action: 'activate' | 'finish') {
    if (
      action === 'finish' &&
      !window.confirm(
        `Завершить лекцию «${lecture.title}»? Регистрация посещений остановится, пока лекцию снова не активируют.`,
      )
    ) {
      return
    }
    transition.mutate(
      { lectureId: lecture.id, action },
      {
        onSuccess: () =>
          toast.success(
            action === 'activate'
              ? lecture.status === 'FINISHED'
                ? 'Лекция снова активна'
                : 'Регистрация активирована'
              : 'Лекция завершена',
          ),
        onError: () => toast.error('Не удалось изменить статус лекции'),
      },
    )
  }

  function deleteLecture(lecture: Lecture) {
    if (!window.confirm(`Удалить черновик «${lecture.title}»?`)) return
    remove.mutate(lecture.id, {
      onSuccess: () => toast.success('Лекция удалена'),
      onError: () => toast.error('Удалить можно только черновик без посещений'),
    })
  }

  return (
    <section>
      <div className="mb-3 flex items-center justify-between gap-3">
        <div>
          <h2 className="text-[20px] font-semibold text-ink">Лекции и посещаемость</h2>
          <p className="mt-1 text-[13px] text-muted">
            Расписание, окно сканирования, спикеры и направления, для которых открыта лекция.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="secondary" onClick={() => setDirectionsOpen(true)}>
            <Tags className="h-4 w-4" /> Направления
          </Button>
          {canManage && (
            <Button size="sm" onClick={openCreate}>
              <Plus className="h-4 w-4" /> Новая лекция
            </Button>
          )}
        </div>
      </div>

      {lectures.isLoading && <Skeleton className="h-32 w-full" />}
      {lectures.isError && <ErrorState onRetry={() => lectures.refetch()} />}
      {lectures.data?.length === 0 && (
        <EmptyState
          title="Лекций пока нет"
          description="Создайте лекцию и задайте баллы за посещение."
        />
      )}
      {!!lectures.data?.length && (
        <div className="flex flex-col gap-2">
          {lectures.data.map((lecture) => {
            const status = statusMeta[lecture.status]
            const people = lecturePeopleLine(lecture)
            return (
              <Card key={lecture.id}>
                <CardBody className="flex flex-col gap-3 py-4 lg:flex-row lg:items-center">
                  <CalendarClock className="h-5 w-5 shrink-0 text-brand" />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="font-medium text-ink">{lecture.title}</p>
                      <Badge tone={status.tone}>{status.label}</Badge>
                      <Badge tone="warning">+{lecture.points} баллов</Badge>
                      {(lecture.directions?.length ?? 0) === 0 ? (
                        <Badge>Все направления</Badge>
                      ) : (
                        lecture.directions.map((direction) => (
                          <Badge key={direction.id}>{direction.name}</Badge>
                        ))
                      )}
                    </div>
                    <p className="mt-1 text-[13px] text-muted">
                      {lecture.starts_at ? formatDateTime(lecture.starts_at) : 'Время не задано'}
                      {lecture.location && ` · ${lecture.location}`}
                      {lecture.attendance_starts_at &&
                        ` · регистрация с ${formatDateTime(lecture.attendance_starts_at)}`}
                    </p>
                    {people && <p className="mt-1 text-[13px] text-muted">{people}</p>}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {canManage && lecture.status !== 'FINISHED' && (
                      <Button size="sm" variant="ghost" onClick={() => openEdit(lecture)}>
                        <Pencil className="h-4 w-4" /> Изменить
                      </Button>
                    )}
                    {canManage && lecture.status === 'DRAFT' && (
                      <>
                        <Button
                          size="sm"
                          variant="subtle"
                          loading={transition.isPending}
                          onClick={() => changeStatus(lecture, 'activate')}
                        >
                          <Check className="h-4 w-4" /> Активировать
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          loading={remove.isPending}
                          onClick={() => deleteLecture(lecture)}
                        >
                          <Trash2 className="h-4 w-4 text-danger" />
                        </Button>
                      </>
                    )}
                    {lecture.status === 'ACTIVE' && (
                      <>
                        <Button asChild size="sm">
                          <Link to={`/admin/contests/${contestId}/lectures/${lecture.id}/scanner`}>
                            <QrCode className="h-4 w-4" /> Сканер
                          </Link>
                        </Button>
                        {canManage && (
                        <Button
                          size="sm"
                          variant="secondary"
                          loading={transition.isPending}
                          onClick={() => changeStatus(lecture, 'finish')}
                        >
                          <Flag className="h-4 w-4" /> Завершить
                        </Button>
                        )}
                      </>
                    )}
                    {lecture.status === 'FINISHED' && (
                      <>
                        {canManage && (
                          <Button
                            size="sm"
                            variant="subtle"
                            loading={transition.isPending}
                            onClick={() => changeStatus(lecture, 'activate')}
                          >
                            <Check className="h-4 w-4" /> Снова активировать
                          </Button>
                        )}
                        <Button asChild size="sm" variant="secondary">
                          <Link to={`/admin/contests/${contestId}/lectures/${lecture.id}/scanner`}>
                            Посещения
                          </Link>
                        </Button>
                      </>
                    )}
                  </div>
                </CardBody>
              </Card>
            )
          })}
        </div>
      )}

      <LectureDialog
        contestId={contestId}
        lecture={editing}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
      <EventDirectionsDialog
        contestId={contestId}
        open={directionsOpen}
        onOpenChange={setDirectionsOpen}
        canEdit={canEditDirections}
      />
    </section>
  )
}
