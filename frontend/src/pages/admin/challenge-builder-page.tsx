import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, Plus, Rocket, Lock, Archive, Eye, PencilRuler, Inbox, Pencil, Copy, Scale, Radio, ClipboardList } from 'lucide-react'
import {
  useAdminChallenge,
  useChallengeFields,
  useDuplicateChallenge,
  useTransitionChallenge,
} from '@/entities/challenge/admin-queries'
import { useAdminContest } from '@/entities/contest/queries'
import { canEditContest } from '@/entities/contest/types'
import { schemeHasLive } from '@/entities/evaluation/types'
import { useAppConfig } from '@/shared/config/use-app-config'
import { Card, CardBody } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'
import { toast } from 'sonner'
import { formatDateTime } from '@/shared/lib/format'
import { ApiRequestError } from '@/shared/api/client'
import type { AdminField, ChallengeStatus } from '@/entities/challenge/admin-types'
import { challengeStatusMeta } from './challenge-status'
import { FieldsList } from './fields-list'
import { FieldEditorDialog } from './field-editor-dialog'
import { EditChallengeDialog } from './edit-challenge-dialog'
import { ChallengePreview } from './challenge-preview'
import { SubmissionsSection } from './submissions-section'
import { EvaluationSection } from './evaluation-section'
import { EvaluationLiveSection } from './evaluation-live-section'
import { EvaluationScoresSection } from './evaluation-scores-section'
import { ChallengeDangerZone } from './challenge-danger-zone'

/** Доступные переходы по статусу (зеркалит матрицу бэкенда). */
const actionsByStatus: Record<ChallengeStatus, Array<'publish' | 'close' | 'archive'>> = {
  DRAFT: ['publish', 'archive'],
  PUBLISHED: ['close', 'archive'],
  CLOSED: ['publish', 'archive'],
  ARCHIVED: [],
}

const actionMeta = {
  publish: { label: 'Опубликовать приём', icon: Rocket, variant: 'primary' as const },
  close: { label: 'Закрыть приём', icon: Lock, variant: 'secondary' as const },
  archive: { label: 'В архив', icon: Archive, variant: 'secondary' as const },
}

export function ChallengeIntakePage() {
  return <ChallengeWorkspace section="intake" />
}

export function ChallengeRunPage() {
  return <ChallengeWorkspace section="run" />
}

export function ChallengeBuilderPage() {
  return <ChallengeWorkspace section="intake" />
}

