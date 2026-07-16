import { useRef, useState } from 'react'
import { KeyRound, Ban, UserPlus, Upload, Download, RotateCcw, Trash2 } from 'lucide-react'
import { useContestants, useImportContestants, useRemoveContestant } from '@/entities/contestant/queries'
import { useResetPassword, useUserStatusMutation } from '@/entities/user/queries'
import { exportContestants } from '@/entities/contestant/api'
import { Card } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'
import { toast } from 'sonner'
import { AddContestantDialog } from './add-contestant-dialog'
import type { Contestant } from '@/entities/contestant/types'

// canManage — управление составом участников доступно только владельцу/меге (§3.6).
// Экспорт read-only и доступен всем с доступом к конкурсу.
export function ContestantsTable({ contestId, canManage }: { contestId: string; canManage: boolean }) {
  const { data, isLoading, isError, refetch } = useContestants(contestId)
  const [addOpen, setAddOpen] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)
  const importer = useImportContestants(contestId)

  function onImportFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    file.text().then((csv) => {
      importer.mutate(csv, {
        onSuccess: (r) =>
          (r.failed ? toast.info : toast.success)(
            `Импорт: добавлено ${r.created}, ошибок ${r.failed}`,
          ),
        onError: () => toast.error('Не удалось импортировать файл'),
      })
    })
    e.target.value = ''
  }

  async function onExport() {
    try {
      const csv = await exportContestants(contestId)
      const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }))
      const a = document.createElement('a')
      a.href = url
      a.download = 'contestants.csv'
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      toast.error('Не удалось выгрузить')
    }
  }

  if (isLoading) return <Skeleton className="h-48 w-full" />
  if (isError) return <ErrorState onRetry={() => refetch()} />

  const Toolbar = (
    <div className="flex flex-wrap gap-2">
      {canManage && (
        <>
          <input ref={fileRef} type="file" accept=".csv,text/csv" hidden onChange={onImportFile} />
          <Button size="sm" variant="outline" loading={importer.isPending} onClick={() => fileRef.current?.click()}>
            <Upload className="h-4 w-4" /> Импорт CSV
          </Button>
        </>
      )}
      {!!data?.length && (
        <Button size="sm" variant="outline" onClick={onExport}>
          <Download className="h-4 w-4" /> Экспорт
        </Button>
      )}
      {canManage && (
        <Button size="sm" onClick={() => setAddOpen(true)}>
          <UserPlus className="h-4 w-4" /> Добавить
        </Button>
      )}
    </div>
  )

  return (
    <>
      {!data?.length ? (
        <EmptyState
          icon={UserPlus}
          title="Конкурсантов пока нет"
          description={
            canManage
              ? 'Добавьте участников вручную или импортом из CSV.'
              : 'Список участников пуст.'
          }
          action={canManage ? Toolbar : undefined}
        />
      ) : (
        <Card className="overflow-hidden">
          <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-3">
            <span className="text-[13px] text-muted">{data.length} участников</span>
            {Toolbar}
          </div>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[480px] text-left text-[14px]">
              <thead className="text-[12px] uppercase tracking-wide text-muted-2">
                <tr className="border-b border-border">
                  <th className="px-4 py-2 font-medium">Конкурсант</th>
                  <th className="px-4 py-2 font-medium">Статус</th>
                  {canManage && <th className="px-4 py-2 text-right font-medium">Действия</th>}
                </tr>
              </thead>
              <tbody>
                {data.map((c) => (
                  <ContestantRow key={c.user_id} c={c} contestId={contestId} canManage={canManage} />
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
      <AddContestantDialog contestId={contestId} open={addOpen} onOpenChange={setAddOpen} />
    </>
  )
}

function ContestantRow({ c, contestId, canManage }: { c: Contestant; contestId: string; canManage: boolean }) {
  const reset = useResetPassword()
  const status = useUserStatusMutation()
  const remove = useRemoveContestant(contestId)
  const blocked = c.user_status === 'BLOCKED'

  function onReset() {
    reset.mutate(c.user_id, {
      onSuccess: (r) =>
        toast.success(`Пароль сброшен: ${c.login}`, {
          description: `Временный пароль: ${r.temp_password}`,
        }),
      onError: () => toast.error('Не удалось сбросить пароль'),
    })
  }

  function onToggleBlock() {
    status.mutate(
      { userId: c.user_id, block: !blocked },
      {
        onSuccess: () =>
          toast.info(blocked ? `Разблокирован: ${c.login}` : `Заблокирован: ${c.login}`),
        onError: () => toast.error('Не удалось изменить статус'),
      },
    )
  }

  function onRemove() {
    if (!confirm(`Убрать конкурсанта ${c.login} из конкурса?`)) return
    remove.mutate(c.user_id, {
      onSuccess: () => toast.info(`Убран: ${c.login}`),
      onError: () => toast.error('Не удалось убрать'),
    })
  }

  return (
    <tr className="border-b border-border last:border-0 hover:bg-muted/5">
      <td className="px-4 py-3">
        <p className="font-medium text-ink">{c.full_name}</p>
        <p className="text-[13px] text-muted-2">
          {c.login}
          {c.organization ? ` · ${c.organization}` : ''}
        </p>
      </td>
      <td className="px-4 py-3">
        {blocked ? <Badge tone="danger">Заблокирован</Badge> : <Badge tone="success">Активен</Badge>}
      </td>
      {canManage && (
        <td className="px-4 py-3">
          <div className="flex justify-end gap-1">
            <IconBtn title="Сбросить пароль" onClick={onReset} disabled={reset.isPending}>
              <KeyRound className="h-4 w-4" />
            </IconBtn>
            <IconBtn title={blocked ? 'Разблокировать' : 'Заблокировать'} onClick={onToggleBlock} disabled={status.isPending} danger>
              {blocked ? <RotateCcw className="h-4 w-4" /> : <Ban className="h-4 w-4" />}
            </IconBtn>
            <IconBtn title="Убрать из конкурса" onClick={onRemove} disabled={remove.isPending} danger>
              <Trash2 className="h-4 w-4" />
            </IconBtn>
          </div>
        </td>
      )}
    </tr>
  )
}

function IconBtn({
  title,
  onClick,
  disabled,
  danger,
  children,
}: {
  title: string
  onClick: () => void
  disabled?: boolean
  danger?: boolean
  children: React.ReactNode
}) {
  return (
    <button
      title={title}
      onClick={onClick}
      disabled={disabled}
      className={`rounded-btn p-2 text-muted-2 hover:bg-muted/10 disabled:opacity-40 ${danger ? 'hover:text-danger' : 'hover:text-brand'}`}
    >
      {children}
    </button>
  )
}
