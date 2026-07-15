import { useState } from 'react'
import { Building2, Search } from 'lucide-react'
import { useAdminUsers } from '@/entities/user/queries'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Card } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { Skeleton, ErrorState, EmptyState } from '@/shared/ui/states'
import { formatDate } from '@/shared/lib/format'
import { CreateOrganizerDialog } from './create-organizer-dialog'

const PAGE = 50

/** Экран организаторов (только MEGA_ADMIN): реестр SUPER_ADMIN + создание. */
export function OrganizersPage() {
  const [createOpen, setCreateOpen] = useState(false)
  const [search, setSearch] = useState('')

  const { data, isLoading, isError, refetch } = useAdminUsers({
    role: 'SUPER_ADMIN',
    search: search || undefined,
    limit: PAGE,
  })

  return (
    <div>
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-[28px] font-bold tracking-tight text-ink">Организаторы</h1>
          <p className="mt-1 text-[15px] text-muted">
            Суперадмины-организаторы. Каждый видит только свои конкурсы и пользователей.
          </p>
        </div>
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <Building2 className="h-4 w-4" /> Новый организатор
        </Button>
      </header>

      <div className="mb-4 flex flex-wrap gap-2">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-2" />
          <Input
            className="pl-9"
            placeholder="Поиск по логину или ФИО"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      </div>

      {isLoading ? (
        <Skeleton className="h-72 w-full" />
      ) : isError ? (
        <ErrorState onRetry={() => refetch()} />
      ) : !data?.users.length ? (
        <EmptyState
          title="Организаторов пока нет"
          description="Создайте первого суперадмина-организатора."
        />
      ) : (
        <Card className="overflow-hidden">
          <table className="w-full text-left text-[14px]">
            <thead className="text-[12px] uppercase tracking-wide text-muted-2">
              <tr className="border-b border-border">
                <th className="px-4 py-2 font-medium">Организатор</th>
                <th className="px-4 py-2 font-medium">Организация</th>
                <th className="hidden px-4 py-2 font-medium md:table-cell">Создан</th>
                <th className="px-4 py-2 font-medium">Статус</th>
              </tr>
            </thead>
            <tbody>
              {data.users.map((u) => (
                <tr key={u.id} className="border-b border-border last:border-0 hover:bg-muted/5">
                  <td className="px-4 py-3">
                    <p className="font-medium text-ink">{u.full_name}</p>
                    <p className="text-[13px] text-muted-2">{u.login}</p>
                  </td>
                  <td className="px-4 py-3 text-muted">{u.org_name ?? '—'}</td>
                  <td className="hidden px-4 py-3 text-muted md:table-cell">
                    {formatDate(u.created_at)}
                  </td>
                  <td className="px-4 py-3">
                    {u.status === 'BLOCKED' ? (
                      <Badge tone="danger">Заблокирован</Badge>
                    ) : (
                      <Badge tone="success">Активен</Badge>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}

      <CreateOrganizerDialog open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  )
}
