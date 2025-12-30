<!--
 * 文件作用：主布局组件，定义管理后台整体布局结构
 * 负责功能：
 *   - 侧边栏导航菜单
 *   - 顶部用户信息栏
 *   - 内容区路由出口
 *   - 菜单折叠控制
 * 重要程度：⭐⭐⭐⭐ 重要（主布局框架）
 * 依赖模块：element-plus, vue-router, user store
-->
<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '200px'" class="layout-aside">
      <div class="logo">
        <span v-if="!isCollapse">AIProxy</span>
        <span v-else>AP</span>
      </div>

      <el-menu
        :default-active="route.path"
        :collapse="isCollapse"
        router
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409eff"
      >
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
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { prefetchChunk } from '@/prefetch'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const isCollapse = ref(false)

function prefetchFor(path) {
  const loaders = {
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
  background-color: #304156;
  transition: width 0.3s;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 20px;
  font-weight: bold;
  background-color: #263445;
}

.layout-header {
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
  padding: 0 20px;
}

.collapse-btn {
  font-size: 20px;
  cursor: pointer;
}

.user-info {
  display: flex;
  align-items: center;
  cursor: pointer;
}

.username {
  margin-left: 8px;
  color: #333;
}

.layout-main {
  background-color: #f0f2f5;
}

.el-menu {
  border-right: none;
}
</style>
