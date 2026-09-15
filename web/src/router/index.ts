import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import { redirectToPortal, ensureSession, clearToken } from '../utils/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/auth/callback',
      name: 'AuthCallback',
      component: () => import('../views/AuthCallback.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/logout',
      name: 'AuthLogout',
      component: () => import('../views/AuthLogout.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/shops',
      children: [
        { path: 'shops', name: 'Shops', component: () => import('../views/Shops.vue'), meta: { title: '店铺' } },
        { path: 'conversations', name: 'Conversations', component: () => import('../views/Conversations.vue'), meta: { title: '会话' } },
        { path: 'auto-reply', name: 'AutoReply', component: () => import('../views/AutoReply.vue'), meta: { title: '自动回复' } },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  const ok = await ensureSession()
  if (!ok) {
    clearToken()
    redirectToPortal()
    return false
  }
  return true
})

export default router
