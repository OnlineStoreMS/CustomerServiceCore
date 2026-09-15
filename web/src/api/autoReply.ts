import client, { unwrap } from './client'

export interface AutoReplyRuleItem {
  id: number
  shopId: number
  shopName?: string
  name: string
  enabled: boolean
  matchMode: 'exact' | 'contains' | string
  keywords: string
  replyText: string
  priority: number
  cooldownSec: number
  createdAt: string
  updatedAt: string
}

export interface AutoReplyRuleInput {
  shopId?: number
  name: string
  enabled?: boolean
  matchMode?: string
  keywords: string
  replyText: string
  priority?: number
  cooldownSec?: number
}

export async function listAutoReplyRules() {
  return unwrap<AutoReplyRuleItem[]>(await client.get('/auto-reply-rules'))
}

export async function createAutoReplyRule(data: AutoReplyRuleInput) {
  return unwrap<AutoReplyRuleItem>(await client.post('/auto-reply-rules', data))
}

export async function updateAutoReplyRule(id: number, data: Partial<AutoReplyRuleInput> & { enabled?: boolean }) {
  return unwrap<AutoReplyRuleItem>(await client.patch(`/auto-reply-rules/${id}`, data))
}

export async function deleteAutoReplyRule(id: number) {
  return unwrap<{ ok: boolean }>(await client.delete(`/auto-reply-rules/${id}`))
}

export async function seedAutoReplyPresets() {
  return unwrap<AutoReplyRuleItem[]>(await client.post('/auto-reply-rules/presets'))
}
