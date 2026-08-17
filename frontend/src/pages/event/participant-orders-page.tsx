import { PackageCheck, ShoppingBag, X } from 'lucide-react'
import { toast } from 'sonner'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import { useCancelParticipantMerchOrder, useParticipantMerchOrders } from '@/entities/merch/queries'
import type { MerchOrderStatus } from '@/entities/merch/types'
import { formatPoints } from '@/entities/points/format'
import { formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'

const statusMeta: Record<
  MerchOrderStatus,
  { label: string; tone: 'warning' | 'success' | 'danger' | 'neutral'; description: string }
> = {
  RESERVED: {
    label: 'Ожидает выдачи',
    tone: 'warning',
    description: 'Баллы и товар зарезервированы.',
  },
  ISSUED: { label: 'Выдан', tone: 'success', description: 'Заказ получен, баллы списаны.' },
  REJECTED: {
    label: 'Отклонён',
    tone: 'danger',
    description: 'Резерв товара и баллов освобождён.',
  },
  CANCELLED: {
    label: 'Отменён',
    tone: 'neutral',
    description: 'Резерв товара и баллов освобождён.',
  },
}

export function ParticipantOrdersPage() {
  const { session } = useParticipantAuth()
  const eventId = session?.event.id ?? ''
  const participantId = session?.participant.id ?? ''
  const orders = useParticipantMerchOrders(eventId, participantId)
  const cancel = useCancelParticipantMerchOrder(eventId, participantId)

  if (!session) return null

  function cancelOrder(orderId: string) {
    if (!window.confirm('Отменить резерв заказа?')) return
    cancel.mutate(orderId, {
      onSuccess: () => toast.success('Заказ отменён, резерв освобождён'),
      onError: () => toast.error('Этот заказ уже нельзя отменить'),
    })
  }

  return (
    <div className="flex flex-col gap-5">
      <header>
        <p className="text-[13px] font-medium text-brand">Магазин</p>
        <h1 className="mt-1 text-[28px] font-bold text-ink">Мои заказы</h1>
        <p className="mt-1 text-[14px] text-muted">
          Покажите ожидающий заказ организатору при выдаче.
        </p>
      </header>
      {orders.isLoading && <Skeleton className="h-40 w-full" />}
      {orders.isError && <ErrorState onRetry={() => orders.refetch()} />}
      {orders.data?.length === 0 && (
        <EmptyState
          title="Заказов пока нет"
          description="Зарезервированные товары появятся здесь."
        />
      )}
      {!!orders.data?.length && (
        <div className="flex flex-col gap-3">
          {orders.data.map((order) => {
            const status = statusMeta[order.status]
            return (
              <Card key={order.id}>
                <CardBody>
                  <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        {order.status === 'ISSUED' ? (
                          <PackageCheck className="h-5 w-5 text-success" />
                        ) : (
                          <ShoppingBag className="h-5 w-5 text-brand" />
                        )}
                        <p className="font-semibold text-ink">
                          Заказ от {formatDateTime(order.created_at)}
                        </p>
                        <Badge tone={status.tone}>{status.label}</Badge>
                      </div>
                      <div className="mt-3 flex flex-col gap-2">
                        {order.items.map((item) => (
                          <div
                            key={item.id}
                            className="flex items-center justify-between gap-4 text-[14px]"
                          >
                            <span className="text-ink">
                              {item.product_title} × {item.quantity}
                            </span>
                            <span className="whitespace-nowrap text-muted">
                              {formatPoints(item.total_points)} б.
                            </span>
                          </div>
                        ))}
                      </div>
                      <p className="mt-3 text-[12px] text-muted">{status.description}</p>
                      {order.rejection_reason && (
                        <p className="mt-1 text-[12px] text-danger">
                          Причина: {order.rejection_reason}
                        </p>
                      )}
                    </div>
                    <div className="shrink-0 text-left sm:text-right">
                      <p className="text-[12px] text-muted">Итого</p>
                      <p className="text-[20px] font-bold text-brand">
                        {formatPoints(order.points_total)} баллов
                      </p>
                      {order.status === 'RESERVED' && (
                        <Button
                          className="mt-3"
                          size="sm"
                          variant="secondary"
                          loading={cancel.isPending}
                          onClick={() => cancelOrder(order.id)}
                        >
                          <X className="h-4 w-4" /> Отменить
                        </Button>
                      )}
                    </div>
                  </div>
                </CardBody>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
