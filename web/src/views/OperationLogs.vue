<!--
 * 文件作用：操作日志页面，记录系统所有操作行为
 * 负责功能：
 *   - 操作日志列表和筛选
 *   - 模块/操作类型分类
 *   - 日志详情查看
 *   - 历史日志清理
 * 重要程度：⭐⭐ 辅助（审计日志）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="operation-logs-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h2>操作日志</h2>
        <p class="header-desc">记录系统所有操作行为，便于审计和追溯</p>
      </div>
      <div class="header-actions">
        <el-button @click="loadLogs">
          <i class="fa-solid fa-sync-alt" :class="{ 'fa-spin': loading }"></i>
          刷新
        </el-button>
        <el-popconfirm
          title="确定要清理90天前的日志吗？"
          confirm-button-text="确定"
          cancel-button-text="取消"
          @confirm="handleCleanup"
        >
          <template #reference>
            <el-button type="warning">
              <i class="fa-solid fa-broom"></i>
              清理旧日志
            </el-button>
          </template>
        </el-popconfirm>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-select v-model="filters.module" clearable placeholder="模块" @change="handleFilterChange" style="width: 120px">
        <el-option label="认证" value="auth" />
        <el-option label="用户" value="user" />
        <el-option label="账户" value="account" />
        <el-option label="API Key" value="apikey" />
        <el-option label="模型" value="model" />
        <el-option label="配置" value="config" />
        <el-option label="缓存" value="cache" />
        <el-option label="代理" value="proxy" />
        <el-option label="套餐" value="package" />
        <el-option label="分组" value="group" />
      </el-select>
      <el-select v-model="filters.action" clearable placeholder="操作" @change="handleFilterChange" style="width: 120px">
        <el-option label="登录" value="login" />
        <el-option label="创建" value="create" />
        <el-option label="更新" value="update" />
        <el-option label="删除" value="delete" />
        <el-option label="清除" value="clear" />
      </el-select>
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        format="YYYY-MM-DD"
        value-format="YYYY-MM-DD"
        style="width: 260px"
        @change="handleDateChange"
      />
      <el-input
        v-model="filters.search"
        placeholder="搜索用户/描述/路径..."
        clearable
        style="width: 200px"
        @input="handleSearch"
      >
        <template #prefix>
          <i class="fa-solid fa-search"></i>
        </template>
      </el-input>
    </div>

    <!-- 日志列表 -->
    <el-card class="logs-table-card" shadow="never">
      <el-table :data="logs" v-loading="loading" stripe>
        <el-table-column label="#" width="60" align="center">
          <template #default="{ $index }">
            <span class="row-index">{{ (pagination.page - 1) * pagination.pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>

        <el-table-column label="时间" width="170">
          <template #default="{ row }">
            <span class="log-time">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="用户" width="120">
          <template #default="{ row }">
            <div class="user-cell">
              <span class="username">{{ row.username || '-' }}</span>
              <span class="user-ip">{{ row.ip }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="模块" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="getModuleType(row.module)">
              {{ getModuleLabel(row.module) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="getActionType(row.action)">
              {{ getActionLabel(row.action) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="描述" min-width="250">
          <template #default="{ row }">
            <div class="description-cell">
              <span class="description">{{ row.description }}</span>
              <span class="target" v-if="row.target_name">
                <i class="fa-solid fa-arrow-right"></i>
                {{ row.target_name }}
              </span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="路径" width="200">
          <template #default="{ row }">
            <div class="path-cell">
              <el-tag size="small" :type="getMethodType(row.method)">{{ row.method }}</el-tag>
              <span class="path">{{ row.path }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="结果" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.response_code === 0 ? 'success' : 'danger'">
              {{ row.response_code === 0 ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="耗时" width="80" align="right">
          <template #default="{ row }">
            <span class="duration">{{ row.duration }}ms</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="showDetail(row)">
              <i class="fa-solid fa-eye"></i> 详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="table-footer">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @change="loadLogs"
        />
      </div>
    </el-card>

    <!-- 详情弹窗 -->
    <el-dialog v-model="showDetailDialog" title="操作日志详情" width="600px">
      <el-descriptions :column="2" border v-if="currentLog">
        <el-descriptions-item label="ID">{{ currentLog.id }}</el-descriptions-item>
        <el-descriptions-item label="时间">{{ formatTime(currentLog.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="用户">{{ currentLog.username }} (ID: {{ currentLog.user_id }})</el-descriptions-item>
        <el-descriptions-item label="IP">{{ currentLog.ip }}</el-descriptions-item>
        <el-descriptions-item label="模块">{{ getModuleLabel(currentLog.module) }}</el-descriptions-item>
        <el-descriptions-item label="操作">{{ getActionLabel(currentLog.action) }}</el-descriptions-item>
        <el-descriptions-item label="目标ID">{{ currentLog.target_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="目标名称">{{ currentLog.target_name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">{{ currentLog.description }}</el-descriptions-item>
        <el-descriptions-item label="请求方法">{{ currentLog.method }}</el-descriptions-item>
        <el-descriptions-item label="请求路径">{{ currentLog.path }}</el-descriptions-item>
        <el-descriptions-item label="响应码">{{ currentLog.response_code }}</el-descriptions-item>
        <el-descriptions-item label="响应消息">{{ currentLog.response_msg || '-' }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ currentLog.duration }}ms</el-descriptions-item>
        <el-descriptions-item label="User-Agent" :span="2">
          <div class="ua-text">{{ currentLog.user_agent || '-' }}</div>
        </el-descriptions-item>
        <el-descriptions-item label="请求体" :span="2" v-if="currentLog.request_body">
          <pre class="request-body">{{ formatJSON(currentLog.request_body) }}</pre>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ensureFontAwesomeLoaded } from '@/utils/fontawesome'
ensureFontAwesomeLoaded()

import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const logs = ref([])
const showDetailDialog = ref(false)
const currentLog = ref(null)
const dateRange = ref(null)

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const filters = reactive({
  module: '',
  action: '',
  search: '',
  start_time: '',
  end_time: ''
})

// 模块标签
const moduleLabels = {
  auth: '认证',
  user: '用户',
  account: '账户',
  apikey: 'API Key',
  model: '模型',
  config: '配置',
  cache: '缓存',
  proxy: '代理',
  package: '套餐',
  group: '分组',
  system: '系统'
}

// 操作标签
const actionLabels = {
  login: '登录',
  create: '创建',
  update: '更新',
  delete: '删除',
  clear: '清除'
}

function getModuleLabel(module) {
  return moduleLabels[module] || module
}

function getModuleType(module) {
  const types = {
    auth: 'warning',
    user: 'primary',
    account: 'success',
    apikey: 'info',
    model: '',
    config: 'warning',
    cache: 'info',
    proxy: 'danger',
    package: 'success',
    group: ''
  }
  return types[module] || ''
}

function getActionLabel(action) {
  return actionLabels[action] || action
}

function getActionType(action) {
  const types = {
    login: 'warning',
    create: 'success',
    update: 'primary',
    delete: 'danger',
    clear: 'warning'
  }
  return types[action] || ''
}

function getMethodType(method) {
  const types = {
    POST: 'success',
    PUT: 'warning',
    DELETE: 'danger',
    GET: 'info'
  }
  return types[method] || ''
}

function formatTime(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function formatJSON(str) {
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

let searchTimer = null
function handleSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    pagination.page = 1  // 搜索时重置到第一页
    loadLogs()
  }, 300)
}

onUnmounted(() => {
  if (searchTimer) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
})

function handleDateChange(val) {
  if (val) {
    filters.start_time = val[0]
    filters.end_time = val[1]
  } else {
    filters.start_time = ''
    filters.end_time = ''
  }
  pagination.page = 1  // 重置到第一页
  loadLogs()
}

// 筛选条件改变时重置分页
function handleFilterChange() {
  pagination.page = 1
  loadLogs()
}

async function loadLogs() {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize,
      ...filters
    }
    // 移除空值
    Object.keys(params).forEach(key => {
      if (params[key] === '' || params[key] === null) {
        delete params[key]
      }
    })

    const res = await api.getOperationLogs(params)
    logs.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (e) {
    console.error('Failed to load operation logs:', e)
  } finally {
    loading.value = false
  }
}

function showDetail(row) {
  currentLog.value = row
  showDetailDialog.value = true
}

async function handleCleanup() {
  try {
    const res = await api.cleanupOperationLogs(90)
    ElMessage.success(`清理完成，删除了 ${res.data.deleted} 条记录`)
    loadLogs()
  } catch (e) {
    ElMessage.error('清理失败')
  }
}

onMounted(() => {
  loadLogs()
})
</script>

<style scoped>
.operation-logs-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left h2 {
  margin: 0 0 4px;
  font-size: 24px;
  font-weight: 700;
  color: #1f2937;
}

.header-desc {
  margin: 0;
  font-size: 14px;
  color: #6b7280;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.logs-table-card {
  border-radius: 12px;
}

.row-index {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
  color: #9ca3af;
}

.log-time {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
  color: #6b7280;
}

.user-cell {
  display: flex;
  flex-direction: column;
}

.username {
  font-weight: 600;
  color: #1f2937;
}

.user-ip {
  font-size: 12px;
  color: #9ca3af;
  font-family: 'SF Mono', Monaco, monospace;
}

.description-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.description {
  color: #1f2937;
}

.target {
  font-size: 12px;
  color: #6b7280;
}

.target i {
  font-size: 10px;
  margin-right: 4px;
}

.path-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.path {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 12px;
  color: #6b7280;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.duration {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
  color: #6b7280;
}

.table-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.ua-text {
  font-size: 12px;
  word-break: break-all;
  color: #6b7280;
}

.request-body {
  background: #f3f4f6;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  font-family: 'SF Mono', Monaco, monospace;
  overflow-x: auto;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
