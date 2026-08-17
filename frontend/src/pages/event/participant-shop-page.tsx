import { Link } from 'react-router-dom'
import { Bookmark, BookmarkCheck, ShoppingBag } from 'lucide-react'
import { toast } from 'sonner'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import { useParticipantMerch, useParticipantSavingTarget } from '@/entities/merch/queries'
import { formatPoints } from '@/entities/points/format'
import { useParticipantPoints } from '@/entities/points/queries'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'

export function ParticipantShopPage() {
  const { session } = useParticipantAuth()
  const eventId = session?.event.id ?? ''
  const participantId = session?.participant.id ?? ''
  const products = useParticipantMerch(eventId, participantId)
  const points = useParticipantPoints(eventId, participantId)
  const target = useParticipantSavingTarget(eventId, participantId)

  if (!session) return null
  const base = `/event/${encodeURIComponent(session.event.slug)}`

  function toggleTarget(productId: string, active: boolean) {
    target.mutate(active ? null : productId, {
      onSuccess: () => toast.success(active ? 'Цель накопления убрана' : 'Цель накопления выбрана'),
      onError: () => toast.error('Не удалось изменить цель накопления'),
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-col gap-4 rounded-card bg-gradient-to-br from-brand-deep to-brand p-6 text-white sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-[13px] font-semibold uppercase tracking-[0.12em] text-white/70">
            Магазин
          </p>
          <h1 className="mt-2 text-[28px] font-bold">Мерч за баллы</h1>
          <p className="mt-1 max-w-xl text-[14px] text-white/80">
            Выберите товар, накопите баллы и зарезервируйте его до выдачи организатором.
          </p>
        </div>
        <div className="rounded-[12px] bg-white/10 px-4 py-3">
          <p className="text-[12px] text-white/70">Доступно</p>
          <p className="text-[24px] font-bold">
            {points.data ? formatPoints(points.data.balance.available_points) : '—'} баллов
          </p>
        </div>
      </header>

      {products.isLoading && <Skeleton className="h-56 w-full" />}
      {products.isError && <ErrorState onRetry={() => products.refetch()} />}
      {products.data?.length === 0 && (
        <EmptyState
          title="Витрина пока пуста"
          description="Организаторы ещё не опубликовали товары."
        />
      )}
      {!!products.data?.length && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {products.data.map((product) => (
            <Card key={product.id} className="overflow-hidden">
              <Link to={`${base}/shop/${encodeURIComponent(product.slug)}`}>
                {product.images[0]?.url ? (
                  <img
                    src={product.images[0].url!}
                    alt={product.title}
                    className="h-52 w-full object-cover"
                  />
                ) : (
                  <div className="flex h-52 items-center justify-center bg-brand-subtle">
                    <ShoppingBag className="h-12 w-12 text-brand/60" />
                  </div>
                )}
              </Link>
              <CardBody>
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <Link
                      to={`${base}/shop/${encodeURIComponent(product.slug)}`}
                      className="text-[17px] font-semibold text-ink hover:text-brand"
                    >
                      {product.title}
                    </Link>
                    <div className="mt-1 flex flex-wrap items-center gap-2">
                      <span className="text-[16px] font-bold text-brand">
                        {formatPoints(product.effective_price_points)} баллов
                      </span>
                      {product.discount_price_points && (
                        <span className="text-[12px] text-muted line-through">
                          {formatPoints(product.price_points)}
                        </span>
                      )}
                    </div>
                  </div>
                  <button
                    type="button"
                    disabled={target.isPending}
                    onClick={() => toggleTarget(product.id, product.is_saving_target)}
                    className="rounded-btn p-2 text-brand hover:bg-brand-subtle disabled:opacity-50"
                    aria-label={product.is_saving_target ? 'Убрать цель' : 'Выбрать целью'}
                  >
                    {product.is_saving_target ? (
                      <BookmarkCheck className="h-5 w-5" />
                    ) : (
                      <Bookmark className="h-5 w-5" />
                    )}
                  </button>
                </div>
                <p className="mt-2 line-clamp-2 text-[13px] text-muted">{product.description}</p>
                <div className="mt-4 flex items-center justify-between gap-3">
                  <Badge tone={product.available_quantity > 0 ? 'success' : 'danger'}>
                    {product.available_quantity > 0
                      ? `Осталось: ${product.available_quantity}`
                      : 'Нет в наличии'}
                  </Badge>
                  <Button asChild size="sm" variant="subtle">
                    <Link to={`${base}/shop/${encodeURIComponent(product.slug)}`}>Подробнее</Link>
                  </Button>
                </div>
              </CardBody>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
