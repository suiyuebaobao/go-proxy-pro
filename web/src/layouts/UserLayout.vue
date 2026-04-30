<!--
 * 文件作用：用户中心布局组件
 * 负责功能：
 *   - 用户中心页面框架
 *   - 顶部导航栏和主题切换
 *   - 侧边菜单
 *   - 内容区域
 * 重要程度：⭐⭐⭐⭐ 重要（用户界面框架）
-->
<template>
  <el-container class="user-layout">
    <!-- 顶部导航 -->
    <el-header class="header">
      <div class="header-left">
        <div class="logo">
          <el-icon :size="24"><Monitor /></el-icon>
          <span class="logo-text">{{ siteStore.siteName }} 用户中心</span>
        </div>
      </div>
      <div class="header-right">
        <ThemeSwitcher />

        <el-dropdown @command="handleCommand">
          <span class="user-info">
            <el-avatar :size="32" class="avatar">
              {{ userStore.user?.username?.charAt(0)?.toUpperCase() || 'U' }}
            </el-avatar>
            <span class="username">{{ userStore.user?.username || '用户' }}</span>
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">
                <el-icon><User /></el-icon> 个人资料
              </el-dropdown-item>
              <el-dropdown-item v-if="userStore.user?.role === 'admin'" command="admin" divided>
                <el-icon><Setting /></el-icon> 管理后台
              </el-dropdown-item>
              <el-dropdown-item command="logout" divided>
                <el-icon><SwitchButton /></el-icon> 退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>

    <el-container class="main-container">
      <!-- 侧边菜单 -->
      <el-aside :width="isCollapse ? '64px' : '200px'" class="aside">
        <el-menu
          :default-active="activeMenu"
          :collapse="isCollapse"
          :router="true"
          class="side-menu"
        >
          <el-menu-item index="/user/dashboard" @mouseenter="prefetchFor('/user/dashboard')">
            <el-icon><DataAnalysis /></el-icon>
            <template #title>仪表盘</template>
          </el-menu-item>
          <el-menu-item index="/user/api-keys" @mouseenter="prefetchFor('/user/api-keys')">
            <el-icon><Key /></el-icon>
            <template #title>我的 API Key</template>
          </el-menu-item>
          <el-menu-item index="/user/packages" @mouseenter="prefetchFor('/user/packages')">
            <el-icon><Box /></el-icon>
            <template #title>我的套餐</template>
          </el-menu-item>
          <el-menu-item index="/user/records" @mouseenter="prefetchFor('/user/records')">
            <el-icon><Document /></el-icon>
            <template #title>使用记录</template>
          </el-menu-item>
          <el-menu-item index="/user/api-docs" @mouseenter="prefetchFor('/user/api-docs')">
            <el-icon><Reading /></el-icon>
            <template #title>API 文档</template>
          </el-menu-item>
        </el-menu>

        <div class="collapse-btn" @click="isCollapse = !isCollapse">
          <el-icon v-if="isCollapse"><Expand /></el-icon>
          <el-icon v-else><Fold /></el-icon>
        </div>
      </el-aside>

      <!-- 主内容区 -->
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useSiteStore } from '@/stores/site'
import { prefetchChunk } from '@/prefetch'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'

const siteStore = useSiteStore()
siteStore.fetchSiteName()
import {
  Monitor, ArrowDown, User, Setting, SwitchButton,
  DataAnalysis, Key, Box, Document, Expand, Fold, Reading
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const isCollapse = ref(false)

const activeMenu = computed(() => route.path)

function prefetchFor(path) {
  const loaders = {
    '/user/dashboard': () => import('@/views/user/UserDashboard.vue'),
    '/user/api-keys': () => import('@/views/user/MyAPIKeys.vue'),
    '/user/packages': () => import('@/views/user/MyPackages.vue'),
    '/user/records': () => import('@/views/user/MyUsageRecords.vue'),
    '/user/api-docs': () => import('@/views/user/ApiDocs.vue'),
    '/user/profile': () => import('@/views/Profile.vue'),
    '/admin/system-monitor': () => import('@/views/SystemMonitor.vue')
  }
  const loader = loaders[path]
  if (!loader) return
  prefetchChunk(path, loader)
}

const handleCommand = (command) => {
  switch (command) {
    case 'profile':
      prefetchFor('/user/profile')
      router.push('/user/profile')
      break
    case 'admin':
      prefetchFor('/admin/system-monitor')
      router.push('/admin/system-monitor')
      break
    case 'logout':
      userStore.logout()
      router.push('/login')
      break
  }
}
</script>

<style scoped>
.user-layout {
  height: 100vh;
  background: var(--pink-bg, #fdf8f9);
}

.header {
  background: var(--pink-surface, #ffffff);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 24px;
  border-bottom: 1px solid var(--pink-border, #f0dde2);
  height: 60px;
}

.header-left {
  display: flex;
  align-items: center;
}

.logo {
  display: flex;
  align-items: center;
  color: var(--pink-accent, #c97b8b);
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.5px;
}

.logo-text {
  margin-left: 10px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-info {
  display: flex;
  align-items: center;
  color: var(--pink-text, #3a3045);
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 8px;
  transition: background 0.2s;
}

.user-info:hover {
  background: var(--pink-accent-light, #faf2f4);
}

.avatar {
  background: var(--pink-accent, #c97b8b);
  color: white;
  font-weight: 600;
}

.username {
  margin: 0 8px;
  font-size: 14px;
  font-weight: 500;
}

.main-container {
  height: calc(100vh - 60px);
}

.aside {
  background: var(--pink-surface, #ffffff);
  border-right: 1px solid var(--pink-border, #f0dde2);
  display: flex;
  flex-direction: column;
  transition: width 0.3s;
}

.side-menu {
  flex: 1;
  border-right: none;
}

.side-menu:not(.el-menu--collapse) {
  width: 200px;
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

.collapse-btn {
  padding: 15px;
  text-align: center;
  cursor: pointer;
  border-top: 1px solid var(--pink-border, #f0dde2);
  color: var(--pink-text-secondary, #8b7d92);
  transition: all 0.2s;
}

.collapse-btn:hover {
  background: var(--pink-accent-light, #faf2f4);
  color: var(--pink-accent, #c97b8b);
}

.main {
  padding: 20px;
  background: var(--pink-bg, #fdf8f9);
  overflow-y: auto;
}
</style>
