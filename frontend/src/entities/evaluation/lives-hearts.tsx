import { Heart } from 'lucide-react'

export function eliminatedLabel(question?: number | null): string {
  if (question != null && question > 0) return `выбыл на вопросе ${question}`
  return 'выбыл'
}

export function LivesHearts({ lives, starting }: { lives: number; starting: number }) {
  const filled = Math.max(0, lives)
  const total = Math.max(starting, filled)
  return (
    <span className="inline-flex items-center gap-0.5" aria-label={`Жизней: ${filled}`}>
      {Array.from({ length: total }, (_, i) => (
        <Heart
          key={i}
          className={'h-4 w-4 ' + (i < filled ? 'fill-danger text-danger' : 'text-muted-2')}
        />
      ))}
    </span>
  )
}
