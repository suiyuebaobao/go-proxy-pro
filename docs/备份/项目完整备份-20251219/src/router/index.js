import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { guest: true }
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue')
      },
      {
        path: 'accounts',
        name: 'Accounts',
        component: () => import('@/views/Accounts.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'models',
        name: 'Models',
        component: () => import('@/views/Models.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/Users.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'profile',
        name: 'Profile',
        component: () => import('@/views/Profile.vue')
      },
      {
        path: 'usage',
        name: 'Usage',
        component: () => import('@/views/Usage.vue')
      },
      {
        path: 'api-keys',
        name: 'APIKeys',
        component: () => import('@/views/APIKeys.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'proxy-test',
        name: 'ProxyTest',
        component: () => import('@/views/ProxyTest.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'request-logs',
        name: 'RequestLogs',
        component: () => import('@/views/RequestLogs.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'account-load',
        name: 'AccountLoad',
        component: () => import('@/views/AccountLoad.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'cache',
        name: 'Cache',
        component: () => import('@/views/Cache.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/Settings.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'packages',
        name: 'Packages',
        component: () => import('@/views/Packages.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'proxies',
        name: 'Proxies',
        component: () => import('@/views/Proxies.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'operation-logs',
        name: 'OperationLogs',
        component: () => import('@/views/OperationLogs.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'client-filter',
        name: 'ClientFilter',
        component: () => import('@/views/ClientFilter.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'system-monitor',
        name: 'SystemMonitor',
        component: () => import('@/views/SystemMonitor.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'error-messages',
        name: 'ErrorMessages',
        component: () => import('@/views/ErrorMessages.vue'),
        meta: { requiresAdmin: true }
      },
      {
        path: 'system-logs',
        name: 'SystemLogs',
        component: () => import('@/views/SystemLogs.vue'),
        meta: { requiresAdmin: true }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const userStore = useUserStore()

  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    next('/login')
  } else if (to.meta.guest && userStore.isLoggedIn) {
    next('/')
  } else if (to.meta.requiresAdmin && userStore.user?.role !== 'admin') {
    next('/')
  } else {
    next()
  }
})

export default router
