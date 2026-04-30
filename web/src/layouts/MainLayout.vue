<!--
 * 文件作用：主布局组件，定义管理后台整体布局结构
 * 负责功能：
 *   - 侧边栏导航菜单
 *   - 顶部用户信息栏和主题切换
 *   - 内容区路由出口
 *   - 菜单折叠控制
 * 重要程度：⭐⭐⭐⭐ 重要（主布局框架）
 * 依赖模块：element-plus, vue-router, user store, theme store, ThemeSwitcher
-->
<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '200px'" class="layout-aside">
      <div class="logo">
        <span v-if="!isCollapse">{{ siteStore.siteName }}</span>
        <span v-else>{{ siteStore.siteName.charAt(0) }}</span>
      </div>

      <el-menu
        :default-active="route.path"
        :collapse="isCollapse"
        router
        background-color="transparent"
        :text-color="menuTextColor"
        :active-text-color="menuActiveColor"
      >
        <el-menu-item index="/admin/overview" @mouseenter="prefetchFor('/admin/overview')">
          <el-icon><DataAnalysis /></el-icon>
          <span>数据概览</span>
        </el-menu-item>

        <el-menu-item index="/admin/system-monitor" @mouseenter="prefetchFor('/admin/system-monitor')">
          <el-icon><Monitor /></el-icon>
          <span>系统监控</span>
        </el-menu-item>

        <el-menu-item index="/admin/accounts" @mouseenter="prefetchFor('/admin/accounts')">
          <el-icon><Key /></el-icon>
          <span>账户管理</span>
        </el-menu-item>

        <el-menu-item index="/admin/proxies" @mouseenter="prefetchFor('/admin/proxies')">
          <el-icon><Position /></el-icon>
          <span>代理管理</span>
        </el-menu-item>

        <el-menu-item index="/admin/models" @mouseenter="prefetchFor('/admin/models')">
          <el-icon><Cpu /></el-icon>
          <span>模型管理</span>
        </el-menu-item>

        <el-menu-item index="/admin/users" @mouseenter="prefetchFor('/admin/users')">
          <el-icon><User /></el-icon>
          <span>用户管理</span>
        </el-menu-item>

        <el-menu-item index="/admin/request-logs" @mouseenter="prefetchFor('/admin/request-logs')">
          <el-icon><Document /></el-icon>
          <span>请求日志</span>
        </el-menu-item>

        <el-menu-item index="/admin/account-load" @mouseenter="prefetchFor('/admin/account-load')">
          <el-icon><TrendCharts /></el-icon>
          <span>账户负载</span>
        </el-menu-item>

        <el-menu-item index="/admin/cache" @mouseenter="prefetchFor('/admin/cache')">
          <el-icon><Box /></el-icon>
          <span>缓存管理</span>
        </el-menu-item>

        <el-menu-item index="/admin/api-keys" @mouseenter="prefetchFor('/admin/api-keys')">
          <el-icon><Tickets /></el-icon>
          <span>API Key 管理</span>
        </el-menu-item>

        <el-menu-item index="/admin/packages" @mouseenter="prefetchFor('/admin/packages')">
          <el-icon><ShoppingBag /></el-icon>
          <span>套餐管理</span>
        </el-menu-item>

        <el-menu-item index="/admin/settings" @mouseenter="prefetchFor('/admin/settings')">
          <el-icon><Tools /></el-icon>
          <span>系统设置</span>
        </el-menu-item>

        <el-menu-item index="/admin/error-messages" @mouseenter="prefetchFor('/admin/error-messages')">
          <el-icon><Warning /></el-icon>
          <span>错误消息</span>
        </el-menu-item>

        <el-menu-item index="/admin/operation-logs" @mouseenter="prefetchFor('/admin/operation-logs')">
          <el-icon><Notebook /></el-icon>
          <span>操作日志</span>
        </el-menu-item>

        <el-menu-item index="/admin/alerts" @mouseenter="prefetchFor('/admin/alerts')">
          <el-icon><Warning /></el-icon>
          <span>告警管理</span>
        </el-menu-item>

        <el-menu-item index="/admin/system-logs" @mouseenter="prefetchFor('/admin/system-logs')">
          <el-icon><Files /></el-icon>
          <span>系统日志</span>
        </el-menu-item>

        <el-menu-item index="/admin/client-filter" @mouseenter="prefetchFor('/admin/client-filter')">
          <el-icon><Filter /></el-icon>
          <span>客户端过滤</span>
        </el-menu-item>

        <el-divider />

        <el-menu-item index="/user/dashboard" @mouseenter="prefetchFor('/user/dashboard')">
          <el-icon><SwitchButton /></el-icon>
          <span>用户中心</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 主内容 -->
    <el-container>
      <!-- 顶栏 -->
      <el-header class="layout-header">
        <div class="header-left">
          <el-icon class="collapse-btn" @click="isCollapse = !isCollapse">
            <Expand v-if="isCollapse" />
            <Fold v-else />
          </el-icon>
        </div>

        <div class="header-right">
          <ThemeSwitcher />

          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-avatar :size="32" icon="User" />
              <span class="username">{{ userStore.user?.username }}</span>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">个人设置</el-dropdown-item>
                <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 内容区 -->
      <el-main class="layout-main">
        <router-view v-slot="{ Component }">
          <keep-alive :max="5" :include="['Profile']">
            <component :is="Component" />
          </keep-alive>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'
