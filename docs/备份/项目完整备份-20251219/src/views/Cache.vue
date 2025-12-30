<template>
  <div class="cache-page">
    <div class="page-header">
      <h2>缓存管理</h2>
      <el-button @click="refreshAll">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <!-- 缓存配置 -->
    <el-card class="config-card">
      <template #header>
        <span>缓存配置</span>
      </template>
      <el-form :inline="true" :model="configForm" v-loading="loadingConfig">
        <el-form-item label="粘性会话TTL">
          <el-input-number v-model="configForm.session_ttl" :min="1" :max="1440" />
          <span class="unit">分钟</span>
        </el-form-item>
        <el-form-item label="会话续期阈值">
          <el-input-number v-model="configForm.session_renewal_ttl" :min="1" :max="60" />
          <span class="unit">分钟</span>
        </el-form-item>
        <el-form-item label="不可用标记TTL">
          <el-input-number v-model="configForm.unavailable_ttl" :min="1" :max="60" />
          <span class="unit">分钟</span>
        </el-form-item>
        <el-form-item label="并发计数TTL">
          <el-input-number v-model="configForm.concurrency_ttl" :min="1" :max="60" />
          <span class="unit">分钟</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveConfig">保存配置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 两个Tab：账号缓存 / 用户缓存 -->
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <!-- 账号缓存 -->
      <el-tab-pane label="账号缓存" name="accounts">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>账号缓存列表 ({{ accountList.length }})</span>
              <div class="header-actions">
                <el-input
                  v-model="accountSearch"
                  placeholder="搜索账号ID"
                  clearable
                  style="width: 150px"
                >
                  <template #prefix><el-icon><Search /></el-icon></template>
                </el-input>
                <el-popconfirm title="清除所有账号缓存?" @confirm="clearAllAccountCache">
                  <template #reference>
                    <el-button type="danger" size="small">清除全部</el-button>
                  </template>
                </el-popconfirm>
              </div>
            </div>
          </template>

          <el-table :data="filteredAccounts" v-loading="loadingAccounts" size="small">
            <el-table-column prop="account_id" label="账号ID" width="100" align="center" />
            <el-table-column label="并发" width="120" align="center">
              <template #default="{ row }">
                <span :class="getConcurrencyClass(row.concurrency, row.max_concurrency)">
                  {{ row.concurrency }}
                </span> / {{ row.max_concurrency }}
              </template>
            </el-table-column>
            <el-table-column label="会话数" width="100" align="center">
              <template #default="{ row }">
                <el-tag size="small">{{ row.session_count }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="使用此账号的用户" min-width="150">
              <template #default="{ row }">
                <el-tag v-for="uid in row.users" :key="uid" size="small" class="user-tag">
                  用户{{ uid }}
                </el-tag>
                <span v-if="!row.users?.length" class="no-data">-</span>
              </template>
            </el-table-column>
            <el-table-column label="会话详情" min-width="200">
              <template #default="{ row }">
                <el-popover placement="left" :width="400" trigger="click">
                  <template #reference>
                    <el-button link type="primary" size="small">查看会话</el-button>
                  </template>
                  <el-table :data="row.sessions" size="small" max-height="300">
                    <el-table-column prop="session_id" label="会话ID" show-overflow-tooltip />
                    <el-table-column prop="user_id" label="用户" width="60" />
                    <el-table-column label="绑定时间" width="120">
                      <template #default="{ row: s }">{{ formatTime(s.bound_at) }}</template>
                    </el-table-column>
                  </el-table>
                </el-popover>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="center">
              <template #default="{ row }">
                <el-popconfirm title="清除此账号缓存?" @confirm="clearAccountCache(row.account_id)">
                  <template #reference>
                    <el-button link type="danger" size="small">清除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>

          <el-empty v-if="!loadingAccounts && accountList.length === 0" description="暂无账号缓存" />
        </el-card>
      </el-tab-pane>

      <!-- 用户缓存 -->
      <el-tab-pane label="用户缓存" name="users">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>用户缓存列表 ({{ userList.length }})</span>
              <div class="header-actions">
                <el-input
                  v-model="userSearch"
                  placeholder="搜索用户ID"
                  clearable
                  style="width: 150px"
                >
                  <template #prefix><el-icon><Search /></el-icon></template>
                </el-input>
                <el-popconfirm title="清除所有用户缓存?" @confirm="clearAllUserCache">
                  <template #reference>
                    <el-button type="danger" size="small">清除全部</el-button>
                  </template>
                </el-popconfirm>
              </div>
            </div>
          </template>

          <el-table :data="filteredUsers" v-loading="loadingUsers" size="small">
            <el-table-column prop="user_id" label="用户ID" width="100" align="center" />
            <el-table-column label="并发" width="120" align="center">
              <template #default="{ row }">
                <span :class="getConcurrencyClass(row.concurrency, 10)">
                  {{ row.concurrency }}
                </span> / 10
              </template>
            </el-table-column>
            <el-table-column label="会话数" width="100" align="center">
              <template #default="{ row }">
                <el-tag size="small">{{ row.session_count }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="使用的账号" min-width="150">
              <template #default="{ row }">
                <el-tag v-for="aid in row.accounts" :key="aid" size="small" type="warning" class="account-tag">
                  账号{{ aid }}
                </el-tag>
                <span v-if="!row.accounts?.length" class="no-data">-</span>
              </template>
            </el-table-column>
            <el-table-column label="会话详情" min-width="200">
              <template #default="{ row }">
                <el-popover placement="left" :width="400" trigger="click">
                  <template #reference>
                    <el-button link type="primary" size="small">查看会话</el-button>
                  </template>
                  <el-table :data="row.sessions" size="small" max-height="300">
                    <el-table-column prop="session_id" label="会话ID" show-overflow-tooltip />
                    <el-table-column prop="account_id" label="账号" width="60" />
                    <el-table-column label="绑定时间" width="120">
                      <template #default="{ row: s }">{{ formatTime(s.bound_at) }}</template>
                    </el-table-column>
                  </el-table>
                </el-popover>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="center">
              <template #default="{ row }">
                <el-popconfirm title="清除此用户缓存?" @confirm="clearUserCache(row.user_id)">
                  <template #reference>
                    <el-button link type="danger" size="small">清除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>

          <el-empty v-if="!loadingUsers && userList.length === 0" description="暂无用户缓存" />
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import api from '@/api'

// 配置
const loadingConfig = ref(false)
const configForm = reactive({
  session_ttl: 60,
  session_renewal_ttl: 14,
  unavailable_ttl: 5,
  concurrency_ttl: 5
})

// Tab
const activeTab = ref('accounts')

// 账号缓存列表
const loadingAccounts = ref(false)
const accountList = ref([])
const accountSearch = ref('')

// 用户缓存列表
const loadingUsers = ref(false)
const userList = ref([])
const userSearch = ref('')

// 过滤
const filteredAccounts = computed(() => {
  if (!accountSearch.value) return accountList.value
  return accountList.value.filter(a => String(a.account_id).includes(accountSearch.value))
})

const filteredUsers = computed(() => {
  if (!userSearch.value) return userList.value
  return userList.value.filter(u => String(u.user_id).includes(userSearch.value))
})

function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit'
  })
}

