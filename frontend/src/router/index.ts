import { createRouter, createWebHistory } from 'vue-router'
import { tokenKey } from '../api/client'

const titles: Record<string, string> = {
  points: '监测点', measurements: '测量工作台', sources: '声源谱', attribution: '贡献归因', audit: '审计中心', login: '登录',
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/points' },
    { path: '/login', name: 'login', component: () => import('../pages/LoginPage.vue'), meta: { public: true } },
    { path: '/points', name: 'points', component: () => import('../pages/PointsPage.vue') },
    { path: '/measurements', name: 'measurements', component: () => import('../pages/MeasurementsPage.vue') },
    { path: '/sources', name: 'sources', component: () => import('../pages/SourcesPage.vue') },
    { path: '/attribution', name: 'attribution', component: () => import('../pages/AttributionPage.vue') },
    { path: '/audit', name: 'audit', component: () => import('../pages/AuditPage.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/points' },
  ],
})

router.beforeEach((to) => {
  document.title = `${titles[String(to.name)] ?? '工作台'} · NoiseTrace 噪声源贡献归因台`
  const authenticated = Boolean(localStorage.getItem(tokenKey))
  if (!to.meta.public && !authenticated) return { name: 'login', query: { redirect: to.fullPath } }
  if (to.name === 'login' && authenticated) return { name: 'points' }
})

window.addEventListener('noisetrace-auth-expired', () => router.push('/login'))
export default router