import { useSiteStore } from '@/stores/site'
import { prefetchChunk } from '@/prefetch'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const themeStore = useThemeStore()
const siteStore = useSiteStore()
siteStore.fetchSiteName()

const isCollapse = ref(false)

const menuTextColor = computed(() => themeStore.getThemeMeta().vars['--el-text-color-regular'])
const menuActiveColor = computed(() => themeStore.getThemeMeta().vars['--pink-accent'])

function prefetchFor(path) {
  const loaders = {
    '/admin/overview': () => import('@/views/Overview.vue'),
    '/admin/system-monitor': () => import('@/views/SystemMonitor.vue'),
    '/admin/accounts': () => import('@/views/Accounts.vue'),
    '/admin/proxies': () => import('@/views/Proxies.vue'),
    '/admin/models': () => import('@/views/Models.vue'),
    '/admin/users': () => import('@/views/Users.vue'),
    '/admin/request-logs': () => import('@/views/RequestLogs.vue'),
    '/admin/account-load': () => import('@/views/AccountLoad.vue'),
    '/admin/cache': () => import('@/views/Cache.vue'),
    '/admin/api-keys': () => import('@/views/APIKeys.vue'),
    '/admin/packages': () => import('@/views/Packages.vue'),
    '/admin/settings': () => import('@/views/Settings.vue'),
    '/admin/error-messages': () => import('@/views/ErrorMessages.vue'),
    '/admin/operation-logs': () => import('@/views/OperationLogs.vue'),
    '/admin/system-logs': () => import('@/views/SystemLogs.vue'),
    '/admin/client-filter': () => import('@/views/ClientFilter.vue'),
    '/user/dashboard': () => import('@/views/user/UserDashboard.vue')
  }
  const loader = loaders[path]
  if (!loader) return
  prefetchChunk(path, loader)
}

function handleCommand(cmd) {
  if (cmd === 'logout') {
    userStore.logout()
    router.push('/login')
  } else if (cmd === 'profile') {
    router.push('/user/profile')
  }
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.layout-aside {
  background-color: var(--pink-sidebar, #ffffff);
  border-right: 1px solid var(--pink-border, #f0dde2);
  transition: width 0.3s;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--pink-accent, #c97b8b);
  font-size: 20px;
  font-weight: 700;
  letter-spacing: 1px;
  border-bottom: 1px solid var(--pink-border, #f0dde2);
}

.layout-header {
  background: var(--pink-surface, #ffffff);
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--pink-border, #f0dde2);
  padding: 0 24px;
  height: 60px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.collapse-btn {
  font-size: 20px;
  cursor: pointer;
  color: var(--pink-text-secondary, #8b7d92);
  transition: color 0.2s;
}

.collapse-btn:hover {
  color: var(--pink-accent, #c97b8b);
}

.user-info {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 6px 12px;
  border-radius: 8px;
  transition: background 0.2s;
}

.user-info:hover {
  background: var(--pink-accent-light, #faf2f4);
}

.username {
  margin-left: 8px;
  color: var(--pink-text, #3a3045);
  font-weight: 500;
}

.layout-main {
  background-color: var(--pink-bg, #fdf8f9);
  padding: 20px;
}

.el-menu {
  border-right: none;
}

:deep(.el-menu-item) {
  margin: 2px 8px;
  border-radius: 8px;
  height: 44px;
  line-height: 44px;
  transition: all 0.2s;
}

:deep(.el-menu-item:hover) {
  background-color: var(--pink-accent-light, #faf2f4) !important;
}

:deep(.el-menu-item.is-active) {
  background-color: var(--pink-accent-light, #faf2f4) !important;
  color: var(--pink-accent, #c97b8b) !important;
  font-weight: 600;
  border-left: 3px solid var(--pink-accent, #c97b8b);
}

:deep(.el-divider) {
  margin: 8px 16px;
  border-top-color: var(--pink-border, #f0dde2);
}
</style>
