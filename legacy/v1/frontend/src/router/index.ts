import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('@/views/DashboardView.vue'),
      meta: { title: '看板' },
    },
    {
      path: '/problems',
      name: 'problems',
      component: () => import('@/views/ProblemsView.vue'),
      meta: { title: '问题清单' },
    },
  ],
})

router.afterEach((to) => {
  if (to.meta?.title) document.title = `${to.meta.title} · 问题协同研究解决平台`
})

export default router
