import { useRef } from 'react'
import { UploadCloud, FileIcon, X, Loader2, CheckCircle2, AlertCircle } from 'lucide-react'
import { cn } from '@/shared/lib/cn'
import { formatBytes } from '@/shared/lib/format'

// Статусы файла — подмножество SITE.md §8 (File).
export type UploadStatus = 'UPLOADING' | 'READY' | 'REJECTED'
export interface UploadedFile {
  id: string
  name: string
  size: number
  status: UploadStatus
  progress: number
}

interface FileUploadProps {
  files: UploadedFile[]
  hint?: string
  accept?: string
  onAdd: (files: FileList) => void
  onRemove: (id: string) => void
}

const statusIcon = {
  UPLOADING: <Loader2 className="h-4 w-4 animate-spin text-brand" />,
  READY: <CheckCircle2 className="h-4 w-4 text-success" />,
  REJECTED: <AlertCircle className="h-4 w-4 text-danger" />,
}

export function FileUpload({ files, hint, accept, onAdd, onRemove }: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  return (
    <div className="flex flex-col gap-3">
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        className="flex flex-col items-center gap-2 rounded-card border border-dashed border-border bg-surface-2 px-6 py-8 text-center transition-colors hover:border-brand hover:bg-brand-subtle"
      >
        <UploadCloud className="h-7 w-7 text-brand" />
        <span className="text-[15px] font-medium text-ink">Нажмите, чтобы загрузить файлы</span>
        {hint && <span className="text-[13px] text-muted">{hint}</span>}
      </button>
      <input
        ref={inputRef}
        type="file"
        multiple
        accept={accept}
        className="sr-only"
        onChange={(e) => e.target.files && onAdd(e.target.files)}
      />
      {files.length > 0 && (
        <ul className="flex flex-col gap-2">
          {files.map((f) => (
            <li
              key={f.id}
              className="flex items-center gap-3 rounded-[10px] border border-border bg-surface px-3 py-2.5"
            >
              <FileIcon className="h-5 w-5 shrink-0 text-muted-2" />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <p className="truncate text-[14px] font-medium text-ink">{f.name}</p>
                  {statusIcon[f.status]}
                </div>
                {f.status === 'UPLOADING' ? (
                  <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-muted/15">
                    <div
                      className="h-full rounded-full bg-brand transition-all"
                      style={{ width: `${f.progress}%` }}
                    />
                  </div>
                ) : (
                  <p
                    className={cn(
                      'text-[12px]',
                      f.status === 'REJECTED' ? 'text-danger' : 'text-muted',
                    )}
                  >
                    {f.status === 'REJECTED' ? 'Файл отклонён' : formatBytes(f.size)}
                  </p>
                )}
              </div>
              <button
                type="button"
                aria-label="Удалить файл"
                onClick={() => onRemove(f.id)}
                className="text-muted-2 hover:text-danger"
              >
                <X className="h-4 w-4" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
