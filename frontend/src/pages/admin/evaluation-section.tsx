import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Trash2, Pencil, Check, X } from 'lucide-react'
import {
  useAddCriterion,
  useDeleteCriterion,
  useEvaluation,
  usePutRemoteJury,
  usePutStageLink,
  useSaveEvaluation,
  useUpdateCriterion,
} from '@/entities/evaluation/admin-queries'
import {
  evaluationTypeDescriptions,
  evaluationTypeLabels,
  evaluationTypeOrder,
  type EvaluationType,
  type CorridorMode,
  type EvaluationCriterion,
  type CriterionInput,
  type SchemeInput,
  type AdminScoreboardJury,
  type CombineMode,
  type StageLink,
} from '@/entities/evaluation/types'
import { Card, CardBody } from '@/shared/ui/card'
import { Button } from '@/shared/ui/button'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { ApiRequestError } from '@/shared/api/client'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'
import { cn } from '@/shared/lib/cn'
import { toast } from 'sonner'

const DEFAULT_LIVES = 3

export function EvaluationSection({ challengeId, canEdit }: { challengeId: string; canEdit: boolean }) {
  const q = useEvaluation(challengeId)
  if (q.isLoading) return <Skeleton className="h-48 w-full" />
  if (q.isError) return <ErrorState onRetry={() => q.refetch()} />
  const scheme = q.data?.scheme
  const jury = q.data?.jury ?? []
  return (
    <SchemeEditor
      key={`${scheme?.updated_at ?? 'new'}-${scheme?.operator_user_id ?? ''}`}
      challengeId={challengeId}
      canEdit={canEdit}
      configured={q.data?.configured ?? false}
      schemeType={scheme?.type ?? 'CRITERIA_SCORING'}
      corridor={scheme?.corridor_mode ?? 'NONE'}
      minScore={scheme?.min_score ?? 1}
      maxScore={scheme?.max_score ?? (scheme?.type === 'NUMERIC_RESULT' ? 100 : 10)}
      startingLives={scheme?.starting_lives ?? DEFAULT_LIVES}
      operatorUserId={scheme?.operator_user_id ?? ''}
      jury={q.data?.contest_jury ?? jury}
      assignedJury={jury}
      remoteJury={q.data?.remote_jury ?? []}
      criteria={scheme?.criteria ?? []}
      contestId={scheme?.contest_id ?? ''}
      stageLink={q.data?.stage_link ?? null}
      linkedFrom={q.data?.linked_from ?? null}
      remoteStageOptions={q.data?.remote_stage_options ?? []}
    />
  )
}

