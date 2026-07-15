// Конкурс глазами конкурсанта (кабинет).
export interface MyContest {
  id: string
  name: string
  slug: string
  description?: string | null
  status: 'ACTIVE' | 'FINISHED'
  start_at?: string | null
  end_at?: string | null
  image_url?: string | null
}
