import client, { unwrap } from './client'

export interface ShopItem {
  id: number
  name: string
  platform: string
  platformLabel: string
  platformShopId?: string
  platformShopName?: string
  bindCode: string
  pluginStatus: string
  monitorEnabled: boolean
  lastSeenAt?: string
  remark?: string
  createdAt: string
  updatedAt: string
}

export interface ShopCreateInput {
  name: string
  platform?: string
  platformShopId?: string
  platformShopName?: string
  remark?: string
}

export async function listShops() {
  return unwrap<ShopItem[]>(await client.get('/shops'))
}

export async function createShop(data: ShopCreateInput) {
  return unwrap<ShopItem>(await client.post('/shops', data))
}

export async function updateShop(id: number, data: { name?: string; remark?: string; monitorEnabled?: boolean }) {
  return unwrap<ShopItem>(await client.patch(`/shops/${id}`, data))
}

export async function rotateBindCode(id: number) {
  return unwrap<ShopItem>(await client.post(`/shops/${id}/rotate-bind-code`))
}

export async function resetPlugin(id: number) {
  return unwrap<ShopItem>(await client.post(`/shops/${id}/reset-plugin`))
}
