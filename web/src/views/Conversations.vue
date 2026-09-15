<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { listShops, type ShopItem } from '../api/shops'
import {
  listConversations,
  listMessages,
  type ConversationItem,
  type MessageItem,
} from '../api/conversations'

const shops = ref<ShopItem[]>([])
const shopId = ref<number | undefined>()
const loading = ref(false)
const list = ref<ConversationItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)

const active = ref<ConversationItem | null>(null)
const msgLoading = ref(false)
const messages = ref<MessageItem[]>([])
const msgTotal = ref(0)
const msgPage = ref(1)
const msgPageSize = ref(100)

function normalizeBuyerName(raw: string | undefined): string {
  let s = (raw || '').trim().replace(/^(name|uid):/i, '')
  s = s.replace(/重复来访/g, ' ').replace(/进线/g, ' ')
  s = s.replace(/\b\d{1,2}:\d{2}(:\d{2})?\b/g, ' ')
  s = s.replace(/\b\d{1,2}\s*s\b/gi, ' ').replace(/\d{1,2}\s*秒/g, ' ')
  s = s.replace(/\s+/g, ' ').trim()
  return (s.split(/\s+/)[0] || '').trim()
}

function conversationGroupKey(row: ConversationItem): string {
  const name = normalizeBuyerName(row.buyerName || row.platformBuyerId)
  return [row.shopId || 0, row.platform || '', row.platformShopId || '', name].join('|')
}

function mergeConversationRows(rows: ConversationItem[]): ConversationItem[] {
  const map = new Map<string, ConversationItem>()
  for (const row of rows) {
    const name = normalizeBuyerName(row.buyerName || row.platformBuyerId)
    const key = name ? conversationGroupKey(row) : `id:${row.id}`
    const prev = map.get(key)
    const ids = new Set<number>([...(prev?.mergedIds || []), ...(row.mergedIds || []), row.id, prev?.id].filter(Boolean) as number[])
    if (!prev) {
      map.set(key, { ...row, buyerName: name || row.buyerName, mergedIds: [...ids] })
      continue
    }
    const newer = (row.lastMessageAt || '') > (prev.lastMessageAt || '')
    map.set(key, {
      ...(newer ? row : prev),
      buyerName: name || (newer ? row.buyerName : prev.buyerName),
      lastMessageAt: newer ? row.lastMessageAt : prev.lastMessageAt,
      lastMessagePreview: newer ? row.lastMessagePreview : prev.lastMessagePreview,
      mergedIds: [...ids],
    })
  }
  return [...map.values()].sort((a, b) => (b.lastMessageAt || '').localeCompare(a.lastMessageAt || ''))
}

function dedupeMessages(rows: MessageItem[]): MessageItem[] {
  const seen = new Set<string>()
  const out: MessageItem[] = []
  for (const row of rows) {
    const key = `${row.direction}\n${(row.content || '').trim()}`
    if (seen.has(key)) continue
    seen.add(key)
    out.push(row)
  }
  return out.sort((a, b) => (a.sentAt || '').localeCompare(b.sentAt || '') || a.id - b.id)
}

async function loadShops() {
  try {
    shops.value = (await listShops()) || []
  } catch {
    // ignore
  }
}

async function load() {
  loading.value = true
  try {
    const res = await listConversations({
      shopId: shopId.value,
      page: 1,
      pageSize: 200,
    })
    const merged = mergeConversationRows(res.list || [])
    total.value = merged.length
    const start = (page.value - 1) * pageSize.value
    list.value = merged.slice(start, start + pageSize.value)
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadMessages() {
  if (!active.value) {
    messages.value = []
    return
  }
  msgLoading.value = true
  try {
    const ids = [...new Set([active.value.id, ...(active.value.mergedIds || [])])].filter(Boolean)
    const chunks = await Promise.all(
      ids.map((id) => listMessages(id, { page: 1, pageSize: 200 }))
    )
    const merged = dedupeMessages(chunks.flatMap((c) => c.list || []))
    msgTotal.value = merged.length
    const start = (msgPage.value - 1) * msgPageSize.value
    messages.value = merged.slice(start, start + msgPageSize.value)
  } catch (e: any) {
    ElMessage.error(e?.message || '加载消息失败')
  } finally {
    msgLoading.value = false
  }
}

function selectConv(row: ConversationItem) {
  active.value = row
  msgPage.value = 1
  void loadMessages()
}

watch(shopId, () => {
  page.value = 1
  active.value = null
  void load()
})

function messageImageSrc(content: string | undefined): string {
  const raw = (content || '').trim()
  if (!raw) return ''
  const mdData = raw.match(/!\[[^\]]*\]\(\s*(data:image\/[a-zA-Z0-9.+-]+;base64,[A-Za-z0-9+/=\s]+)\)/s)
  if (mdData?.[1]) return mdData[1].replace(/\s+/g, '')
  const mdHttp = raw.match(/!\[[^\]]*\]\(\s*(https?:\/\/[^)\s]+)\)/)
  if (mdHttp?.[1]) return mdHttp[1]
  if (raw.startsWith('data:image/')) return raw.replace(/\s+/g, '')
  if (/^https?:\/\//i.test(raw) && /(image|\.png|\.jpe?g|\.gif|\.webp)/i.test(raw)) return raw
  return ''
}

