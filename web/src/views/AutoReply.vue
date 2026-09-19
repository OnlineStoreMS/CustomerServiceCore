<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listShops, type ShopItem } from '../api/shops'
import {
  createAutoReplyRule,
  deleteAutoReplyRule,
  getLlmSetting,
  listAutoReplyRules,
  saveLlmSetting,
  seedAutoReplyPresets,
  updateAutoReplyRule,
  type AutoReplyRuleItem,
  type LlmSetting,
} from '../api/autoReply'

const loading = ref(false)
const list = ref<AutoReplyRuleItem[]>([])
const shops = ref<ShopItem[]>([])
const llm = ref<LlmSetting | null>(null)
const llmSaving = ref(false)
const dialogVisible = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({
  name: '',
  shopId: 0,
  matchMode: 'exact',
  keywords: '',
  replyText: '',
  enabled: true,
  priority: 100,
  cooldownSec: 30,
})

async function load() {
  loading.value = true
  try {
    shops.value = (await listShops()) || []
    list.value = (await listAutoReplyRules()) || []
    llm.value = await getLlmSetting()
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.name = ''
  form.shopId = 0
  form.matchMode = 'exact'
  form.keywords = ''
  form.replyText = ''
  form.enabled = true
  form.priority = 100
  form.cooldownSec = 30
  dialogVisible.value = true
}

function openEdit(row: AutoReplyRuleItem) {
  editingId.value = row.id
  form.name = row.name
  form.shopId = row.shopId || 0
  form.matchMode = row.matchMode || 'exact'
  form.keywords = row.keywords
  form.replyText = row.replyText
  form.enabled = row.enabled
  form.priority = row.priority
  form.cooldownSec = row.cooldownSec
  dialogVisible.value = true
}

async function submit() {
  if (!form.name.trim() || !form.keywords.trim() || !form.replyText.trim()) {
    ElMessage.warning('请填写名称、关键词和回复内容')
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      shopId: form.shopId,
      matchMode: form.matchMode,
      keywords: form.keywords.trim(),
      replyText: form.replyText.trim(),
      enabled: form.enabled,
      priority: form.priority,
      cooldownSec: form.cooldownSec,
    }
    if (editingId.value) {
      await updateAutoReplyRule(editingId.value, payload)
      ElMessage.success('已保存')
    } else {
      await createAutoReplyRule(payload)
      ElMessage.success('已创建')
    }
    dialogVisible.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(row: AutoReplyRuleItem, val: boolean) {
  try {
    await updateAutoReplyRule(row.id, { enabled: val, name: row.name, keywords: row.keywords, replyText: row.replyText })
    row.enabled = val
  } catch (e: any) {
    ElMessage.error(e?.message || '更新失败')
    await load()
  }
}

async function onDelete(row: AutoReplyRuleItem) {
  try {
    await ElMessageBox.confirm(`删除规则「${row.name}」？`, '删除')
    await deleteAutoReplyRule(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e?.message || '删除失败')
  }
}

async function onSeed() {
  try {
    await seedAutoReplyPresets()
    ElMessage.success('已写入寒暄模板（仅在当前没有任何规则时生效）')
    await load()
  } catch (e: any) {
    ElMessage.error(e?.message || '写入失败')
  }
}

async function saveLlm() {
  if (!llm.value) return
  llmSaving.value = true
  try {
    llm.value = await saveLlmSetting({
      enabled: llm.value.enabled,
      styleHint: llm.value.styleHint,
      cooldownSec: llm.value.cooldownSec,
      systemPrompt: llm.value.systemPrompt,
      model: llm.value.model,
      maxChars: llm.value.maxChars,
      maxTokens: llm.value.maxTokens,
      timeoutSec: llm.value.timeoutSec,
      temperature: llm.value.temperature,
      thinkingEnabled: llm.value.thinkingEnabled,
      historyCount: llm.value.historyCount,
      inboundMaxChars: llm.value.inboundMaxChars,
      useProductContext: llm.value.useProductContext,
      retryStall: llm.value.retryStall,
    })
    ElMessage.success('已保存 DeepSeek 设置')
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
  } finally {
    llmSaving.value = false
  }
}

function restoreDefaultPrompt() {
  if (!llm.value) return
  llm.value.systemPrompt = llm.value.defaultSystemPrompt || ''
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="head">
      <div>
        <h2>自动回复</h2>
        <p class="hint">
          寒暄短句走关键词规则秒回，命中后应在 30 秒内发到飞鸽。DeepSeek 简短问答也尽量 30 秒内发出；接口本身慢了这轮就不强求。DeepSeek 刚回过也不会挡住「好 / 收到 / 再见」。没匹配上的问句才交给模型。
        </p>
      </div>
      <div class="actions">
        <el-button @click="load">刷新</el-button>
        <el-button @click="onSeed">写入寒暄模板</el-button>
        <el-button type="primary" @click="openCreate">新建规则</el-button>
      </div>
    </div>

    <el-card v-if="llm" class="llm-card" shadow="never">
      <div class="llm-row">
        <div>
          <div class="llm-title">DeepSeek 简短问答（可调参）</div>
          <p class="hint">
            关键词没中时才调用。API Key 仍在服务器，这里只调回复行为。
            当前约 {{ llm.maxChars }} 字、模型 {{ llm.model }}。
            <span v-if="!llm.configured">当前云端未配置 API Key，开关打开也不会发。</span>
          </p>
        </div>
        <el-switch
          v-model="llm.enabled"
          :disabled="!llm.configured"
          active-text="启用"
          @change="saveLlm"
        />
      </div>
      <el-form label-width="130px" class="llm-form">
        <el-form-item label="预设角色">
          <el-input
            v-model="llm.systemPrompt"
            type="textarea"
            :rows="8"
            maxlength="4000"
            show-word-limit
            placeholder="告诉模型它是谁、怎么回买家"
          />
          <el-button class="prompt-reset" link type="primary" @click="restoreDefaultPrompt">
            恢复默认提示词
          </el-button>
        </el-form-item>
        <el-form-item label="口吻补充">
          <el-input
            v-model="llm.styleHint"
            placeholder="例如：别叫亲，像杭州本地客服那样说话"
            maxlength="120"
            show-word-limit
          />
        </el-form-item>
        <el-form-item label="模型">
          <el-select v-model="llm.model" filterable allow-create default-first-option style="width: 280px">
            <el-option label="deepseek-flash（快、便宜）" value="deepseek-flash" />
            <el-option label="deepseek-v4-pro" value="deepseek-v4-pro" />
          </el-select>
        </el-form-item>
        <div class="llm-grid">
          <el-form-item label="回复字数">
            <el-input-number v-model="llm.maxChars" :min="20" :max="200" />
          </el-form-item>
          <el-form-item label="生成 Token">
            <el-input-number v-model="llm.maxTokens" :min="32" :max="800" :step="10" />
          </el-form-item>
          <el-form-item label="超时秒数">
            <el-input-number v-model="llm.timeoutSec" :min="5" :max="60" />
            <span class="muted">建议 20，整轮尽量 30 秒内发出</span>
          </el-form-item>
          <el-form-item label="温度">
            <el-input-number v-model="llm.temperature" :min="0.05" :max="1.2" :step="0.05" :precision="2" />
          </el-form-item>
          <el-form-item label="上下文条数">
            <el-input-number v-model="llm.historyCount" :min="2" :max="30" />
          </el-form-item>
          <el-form-item label="买家字数上限">
            <el-input-number v-model="llm.inboundMaxChars" :min="20" :max="200" />
            <span class="muted">超过不走模型</span>
          </el-form-item>
          <el-form-item label="冷却秒数">
            <el-input-number v-model="llm.cooldownSec" :min="10" :max="180" />
            <span class="muted">同会话间隔内不连回</span>
          </el-form-item>
        </div>
        <el-form-item label="思考模式">
          <el-switch v-model="llm.thinkingEnabled" active-text="开" inactive-text="关" />
          <span class="muted">客服短句建议关，开了更慢更贵</span>
        </el-form-item>
        <el-form-item label="咨询宝贝">
          <el-switch v-model="llm.useProductContext" active-text="写入背景" />
          <span class="muted">把飞鸽右侧咨询宝贝 / 浏览足迹给模型</span>
        </el-form-item>
        <el-form-item label="空话重试">
          <el-switch v-model="llm.retryStall" active-text="重写稍等句" />
          <span class="muted">「帮你看下 / 稍等」会再要一句实答</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="llmSaving" :disabled="!llm.configured" @click="saveLlm">
            保存模型设置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="120" />
      <el-table-column label="店铺" min-width="120">
        <template #default="{ row }">{{ row.shopName || (row.shopId ? row.shopId : '全部店铺') }}</template>
      </el-table-column>
      <el-table-column label="匹配" width="90">
        <template #default="{ row }">{{ row.matchMode === 'contains' ? '包含' : '完全匹配' }}</template>
      </el-table-column>
      <el-table-column prop="keywords" label="关键词" min-width="180" show-overflow-tooltip />
      <el-table-column prop="replyText" label="回复内容" min-width="200" show-overflow-tooltip />
      <el-table-column prop="priority" label="优先级" width="80" />
      <el-table-column label="启用" width="90">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="(v: boolean) => toggleEnabled(row, v)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑规则' : '新建规则'" width="560px">
      <el-form label-width="110px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="如：寒暄收到" />
        </el-form-item>
        <el-form-item label="店铺">
          <el-select v-model="form.shopId" style="width: 100%">
            <el-option label="全部店铺" :value="0" />
            <el-option v-for="s in shops" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="匹配方式">
          <el-select v-model="form.matchMode" style="width: 100%">
            <el-option label="完全匹配（去标点后相等）" value="exact" />
            <el-option label="包含关键词" value="contains" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词" required>
          <el-input
            v-model="form.keywords"
            type="textarea"
            :rows="2"
            placeholder="逗号分隔，如：好的,谢谢,收到"
          />
        </el-form-item>
        <el-form-item label="回复内容" required>
          <el-input v-model="form.replyText" type="textarea" :rows="3" placeholder="发到飞鸽的固定语句" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="form.priority" :min="1" :max="999" />
          <span class="muted">数字越小越先匹配</span>
        </el-form-item>
        <el-form-item label="冷却秒数">
          <el-input-number v-model="form.cooldownSec" :min="0" :max="600" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  gap: 12px;
}
.hint {
  margin: 4px 0 0;
  color: #909399;
  font-size: 13px;
  max-width: 860px;
}
.actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.muted {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}
.llm-card {
  margin-bottom: 16px;
}
.llm-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 8px;
}
.llm-title {
  font-weight: 600;
}
.llm-form {
  max-width: 920px;
}
.llm-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
}
.prompt-reset {
  margin-top: 4px;
}
</style>
