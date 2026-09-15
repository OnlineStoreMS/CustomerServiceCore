import client, { unwrap, type PageData } from './client'

export interface ConversationItem {
  id: number
  shopId: number
  shopName?: string
  platform: string
  platformShopId: string
  platformShopName?: string
  platformBuyerId: string
  buyerName: string
  platformConversationId?: string
  lastMessageAt?: string
  lastMessagePreview?: string
  unreadHint: number
  mergedIds?: number[]
  createdAt: string
  updatedAt: string
}

export interface MessageItem {
  id: number
  conversationId: number
  platformMessageId: string
  platformBuyerId: string
  direction: string
  content: string
  sentAt: string
  createdAt: string
}

export async function listConversations(params: { shopId?: number; page?: number; pageSize?: number }) {
  return unwrap<PageData<ConversationItem>>(await client.get('/conversations', { params }))
}

export async function listMessages(conversationId: number, params?: { page?: number; pageSize?: number }) {
  return unwrap<PageData<MessageItem>>(await client.get(`/conversations/${conversationId}/messages`, { params }))
}
