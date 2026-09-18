<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { Close, CopyDocument, Download, ZoomIn } from '@element-plus/icons-vue'
import { listShops, type ShopItem } from '../api/shops'
import {
  listConversations,
  listMessages,
  replyConversation,
  clearAllConversations,
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
const draft = ref('')
const sending = ref(false)
const msgListEl = ref<HTMLElement | null>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null
let polling = false
let seenPrimed = false
const seenStamp = new Map<string, string>()
const defaultTitle = typeof document !== 'undefined' ? document.title : '客服中心'
let titleTimer: ReturnType<typeof setTimeout> | null = null

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

async function load(opts?: { silent?: boolean }) {
  const silent = !!opts?.silent
  if (!silent) loading.value = true
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
    notifyNewInbound(merged)
    if (active.value) {
      const next = findSameConversation(merged, active.value)
      if (next) active.value = next
    }
  } catch (e: any) {
    if (!silent) ElMessage.error(e?.message || '加载失败')
  } finally {
    if (!silent) loading.value = false
  }
}

async function loadMessages(opts?: { silent?: boolean }) {
  if (!active.value) {
    messages.value = []
    return
  }
  const silent = !!opts?.silent
  if (!silent) msgLoading.value = true
  const prevCount = messages.value.length
  const prevLast = messages.value[messages.value.length - 1]
  try {
    const ids = [...new Set([active.value.id, ...(active.value.mergedIds || [])])].filter(Boolean)
    const chunks = await Promise.all(
      ids.map((id) => listMessages(id, { page: 1, pageSize: 200 }))
    )
    const merged = dedupeMessages(chunks.flatMap((c) => c.list || []))
    msgTotal.value = merged.length
    const start = (msgPage.value - 1) * msgPageSize.value
    messages.value = merged.slice(start, start + msgPageSize.value)
    const last = messages.value[messages.value.length - 1]
    const grew = messages.value.length > prevCount || (last && last.id !== prevLast?.id)
    if (grew && msgPage.value === 1) scrollMessagesToBottom()
  } catch (e: any) {
    if (!silent) ElMessage.error(e?.message || '加载消息失败')
  } finally {
    if (!silent) msgLoading.value = false
  }
}

function findSameConversation(rows: ConversationItem[], current: ConversationItem): ConversationItem | undefined {
  const ids = new Set<number>([current.id, ...(current.mergedIds || [])].filter(Boolean))
  const key = conversationGroupKey(current)
  return (
    rows.find((row) => ids.has(row.id) || (row.mergedIds || []).some((id) => ids.has(id)))
    || rows.find((row) => conversationGroupKey(row) === key)
  )
}

function notifyNewInbound(rows: ConversationItem[]) {
  for (const row of rows) {
    const key = conversationGroupKey(row)
    const stamp = row.lastMessageAt || ''
    const prev = seenStamp.get(key)
    if (seenPrimed && stamp && stamp !== prev) {
      const looking = !!active.value && conversationGroupKey(active.value) === key
      if (!looking) {
        ElNotification({
          title: '新消息',
          message: `${normalizeBuyerName(row.buyerName || row.platformBuyerId) || '买家'}：${row.lastMessagePreview || ''}`,
          type: 'warning',
          duration: 8000,
        })
        flashTitle()
      }
    }
    if (stamp) seenStamp.set(key, stamp)
  }
  seenPrimed = true
}

function flashTitle() {
  document.title = '【新消息】客服中心'
  if (titleTimer) clearTimeout(titleTimer)
  titleTimer = setTimeout(() => {
    restoreTitle()
  }, 15000)
}

function restoreTitle() {
  document.title = defaultTitle
}

