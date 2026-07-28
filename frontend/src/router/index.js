import { createRouter, createWebHistory } from 'vue-router'
import { routes } from 'vue-router/auto-routes'
import { getIngressBase } from '@/utils/ingress'

function getRouterBase() {
  const base = getIngressBase()
  return base ? base + '/' : import.meta.env.BASE_URL
}

const router = createRouter({
  history: createWebHistory(getRouterBase()),
  routes: [
    ...routes,
    { path: '/server', redirect: '/settings/server' },
    { path: '/audio', redirect: '/settings/audio' },
    { path: '/channels', redirect: '/' },
    { path: '/chat', redirect: '/' },
  ],
})

export default router
