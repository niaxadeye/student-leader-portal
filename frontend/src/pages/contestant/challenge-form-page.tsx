import { useParams } from 'react-router-dom'
import { Save, Send } from 'lucide-react'
import { useChallenge } from '@/entities/challenge/queries'
import { useSubmission, useSubmit } from '@/entities/submission/queries'
import { useSubmissionForm } from '@/features/submit-form/use-submission-form'
import { FieldRenderer } from '@/features/submit-form/field-renderer'
import { ChallengeFormHeader } from './challenge-form-header'
import { Card, CardBody } from '@/shared/ui/card'
import { Button } from '@/shared/ui/button'
import { Skeleton } from '@/shared/ui/states'
import { Dialog, DialogContent, DialogTrigger, DialogClose } from '@/shared/ui/dialog'
import { useToast } from '@/shared/ui/toast'
import { ApiRequestError } from '@/shared/api/client'

export function ChallengeFormPage() {
  const { challengeId } = useParams()
  const { data: challenge, isLoading } = useChallenge(challengeId)
  const { data: sub, isLoading: subLoading } = useSubmission(challengeId)
  const toast = useToast()
  const form = useSubmissionForm(challenge, sub)
  const submitMut = useSubmit(challengeId ?? '')

  if (isLoading || subLoading || !challenge || !sub) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-96 w-full" />
      </div>
    )
  }

  const readOnly = sub.locked
  const revision = sub.current_revision_number

  async function saveDraft() {
    try {
      await form.saveNow()
      toast({ tone: 'success', title: 'Черновик сохранён' })
    } catch {
      toast({ tone: 'error', title: 'Не удалось сохранить' })
    }
  }

  function submit() {
    if (!form.validate()) {
      toast({ tone: 'error', title: 'Проверьте форму', description: 'Не все обязательные поля заполнены' })
      return
    }
    submitMut.mutate(form.answers, {
      onSuccess: (data) => {
        toast({
          tone: 'success',
          title: data.current_revision_number === 1 ? 'Форма отправлена' : 'Форма обновлена',
          description: `Создана ревизия №${data.current_revision_number}`,
        })
      },
      onError: (e) => {
        const msg = e instanceof ApiRequestError ? e.message : 'Не удалось отправить'
        toast({ tone: 'error', title: 'Ошибка отправки', description: msg })
      },
    })
  }

  return (
    <div className="flex flex-col gap-6 pb-24">
      <ChallengeFormHeader
        challenge={challenge}
        status={sub.status}
        saveState={form.saveState}
        revision={revision}
      />

      <Card>
        <CardBody className="grid grid-cols-1 gap-5 md:grid-cols-2">
          {challenge.fields.map((field) => (
            <FieldRenderer
              key={field.id}
              field={field}
              value={form.answers[field.key]}
              files={form.files[field.key] ?? []}
              error={form.errors[field.key]}
              disabled={readOnly}
              onChange={(v) => form.setAnswer(field.key, v)}
              onAddFiles={(list) => form.addFiles(field.key, list)}
              onRemoveFile={(id) => form.removeFile(field.key, id)}
            />
          ))}
        </CardBody>
      </Card>

      {!readOnly && (
        <div className="fixed inset-x-0 bottom-0 z-20 border-t border-border bg-surface/90 backdrop-blur">
          <div className="mx-auto flex max-w-5xl items-center justify-between gap-3 px-4 py-3">
            <p className="hidden text-[13px] text-muted sm:block">
              {form.hasUploading
                ? 'Дождитесь окончания загрузки файлов'
                : 'Черновик не является отправкой'}
            </p>
            <div className="flex w-full gap-3 sm:w-auto">
              <Button variant="outline" onClick={saveDraft} className="flex-1 sm:flex-none">
                <Save className="h-4 w-4" /> Сохранить черновик
              </Button>
              <Dialog>
                <DialogTrigger asChild>
                  <Button disabled={form.hasUploading} className="flex-1 sm:flex-none">
                    <Send className="h-4 w-4" />
                    {revision > 0 ? 'Обновить отправку' : 'Отправить'}
                  </Button>
                </DialogTrigger>
                <DialogContent
                  title={revision > 0 ? 'Обновить отправку?' : 'Отправить форму?'}
                  description="После отправки дирекция получит уведомление и увидит вашу работу. Будет создана новая ревизия."
                >
                  <div className="flex justify-end gap-3">
                    <DialogClose asChild>
                      <Button variant="secondary">Отмена</Button>
                    </DialogClose>
                    <DialogClose asChild>
                      <Button loading={submitMut.isPending} onClick={submit}>
                        Подтвердить
                      </Button>
                    </DialogClose>
                  </div>
                </DialogContent>
              </Dialog>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
