import { useEffect, type ComponentType } from 'react'
import { Link } from 'react-router-dom'
import {
  CalendarDays,
  CheckCircle2,
  History,
  QrCode,
  ShoppingBag,
  Sparkles,
  Target,
} from 'lucide-react'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import { formatPoints, signedPoints } from '@/entities/points/format'
import { useParticipantPoints } from '@/entities/points/queries'
import { useParticipantLectures } from '@/entities/lecture/queries'
import { ApiRequestError } from '@/shared/api/client'
import { formatDateOnly, formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody, CardHeader, CardTitle } from '@/shared/ui/card'
import { Skeleton } from '@/shared/ui/states'

function DataRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-border py-3 last:border-b-0 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
      <span className="text-[13px] text-muted">{label}</span>
      <span className="break-all text-[14px] font-medium text-ink sm:text-right">{value}</span>
    </div>
  )
}

function UpcomingFeature({
  icon: Icon,
  title,
  description,
}: {
  icon: ComponentType<{ className?: string }>
  title: string
  description: string
}) {
  return (
    <Card className="p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-btn bg-brand-subtle">
          <Icon className="h-5 w-5 text-brand" aria-hidden />
        </div>
        <Badge>Следующий этап</Badge>
      </div>
      <h3 className="mt-4 text-[17px] font-semibold text-ink">{title}</h3>
      <p className="mt-1 text-[13px] leading-relaxed text-muted">{description}</p>
    </Card>
  )
}

