import { useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Button } from '@/shared/ui/button'
import { Badge } from '@/shared/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'
import { Skeleton, ErrorState } from '@/shared/ui/states'
import { useAdminContests } from '@/entities/contest/queries'
import {
  useClearStaffPermissions,
  useReplaceStaffPermissions,
  useStaffPermissions,
} from '@/entities/user/queries'
import { STAFF_PERMISSION_OPTIONS } from '@/entities/user/staff-api'
import type { StaffPermission } from '@/entities/auth/types'

export function ManageStaffDialog({
  userId,
  login,
  open,
  onOpenChange,
}: {
  userId: string
  login: string
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title="Права на мероприятия" description={login}>
        {open && <StaffBody userId={userId} />}
      </DialogContent>
    </Dialog>
  )
}

function StaffBody({ userId }: { userId: string }) {
  const contests = useAdminContests()
  const grants = useStaffPermissions(userId)
  const replace = useReplaceStaffPermissions(userId)
  const clear = useClearStaffPermissions(userId)
  const [contestId, setContestId] = useState('')
  const [selected, setSelected] = useState<StaffPermission[]>([])

  const contestName = (id: string) => contests.data?.find((c) => c.id === id)?.name ?? id.slice(0, 8)
  const labelOf = (perm: string) =>
    STAFF_PERMISSION_OPTIONS.find((o) => o.value === perm)?.label ?? perm

  function toggle(perm: StaffPermission) {
    setSelected((cur) => (cur.includes(perm) ? cur.filter((p) => p !== perm) : [...cur, perm]))
  }

  function onSave() {
    if (!contestId || selected.length === 0) return
    replace.mutate(
      { contestId, permissions: selected },
      {
        onSuccess: () => {
          toast.success('Права сохранены')
          setSelected([])
        },
        onError: () => toast.error('Не удалось сохранить права'),
      },
    )
  }

  if (grants.isLoading) return <Skeleton className="h-40 w-full" />
  if (grants.isError) return <ErrorState onRetry={() => grants.refetch()} />

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-2">
        {!grants.data?.length ? (
          <p className="text-[14px] text-muted">Права на мероприятия ещё не выданы.</p>
        ) : (
          grants.data.map((g) => (
            <div
              key={g.contest_id}
              className="flex items-start justify-between gap-3 rounded-[10px] border border-border px-3 py-2"
            >
              <div>
                <p className="text-[14px] font-medium text-ink">{contestName(g.contest_id)}</p>
                <div className="mt-1 flex flex-wrap gap-1">
                  {g.permissions.map((p) => (
                    <Badge key={p} tone="neutral">
                      {labelOf(p)}
                    </Badge>
                  ))}
                </div>
              </div>
              <button
                title="Снять все права с мероприятия"
                onClick={() =>
                  clear.mutate(g.contest_id, {
                    onSuccess: () => toast.info('Права сняты'),
                    onError: () => toast.error('Не удалось снять права'),
                  })
                }
                disabled={clear.isPending}
                className="rounded-btn p-1.5 text-muted-2 hover:bg-muted/10 hover:text-danger disabled:opacity-40"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))
        )}
      </div>

      <div className="border-t border-border pt-4">
        <p className="mb-3 text-[14px] font-medium text-ink">Выдать права</p>
        <div className="flex flex-col gap-3">
          <Field label="Мероприятие" required>
            {(p) => (
              <Select value={contestId} onValueChange={setContestId}>
                <SelectTrigger id={p.id}>
                  <SelectValue placeholder="Выберите конкурс" />
                </SelectTrigger>
                <SelectContent>
                  {(contests.data ?? []).map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </Field>
          <div className="grid gap-2">
            {STAFF_PERMISSION_OPTIONS.map((opt) => (
              <label key={opt.value} className="flex items-start gap-2 text-[14px] text-ink">
                <input
                  type="checkbox"
                  className="mt-1"
                  checked={selected.includes(opt.value)}
                  onChange={() => toggle(opt.value)}
                />
                <span>
                  {opt.label}
                  <span className="block text-[12px] text-muted">{opt.hint}</span>
                </span>
              </label>
            ))}
          </div>
          <div className="flex justify-end">
            <Button
              size="sm"
              onClick={onSave}
              loading={replace.isPending}
              disabled={!contestId || selected.length === 0}
            >
              <Plus className="h-4 w-4" /> Сохранить
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
