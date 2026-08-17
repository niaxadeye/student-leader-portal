import { useState } from 'react'
import { ArrowLeft, Bookmark, BookmarkCheck, PackageCheck, ShoppingBag } from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import {
  useParticipantMerchProduct,
  useParticipantSavingTarget,
  useReserveParticipantMerch,
} from '@/entities/merch/queries'
import { formatPoints } from '@/entities/points/format'
import { useParticipantPoints } from '@/entities/points/queries'
import { ApiRequestError } from '@/shared/api/client'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'

export function ParticipantProductPage() {
  const { productSlug } = useParams()
  const navigate = useNavigate()
  const { session } = useParticipantAuth()
  const eventId = session?.event.id ?? ''
  const participantId = session?.participant.id ?? ''
  const product = useParticipantMerchProduct(eventId, participantId, productSlug)
  const points = useParticipantPoints(eventId, participantId)
  const target = useParticipantSavingTarget(eventId, participantId)
  const reserve = useReserveParticipantMerch(eventId, participantId)
  const [quantity, setQuantity] = useState(1)

  if (!session) return null
  const base = `/event/${encodeURIComponent(session.event.slug)}`
  if (product.isLoading) return <Skeleton className="h-96 w-full" />
  if (product.isError) return <ErrorState onRetry={() => product.refetch()} />
  if (!product.data) return <EmptyState title="Товар не найден" />

  const item = product.data
  const total = item.effective_price_points * quantity
  const availablePoints = points.data?.balance.available_points ?? 0

  function saveTarget() {
    target.mutate(item.is_saving_target ? null : item.id, {
      onSuccess: () => toast.success(item.is_saving_target ? 'Цель убрана' : 'Товар выбран целью'),
      onError: () => toast.error('Не удалось изменить цель'),
    })
  }

  function reserveOrder() {
    reserve.mutate(
      {
        items: [{ product_id: item.id, quantity }],
        idempotency_key: `web-${crypto.randomUUID()}`,
      },
      {
        onSuccess: () => {
          toast.success('Товар зарезервирован до выдачи')
          navigate(`${base}/orders`)
        },
        onError: (error) => {
          if (error instanceof ApiRequestError && error.code === 'INSUFFICIENT_POINTS') {
            toast.error('Недостаточно доступных баллов')
          } else if (error instanceof ApiRequestError && error.code === 'INSUFFICIENT_STOCK') {
            toast.error('Товар уже закончился или был зарезервирован')
          } else {
            toast.error('Не удалось оформить резерв')
          }
        },
      },
    )
  }

  return (
    <div>
      <Link
        to={`${base}/shop`}
        className="mb-4 inline-flex items-center gap-1 text-[14px] text-muted hover:text-ink"
      >
        <ArrowLeft className="h-4 w-4" /> В магазин
      </Link>
      <Card className="overflow-hidden">
        <div className="grid lg:grid-cols-2">
          <div className="bg-brand-subtle">
            {item.images[0]?.url ? (
              <img
                src={item.images[0].url!}
                alt={item.title}
                className="h-full min-h-80 w-full object-cover"
              />
            ) : (
              <div className="flex min-h-80 items-center justify-center">
                <ShoppingBag className="h-20 w-20 text-brand/50" />
              </div>
            )}
          </div>
          <CardBody className="flex flex-col justify-center p-6 sm:p-8">
            <Badge tone={item.available_quantity > 0 ? 'success' : 'danger'} className="self-start">
              {item.available_quantity > 0
                ? `В наличии: ${item.available_quantity}`
                : 'Нет в наличии'}
            </Badge>
            <h1 className="mt-3 text-[28px] font-bold text-ink">{item.title}</h1>
            <div className="mt-2 flex items-center gap-3">
              <span className="text-[24px] font-bold text-brand">
                {formatPoints(item.effective_price_points)} баллов
              </span>
              {item.discount_price_points && (
                <span className="text-[14px] text-muted line-through">
                  {formatPoints(item.price_points)}
                </span>
              )}
            </div>
            <p className="mt-5 whitespace-pre-wrap text-[14px] leading-relaxed text-muted">
              {item.description}
            </p>
            <div className="mt-6 rounded-[12px] bg-surface-2 p-4">
              <div className="flex items-center justify-between gap-3 text-[13px]">
                <span className="text-muted">Ваш доступный баланс</span>
                <span className="font-semibold text-ink">
                  {formatPoints(availablePoints)} баллов
                </span>
              </div>
              <div className="mt-3 flex items-center gap-3">
                <Input
                  type="number"
                  min={1}
                  max={Math.min(99, item.available_quantity)}
                  value={quantity}
                  onChange={(e) =>
                    setQuantity(Math.max(1, Math.min(99, Number(e.target.value) || 1)))
                  }
                  className="w-24"
                  aria-label="Количество"
                />
                <div>
                  <p className="text-[12px] text-muted">Итого</p>
                  <p className="font-semibold text-ink">{formatPoints(total)} баллов</p>
                </div>
              </div>
            </div>
            <div className="mt-5 flex flex-wrap gap-2">
              <Button
                loading={reserve.isPending}
                disabled={item.available_quantity < quantity || availablePoints < total}
                onClick={reserveOrder}
              >
                <PackageCheck className="h-4 w-4" /> Зарезервировать
              </Button>
              <Button variant="secondary" loading={target.isPending} onClick={saveTarget}>
                {item.is_saving_target ? (
                  <BookmarkCheck className="h-4 w-4" />
                ) : (
                  <Bookmark className="h-4 w-4" />
                )}
                {item.is_saving_target ? 'Цель выбрана' : 'Копить на товар'}
              </Button>
            </div>
            {availablePoints < total && (
              <p className="mt-3 text-[12px] text-danger">
                Не хватает {formatPoints(total - availablePoints)} баллов.
              </p>
            )}
          </CardBody>
        </div>
      </Card>
    </div>
  )
}
