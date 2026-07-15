import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { UserPlus, Search } from 'lucide-react'
import { useAdminUsers } from '@/entities/user/queries'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Skeleton, ErrorState, EmptyState } from '@/shared/ui/states'
import { CreateUserDialog } from './create-user-dialog'
import { UsersTable } from './users-table'

const PAGE = 20

export function AdminUsersPage() {
  const [params, setParams] = useSearchParams()
  const [createOpen, setCreateOpen] = useState(false)

  const search = params.get('search') ?? ''
  const role = params.get('role') ?? ''
  const status = params.get('status') ?? ''
  const offset = Number(params.get('offset') ?? 0)

  function patch(next: Record<string, string>) {
    const p = new URLSearchParams(params)
    for (const [k, v] of Object.entries(next)) {
      if (v) p.set(k, v)
      else p.delete(k)
    }
    if (!('offset' in next)) p.delete('offset') // смена фильтра сбрасывает страницу
    setParams(p, { replace: true })
  }

  const { data, isLoading, isError, refetch } = useAdminUsers({
    search: search || undefined,
    role: role || undefined,
    status: status || undefined,
    limit: PAGE,
    offset,
  })

  const total = data?.total ?? 0
  const from = total === 0 ? 0 : offset + 1
  const to = Math.min(offset + PAGE, total)

  return (
    <div>
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-[28px] font-bold tracking-tight text-ink">Пользователи</h1>
          <p className="mt-1 text-[15px] text-muted">
            Реестр всех учётных записей. Доступно только суперадмину.
          </p>
        </div>
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <UserPlus className="h-4 w-4" /> Новый пользователь
        </Button>
      </header>

      <div className="mb-4 flex flex-wrap gap-2">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-2" />
          <Input
            className="pl-9"
            placeholder="Поиск по логину или ФИО"
            defaultValue={search}
            onChange={(e) => patch({ search: e.target.value })}
          />
        </div>
        <FilterSelect value={role} onChange={(v) => patch({ role: v })} label="Все роли" options={[
          ['SUPER_ADMIN', 'Суперадмины'],
          ['ADMIN', 'Админы'],
          ['CONTESTANT', 'Конкурсанты'],
        ]} />
        <FilterSelect value={status} onChange={(v) => patch({ status: v })} label="Все статусы" options={[
          ['ACTIVE', 'Активные'],
          ['BLOCKED', 'Заблокированные'],
        ]} />
      </div>

      {isLoading ? (
        <Skeleton className="h-72 w-full" />
      ) : isError ? (
        <ErrorState onRetry={() => refetch()} />
      ) : !data?.users.length ? (
        <EmptyState title="Ничего не найдено" description="Измените параметры поиска или фильтры." />
      ) : (
        <>
          <UsersTable users={data.users} />
          <div className="mt-4 flex items-center justify-between text-[13px] text-muted">
            <span>
              {from}–{to} из {total}
            </span>
            <div className="flex gap-2">
              <Button size="sm" variant="secondary" disabled={offset === 0} onClick={() => patch({ offset: String(Math.max(0, offset - PAGE)) })}>
                Назад
              </Button>
              <Button size="sm" variant="secondary" disabled={to >= total} onClick={() => patch({ offset: String(offset + PAGE) })}>
                Вперёд
              </Button>
            </div>
          </div>
        </>
      )}

      <CreateUserDialog open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  )
}

function FilterSelect({
  value,
  onChange,
  label,
  options,
}: {
  value: string
  onChange: (v: string) => void
  label: string
  options: Array<[string, string]>
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="h-11 rounded-[10px] border border-border bg-surface px-3 text-[14px] text-ink focus-visible:border-brand focus-visible:outline-none"
    >
      <option value="">{label}</option>
      {options.map(([v, l]) => (
        <option key={v} value={v}>
          {l}
        </option>
      ))}
    </select>
  )
}