function SchemeEditor({
  challengeId,
  canEdit,
  configured,
  schemeType,
  corridor,
  minScore,
  maxScore,
  startingLives,
  operatorUserId,
  jury,
  assignedJury,
  remoteJury,
  criteria,
  contestId,
  stageLink,
  linkedFrom,
  remoteStageOptions,
}: {
  challengeId: string
  canEdit: boolean
  configured: boolean
  schemeType: EvaluationType
  corridor: CorridorMode
  minScore: number
  maxScore: number
  startingLives: number
  operatorUserId: string
  jury: AdminScoreboardJury[]
  assignedJury: AdminScoreboardJury[]
  remoteJury: AdminScoreboardJury[]
  criteria: EvaluationCriterion[]
  contestId: string
  stageLink: StageLink | null
  linkedFrom: { id: string; title: string } | null
  remoteStageOptions: Array<{ id: string; title: string }>
}) {
  const save = useSaveEvaluation(challengeId)
  const add = useAddCriterion(challengeId)
  const update = useUpdateCriterion(challengeId)
  const remove = useDeleteCriterion(challengeId)
  const [type, setType] = useState<EvaluationType>(schemeType)
  const [corridorMode, setCorridorMode] = useState<CorridorMode>(corridor)
  const [min, setMin] = useState(String(minScore))
  const [max, setMax] = useState(String(maxScore))
  const [lives, setLives] = useState(String(startingLives || DEFAULT_LIVES))
  const [operator, setOperator] = useState(operatorUserId || 'none')
  const [title, setTitle] = useState('')
  const criteriaType = type === 'CRITERIA_SCORING' || type === 'REMOTE_CRITERIA'

  function schemePayload(): SchemeInput | null {
    if (type === 'ELIMINATION_LIVES') {
      const n = Number(lives)
      if (!Number.isInteger(n) || n < 1 || n > 20) {
        toast.error('Количество жизней — целое число от 1 до 20')
        return null
      }
      return {
        type,
        scoring_unit: 'LIVES',
        corridor_mode: 'NONE',
        starting_lives: n,
        operator_user_id: operator === 'none' ? null : operator,
      }
    }
    if (type === 'NUMERIC_RESULT') {
      const n = Number(max)
      if (!Number.isFinite(n) || n <= 0 || n > 10000) {
        toast.error('Максимальный балл — число больше 0')
        return null
      }
      return {
        type,
        scoring_unit: 'POINTS',
        corridor_mode: 'NONE',
        min_score: 0,
        max_score: n,
      }
    }
    return {
      type,
      corridor_mode: corridorMode,
      min_score: Number(min) || null,
      max_score: Number(max) || null,
    }
  }

  function persist() {
    const payload = schemePayload()
    if (!payload) return
    save.mutate(payload, {
      onSuccess: () => toast.success(configured ? 'Схема обновлена' : 'Схема создана'),
      onError: (err) => {
        const msg =
          err instanceof ApiRequestError && err.code === 'EVALUATION_SCHEME_LOCKED'
            ? 'Схему нельзя менять после старта live-сессии'
            : 'Не удалось сохранить схему'
        toast.error(msg)
      },
    })
  }

  async function addCriterion() {
    const name = title.trim()
    if (!name) return
    const payload = schemePayload()
    if (!payload) return
    try {
      if (!configured) {
        await save.mutateAsync(payload)
      }
      await add.mutateAsync({ title: name, min_score: Number(min) || 1, max_score: Number(max) || 10 })
      setTitle('')
      toast.success('Критерий добавлен')
    } catch {
      toast.error(configured ? 'Не удалось добавить критерий' : 'Не удалось сохранить схему')
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <Card>
        <CardBody className="flex flex-col gap-4 py-5">
          <div>
            <h2 className="text-[18px] font-semibold text-ink">Тип оценивания</h2>
            <p className="mt-1 text-[13px] text-muted">
              Сначала выберите формат. Поля настроек появятся ниже.
            </p>
          </div>
          <div className="grid gap-2 sm:grid-cols-2">
            {evaluationTypeOrder.map((t) => {
              const active = type === t
              return (
                <button
                  key={t}
                  type="button"
                  disabled={!canEdit}
                  onClick={() => {
                    setType(t)
                    if (t === 'NUMERIC_RESULT' && !configured) setMax('100')
                  }}
                  className={cn(
                    'rounded-card border px-4 py-3 text-left transition-colors disabled:opacity-50',
                    active
                      ? 'border-brand bg-brand-subtle'
                      : 'border-border hover:bg-muted/5',
                  )}
                >
                  <p className="text-[14px] font-medium text-ink">{evaluationTypeLabels[t]}</p>
                  <p className="mt-1 text-[13px] text-muted">{evaluationTypeDescriptions[t]}</p>
                </button>
              )
            })}
          </div>
        </CardBody>
      </Card>

      {criteriaType && (
        <Card>
          <CardBody className="flex flex-col gap-4 py-5">
            <div>
              <h2 className="text-[18px] font-semibold text-ink">Шкала</h2>
              <p className="mt-1 text-[13px] text-muted">
                Диапазон баллов и коридор расхождений жюри.
              </p>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Коридор баллов" helpText="Пока сохраняется в схеме; проверка на live — позже.">
                {(p) => (
                  <Select
                    value={corridorMode}
                    onValueChange={(v) => setCorridorMode(v as CorridorMode)}
                    disabled={!canEdit}
                  >
                    <SelectTrigger id={p.id}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="NONE">Нет</SelectItem>
                      <SelectItem value="WARN">Предупреждение</SelectItem>
                      <SelectItem value="STRICT">Строгий</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              </Field>
              <div className="hidden sm:block" />
              <Field label="Мин. балл">
                {(p) => (
                  <Input
                    {...p}
                    type="number"
                    value={min}
                    disabled={!canEdit}
                    onChange={(e) => setMin(e.target.value)}
                  />
                )}
              </Field>
              <Field label="Макс. балл">
                {(p) => (
                  <Input
                    {...p}
                    type="number"
                    value={max}
                    disabled={!canEdit}
                    onChange={(e) => setMax(e.target.value)}
                  />
                )}
              </Field>
            </div>
            {canEdit && (
              <div className="flex justify-end">
                <Button size="sm" onClick={persist} loading={save.isPending}>
                  Сохранить схему
                </Button>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      {type === 'ELIMINATION_LIVES' && (
        <Card>
          <CardBody className="flex flex-col gap-4 py-5">
            <div>
              <h2 className="text-[18px] font-semibold text-ink">Настройки «2 к 1»</h2>
              <p className="mt-1 text-[13px] text-muted">
                Сколько жизней у каждого конкурсанта в начале испытания.
              </p>
            </div>
            <Field
              label="Количество жизней"
              required
              helpText="Неверный ответ снимает жизнь. При нуле конкурсант выбывает."
            >
              {(p) => (
                <Input
                  {...p}
                  type="number"
                  min={1}
                  max={20}
                  step={1}
                  value={lives}
                  disabled={!canEdit}
                  onChange={(e) => setLives(e.target.value)}
                />
              )}
            </Field>
            <Field
              label="Ответственное жюри за 2 к 1"
              required
              helpText="Лог этого члена жюри видит администратор испытания. В «Оценках» доступны логи всех."
            >
              {(p) => (
                <Select value={operator} onValueChange={setOperator} disabled={!canEdit}>
                  <SelectTrigger id={p.id} aria-invalid={p['aria-invalid']}>
                    <SelectValue placeholder="Выберите члена жюри" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">Не назначено</SelectItem>
                    {jury.map((j) => (
                      <SelectItem key={j.user_id} value={j.user_id}>
                        {j.full_name.trim() || j.login}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </Field>
            {jury.length === 0 ? (
              <p className="text-[13px] text-muted">
                Сначала назначьте жюри на конкурс — затем выберите ответственного здесь.
              </p>
            ) : null}
            {canEdit && (
              <div className="flex justify-end">
                <Button size="sm" onClick={persist} loading={save.isPending}>
                  Сохранить схему
                </Button>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      {type === 'NUMERIC_RESULT' && (
        <Card>
          <CardBody className="flex flex-col gap-4 py-5">
            <div>
              <h2 className="text-[18px] font-semibold text-ink">Числовой результат</h2>
              <p className="mt-1 text-[13px] text-muted">
                Задайте максимум. Баллы каждого конкурсанта выставляет администратор на вкладке «Оценки».
                Live-пульта и жеребьёвки нет.
              </p>
            </div>
            <Field
              label="Максимальный балл"
              required
              helpText="Результат конкурсанта не может быть выше этого значения. Допускаются дробные баллы, ноль тоже."
            >
              {(p) => (
                <Input
                  {...p}
                  type="number"
                  min={0.0001}
                  step="any"
                  value={max}
                  disabled={!canEdit}
                  onChange={(e) => setMax(e.target.value)}
                />
              )}
            </Field>
            {canEdit && (
              <div className="flex justify-end">
                <Button size="sm" onClick={persist} loading={save.isPending}>
                  Сохранить схему
                </Button>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      {type !== 'CRITERIA_SCORING' &&
        type !== 'REMOTE_CRITERIA' &&
        type !== 'ELIMINATION_LIVES' &&
        type !== 'NUMERIC_RESULT' && (
        <Card>
          <CardBody className="flex flex-col gap-4 py-5">
            <div>
              <h2 className="text-[18px] font-semibold text-ink">{evaluationTypeLabels[type]}</h2>
              <p className="mt-1 text-[13px] text-muted">
                Поля настроек для этого типа появятся позже. Тип можно сохранить сейчас.
              </p>
            </div>
            {canEdit && (
              <div className="flex justify-end">
                <Button size="sm" onClick={persist} loading={save.isPending}>
                  Сохранить тип
                </Button>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      {criteriaType && (
        <div>
          <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
            <h2 className="text-[18px] font-semibold text-ink">Критерии</h2>
            {canEdit && (
              <div className="flex gap-2">
                <Input
                  placeholder="Название критерия"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      void addCriterion()
                    }
                  }}
                />
                <Button
                  size="sm"
                  onClick={() => void addCriterion()}
                  loading={add.isPending || save.isPending}
                  disabled={!title.trim()}
                >
                  <Plus className="h-4 w-4" /> Добавить
                </Button>
              </div>
            )}
          </div>
          {criteria.length === 0 ? (
            <EmptyState
              title="Критериев пока нет"
              description={
                type === 'REMOTE_CRITERIA'
                  ? 'Сохраните схему и добавьте критерии — по ним заочное жюри оценит сданные работы.'
                  : 'Сохраните схему и добавьте первый критерий — его увидит жюри на live-сессии.'
              }
            />
          ) : (
            <ul className="flex flex-col gap-2">
              {criteria.map((c) => (
                <CriterionRow
                  key={c.id}
                  criterion={c}
                  canEdit={canEdit}
                  saving={update.isPending}
                  onSave={(input) =>
                    update.mutate(
                      { id: c.id, input },
                      {
                        onSuccess: () => toast.success('Критерий обновлён'),
                        onError: (err) => {
                          const msg =
                            err instanceof ApiRequestError && err.code === 'EVALUATION_SCHEME_LOCKED'
                              ? 'Схему нельзя менять после старта live-сессии'
                              : 'Не удалось сохранить критерий'
                          toast.error(msg)
                        },
                      },
                    )
                  }
                  onDelete={() =>
                    remove.mutate(c.id, {
                      onSuccess: () => toast.info('Критерий удалён'),
                      onError: () => toast.error('Не удалось удалить'),
                    })
                  }
                />
              ))}
            </ul>
          )}
        </div>
      )}

      {type === 'REMOTE_CRITERIA' && linkedFrom && contestId && (
        <Card>
          <CardBody className="py-5">
            <h2 className="text-[18px] font-semibold text-ink">Основной этап</h2>
            <p className="mt-1 text-[13px] text-muted">
              Это заочный этап привязан к испытанию{' '}
              <Link
                to={`/admin/contests/${contestId}/challenges/${linkedFrom.id}/run`}
                className="text-brand hover:underline"
              >
                {linkedFrom.title}
              </Link>
              . Итоговый рейтинг считается там.
            </p>
          </CardBody>
        </Card>
      )}

      {type !== 'REMOTE_CRITERIA' && (
        <StageLinkPanel
          key={`${stageLink?.remote_challenge_id ?? ''}-${stageLink?.combine_mode ?? ''}`}
          challengeId={challengeId}
          canEdit={canEdit}
          configured={configured && schemeType !== 'REMOTE_CRITERIA'}
          options={remoteStageOptions}
          stageLink={stageLink}
        />
      )}

      {type === 'REMOTE_CRITERIA' && (
        <RemoteJuryPanel
          key={assignedJury.map((j) => j.user_id).join(',')}
          challengeId={challengeId}
          canEdit={canEdit}
          configured={configured && schemeType === 'REMOTE_CRITERIA'}
          remoteJury={remoteJury}
          assignedJury={assignedJury}
        />
      )}
    </div>
  )
}

function personLabel(j: AdminScoreboardJury): string {
  return j.full_name.trim() || j.login
}

function StageLinkPanel({
  challengeId,
  canEdit,
  configured,
  options,
  stageLink,
}: {
  challengeId: string
  canEdit: boolean
  configured: boolean
  options: Array<{ id: string; title: string }>
  stageLink: StageLink | null
}) {
  const save = usePutStageLink(challengeId)
  const [remoteId, setRemoteId] = useState(stageLink?.remote_challenge_id ?? 'none')
  const [mainWeight, setMainWeight] = useState(String(stageLink?.main_weight ?? 1))
  const [remoteWeight, setRemoteWeight] = useState(String(stageLink?.remote_weight ?? 1))
  const [mode, setMode] = useState<CombineMode>(stageLink?.combine_mode ?? 'RANK_SUM')

  function persist() {
    if (remoteId === 'none') {
      save.mutate(
        { remote_challenge_id: null, main_weight: 1, remote_weight: 1, combine_mode: mode },
        {
          onSuccess: () => toast.success('Связь с заочным этапом снята'),
          onError: () => toast.error('Не удалось сохранить связь этапов'),
        },
      )
      return
    }
    const mw = Number(mainWeight.replace(',', '.'))
    const rw = Number(remoteWeight.replace(',', '.'))
    if (!Number.isFinite(mw) || !Number.isFinite(rw) || mw <= 0 || rw <= 0) {
      toast.error('Веса должны быть положительными числами')
      return
    }
    save.mutate(
      { remote_challenge_id: remoteId, main_weight: mw, remote_weight: rw, combine_mode: mode },
      {
        onSuccess: () => toast.success('Заочный этап привязан'),
        onError: () => toast.error('Не удалось сохранить связь этапов'),
      },
    )
  }

  return (
    <Card>
      <CardBody className="flex flex-col gap-4 py-5">
        <div>
          <h2 className="text-[18px] font-semibold text-ink">Заочный этап</h2>
          <p className="mt-1 text-[13px] text-muted">
            Привяжите заочное испытание этого конкурса. Итоговый рейтинг считается на вкладке «Оценки».
          </p>
        </div>
        {!configured ? (
          <p className="text-[14px] text-muted">Сначала сохраните схему основного испытания.</p>
        ) : (
          <>
            <Field label="Заочное испытание">
              {(p) => (
                <Select value={remoteId} onValueChange={setRemoteId} disabled={!canEdit}>
                  <SelectTrigger id={p.id}>
                    <SelectValue placeholder="Не привязано" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">Не привязано</SelectItem>
                    {options.map((o) => (
                      <SelectItem key={o.id} value={o.id}>
                        {o.title}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </Field>
            {options.length === 0 && remoteId === 'none' && (
              <p className="text-[13px] text-muted">
                Нет свободных заочных испытаний. Создайте испытание с типом «Заочное оценивание».
              </p>
            )}
            {remoteId !== 'none' && (
              <>
                <div className="grid gap-3 sm:grid-cols-2">
                  <Field label="Вес основного этапа" helpText="Чем больше вес, тем сильнее влияет этот этап.">
                    {(p) => (
                      <Input
                        {...p}
                        type="number"
                        min={0.0001}
                        step="any"
                        value={mainWeight}
                        disabled={!canEdit}
                        onChange={(e) => setMainWeight(e.target.value)}
                      />
                    )}
                  </Field>
                  <Field label="Вес заочного этапа">
                    {(p) => (
                      <Input
                        {...p}
                        type="number"
                        min={0.0001}
                        step="any"
                        value={remoteWeight}
                        disabled={!canEdit}
                        onChange={(e) => setRemoteWeight(e.target.value)}
                      />
                    )}
                  </Field>
                </div>
                <Field label="Как сводить рейтинг">
                  {() => (
                    <div className="flex flex-col gap-2 text-[14px]">
                      <label className="flex items-start gap-2">
                        <input
                          type="radio"
                          name="combine-mode"
                          checked={mode === 'RANK_SUM'}
                          disabled={!canEdit}
                          onChange={() => setMode('RANK_SUM')}
                          className="mt-1"
                        />
                        <span>
                          Сумма рейтингов — у кого меньше сумма мест (с учётом весов), тот выше.
                        </span>
                      </label>
                      <label className="flex items-start gap-2">
                        <input
                          type="radio"
                          name="combine-mode"
                          checked={mode === 'SCORE_SUM'}
                          disabled={!canEdit}
                          onChange={() => setMode('SCORE_SUM')}
                          className="mt-1"
                        />
                        <span>
                          Сумма баллов — у кого больше сумма баллов (с учётом весов), тот выше.
                        </span>
                      </label>
                    </div>
                  )}
                </Field>
              </>
            )}
            {canEdit && (
              <div className="flex justify-end">
                <Button size="sm" loading={save.isPending} onClick={persist}>
                  Сохранить связь
                </Button>
              </div>
            )}
          </>
        )}
      </CardBody>
    </Card>
  )
}

function RemoteJuryPanel({
  challengeId,
  canEdit,
  configured,
  remoteJury,
  assignedJury,
}: {
  challengeId: string
  canEdit: boolean
  configured: boolean
  remoteJury: AdminScoreboardJury[]
  assignedJury: AdminScoreboardJury[]
}) {
  const save = usePutRemoteJury(challengeId)
  const [selected, setSelected] = useState<string[]>(() => assignedJury.map((j) => j.user_id))

  function toggle(id: string) {
    setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  return (
    <Card>
      <CardBody className="flex flex-col gap-4 py-5">
        <div>
          <h2 className="text-[18px] font-semibold text-ink">Заочное жюри</h2>
          <p className="mt-1 text-[13px] text-muted">
            Отметьте, кто оценивает именно это испытание. Список — пользователи с ролью «Заочное жюри»
            на этот конкурс. Они не входят в live-жюри и не видят очные испытания.
          </p>
        </div>
        {!configured ? (
          <p className="text-[14px] text-muted">Сохраните схему, затем отметьте жюри.</p>
        ) : remoteJury.length === 0 ? (
          <p className="text-[14px] text-muted">
            На конкурс пока нет заочного жюри. Создайте пользователя с ролью «Заочное жюри» в разделе
            «Пользователи».
          </p>
        ) : (
          <div className="flex max-h-64 flex-col gap-1 overflow-auto rounded-btn border border-border p-2">
            {remoteJury.map((j) => (
              <label
                key={j.user_id}
                className="flex cursor-pointer items-center gap-2 rounded-btn px-2 py-1.5 text-[14px] hover:bg-muted/10"
              >
                <input
                  type="checkbox"
                  checked={selected.includes(j.user_id)}
                  disabled={!canEdit}
                  onChange={() => toggle(j.user_id)}
                />
                <span className="min-w-0 truncate">{personLabel(j)}</span>
                <span className="text-[12px] text-muted">{j.login}</span>
              </label>
            ))}
          </div>
        )}
        {configured && canEdit && remoteJury.length > 0 && (
          <div className="flex justify-end">
            <Button
              size="sm"
              loading={save.isPending}
              onClick={() =>
                save.mutate(selected, {
                  onSuccess: () =>
                    toast.success(
                      selected.length
                        ? `Назначено жюри: ${selected.length}`
                        : 'Список заочного жюри очищен — оценки в этом испытании никто не ставит',
                    ),
                  onError: () => toast.error('Не удалось сохранить состав жюри'),
                })
              }
            >
              Сохранить жюри
            </Button>
          </div>
        )}
      </CardBody>
    </Card>
  )
}

function CriterionRow({
  criterion,
  canEdit,
  saving,
  onSave,
  onDelete,
}: {
  criterion: EvaluationCriterion
  canEdit: boolean
  saving: boolean
  onSave: (input: CriterionInput) => void
  onDelete: () => void
}) {
  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState(criterion.title)
  const [description, setDescription] = useState(criterion.description ?? '')
  const [min, setMin] = useState(String(criterion.min_score))
  const [max, setMax] = useState(String(criterion.max_score))
  const [weight, setWeight] = useState(String(criterion.weight))

  function save() {
    const name = title.trim()
    if (!name) {
      toast.error('Укажите название критерия')
      return
    }
    onSave({
      title: name,
      description: description.trim() || null,
      min_score: Number(min) || 1,
      max_score: Number(max) || 10,
      weight: Number(weight) || 1,
      is_required: criterion.is_required,
      bands: criterion.bands.map((b) => ({
        min_score: b.min_score,
        max_score: b.max_score,
        description: b.description,
      })),
    })
    setOpen(false)
  }

  return (
    <li className="rounded-card border border-border bg-surface px-4 py-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[14px] font-medium text-ink">{criterion.title}</p>
          <p className="text-[12px] text-muted">
            {criterion.min_score}–{criterion.max_score} · вес {criterion.weight}
            {criterion.bands.length ? ` · ${criterion.bands.length} пояснения шкалы` : ''}
          </p>
          {criterion.description && !open && (
            <p className="mt-1 text-[13px] text-muted">{criterion.description}</p>
          )}
        </div>
        {canEdit && (
          <div className="flex shrink-0 gap-1">
            <button
              type="button"
              title={open ? 'Свернуть' : 'Изменить'}
              onClick={() => setOpen((v) => !v)}
              className="rounded-btn p-1.5 text-muted-2 hover:bg-muted/10 hover:text-ink"
            >
              {open ? <X className="h-4 w-4" /> : <Pencil className="h-4 w-4" />}
            </button>
            <button
              type="button"
              title="Удалить"
              onClick={onDelete}
              className="rounded-btn p-1.5 text-muted-2 hover:bg-muted/10 hover:text-danger"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        )}
      </div>
      {open && canEdit && (
        <div className="mt-3 flex flex-col gap-3 border-t border-border pt-3">
          <Field label="Название" required>
            {(p) => <Input {...p} value={title} onChange={(e) => setTitle(e.target.value)} />}
          </Field>
          <Field label="Описание">
            {(p) => (
              <Textarea {...p} rows={2} value={description} onChange={(e) => setDescription(e.target.value)} />
            )}
          </Field>
          <div className="grid gap-3 sm:grid-cols-3">
            <Field label="Мин.">
              {(p) => <Input {...p} type="number" value={min} onChange={(e) => setMin(e.target.value)} />}
            </Field>
            <Field label="Макс.">
              {(p) => <Input {...p} type="number" value={max} onChange={(e) => setMax(e.target.value)} />}
            </Field>
            <Field label="Вес">
              {(p) => <Input {...p} type="number" value={weight} onChange={(e) => setWeight(e.target.value)} />}
            </Field>
          </div>
          <div className="flex justify-end">
            <Button size="sm" onClick={save} loading={saving}>
              <Check className="h-4 w-4" /> Сохранить критерий
            </Button>
          </div>
        </div>
      )}
    </li>
  )
}
