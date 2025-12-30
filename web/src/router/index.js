/**
 * 文件作用：Vue路由配置，定义应用页面路由和导航守卫
 * 负责功能：
 *   - 页面路由定义
 *   - 权限路由守卫
 *   - 管理员/用户路由分离
 *   - 登录状态检查
 * 重要程度：⭐⭐⭐⭐ 重要（前端路由核心）
 * 依赖模块：vue-router, user store
 */
import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { guest: true }
  },
  // 首页（公共页面）
  {
    path: '/',
    name: 'Home',
    component: () => import('@/views/Home.vue'),
    meta: { guest: true }
  },
  // 后台入口（根据角色重定向）
  {
    path: '/dashboard',
    redirect: to => {
      const userStore = useUserStore()
      if (userStore.user?.role === 'admin') {
        return '/admin/system-monitor'
      }
      return '/user/dashboard'
    }
  },
  // 管理员后台路由
  {
    path: '/admin',
    component: () => import('@/layouts/MainLayout.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
    children: [
      {
        path: '',
        redirect: '/admin/system-monitor'
      },
      {
        path: 'system-monitor',
        name: 'SystemMonitor',
        component: () => import('@/views/SystemMonitor.vue')
      },
      {
        path: 'accounts',
        name: 'Accounts',
        component: () => import('@/views/Accounts.vue')
      },
      {
        path: 'models',
        name: 'Models',
        component: () => import('@/views/Models.vue')
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/Users.vue')
      },
      {
        path: 'api-keys',
        name: 'AdminAPIKeys',
        component: () => import('@/views/APIKeys.vue')
      },
      {
        path: 'request-logs',
        name: 'RequestLogs',
        component: () => import('@/views/RequestLogs.vue')
      },
      {
        path: 'account-load',
        name: 'AccountLoad',
        component: () => import('@/views/AccountLoad.vue')
      },
      {
        path: 'cache',
        name: 'Cache',
        component: () => import('@/views/Cache.vue')
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/Settings.vue')
      },
      {
        path: 'packages',
        name: 'Packages',
        component: () => import('@/views/Packages.vue')
      },
      {
        path: 'proxies',
        name: 'Proxies',
        component: () => import('@/views/Proxies.vue')
      },
      {
        path: 'operation-logs',
        name: 'OperationLogs',
        component: () => import('@/views/OperationLogs.vue')
      },
      {
        path: 'client-filter',
        name: 'ClientFilter',
        component: () => import('@/views/ClientFilter.vue')
      },
      {
        path: 'error-messages',
        name: 'ErrorMessages',
        component: () => import('@/views/ErrorMessages.vue')
      },
      {
        path: 'system-logs',
        name: 'SystemLogs',
        component: () => import('@/views/SystemLogs.vue')
      }
    ]
  },
  // 用户中心路由
  {
    path: '/user',
    component: () => import('@/layouts/UserLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: '/user/dashboard'
      },
      {
        path: 'dashboard',
        name: 'UserDashboard',
        component: () => import('@/views/user/UserDashboard.vue')
      },
      {
        path: 'api-keys',
        name: 'UserAPIKeys',
        component: () => import('@/views/user/MyAPIKeys.vue')
      },
      {
        path: 'packages',
        name: 'UserPackages',
        component: () => import('@/views/user/MyPackages.vue')
      },
      {
        path: 'records',
        name: 'UserRecords',
        component: () => import('@/views/user/MyUsageRecords.vue')
      },
      {
        path: 'profile',
        name: 'UserProfile',
        component: () => import('@/views/Profile.vue')
      }
    ]
  },
  // 旧路由兼容重定向
  {
    path: '/system-monitor',
    redirect: '/admin/system-monitor'
  },
  {
    path: '/accounts',
    redirect: '/admin/accounts'
  },
  {
    path: '/models',
    redirect: '/admin/models'
  },
  {
    path: '/users',
    redirect: '/admin/users'
  },
  {
    path: '/api-keys',
    redirect: '/admin/api-keys'
  },
  {
    path: '/request-logs',
    redirect: '/admin/request-logs'
  },
  {
    path: '/packages',
    redirect: '/admin/packages'
  },
  {
    path: '/settings',
    redirect: '/admin/settings'
  },
  // 404 处理
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const userStore = useUserStore()

  // 需要登录但未登录
  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    next('/login')
    return
  }

  // 已登录访问登录页，跳转到后台
  if (to.path === '/login' && userStore.isLoggedIn) {
    if (userStore.user?.role === 'admin') {
      next('/admin/system-monitor')
    } else {
      next('/user/dashboard')
    }
    return
  }

  // 需要管理员权限但非管理员
  if (to.meta.requiresAdmin && userStore.user?.role !== 'admin') {
    next('/user/dashboard')
    return
  }

  next()
})

export default router