export function ParticipantMePage() {
  const { session, refresh } = useParticipantAuth()
  const points = useParticipantPoints(session?.event.id, session?.participant.id)
  const lectures = useParticipantLectures(session?.event.id, session?.participant.id)

  useEffect(() => {
    if (points.error instanceof ApiRequestError && points.error.status === 401) {
      void refresh()
    }
  }, [points.error, refresh])

  if (!session) return null

  const { participant, event } = session
  return (
    <div className="flex flex-col gap-7">
      <section className="overflow-hidden rounded-card bg-gradient-to-br from-brand-deep to-brand p-6 text-white shadow-subtle sm:p-8">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-[13px] font-semibold uppercase tracking-[0.12em] text-white/70">
              {event.name}
            </p>
            <h1 className="mt-2 text-[28px] font-bold sm:text-[34px]">Здравствуйте!</h1>
            <p className="mt-1 text-[16px] text-white/85">{participant.full_name}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              asChild
              variant="secondary"
              className="border border-white/25 bg-white/10 text-white"
            >
              <Link to={`/event/${encodeURIComponent(event.slug)}/tasks`}>
                <Target className="h-4 w-4" /> Задания
              </Link>
            </Button>
            <Button
              asChild
              variant="secondary"
              className="border border-white/25 bg-white/10 text-white"
            >
              <Link to={`/event/${encodeURIComponent(event.slug)}/shop`}>
                <ShoppingBag className="h-4 w-4" /> Магазин
              </Link>
            </Button>
            <Button
              asChild
              variant="secondary"
              className="border border-white/25 bg-white/10 text-white"
            >
              <Link to={`/event/${encodeURIComponent(event.slug)}/me/qr`}>
                <QrCode className="h-4 w-4" /> Мой QR
              </Link>
            </Button>
          </div>
        </div>
      </section>

      <div className="grid gap-5 lg:grid-cols-[minmax(0,1.1fr)_minmax(280px,0.9fr)]">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-3">
            <CardTitle>Профиль участника</CardTitle>
            <Badge tone="success">
              <CheckCircle2 className="h-3 w-3" /> Активен
            </Badge>
          </CardHeader>
          <CardBody className="pt-3">
            <DataRow label="ФИО" value={participant.full_name} />
            <DataRow label="Дата рождения" value={formatDateOnly(participant.birth_date)} />
            <DataRow
              label="Профсоюзный билет"
              value={participant.union_card_number || 'Не указан'}
            />
            <DataRow label="Barcode СКС" value={participant.sks_barcode || 'Не указан'} />
            <DataRow label="Направление" value={participant.direction_name || 'Не указано'} />
          </CardBody>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-3">
            <CardTitle>Баланс</CardTitle>
            <Badge>PointsLedger</Badge>
          </CardHeader>
          <CardBody>
            {points.isLoading ? (
              <Skeleton className="h-32 w-full" />
            ) : points.isError || !points.data ? (
              <div className="rounded-[12px] bg-danger/10 p-4 text-[13px] text-danger">
                Не удалось загрузить баланс.
              </div>
            ) : (
              <>
                <div className="rounded-[12px] bg-brand-subtle p-4">
                  <p className="text-[13px] text-muted">Доступные баллы</p>
                  <p className="mt-1 text-[30px] font-bold text-brand">
                    {formatPoints(points.data.balance.available_points)}
                  </p>
                </div>
                <div className="mt-3 grid grid-cols-2 gap-2 text-[12px]">
                  <div className="rounded-[10px] bg-surface-2 p-3">
                    <p className="text-muted">Общий баланс</p>
                    <p className="mt-1 font-semibold text-ink">
                      {formatPoints(points.data.balance.ledger_balance)}
                    </p>
                  </div>
                  <div className="rounded-[10px] bg-surface-2 p-3">
                    <p className="text-muted">В резерве</p>
                    <p className="mt-1 font-semibold text-ink">
                      {formatPoints(points.data.balance.reserved_points)}
                    </p>
                  </div>
                </div>
              </>
            )}
          </CardBody>
        </Card>
      </div>

      {points.data && (
        <Card>
          <CardHeader className="flex flex-row items-center gap-2">
            <History className="h-5 w-5 text-brand" />
            <CardTitle>История баллов</CardTitle>
          </CardHeader>
          <CardBody>
            {!points.data.entries.length ? (
              <p className="rounded-[12px] bg-surface-2 px-4 py-6 text-center text-[13px] text-muted">
                Операций пока нет.
              </p>
            ) : (
              <div className="divide-y divide-border">
                {points.data.entries.map((entry) => (
                  <div
                    key={entry.id}
                    className="flex items-start justify-between gap-4 py-3 first:pt-0 last:pb-0"
                  >
                    <div>
                      <p className="text-[14px] font-medium text-ink">{entry.description}</p>
                      <p className="mt-0.5 text-[12px] text-muted">
                        {formatDateTime(entry.created_at)}
                      </p>
                    </div>
                    <span
                      className={`whitespace-nowrap text-[14px] font-semibold ${entry.amount > 0 ? 'text-success' : 'text-danger'}`}
                    >
                      {signedPoints(entry.amount)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </CardBody>
        </Card>
      )}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <CalendarDays className="h-5 w-5 text-brand" />
            <CardTitle>Лекции</CardTitle>
          </div>
          <Badge tone="brand">
            Посещено: {lectures.data?.filter((item) => item.attendance).length ?? 0}
          </Badge>
        </CardHeader>
        <CardBody>
          {lectures.isLoading && <Skeleton className="h-24 w-full" />}
          {lectures.isError && (
            <p className="rounded-[12px] bg-danger/10 p-4 text-[13px] text-danger">
              Не удалось загрузить лекции.
            </p>
          )}
          {lectures.data?.length === 0 && (
            <p className="rounded-[12px] bg-surface-2 p-5 text-center text-[13px] text-muted">
              Для вашего направления пока нет опубликованных лекций.
            </p>
          )}
          {!!lectures.data?.length && (
            <div className="divide-y divide-border">
              {lectures.data.map(({ lecture, attendance }) => (
                <div
                  key={lecture.id}
                  className="flex items-start justify-between gap-4 py-3 first:pt-0 last:pb-0"
                >
                  <div>
                    <p className="text-[14px] font-medium text-ink">{lecture.title}</p>
                    <p className="mt-0.5 text-[12px] text-muted">
                      {lecture.starts_at ? formatDateTime(lecture.starts_at) : 'Время не задано'} ·
                      +{lecture.points} баллов
                    </p>
                  </div>
                  {attendance ? (
                    <Badge tone="success">
                      <CheckCircle2 className="h-3 w-3" /> Посещено
                    </Badge>
                  ) : (
                    <Badge>{lecture.status === 'ACTIVE' ? 'Активна' : 'Завершена'}</Badge>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>

      <section>
        <div className="mb-4 flex items-end justify-between gap-4">
          <div>
            <p className="text-[13px] font-medium text-brand">Возможности платформы</p>
            <h2 className="mt-1 text-[22px] font-semibold text-ink">Скоро в кабинете</h2>
          </div>
          <Badge tone="brand">Кабинет подключён</Badge>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <UpcomingFeature
            icon={Sparkles}
            title="Достижения"
            description="Прогресс и достижения участника мероприятия."
          />
        </div>
      </section>
    </div>
  )
}