function scrollMessagesToBottom() {
  void nextTick(() => {
    const el = msgListEl.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

function selectConv(row: ConversationItem) {
  active.value = row
  msgPage.value = 1
  draft.value = ''
  void loadMessages().then(() => scrollMessagesToBottom())
}

async function clearAll() {
  try {
    await ElMessageBox.confirm('将删除本租户下全部会话、消息和待发送回复，且无法恢复。', '清空全部会话', {
      type: 'warning',
      confirmButtonText: '清空',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  loading.value = true
  try {
    await clearAllConversations()
    active.value = null
    messages.value = []
    list.value = []
    total.value = 0
    ElMessage.success('会话已清空')
    await load({ silent: true })
  } catch (e: any) {
    ElMessage.error(e?.message || '清空失败')
  } finally {
    loading.value = false
  }
}

async function sendReply() {
  const text = draft.value.trim()
  if (!active.value || !text || sending.value) return
  sending.value = true
  try {
    await replyConversation(active.value.id, text)
    draft.value = ''
    ElMessage.success('已排队，将由本机飞鸽发出')
    await loadMessages()
    await load({ silent: true })
    scrollMessagesToBottom()
  } catch (e: any) {
    ElMessage.error(e?.message || '发送失败')
  } finally {
    sending.value = false
  }
}

async function pollTick() {
  if (polling) return
  polling = true
  try {
    const prevStamp = active.value?.lastMessageAt || ''
    await load({ silent: true })
    if (!active.value) return
    const stamp = active.value.lastMessageAt || ''
    if (stamp !== prevStamp || messages.value.length === 0) {
      await loadMessages({ silent: true })
    }
  } finally {
    polling = false
  }
}

function startPoll() {
  stopPoll()
  void pollTick()
  pollTimer = setInterval(() => {
    void pollTick()
  }, 2000)
}

function stopPoll() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
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
  const embedded = raw.match(/data:image\/[a-zA-Z0-9.+-]+;base64,[A-Za-z0-9+/=\s]+/)
  if (embedded?.[0]) return embedded[0].replace(/\s+/g, '').replace(/\)+$/, '')
  if (raw.startsWith('data:image/')) return raw.replace(/\s+/g, '')
  if (/^https?:\/\//i.test(raw) && /(image|\.png|\.jpe?g|\.gif|\.webp)/i.test(raw)) return raw
  return ''
}

const previewSrc = ref('')
const previewBusy = ref('')
const renderedMessages = computed(() =>
  messages.value.map((m) => ({ message: m, imageSrc: messageImageSrc(m.content) })),
)

function openPreview(src: string) {
  if (!src) return
  previewSrc.value = src
}

function closePreview() {
  previewSrc.value = ''
  previewBusy.value = ''
}

function onGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && previewSrc.value) {
    closePreview()
  }
}

function mimeExt(mime: string): string {
  const t = (mime || '').toLowerCase()
  if (t.includes('png')) return 'png'
  if (t.includes('webp')) return 'webp'
  if (t.includes('gif')) return 'gif'
  return 'jpg'
}

async function srcToBlob(src: string): Promise<Blob> {
  const res = await fetch(src)
  if (!res.ok) throw new Error('读取图片失败')
  return res.blob()
}

async function blobToPng(blob: Blob): Promise<Blob> {
  if ((blob.type || '').toLowerCase() === 'image/png') return blob
  const bmp = await createImageBitmap(blob)
  const canvas = document.createElement('canvas')
  canvas.width = bmp.width
  canvas.height = bmp.height
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('无法复制图片')
  ctx.drawImage(bmp, 0, 0)
  bmp.close()
  return await new Promise((resolve, reject) => {
    canvas.toBlob((out) => (out ? resolve(out) : reject(new Error('无法复制图片'))), 'image/png')
  })
}

async function copyImage(src: string) {
  if (!src || previewBusy.value) return
  previewBusy.value = 'copy'
  try {
    const blob = await srcToBlob(src)
    const png = await blobToPng(blob)
    if (!navigator.clipboard?.write || typeof ClipboardItem === 'undefined') {
      throw new Error('当前浏览器不支持复制图片')
    }
    await navigator.clipboard.write([new ClipboardItem({ 'image/png': png })])
    ElMessage.success('图片已复制，可直接粘贴')
  } catch (e: any) {
    ElMessage.error(e?.message || '复制失败')
  } finally {
    previewBusy.value = ''
  }
}

async function saveImage(src: string) {
  if (!src || previewBusy.value) return
  previewBusy.value = 'save'
  try {
    const blob = await srcToBlob(src)
    const ext = mimeExt(blob.type)
    const stamp = new Date().toISOString().replace(/[-:T]/g, '').slice(0, 14)
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `买家图片-${stamp}.${ext}`
    a.click()
    URL.revokeObjectURL(a.href)
    ElMessage.success('已开始保存')
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
  } finally {
    previewBusy.value = ''
  }
}

onMounted(async () => {
  window.addEventListener('keydown', onGlobalKeydown)
  window.addEventListener('focus', restoreTitle)
  await loadShops()
  await load()
  startPoll()
})

onUnmounted(() => {
  stopPoll()
  window.removeEventListener('keydown', onGlobalKeydown)
  window.removeEventListener('focus', restoreTitle)
  restoreTitle()
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
        <el-button type="danger" plain @click="clearAll">清空全部</el-button>
        <span class="sync-hint">自动同步中 · 约 2 秒</span>
      </div>
    </div>

    <div class="split">
      <div class="left">
        <el-table
          :data="list"
          v-loading="loading"
          stripe
          highlight-current-row
          row-key="id"
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
          <div class="msg-list" ref="msgListEl" v-loading="msgLoading">
            <div
              v-for="row in renderedMessages"
              :key="row.message.id"
              class="msg"
              :class="row.message.direction === 'out' ? 'out' : 'in'"
            >
              <div class="bubble">
                <div v-if="row.imageSrc" class="img-wrap">
                  <img
                    :src="row.imageSrc"
                    class="msg-img"
                    alt="图片"
                    title="点击预览"
                    @click="openPreview(row.imageSrc)"
                  />
                  <div class="img-actions">
                    <el-button circle size="small" title="预览" @click.stop="openPreview(row.imageSrc)">
                      <el-icon><ZoomIn /></el-icon>
                    </el-button>
                    <el-button
                      circle
                      size="small"
                      title="复制"
                      :loading="previewBusy === 'copy'"
                      @click.stop="copyImage(row.imageSrc)"
                    >
                      <el-icon><CopyDocument /></el-icon>
                    </el-button>
                    <el-button
                      circle
                      size="small"
                      title="保存"
                      :loading="previewBusy === 'save'"
                      @click.stop="saveImage(row.imageSrc)"
                    >
                      <el-icon><Download /></el-icon>
                    </el-button>
                  </div>
                </div>
                <template v-else>{{ row.message.content || '(空)' }}</template>
              </div>
              <div class="meta">{{ row.message.direction }} · {{ row.message.sentAt }}</div>
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
          <div class="composer">
            <el-input
              v-model="draft"
              type="textarea"
              :rows="2"
              maxlength="800"
              show-word-limit
              placeholder="回复买家，由本机 WindowsAgent 在飞鸽发出"
              @keydown.enter.exact.prevent="sendReply"
            />
            <el-button type="primary" :loading="sending" :disabled="!draft.trim()" @click="sendReply">
              发送
            </el-button>
          </div>
        </template>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="previewSrc" class="img-preview" @click.self="closePreview">
        <div class="img-preview-bar">
          <el-button :loading="previewBusy === 'copy'" @click="copyImage(previewSrc)">
            <el-icon><CopyDocument /></el-icon>
            复制
          </el-button>
          <el-button type="primary" :loading="previewBusy === 'save'" @click="saveImage(previewSrc)">
            <el-icon><Download /></el-icon>
            保存
          </el-button>
          <el-button :icon="Close" circle @click="closePreview" />
        </div>
        <img :src="previewSrc" alt="预览" class="img-preview-full" />
      </div>
    </Teleport>
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
  align-items: center;
}
.sync-hint {
  color: #909399;
  font-size: 12px;
  white-space: nowrap;
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
.img-wrap {
  position: relative;
  display: inline-block;
  max-width: 240px;
}
.msg-img {
  display: block;
  max-width: 240px;
  max-height: 320px;
  border-radius: 6px;
  cursor: zoom-in;
}
.img-actions {
  position: absolute;
  right: 6px;
  bottom: 6px;
  display: flex;
  gap: 4px;
}
@media (hover: hover) {
  .img-actions {
    opacity: 0;
    transition: opacity 0.15s ease;
  }
  .img-wrap:hover .img-actions {
    opacity: 1;
  }
}
.bubble:has(.msg-img) {
  padding: 4px;
  background: transparent;
}
.img-preview {
  position: fixed;
  inset: 0;
  z-index: 4100;
  background: rgba(0, 0, 0, 0.78);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 16px;
}
.img-preview-bar {
  display: flex;
  align-items: center;
  gap: 8px;
}
.img-preview-full {
  max-width: 92vw;
  max-height: calc(100vh - 96px);
  object-fit: contain;
  border-radius: 8px;
  background: #111;
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
.composer {
  display: flex;
  gap: 8px;
  align-items: flex-end;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #ebeef5;
}
.composer .el-textarea {
  flex: 1;
}
@media (max-width: 900px) {
  .split {
    grid-template-columns: 1fr;
    height: auto;
  }
}
</style>