onMounted(async () => {
  await loadShops()
  await load()
})
</script>

<template>
  <div class="page">
    <div class="head">
      <h2>会话</h2>
      <div class="filters">
        <el-select v-model="shopId" clearable placeholder="全部店铺" style="width: 200px">
          <el-option v-for="s in shops" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="split">
      <div class="left">
        <el-table
          :data="list"
          v-loading="loading"
          stripe
          highlight-current-row
          height="100%"
          @row-click="selectConv"
        >
          <el-table-column label="买家" min-width="100">
            <template #default="{ row }">
              {{ normalizeBuyerName(row.buyerName || row.platformBuyerId) || row.buyerName }}
            </template>
          </el-table-column>
          <el-table-column label="店铺" min-width="120" show-overflow-tooltip>
            <template #default="{ row }">
              {{ row.shopName || row.platformShopName || row.platformShopId || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="lastMessagePreview" label="最近消息" min-width="160" show-overflow-tooltip />
          <el-table-column prop="lastMessageAt" label="时间" width="160" />
        </el-table>
        <div class="pager">
          <el-pagination
            v-model:current-page="page"
            v-model:page-size="pageSize"
            :total="total"
            layout="total, prev, pager, next"
            @current-change="load"
          />
        </div>
      </div>

      <div class="right">
        <div v-if="!active" class="empty">选择左侧会话查看消息</div>
        <template v-else>
          <div class="msg-head">
            <strong>{{ normalizeBuyerName(active.buyerName || active.platformBuyerId) || active.buyerName }}</strong>
            <span class="muted">{{ active.shopName || active.platformShopName || active.platformShopId }}</span>
          </div>
          <div class="msg-list" v-loading="msgLoading">
            <div
              v-for="m in messages"
              :key="m.id"
              class="msg"
              :class="m.direction === 'out' ? 'out' : 'in'"
            >
              <div class="bubble">
                <img
                  v-if="messageImageSrc(m.content)"
                  :src="messageImageSrc(m.content)"
                  class="msg-img"
                  alt="图片"
                />
                <template v-else>{{ m.content || '(空)' }}</template>
              </div>
              <div class="meta">{{ m.direction }} · {{ m.sentAt }}</div>
            </div>
          </div>
          <div class="pager">
            <el-pagination
              v-model:current-page="msgPage"
              v-model:page-size="msgPageSize"
              :total="msgTotal"
              layout="total, prev, pager, next"
              @current-change="loadMessages"
            />
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  gap: 12px;
}
.filters {
  display: flex;
  gap: 8px;
}
.split {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  height: calc(100vh - 160px);
  min-height: 420px;
}
.left,
.right {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.pager {
  margin-top: 8px;
  display: flex;
  justify-content: flex-end;
}
.empty {
  color: #909399;
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
}
.msg-head {
  display: flex;
  gap: 12px;
  align-items: baseline;
  margin-bottom: 8px;
}
.muted {
  color: #909399;
  font-size: 12px;
}
.msg-list {
  flex: 1;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 4px 0;
}
.msg {
  max-width: 80%;
}
.msg.in {
  align-self: flex-start;
}
.msg.out {
  align-self: flex-end;
  text-align: right;
}
.bubble {
  display: inline-block;
  padding: 8px 12px;
  border-radius: 8px;
  background: #f0f2f5;
  white-space: pre-wrap;
  word-break: break-word;
  text-align: left;
}
.msg-img {
  display: block;
  max-width: 240px;
  max-height: 320px;
  border-radius: 6px;
}
.bubble:has(.msg-img) {
  padding: 4px;
  background: transparent;
}
.msg.out .bubble {
  background: #ecf5ff;
  color: #303133;
}
.meta {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
@media (max-width: 900px) {
  .split {
    grid-template-columns: 1fr;
    height: auto;
  }
}
</style>
