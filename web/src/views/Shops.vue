<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createShop,
  listShops,
  resetPlugin,
  rotateBindCode,
  updateShop,
  type ShopItem,
} from '../api/shops'

const loading = ref(false)
const list = ref<ShopItem[]>([])
const createVisible = ref(false)
const creating = ref(false)
const form = reactive({
  name: '',
  platform: 'doudian',
  platformShopId: '',
  platformShopName: '',
  remark: '',
})

async function load() {
  loading.value = true
  try {
    list.value = (await listShops()) || []
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.name = ''
  form.platform = 'doudian'
  form.platformShopId = ''
  form.platformShopName = ''
  form.remark = ''
  createVisible.value = true
}

async function submitCreate() {
  if (!form.name.trim() || !form.platformShopId.trim()) {
    ElMessage.warning('请填写店铺名称与平台店铺 ID')
    return
  }
  creating.value = true
  try {
    await createShop({
      name: form.name.trim(),
      platform: form.platform,
      platformShopId: form.platformShopId.trim(),
      platformShopName: form.platformShopName.trim() || undefined,
      remark: form.remark.trim() || undefined,
    })
    ElMessage.success('已创建')
    createVisible.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  } finally {
    creating.value = false
  }
}

async function toggleMonitor(row: ShopItem, val: boolean) {
  try {
    await updateShop(row.id, { monitorEnabled: val })
    row.monitorEnabled = val
    ElMessage.success(val ? '已开启监控' : '已关闭监控，本机将自动关闭飞鸽窗口')
  } catch (e: any) {
    ElMessage.error(e?.message || '更新失败')
    await load()
  }
}

async function onRotateBind(row: ShopItem) {
  try {
    await ElMessageBox.confirm('将重新生成绑定码并清除插件凭证，确认继续？', '轮换绑定码')
    const item = await rotateBindCode(row.id)
    Object.assign(row, item)
    ElMessage.success(`新绑定码：${item.bindCode}`)
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e?.message || '操作失败')
  }
}

async function onResetPlugin(row: ShopItem) {
  try {
    await ElMessageBox.confirm('将清除插件绑定，店铺需重新扫码绑定，确认继续？', '重置插件')
    const item = await resetPlugin(row.id)
    Object.assign(row, item)
    ElMessage.success('已重置插件绑定')
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e?.message || '操作失败')
  }
}

function statusType(status: string) {
  if (status === 'online') return 'success'
  if (status === 'bound') return 'warning'
  if (status === 'offline') return 'info'
  return 'info'
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="head">
      <h2>店铺</h2>
      <div class="actions">
        <el-button @click="load">刷新</el-button>
        <el-button type="primary" @click="openCreate">新建店铺</el-button>
      </div>
    </div>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="platformLabel" label="平台" width="90" />
      <el-table-column prop="platformShopId" label="平台店铺 ID" min-width="140" show-overflow-tooltip />
      <el-table-column prop="bindCode" label="绑定码" width="110">
        <template #default="{ row }">
          <el-tag type="primary" effect="plain">{{ row.bindCode }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="插件状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.pluginStatus)" size="small">{{ row.pluginStatus }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="监控" width="90">
        <template #default="{ row }">
          <el-switch :model-value="row.monitorEnabled" @change="(v: boolean) => toggleMonitor(row, v)" />
        </template>
      </el-table-column>
      <el-table-column prop="lastSeenAt" label="最近心跳" min-width="160" />
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="onRotateBind(row)">轮换绑定码</el-button>
          <el-button link type="danger" @click="onResetPlugin(row)">重置插件</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" title="新建店铺" width="480px">
      <el-form label-width="110px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="店铺显示名" />
        </el-form-item>
        <el-form-item label="平台">
          <el-select v-model="form.platform" style="width: 100%">
            <el-option label="抖店" value="doudian" />
          </el-select>
        </el-form-item>
        <el-form-item label="平台店铺 ID" required>
          <el-input v-model="form.platformShopId" placeholder="与 Agent 上报一致" />
        </el-form-item>
        <el-form-item label="平台店铺名">
          <el-input v-model="form.platformShopName" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>
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
.actions {
  display: flex;
  gap: 8px;
}
</style>
