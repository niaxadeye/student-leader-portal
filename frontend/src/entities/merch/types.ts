export type MerchProductStatus = 'DRAFT' | 'ACTIVE' | 'HIDDEN' | 'SOLD_OUT'
export type MerchOrderStatus = 'RESERVED' | 'ISSUED' | 'REJECTED' | 'CANCELLED'

export interface MerchProductImage {
  id: string
  product_id: string
  original_name: string
  mime_type: string
  size_bytes: number
  sort_order: number
  created_at: string
  url: string | null
}

export interface MerchProduct {
  id: string
  event_id: string
  title: string
  slug: string
  description: string
  price_points: number
  discount_price_points: number | null
  stock_quantity: number
  reserved_quantity: number
  available_quantity: number
  effective_price_points: number
  interested_count: number
  is_saving_target: boolean
  status: MerchProductStatus
  images: MerchProductImage[]
  created_at: string
  updated_at: string
}

export interface MerchProductInput {
  title: string
  description: string
  price_points: number
  discount_price_points: number | null
  stock_quantity: number
}

export interface MerchOrderItem {
  id: string
  product_id: string
  product_title: string
  quantity: number
  price_points: number
  total_points: number
}

export interface MerchOrder {
  id: string
  event_id: string
  participant_id: string
  participant_name?: string
  status: MerchOrderStatus
  points_total: number
  rejection_reason: string | null
  created_at: string
  updated_at: string
  issued_at: string | null
  rejected_at: string | null
  cancelled_at: string | null
  issued_by_user_id: string | null
  rejected_by_user_id: string | null
  items: MerchOrderItem[]
}

export interface MerchOrderResult {
  order: MerchOrder
  replayed: boolean
}

export interface ReserveMerchInput {
  items: Array<{ product_id: string; quantity: number }>
  idempotency_key: string
}
