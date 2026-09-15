<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChatDotRound, ChatLineSquare, Shop } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const collapsed = defineModel<boolean>('collapsed', { default: false })

const activeMenu = computed(() => {
  if (route.path.startsWith('/conversations')) return '/conversations'
  if (route.path.startsWith('/auto-reply')) return '/auto-reply'
  if (route.path.startsWith('/shops')) return '/shops'
  return route.path
})

const menuItems = [
  { path: '/shops', title: '店铺', icon: Shop },
  { path: '/conversations', title: '会话', icon: ChatDotRound },
  { path: '/auto-reply', title: '自动回复', icon: ChatLineSquare },
]

const logoText = computed(() => (collapsed.value ? '客服' : '客服中心'))

function navigate(path: string) {
  router.push(path)
}
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="logo">{{ logoText }}</div>
    <el-menu
      :default-active="activeMenu"
      :collapse="collapsed"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <el-menu-item
        v-for="item in menuItems"
        :key="item.path"
        :index="item.path"
        @click="navigate(item.path)"
      >
        <el-icon><component :is="item.icon" /></el-icon>
        <span>{{ item.title }}</span>
      </el-menu-item>
    </el-menu>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  background: #001529;
  transition: width 0.2s;
  flex-shrink: 0;
}
.sidebar.collapsed {
  width: 64px;
}
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  border-bottom: 1px solid #ffffff14;
}
.sidebar :deep(.el-menu) {
  border-right: none;
}
</style>