function getConcurrencyClass(current, limit) {
  if (current >= limit) return 'danger'
  if (current >= limit * 0.8) return 'warning'
  return 'success'
}

// 加载配置
async function fetchConfig() {
  loadingConfig.value = true
  try {
    const res = await api.getCacheConfig()
    const data = res.data || {}
    configForm.session_ttl = data.session_ttl || 60
    configForm.session_renewal_ttl = data.session_renewal_ttl || 14
    configForm.unavailable_ttl = data.unavailable_ttl || 5
    configForm.concurrency_ttl = data.concurrency_ttl || 5
  } catch (e) {
    console.error('Failed to fetch config:', e)
  } finally {
    loadingConfig.value = false
  }
}

async function saveConfig() {
  loadingConfig.value = true
  try {
    await api.updateCacheConfig(configForm)
    ElMessage.success('配置已保存')
  } catch (e) {
    console.error('Failed to save config:', e)
  } finally {
    loadingConfig.value = false
  }
}

// 加载账号缓存
async function loadAccountCache() {
  loadingAccounts.value = true
  try {
    const res = await api.getCacheAccounts()
    accountList.value = res.data?.accounts || []
  } catch (e) {
    console.error('Failed to load account cache:', e)
  } finally {
    loadingAccounts.value = false
  }
}

// 加载用户缓存
async function loadUserCache() {
  loadingUsers.value = true
  try {
    const res = await api.getCacheUsers()
    userList.value = res.data?.users || []
  } catch (e) {
    console.error('Failed to load user cache:', e)
  } finally {
    loadingUsers.value = false
  }
}

// 清除账号缓存
async function clearAccountCache(accountId) {
  try {
    await api.clearAccountSessions(accountId)
    await api.resetAccountConcurrency(accountId)
    ElMessage.success('账号缓存已清除')
    loadAccountCache()
  } catch (e) {
    console.error('Failed to clear account cache:', e)
  }
}

// 清除所有账号缓存
async function clearAllAccountCache() {
  try {
    await api.clearCache('sessions')
    await api.clearCache('concurrency')
    ElMessage.success('所有账号缓存已清除')
    accountList.value = []
  } catch (e) {
    console.error('Failed to clear all account cache:', e)
  }
}

// 清除用户缓存
async function clearUserCache(userId) {
  try {
    await api.clearUserCache(userId)
    await api.resetUserConcurrency(userId)
    ElMessage.success('用户缓存已清除')
    loadUserCache()
  } catch (e) {
    console.error('Failed to clear user cache:', e)
  }
}

// 清除所有用户缓存
async function clearAllUserCache() {
  try {
    await api.clearCache('sessions')
    await api.clearCache('concurrency')
    ElMessage.success('所有用户缓存已清除')
    userList.value = []
  } catch (e) {
    console.error('Failed to clear all user cache:', e)
  }
}

function handleTabChange(tab) {
  if (tab === 'accounts' && accountList.value.length === 0) {
    loadAccountCache()
  } else if (tab === 'users' && userList.value.length === 0) {
    loadUserCache()
  }
}

function refreshAll() {
  fetchConfig()
  if (activeTab.value === 'accounts') {
    loadAccountCache()
  } else {
    loadUserCache()
  }
}

onMounted(() => {
  fetchConfig()
  loadAccountCache()
})
</script>

<style scoped>
.cache-page {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
}

.config-card {
  margin-bottom: 20px;
}

.config-card .el-form-item {
  margin-bottom: 0;
}

.unit {
  margin-left: 8px;
  color: #909399;
  font-size: 13px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.user-tag,
.account-tag {
  margin-right: 4px;
  margin-bottom: 2px;
}

.no-data {
  color: #c0c4cc;
}

.success {
  color: #67c23a;
  font-weight: 600;
}

.warning {
  color: #e6a23c;
  font-weight: 600;
}

.danger {
  color: #f56c6c;
  font-weight: 600;
}
</style>
