<!--
 * 文件作用：缓存管理页面，管理会话和并发缓存
 * 负责功能：
 *   - 缓存配置管理（TTL设置）
 *   - 会话列表（每个会话一行，显示到TTL结束）
 *   - 会话绑定详情查看
 *   - 不可用账号管理
 * 重要程度：⭐⭐⭐ 一般（缓存管理）
 * 依赖模块：element-plus, api
-->
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

    <!-- 两个Tab：会话列表 / 不可用账号 -->
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <!-- 会话列表 -->
      <el-tab-pane label="会话列表" name="sessions">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>活跃会话 ({{ sessionTotal }})</span>
              <div class="header-actions">
                <el-input
                  v-model="sessionSearch"
                  placeholder="搜索会话/账号/用户"
                  clearable
                  style="width: 200px"
                  @input="filterSessions"
                >
                  <template #prefix><el-icon><Search /></el-icon></template>
                </el-input>
                <el-popconfirm title="清除所有会话缓存?" @confirm="clearAllSessions">
                  <template #reference>
                    <el-button type="danger" size="small">清除全部</el-button>
                  </template>
                </el-popconfirm>
              </div>
            </div>
          </template>

          <el-table :data="filteredSessions" v-loading="loadingSessions" size="small" stripe>
            <el-table-column label="会话ID" min-width="200" show-overflow-tooltip>
              <template #default="{ row }">
                <span class="session-id">{{ parseSessionId(row.session_id) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="账号" width="150">
              <template #default="{ row }">
                <el-tag size="small" type="warning">
                  {{ getAccountName(row.account_id) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="用户" width="120">
              <template #default="{ row }">
                <span v-if="row.user_id">{{ getUserName(row.user_id) }}</span>
                <span v-else class="no-data">-</span>
              </template>
            </el-table-column>
            <el-table-column label="API Key" width="100">
              <template #default="{ row }">
                <el-tooltip
                  v-if="row.api_key_id"
                  :content="getAPIKeyTooltip(row.api_key_id)"
                  placement="top"
                  :show-after="200"
                >
                  <code class="api-key-prefix">{{ getAPIKeyLabel(row.api_key_id) }}</code>
                </el-tooltip>
                <span v-else class="no-data">-</span>
              </template>
            </el-table-column>
            <el-table-column label="平台" width="100">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ row.platform || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="模型" width="150" show-overflow-tooltip>
              <template #default="{ row }">
                {{ row.model || '-' }}
              </template>
            </el-table-column>
            <el-table-column label="绑定时间" width="140">
              <template #default="{ row }">
                {{ formatTime(row.bound_at) }}
              </template>
            </el-table-column>
            <el-table-column label="最后使用" width="140">
              <template #default="{ row }">
                {{ formatTime(row.last_used_at) }}
              </template>
            </el-table-column>
            <el-table-column label="剩余时间" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="getTTLType(row.remaining_ttl)" size="small">
                  {{ formatTTL(row.remaining_ttl) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="客户端IP" width="130" show-overflow-tooltip>
              <template #default="{ row }">
                {{ row.client_ip || '-' }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80" align="center" fixed="right">
              <template #default="{ row }">
                <el-popconfirm title="移除此会话?" @confirm="removeSession(row.session_id)">
                  <template #reference>
                    <el-button link type="danger" size="small">移除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>

          <div class="pagination-wrap" v-if="sessionTotal > 0">
            <el-pagination
              v-model:current-page="sessionPage"
              v-model:page-size="sessionPageSize"
              :total="sessionTotal"
              :page-sizes="[20, 50, 100]"
              layout="total, sizes, prev, pager, next"
              @change="loadSessions"
            />
          </div>

          <el-empty v-if="!loadingSessions && sessions.length === 0" description="暂无活跃会话" />
        </el-card>
      </el-tab-pane>

      <!-- 不可用账号 -->
      <el-tab-pane label="不可用账号" name="unavailable">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>临时不可用账号 ({{ unavailableList.length }})</span>
              <el-button type="primary" size="small" @click="loadUnavailable">
                <el-icon><Refresh /></el-icon> 刷新
              </el-button>
            </div>
          </template>

          <el-table :data="unavailableList" v-loading="loadingUnavailable" size="small">
            <el-table-column label="账号" min-width="150">
              <template #default="{ row }">
                {{ getAccountName(row.account_id) }}
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="200" show-overflow-tooltip />
            <el-table-column label="剩余时间" width="120" align="center">
              <template #default="{ row }">
                <el-tag type="danger" size="small">
                  {{ formatTTL(row.remaining_ttl) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="center">
              <template #default="{ row }">
                <el-popconfirm title="清除不可用标记?" @confirm="clearUnavailable(row.account_id)">
                  <template #reference>
                    <el-button link type="primary" size="small">恢复</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>

          <el-empty v-if="!loadingUnavailable && unavailableList.length === 0" description="暂无不可用账号" />
        </el-card>
      </el-tab-pane>

      <!-- 并发统计 -->
      <el-tab-pane label="并发统计" name="concurrency">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-card>
              <template #header>
                <div class="card-header">
                  <span>账号并发 ({{ accountConcurrencyList.length }})</span>
                </div>
              </template>
              <el-table :data="accountConcurrencyList" size="small" max-height="400">
                <el-table-column label="账号" min-width="120">
                  <template #default="{ row }">
                    {{ getAccountName(row.account_id) }}
                  </template>
                </el-table-column>
                <el-table-column label="并发" width="120" align="center">
                  <template #default="{ row }">
                    <span :class="getConcurrencyClass(row.concurrency, row.max_concurrency)">
                      {{ row.concurrency }}
                    </span> / {{ row.max_concurrency }}
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="80" align="center">
                  <template #default="{ row }">
                    <el-button link type="danger" size="small" @click="resetAccountConcurrency(row.account_id)">
                      重置
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card>
              <template #header>
                <div class="card-header">
                  <span>用户并发 ({{ userConcurrencyList.length }})</span>
                </div>
              </template>
              <el-table :data="userConcurrencyList" size="small" max-height="400">
                <el-table-column label="用户" min-width="120">
                  <template #default="{ row }">
                    {{ getUserName(row.user_id) }}
                  </template>
                </el-table-column>
                <el-table-column label="并发" width="120" align="center">
                  <template #default="{ row }">
                    <span :class="getConcurrencyClass(row.concurrency, row.max_concurrency)">
                      {{ row.concurrency }}
                    </span> / {{ row.max_concurrency }}
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="80" align="center">
                  <template #default="{ row }">
                    <el-button link type="danger" size="small" @click="resetUserConcurrency(row.user_id)">
                      重置
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
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
const activeTab = ref('sessions')

// 会话列表
const loadingSessions = ref(false)
const sessions = ref([])
const sessionTotal = ref(0)
const sessionPage = ref(1)
const sessionPageSize = ref(20)
const sessionSearch = ref('')

// 账号和用户名称缓存
const accountNames = ref({})
const userNames = ref({})
const apiKeyLabels = ref({})
const apiKeyFullMap = ref({})

// 不可用账号
const loadingUnavailable = ref(false)
const unavailableList = ref([])

// 并发统计（从会话列表中计算）
const accountConcurrencyList = computed(() => {
  const map = new Map()
  sessions.value.forEach(s => {
    if (!map.has(s.account_id)) {
      map.set(s.account_id, { account_id: s.account_id, concurrency: 0, max_concurrency: 5 })
    }
    map.get(s.account_id).concurrency++
  })
  return Array.from(map.values())
})

const userConcurrencyList = computed(() => {
  const map = new Map()
  sessions.value.forEach(s => {
    if (s.user_id && !map.has(s.user_id)) {
      map.set(s.user_id, { user_id: s.user_id, concurrency: 0, max_concurrency: 10 })
    }
    if (s.user_id) {
      map.get(s.user_id).concurrency++
    }
  })
  return Array.from(map.values())
})

// 过滤后的会话
const filteredSessions = computed(() => {
  if (!sessionSearch.value) return sessions.value
  const search = sessionSearch.value.toLowerCase()
  return sessions.value.filter(s => {
    return s.session_id?.toLowerCase().includes(search) ||
           getAccountName(s.account_id).toLowerCase().includes(search) ||
           getUserName(s.user_id).toLowerCase().includes(search) ||
           getAPIKeyLabel(s.api_key_id).toLowerCase().includes(search)
  })
})

// 自动刷新定时器
let refreshTimer = null

function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit'
  })
}

function getConcurrencyClass(current, limit) {
  if (current >= limit) return 'danger'
  if (current >= limit * 0.8) return 'warning'
  return 'success'
}

// 解析会话ID，提取可读部分
function parseSessionId(sessionId) {
  if (!sessionId) return '-'
  // 格式: apikey:{apiKeyID}:{x-session-id} 或 apikey:{apiKeyID}
  if (sessionId.startsWith('apikey:')) {
    const parts = sessionId.split(':')
    if (parts.length >= 3) {
      return parts.slice(2).join(':')
    }
    return `Key#${parts[1]}`
  }
  return sessionId
}

// 格式化剩余时间
function formatTTL(seconds) {
  if (!seconds || seconds <= 0) return '已过期'
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分${seconds % 60}秒`
  return `${Math.floor(seconds / 3600)}时${Math.floor((seconds % 3600) / 60)}分`
}

// 获取TTL显示类型
function getTTLType(seconds) {
  if (!seconds || seconds <= 0) return 'danger'
  if (seconds < 300) return 'warning'  // 5分钟内
  return 'success'
}

// 获取账号名称
function getAccountName(accountId) {
  if (!accountId) return '-'
  return accountNames.value[accountId] || `#${accountId}`
}

// 获取用户名称
function getUserName(userId) {
  if (!userId) return '-'
  return userNames.value[userId] || `#${userId}`
}

// 获取 API Key 显示文本
function getAPIKeyLabel(apiKeyId) {
  if (!apiKeyId) return '-'
  return apiKeyLabels.value[apiKeyId] || 'sk-...'
}

function getAPIKeyTooltip(apiKeyId) {
  if (!apiKeyId) return ''
  return apiKeyFullMap.value[apiKeyId] || apiKeyLabels.value[apiKeyId] || ''
}

// 过滤会话
function filterSessions() {
  // 搜索在 computed 中实现
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

// 加载会话列表
async function loadSessions() {
  loadingSessions.value = true
  try {
    const offset = (sessionPage.value - 1) * sessionPageSize.value
    const res = await api.getCacheSessions({ offset, limit: sessionPageSize.value })
    sessions.value = res.data?.sessions || []
    sessionTotal.value = res.data?.total || 0

    // 收集需要查询名称的账号和用户ID
    const accountIds = [...new Set(sessions.value.map(s => s.account_id).filter(Boolean))]
    const userIds = [...new Set(sessions.value.map(s => s.user_id).filter(Boolean))]
    const apiKeyIds = [...new Set(sessions.value.map(s => s.api_key_id).filter(Boolean))]

    // 批量获取账号名称（如果API支持）
    await loadAccountNames(accountIds)
    await loadUserNames(userIds)
    await loadAPIKeyLabels(apiKeyIds)
  } catch (e) {
    console.error('Failed to load sessions:', e)
  } finally {
    loadingSessions.value = false
  }
}

// 加载账号名称
async function loadAccountNames(accountIds) {
  if (!accountIds.length) return
  try {
    const missing = accountIds.filter(id => id && !accountNames.value[id])
    if (!missing.length) return

    // 账户列表接口返回 { items, total, page }，旧代码误用 list 导致一直回退显示 #id
    const res = await api.getAccounts({ page: 1, page_size: 1000 })
    const items = res.data?.items || res.data?.list || []
    items.forEach(acc => {
      if (acc?.id) {
        accountNames.value[acc.id] = acc.name || `账号${acc.id}`
      }
    })

    // 如果分页未覆盖到全部账号，补充按ID查询（仅查当前页缺失的）
    for (const id of missing) {
      if (!accountNames.value[id]) {
        try {
          const r = await api.getAccount(id)
          if (r.data?.id) {
            accountNames.value[r.data.id] = r.data.name || `账号${r.data.id}`
          }
        } catch {
          // ignore
        }
      }
    }
  } catch (e) {
    console.error('Failed to load account names:', e)
  }
}

// 加载用户名称
async function loadUserNames(userIds) {
  if (!userIds.length) return
  try {
    const missing = userIds.filter(id => id && !userNames.value[id])
    if (!missing.length) return

    // 后端提供不分页接口 /admin/users/all，避免分页漏掉导致名称缺失
    const res = await api.getAllUsers()
    const items = res.data?.items || res.data?.list || []
    items.forEach(user => {
      if (user?.id) {
        userNames.value[user.id] = user.username || `用户${user.id}`
      }
    })
  } catch (e) {
    console.error('Failed to load user names:', e)
  }
}

// 加载 API Key 显示文本（名称 + 前缀）
async function loadAPIKeyLabels(apiKeyIds) {
  if (!apiKeyIds.length) return
  const missing = apiKeyIds.filter(id => id && !apiKeyLabels.value[id])
  if (!missing.length) return

  try {
    // 按需批量查询，避免拉全量 API Key 列表导致页面卡顿
    const res = await api.adminLookupAPIKeys(missing)
    const items = res.data?.items || []
    items.forEach(k => {
      if (!k?.id) return
      if (k.key_prefix) {
        apiKeyLabels.value[k.id] = k.key_prefix
      }
      if (k.key_full) {
        apiKeyFullMap.value[k.id] = k.key_full
      }
    })
  } catch (e) {
    console.error('Failed to load API key labels:', e)
  }
}

// 移除会话
async function removeSession(sessionId) {
  try {
    await api.removeCacheSession(sessionId)
    ElMessage.success('会话已移除')
    loadSessions()
  } catch (e) {
    ElMessage.error('移除失败')
  }
}

// 清除所有会话
async function clearAllSessions() {
  try {
    await api.clearCache('sessions')
    ElMessage.success('所有会话已清除')
    sessions.value = []
    sessionTotal.value = 0
  } catch (e) {
    ElMessage.error('清除失败')
  }
}

// 加载不可用账号
async function loadUnavailable() {
  loadingUnavailable.value = true
  try {
    const res = await api.getUnavailableAccounts()
    unavailableList.value = res.data?.accounts || []
  } catch (e) {
    console.error('Failed to load unavailable accounts:', e)
  } finally {
    loadingUnavailable.value = false
  }
}

// 清除不可用标记
async function clearUnavailable(accountId) {
  try {
    await api.clearAccountUnavailable(accountId)
    ElMessage.success('已恢复账号')
    loadUnavailable()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

// 重置账号并发
async function resetAccountConcurrency(accountId) {
  try {
    await api.resetAccountConcurrency(accountId)
    ElMessage.success('已重置')
    loadSessions()
  } catch (e) {
    ElMessage.error('重置失败')
  }
}

// 重置用户并发
async function resetUserConcurrency(userId) {
  try {
    await api.resetUserConcurrency(userId)
    ElMessage.success('已重置')
    loadSessions()
  } catch (e) {
    ElMessage.error('重置失败')
  }
}

function handleTabChange(tab) {
  if (tab === 'sessions') {
    loadSessions()
  } else if (tab === 'unavailable') {
    loadUnavailable()
  } else if (tab === 'concurrency') {
    loadSessions() // 并发统计从会话列表计算
  }
}

function refreshAll() {
  fetchConfig()
  loadSessions()
  if (activeTab.value === 'unavailable') {
    loadUnavailable()
  }
}

// 自动刷新（每30秒）
function startAutoRefresh() {
  refreshTimer = setInterval(() => {
    if (activeTab.value === 'sessions') {
      loadSessions()
    }
  }, 30000)
}

function stopAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

onMounted(() => {
  fetchConfig()
  loadSessions()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
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

.no-data {
  color: #c0c4cc;
}

.api-key-prefix {
  background: #f3f4f6;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  color: #4b5563;
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

.session-id {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: #606266;
}

.pagination-wrap {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
