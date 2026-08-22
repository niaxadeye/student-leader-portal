import { useChallengeDraw } from '@/entities/evaluation/contestant-queries'
import { useAppConfig } from '@/shared/config/use-app-config'
import { Card, CardBody } from '@/shared/ui/card'
import { cn } from '@/shared/lib/cn'

export function ChallengeDrawCard({ challengeId }: { challengeId: string }) {
  const { data: appConfig } = useAppConfig()
  const q = useChallengeDraw(challengeId, appConfig?.features.jury === true)
  const draw = q.data
  if (!draw?.drawn) return null

  return (
    <Card>
      <CardBody className="py-5">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <p className="text-[13px] uppercase tracking-wide text-muted-2">Жеребьёвка</p>
            <p className="mt-1 text-[22px] font-semibold text-ink">
              {draw.my_draw_number != null ? `Ваш номер — №${draw.my_draw_number}` : 'Вы не в жеребьёвке'}
            </p>
            <p className="mt-1 text-[13px] text-muted">из {draw.total} по этому испытанию</p>
          </div>
        </div>
        {draw.order.length > 0 && (
          <ol className="mt-4 divide-y divide-border rounded-[12px] border border-border">
            {draw.order.map((row) => (
              <li
                key={row.draw_number}
                className={cn(
                  'flex items-center justify-between px-3 py-2 text-[14px]',
                  row.is_me ? 'bg-brand-subtle font-medium text-brand' : 'text-ink',
                )}
              >
                <span>
                  №{row.draw_number} {row.full_name}
                  {row.organization?.trim() ? ` · ${row.organization.trim()}` : ''}
                </span>
                {row.is_me && <span className="text-[12px]">вы</span>}
              </li>
            ))}
          </ol>
        )}
      </CardBody>
    </Card>
  )
}