function ChallengeWorkspace({ section }: { section: 'intake' | 'run' }) {
  const { challengeId } = useParams()
  const challengeQ = useAdminChallenge(challengeId)
  const fieldsQ = useChallengeFields(challengeId)
  const { data: contest } = useAdminContest(challengeQ.data?.contest_id)
  const { data: appConfig } = useAppConfig()
  const juryEnabled = appConfig?.features.jury === true
  const transition = useTransitionChallenge(challengeId!, challengeQ.data?.contest_id ?? '')
  const duplicate = useDuplicateChallenge(challengeQ.data?.contest_id ?? '')
  const navigate = useNavigate()
  const [tab, setTab] = useState<'build' | 'preview' | 'submissions' | 'evaluation' | 'live' | 'scores'>(
    section === 'run' ? 'evaluation' : 'build',
  )
  const [editorOpen, setEditorOpen] = useState(false)
  const [editing, setEditing] = useState<AdminField | null>(null)
  const [metaOpen, setMetaOpen] = useState(false)

  if (challengeQ.isLoading) return <Skeleton className="h-64 w-full" />
  if (challengeQ.isError) return <ErrorState onRetry={() => challengeQ.refetch()} />
  const challenge = challengeQ.data
  if (!challenge)
    return (
      <EmptyState
        title="Испытание не найдено"
        description="Возможно, у вас нет доступа к этому испытанию."
      />
    )

  const meta = challengeStatusMeta[challenge.status]
  const actions = actionsByStatus[challenge.status]
  const fields = fieldsQ.data ?? []
  // Уровень доступа берём с родительского конкурса (испытание его не несёт).
  const canEdit = canEditContest(contest?.access_level)
  const hasLive = schemeHasLive(challenge.scheme_type)
  const runTab = tab === 'live' && !hasLive ? 'scores' : tab

  function runTransition(action: 'publish' | 'close' | 'archive') {
    transition.mutate(action, {
      onSuccess: () => toast.success(`${actionMeta[action].label}: готово`),
      onError: (err) => {
        const msg =
          err instanceof ApiRequestError && err.code === 'INVALID_TRANSITION'
            ? 'Такой переход статуса недоступен'
            : 'Не удалось изменить статус'
        toast.error(msg)
      },
    })
  }

  function openCreate() {
    setEditing(null)
    setEditorOpen(true)
  }
  function openEdit(field: AdminField) {
    setEditing(field)
    setEditorOpen(true)
  }

  return (
    <div>
      <Link
        to={`/admin/challenges/${challenge.id}`}
        className="mb-4 inline-flex items-center gap-1 text-[14px] text-muted hover:text-ink"
      >
        <ArrowLeft className="h-4 w-4" /> К испытанию
      </Link>

      <header className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-[28px] font-bold tracking-tight text-ink">{challenge.title}</h1>
            {section === 'intake' && <Badge tone={meta.tone}>{meta.label}</Badge>}
          </div>
          <p className="mt-1 text-[14px] text-muted">
            {section === 'intake' ? 'Приём файлов и ТЗ' : 'Проведение испытания'}
            {challenge.held_at ? ` · ${formatDateTime(challenge.held_at)}` : ''}
            {challenge.venue ? ` · ${challenge.venue}` : ''}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {canEdit && (
            <Button
              size="sm"
              variant="secondary"
              loading={duplicate.isPending}
              onClick={() =>
                duplicate.mutate(challenge.id, {
                  onSuccess: (copy) => {
                    toast.success(`Создана копия: ${copy.title}`)
                    navigate(`/admin/challenges/${copy.id}`)
                  },
                  onError: () => toast.error('Не удалось дублировать испытание'),
                })
              }
            >
              <Copy className="h-4 w-4" /> Дублировать
            </Button>
          )}
          {canEdit && challenge.status !== 'ARCHIVED' && (
            <Button size="sm" variant="secondary" onClick={() => setMetaOpen(true)}>
              <Pencil className="h-4 w-4" /> Редактировать
            </Button>
          )}
          {section === 'intake' &&
            canEdit &&
            actions.map((a) => {
              const Icon = actionMeta[a].icon
              return (
                <Button
                  key={a}
                  size="sm"
                  variant={actionMeta[a].variant}
                  loading={transition.isPending}
                  onClick={() => runTransition(a)}
                >
                  <Icon className="h-4 w-4" /> {actionMeta[a].label}
                </Button>
              )
            })}
        </div>
      </header>

      {section === 'intake' && challenge.status === 'PUBLISHED' && (
        <div className="mb-4 rounded-card bg-amber-50 px-4 py-3 text-[13px] text-amber-800">
          Приём опубликован. Изменение полей создаёт новую версию схемы.
        </div>
      )}

      {section === 'intake' ? (
      <div className="mb-4 flex gap-1 border-b border-border">
        <TabButton active={tab === 'build'} onClick={() => setTab('build')} icon={PencilRuler}>
          Конструктор
        </TabButton>
        <TabButton active={tab === 'preview'} onClick={() => setTab('preview')} icon={Eye}>
          Превью
        </TabButton>
        <TabButton active={tab === 'submissions'} onClick={() => setTab('submissions')} icon={Inbox}>
          Ответы
        </TabButton>
      </div>
      ) : (
      <div className="mb-4 flex gap-1 border-b border-border">
        {juryEnabled && (
          <TabButton active={runTab === 'evaluation'} onClick={() => setTab('evaluation')} icon={Scale}>
            Оценивание
          </TabButton>
        )}
        {juryEnabled && hasLive && (
          <TabButton active={runTab === 'live'} onClick={() => setTab('live')} icon={Radio}>
            Live
          </TabButton>
        )}
        {juryEnabled && (
          <TabButton active={runTab === 'scores'} onClick={() => setTab('scores')} icon={ClipboardList}>
            Оценки
          </TabButton>
        )}
      </div>
      )}

      {section === 'run' && !juryEnabled ? (
        <EmptyState
          title="Оценивание выключено"
          description="Раздел проведения станет доступен, когда включите жюри."
        />
      ) : tab === 'build' ? (
        <div>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-[18px] font-semibold text-ink">Поля формы</h2>
            {canEdit && (
              <Button size="sm" onClick={openCreate}>
                <Plus className="h-4 w-4" /> Добавить поле
              </Button>
            )}
          </div>
          {fieldsQ.isLoading && <Skeleton className="h-32 w-full" />}
          {fields.length === 0 && !fieldsQ.isLoading ? (
            <EmptyState
              title="Полей пока нет"
              description="Добавьте первое поле — оно сразу появится в превью."
            />
          ) : (
            <FieldsList challengeId={challenge.id} fields={fields} onEdit={openEdit} canEdit={canEdit} />
          )}
        </div>
      ) : tab === 'preview' ? (
        <Card>
          <CardBody className="py-6">
            <ChallengePreview fields={fields} />
          </CardBody>
        </Card>
      ) : runTab === 'evaluation' ? (
        <EvaluationSection challengeId={challenge.id} canEdit={canEdit} />
      ) : runTab === 'live' ? (
        <EvaluationLiveSection challengeId={challenge.id} canEdit={canEdit} />
      ) : runTab === 'scores' ? (
        <EvaluationScoresSection challengeId={challenge.id} canEdit={canEdit} />
      ) : (
        <SubmissionsSection challengeId={challenge.id} />
      )}

      {section === 'run' && canEdit && juryEnabled && (
        <div className="mt-8">
          <ChallengeDangerZone challengeId={challenge.id} />
        </div>
      )}

      <FieldEditorDialog
        challengeId={challenge.id}
        field={editing}
        open={editorOpen}
        onOpenChange={setEditorOpen}
      />

      <EditChallengeDialog challenge={challenge} open={metaOpen} onOpenChange={setMetaOpen} />
    </div>
  )
}

function TabButton({
  active,
  onClick,
  icon: Icon,
  children,
}: {
  active: boolean
  onClick: () => void
  icon: typeof Eye
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        '-mb-px inline-flex items-center gap-1.5 border-b-2 px-3 py-2 text-[14px] font-medium transition ' +
        (active
          ? 'border-brand text-brand'
          : 'border-transparent text-muted hover:text-ink')
      }
    >
      <Icon className="h-4 w-4" /> {children}
    </button>
  )
}
