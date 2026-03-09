import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
      meta: { requiresAuth: false, title: '登录' }
    },
    {
      path: '/register',
      name: 'Register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { requiresAuth: false, title: '注册' }
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          redirect: '/dashboard'
        },
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('@/views/DashboardView.vue'),
          meta: { title: '总览' }
        },
        {
          path: 'accounts',
          name: 'Accounts',
          component: () => import('@/views/AccountsView.vue'),
          meta: { title: '账号管理' }
        },
        // Account-specific routes
        {
          path: 'account/:id/home',
          name: 'AccountHome',
          component: () => import('@/views/AccountHomeView.vue'),
          meta: { title: '首页' }
        },
        {
          path: 'account/:id/lands',
          name: 'AccountLands',
          component: () => import('@/views/AccountLandsView.vue'),
          meta: { title: '土地' }
        },
        {
          path: 'account/:id/settings',
          name: 'AccountSettings',
          component: () => import('@/views/AccountSettingsView.vue'),
          meta: { title: '配置' }
        },
        {
          path: 'account/:id/logs',
          name: 'AccountLogs',
          component: () => import('@/views/AccountLogsView.vue'),
          meta: { title: '日志' }
        },
        // Global routes
        {
          path: 'logs',
          name: 'Logs',
          component: () => import('@/views/LogsView.vue'),
          meta: { title: '实时日志' }
        },
        {
          path: 'crop-yield',
          name: 'CropYield',
          component: () => import('@/views/CropYieldView.vue'),
          meta: { title: '种植排行' }
        },
        {
          path: 'level-up-time',
          name: 'LevelUpTime',
          component: () => import('@/views/LevelUpTimeView.vue'),
          meta: { title: '升级计算' }
        },
        {
          path: 'stats',
          name: 'Stats',
          component: () => import('@/views/StatsView.vue'),
          meta: { title: '操作统计' }
        },
        {
          path: 'data-summary',
          name: 'DataSummary',
          component: () => import('@/views/DataSummaryView.vue'),
          meta: { title: '数据汇总' }
        },
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/dashboard'
    }
  ]
})

// Auth guard
router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if ((to.name === 'Login' || to.name === 'Register') && authStore.isAuthenticated) {
    next({ name: 'Dashboard' })
  } else {
    next()
  }
})

export default router
