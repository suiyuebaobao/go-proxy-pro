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
        <el-menu-item index="/">
          <el-icon><DataAnalysis /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/system-monitor">
          <el-icon><Monitor /></el-icon>
          <span>系统监控</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/accounts">
          <el-icon><Key /></el-icon>
          <span>账户管理</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/proxies">
          <el-icon><Position /></el-icon>
          <span>代理管理</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/models">
          <el-icon><Cpu /></el-icon>
          <span>模型管理</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/users">
          <el-icon><User /></el-icon>
          <span>用户管理</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/proxy-test">
          <el-icon><Connection /></el-icon>
          <span>代理测试</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/request-logs">
          <el-icon><Document /></el-icon>
          <span>请求日志</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/account-load">
          <el-icon><TrendCharts /></el-icon>
          <span>账户负载</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/cache">
          <el-icon><Box /></el-icon>
          <span>缓存管理</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/api-keys">
          <el-icon><Tickets /></el-icon>
          <span>API Key 管理</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/packages">
          <el-icon><ShoppingBag /></el-icon>
          <span>套餐管理</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/settings">
          <el-icon><Tools /></el-icon>
          <span>系统设置</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/error-messages">
          <el-icon><Warning /></el-icon>
          <span>错误消息</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/operation-logs">
          <el-icon><Notebook /></el-icon>
          <span>操作日志</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/system-logs">
          <el-icon><Files /></el-icon>
          <span>系统日志</span>
        </el-menu-item>

        <el-menu-item v-if="userStore.user?.role === 'admin'" index="/client-filter">
          <el-icon><Filter /></el-icon>
          <span>客户端过滤</span>
        </el-menu-item>

        <el-menu-item index="/usage">
          <el-icon><Histogram /></el-icon>
          <span>使用统计</span>
        </el-menu-item>

        <el-menu-item index="/profile">
          <el-icon><Setting /></el-icon>
          <span>个人设置</span>
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
          <keep-alive :max="5" :include="['Dashboard', 'Profile']">
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

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const isCollapse = ref(false)

function handleCommand(cmd) {
  if (cmd === 'logout') {
    userStore.logout()
    router.push('/login')
  } else if (cmd === 'profile') {
    router.push('/profile')
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
