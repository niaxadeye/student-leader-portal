import { Trophy, Users, FileCheck, ShieldCheck } from 'lucide-react'
import { useAuth } from '@/entities/auth/auth-context'
import { useAdminContests } from '@/entities/contest/queries'
import { useAdminUsers } from '@/entities/user/queries'
import { StatCard } from '@/widgets/stat-card'
import { Skeleton } from '@/shared/ui/states'

export function AdminDashboardPage() {
  const { user } = useAuth()
  const isSuper = !!user?.roles.includes('SUPER_ADMIN')
  const { data: contests, isLoading } = useAdminContests()
  // Реестр юзеров доступен только SUPER_ADMIN — иначе запрос вернёт 403.
  const { data: usersPage } = useAdminUsers({ role: 'ADMIN', limit: 1 }, isSuper)

  const activeContests = contests?.filter((c) => c.status === 'ACTIVE').length ?? 0
  const totalContestants = contests?.reduce((s, c) => s + c.participants_count, 0) ?? 0

  return (
    <div>
      <header className="mb-6">
        <h1 className="text-[28px] font-bold tracking-tight text-ink">
          Здравствуйте, {user?.full_name?.split(' ')[0] ?? 'коллега'}
        </h1>
        <p className="mt-1 text-[15px] text-muted">
          {isSuper
            ? 'Обзор всей системы: конкурсы, пользователи, доступы.'
            : 'Обзор назначенных вам конкурсов.'}
        </p>
      </header>

      {isLoading ? (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-[76px]" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <StatCard label="Конкурсов" value={contests?.length ?? 0} icon={Trophy} accent />
          <StatCard label="Активных" value={activeContests} icon={FileCheck} />
          <StatCard label="Конкурсантов" value={totalContestants} icon={Users} />
          {isSuper ? (
            <StatCard label="Администраторов" value={usersPage?.total ?? 0} icon={ShieldCheck} />
          ) : (
            <StatCard label="Ваша роль" value="Админ" icon={ShieldCheck} />
          )}
        </div>
      )}
    </div>
  )
}
