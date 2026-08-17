import { useState } from 'react'
import { Check, EyeOff, PackageCheck, Pencil, Plus, ShoppingBag, Trash2, X } from 'lucide-react'
import { toast } from 'sonner'
import {
  useAdminMerch,
  useAdminMerchOrders,
  useDeleteMerch,
  useModerateMerchOrder,
  useTransitionMerch,
} from '@/entities/merch/queries'
import type { MerchOrderStatus, MerchProduct } from '@/entities/merch/types'
import { formatPoints } from '@/entities/points/format'
import { formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'
import { MerchProductDialog } from './merch-product-dialog'

const productStatus = {
  DRAFT: { label: 'Черновик', tone: 'neutral' as const },
  ACTIVE: { label: 'В продаже', tone: 'success' as const },
  HIDDEN: { label: 'Скрыт', tone: 'warning' as const },
  SOLD_OUT: { label: 'Нет в наличии', tone: 'danger' as const },
}

const orderStatus: Record<
  MerchOrderStatus,
  { label: string; tone: 'warning' | 'success' | 'danger' | 'neutral' }
> = {
  RESERVED: { label: 'Зарезервирован', tone: 'warning' },
  ISSUED: { label: 'Выдан', tone: 'success' },
  REJECTED: { label: 'Отклонён', tone: 'danger' },
  CANCELLED: { label: 'Отменён', tone: 'neutral' },
}

export function MerchSection({ contestId }: { contestId: string }) {
  const products = useAdminMerch(contestId)
  const transition = useTransitionMerch(contestId)
  const remove = useDeleteMerch(contestId)
  const [editing, setEditing] = useState<MerchProduct | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [filter, setFilter] = useState<MerchOrderStatus | 'ALL'>('RESERVED')
  const orders = useAdminMerchOrders(contestId, filter)
  const moderate = useModerateMerchOrder(contestId)

  function openProduct(product: MerchProduct | null) {
    setEditing(product)
    setDialogOpen(true)
  }

  function change(product: MerchProduct, action: 'activate' | 'hide') {
    transition.mutate(
      { productId: product.id, action },
      {
        onSuccess: () => toast.success('Статус товара обновлён'),
        onError: () => toast.error('Этот переход сейчас недоступен'),
      },
    )
  }

  function deleteProduct(product: MerchProduct) {
    if (!window.confirm(`Удалить черновик «${product.title}»?`)) return
    remove.mutate(product.id, {
      onSuccess: () => toast.success('Товар удалён'),
      onError: () => toast.error('Удалить можно только неиспользованный черновик'),
    })
  }

  function issue(orderId: string) {
    if (!window.confirm('Подтвердить физическую выдачу заказа? Баллы будут списаны.')) return
    moderate.mutate(
      { orderId, action: 'issue' },
      {
        onSuccess: () => toast.success('Заказ выдан, баллы списаны'),
        onError: () => toast.error('Не удалось выдать заказ'),
      },
    )
  }

  function reject(orderId: string) {
    const reason = window.prompt('Причина отклонения заказа')?.trim()
    if (!reason) return
    moderate.mutate(
      { orderId, action: 'reject', reason },
      {
        onSuccess: () => toast.success('Заказ отклонён, резерв освобождён'),
        onError: () => toast.error('Не удалось отклонить заказ'),
      },
    )
  }

  return (
    <section>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-[20px] font-semibold text-ink">
            <ShoppingBag className="h-5 w-5 text-brand" /> Магазин мерча
          </h2>
          <p className="mt-1 text-[13px] text-muted">Каталог, остатки и резервирование за баллы.</p>
        </div>
        <Button size="sm" onClick={() => openProduct(null)}>
          <Plus className="h-4 w-4" /> Новый товар
        </Button>
      </div>

      {products.isLoading && <Skeleton className="h-32 w-full" />}
      {products.isError && <ErrorState onRetry={() => products.refetch()} />}
      {products.data?.length === 0 && (
        <EmptyState
          title="Товаров пока нет"
          description="Добавьте первый товар в каталог мероприятия."
        />
      )}
      {!!products.data?.length && (
        <div className="grid gap-3 lg:grid-cols-2">
          {products.data.map((product) => {
            const status = productStatus[product.status]
            return (
              <Card key={product.id}>
                <CardBody className="flex gap-4">
                  {product.images[0]?.url ? (
                    <img
                      src={product.images[0].url!}
                      alt=""
                      className="h-24 w-24 shrink-0 rounded-[12px] object-cover"
                    />
                  ) : (
                    <div className="flex h-24 w-24 shrink-0 items-center justify-center rounded-[12px] bg-brand-subtle">
                      <ShoppingBag className="h-7 w-7 text-brand" />
                    </div>
                  )}
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="font-medium text-ink">{product.title}</p>
                      <Badge tone={status.tone}>{status.label}</Badge>
                    </div>
                    <p className="mt-1 text-[14px] font-semibold text-brand">
                      {formatPoints(product.effective_price_points)} баллов
                    </p>
                    <p className="mt-1 text-[12px] text-muted">
                      Доступно {product.available_quantity} · в резерве {product.reserved_quantity}{' '}
                      · целей {product.interested_count}
                    </p>
                    <div className="mt-3 flex flex-wrap gap-1">
                      <Button size="sm" variant="ghost" onClick={() => openProduct(product)}>
                        <Pencil className="h-4 w-4" /> Изменить
                      </Button>
                      {(product.status === 'DRAFT' ||
                        product.status === 'HIDDEN' ||
                        product.status === 'SOLD_OUT') && (
                        <Button
                          size="sm"
                          variant="subtle"
                          onClick={() => change(product, 'activate')}
                        >
                          <Check className="h-4 w-4" /> В продажу
                        </Button>
                      )}
                      {(product.status === 'ACTIVE' || product.status === 'SOLD_OUT') && (
                        <Button
                          size="sm"
                          variant="secondary"
                          onClick={() => change(product, 'hide')}
                        >
                          <EyeOff className="h-4 w-4" /> Скрыть
                        </Button>
                      )}
                      {product.status === 'DRAFT' && (
                        <Button size="sm" variant="ghost" onClick={() => deleteProduct(product)}>
                          <Trash2 className="h-4 w-4 text-danger" />
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

      <div className="mb-3 mt-7 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="flex items-center gap-2 text-[18px] font-semibold text-ink">
            <PackageCheck className="h-5 w-5 text-brand" /> Заказы
          </h3>
          <p className="mt-1 text-[13px] text-muted">
            Баллы списываются только при подтверждении выдачи.
          </p>
        </div>
        <select
          value={filter}
          onChange={(event) => setFilter(event.target.value as MerchOrderStatus | 'ALL')}
          className="h-9 rounded-btn border border-border bg-surface px-3 text-[13px] text-ink"
        >
          <option value="RESERVED">Ожидают выдачи</option>
          <option value="ISSUED">Выданы</option>
          <option value="REJECTED">Отклонены</option>
          <option value="CANCELLED">Отменены</option>
          <option value="ALL">Все</option>
        </select>
      </div>
      {orders.isLoading && <Skeleton className="h-24 w-full" />}
      {orders.isError && <ErrorState onRetry={() => orders.refetch()} />}
      {orders.data?.length === 0 && (
        <div className="rounded-card border border-dashed border-border p-7 text-center text-[13px] text-muted">
          Заказов с таким статусом нет.
        </div>
      )}
      {!!orders.data?.length && (
        <div className="overflow-hidden rounded-card border border-border bg-surface">
          <div className="divide-y divide-border">
            {orders.data.map((order) => (
              <div key={order.id} className="flex flex-col gap-3 p-4 lg:flex-row lg:items-center">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="text-[14px] font-medium text-ink">{order.participant_name}</p>
                    <Badge tone={orderStatus[order.status].tone}>
                      {orderStatus[order.status].label}
                    </Badge>
                  </div>
                  <p className="mt-1 text-[13px] text-muted">
                    {order.items
                      .map((item) => `${item.product_title} × ${item.quantity}`)
                      .join(', ')}
                  </p>
                  <p className="mt-1 text-[12px] text-muted">
                    {formatDateTime(order.created_at)} · {formatPoints(order.points_total)} баллов
                  </p>
                  {order.rejection_reason && (
                    <p className="mt-1 text-[12px] text-danger">{order.rejection_reason}</p>
                  )}
                </div>
                {order.status === 'RESERVED' && (
                  <div className="flex gap-2">
                    <Button size="sm" loading={moderate.isPending} onClick={() => issue(order.id)}>
                      <PackageCheck className="h-4 w-4" /> Выдать
                    </Button>
                    <Button size="sm" variant="secondary" onClick={() => reject(order.id)}>
                      <X className="h-4 w-4" /> Отклонить
                    </Button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      <MerchProductDialog
        contestId={contestId}
        product={editing}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
    </section>
  )
}
