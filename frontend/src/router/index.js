import { createRouter, createWebHistory } from 'vue-router'
import { routes } from 'vue-router/auto-routes'
import { getIngressBase } from '@/utils/ingress'

function getRouterBase() {
  const base = getIngressBase()
  return base ? base + '/' : import.meta.env.BASE_URL
}

const router = createRouter({
  history: createWebHistory(getRouterBase()),
  routes: [...routes],
})

export default router
