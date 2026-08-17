import { participantApiRequest } from '@/entities/event-participant/api'
import { apiPostForm, apiRequest } from '@/shared/api/client'
import type {
  MerchOrder,
  MerchOrderResult,
  MerchOrderStatus,
  MerchProduct,
  MerchProductImage,
  MerchProductInput,
  ReserveMerchInput,
} from './types'

const adminRoot = (contestId: string) => `/admin/contests/${encodeURIComponent(contestId)}/merch`
const productsPath = (contestId: string) => `${adminRoot(contestId)}/products`
const productPath = (contestId: string, productId: string) =>
  `${productsPath(contestId)}/${encodeURIComponent(productId)}`
const ordersPath = (contestId: string) => `${adminRoot(contestId)}/orders`

export const listAdminMerch = (contestId: string): Promise<MerchProduct[]> =>
  apiRequest(productsPath(contestId))

export const createAdminMerch = (
  contestId: string,
  input: MerchProductInput,
): Promise<MerchProduct> => apiRequest(productsPath(contestId), { method: 'POST', body: input })

export const updateAdminMerch = (
  contestId: string,
  productId: string,
  input: MerchProductInput,
): Promise<MerchProduct> =>
  apiRequest(productPath(contestId, productId), { method: 'PATCH', body: input })

export const transitionAdminMerch = (
  contestId: string,
  productId: string,
  action: 'activate' | 'hide',
): Promise<MerchProduct> =>
  apiRequest(`${productPath(contestId, productId)}/${action}`, { method: 'POST' })

export const deleteAdminMerch = (contestId: string, productId: string): Promise<void> =>
  apiRequest(productPath(contestId, productId), { method: 'DELETE' })

export function uploadAdminMerchImage(
  contestId: string,
  productId: string,
  file: File,
): Promise<MerchProductImage> {
  const form = new FormData()
  form.append('image', file)
  return apiPostForm(`${productPath(contestId, productId)}/images`, form)
}

export const deleteAdminMerchImage = (
  contestId: string,
  productId: string,
  imageId: string,
): Promise<void> =>
  apiRequest(`${productPath(contestId, productId)}/images/${encodeURIComponent(imageId)}`, {
    method: 'DELETE',
  })

export const listAdminMerchOrders = (
  contestId: string,
  status: MerchOrderStatus | 'ALL',
): Promise<MerchOrder[]> =>
  apiRequest(`${ordersPath(contestId)}${status === 'ALL' ? '' : `?status=${status}`}`)

export const issueAdminMerchOrder = (
  contestId: string,
  orderId: string,
): Promise<MerchOrderResult> =>
  apiRequest(`${ordersPath(contestId)}/${encodeURIComponent(orderId)}/issue`, { method: 'POST' })

export const rejectAdminMerchOrder = (
  contestId: string,
  orderId: string,
  reason: string,
): Promise<MerchOrderResult> =>
  apiRequest(`${ordersPath(contestId)}/${encodeURIComponent(orderId)}/reject`, {
    method: 'POST',
    body: { reason },
  })

export const listParticipantMerch = (): Promise<MerchProduct[]> =>
  participantApiRequest('/participant/merch')

export const getParticipantMerch = (slug: string): Promise<MerchProduct> =>
  participantApiRequest(`/participant/merch/${encodeURIComponent(slug)}`)

export const setParticipantSavingTarget = (productId: string): Promise<MerchProduct> =>
  participantApiRequest('/participant/merch-saving-target', {
    method: 'PUT',
    body: { product_id: productId },
  })

export const deleteParticipantSavingTarget = (): Promise<void> =>
  participantApiRequest('/participant/merch-saving-target', { method: 'DELETE' })

export const reserveParticipantMerch = (input: ReserveMerchInput): Promise<MerchOrderResult> =>
  participantApiRequest('/participant/orders', {
    method: 'POST',
    headers: { 'Idempotency-Key': input.idempotency_key },
    body: input,
  })

export const listParticipantMerchOrders = (): Promise<MerchOrder[]> =>
  participantApiRequest('/participant/orders')

export const getParticipantMerchOrder = (orderId: string): Promise<MerchOrder> =>
  participantApiRequest(`/participant/orders/${encodeURIComponent(orderId)}`)

export const cancelParticipantMerchOrder = (orderId: string): Promise<MerchOrderResult> =>
  participantApiRequest(`/participant/orders/${encodeURIComponent(orderId)}/cancel`, {
    method: 'POST',
  })
