import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  cancelParticipantMerchOrder,
  createAdminMerch,
  deleteAdminMerch,
  deleteParticipantSavingTarget,
  getParticipantMerch,
  issueAdminMerchOrder,
  listAdminMerch,
  listAdminMerchOrders,
  listParticipantMerch,
  listParticipantMerchOrders,
  rejectAdminMerchOrder,
  reserveParticipantMerch,
  setParticipantSavingTarget,
  transitionAdminMerch,
  updateAdminMerch,
} from './api'
import type { MerchOrderStatus, MerchProductInput, ReserveMerchInput } from './types'

const adminProductsKey = (contestId: string) => ['admin', 'merch-products', contestId]
const adminOrdersKey = (contestId: string) => ['admin', 'merch-orders', contestId]
const participantMerchKey = (eventId: string, participantId: string) => [
  'participant',
  'merch',
  eventId,
  participantId,
]
const participantOrdersKey = (eventId: string, participantId: string) => [
  'participant',
  'merch-orders',
  eventId,
  participantId,
]

export function useAdminMerch(contestId?: string, enabled = true) {
  return useQuery({
    queryKey: adminProductsKey(contestId ?? ''),
    queryFn: () => listAdminMerch(contestId!),
    enabled: !!contestId && enabled,
  })
}

export function useAdminMerchOrders(
  contestId: string,
  status: MerchOrderStatus | 'ALL',
  enabled = true,
) {
  return useQuery({
    queryKey: [...adminOrdersKey(contestId), status],
    queryFn: () => listAdminMerchOrders(contestId, status),
    enabled: !!contestId && enabled,
  })
}

export function useCreateMerch(contestId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: MerchProductInput) => createAdminMerch(contestId, input),
    onSuccess: () => client.invalidateQueries({ queryKey: adminProductsKey(contestId) }),
  })
}

export function useUpdateMerch(contestId: string, productId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: MerchProductInput) => updateAdminMerch(contestId, productId, input),
    onSuccess: () => client.invalidateQueries({ queryKey: adminProductsKey(contestId) }),
  })
}

export function useTransitionMerch(contestId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ productId, action }: { productId: string; action: 'activate' | 'hide' }) =>
      transitionAdminMerch(contestId, productId, action),
    onSuccess: () => client.invalidateQueries({ queryKey: adminProductsKey(contestId) }),
  })
}

export function useDeleteMerch(contestId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (productId: string) => deleteAdminMerch(contestId, productId),
    onSuccess: () => client.invalidateQueries({ queryKey: adminProductsKey(contestId) }),
  })
}

export function useModerateMerchOrder(contestId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({
      orderId,
      action,
      reason,
    }: {
      orderId: string
      action: 'issue' | 'reject'
      reason?: string
    }) =>
      action === 'issue'
        ? issueAdminMerchOrder(contestId, orderId)
        : rejectAdminMerchOrder(contestId, orderId, reason ?? ''),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: adminOrdersKey(contestId) })
      void client.invalidateQueries({ queryKey: adminProductsKey(contestId) })
    },
  })
}

export function useParticipantMerch(eventId?: string, participantId?: string) {
  return useQuery({
    queryKey: participantMerchKey(eventId ?? '', participantId ?? ''),
    queryFn: listParticipantMerch,
    enabled: !!eventId && !!participantId,
  })
}

export function useParticipantMerchProduct(
  eventId?: string,
  participantId?: string,
  slug?: string,
) {
  return useQuery({
    queryKey: [...participantMerchKey(eventId ?? '', participantId ?? ''), slug ?? ''],
    queryFn: () => getParticipantMerch(slug!),
    enabled: !!eventId && !!participantId && !!slug,
  })
}

export function useParticipantSavingTarget(eventId: string, participantId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: async (productId: string | null) => {
      if (productId) {
        await setParticipantSavingTarget(productId)
      } else {
        await deleteParticipantSavingTarget()
      }
    },
    onSuccess: () =>
      client.invalidateQueries({ queryKey: participantMerchKey(eventId, participantId) }),
  })
}

export function useReserveParticipantMerch(eventId: string, participantId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: ReserveMerchInput) => reserveParticipantMerch(input),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: participantMerchKey(eventId, participantId) })
      void client.invalidateQueries({ queryKey: participantOrdersKey(eventId, participantId) })
      void client.invalidateQueries({ queryKey: ['participant', 'points', eventId, participantId] })
    },
  })
}

export function useParticipantMerchOrders(eventId?: string, participantId?: string) {
  return useQuery({
    queryKey: participantOrdersKey(eventId ?? '', participantId ?? ''),
    queryFn: listParticipantMerchOrders,
    enabled: !!eventId && !!participantId,
  })
}

export function useCancelParticipantMerchOrder(eventId: string, participantId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: cancelParticipantMerchOrder,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: participantMerchKey(eventId, participantId) })
      void client.invalidateQueries({ queryKey: participantOrdersKey(eventId, participantId) })
      void client.invalidateQueries({ queryKey: ['participant', 'points', eventId, participantId] })
    },
  })
}
