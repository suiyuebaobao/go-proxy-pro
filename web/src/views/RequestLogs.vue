<!--
 * 文件作用：请求日志页面，展示API请求统计
 * 负责功能：
 *   - 每日请求汇总
 *   - 按模型统计
 *   - 用户详细记录查询
 *   - Token和费用统计
 * 重要程度：⭐⭐⭐ 一般（日志查看）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="logs-page">
    <div class="page-header">
      <h2>请求日志</h2>
      <el-button @click="refreshAll">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <!-- 统计摘要 -->
    <el-row :gutter="20" class="summary-cards">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-item">
            <div class="stat-value">{{ summary.total_requests || 0 }}</div>
            <div class="stat-label">总请求数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-item">
            <div class="stat-value">{{ formatTokens(summary.total_tokens) }}</div>
            <div class="stat-label">总Token</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-item cost">
            <div class="stat-value">${{ (summary.total_cost || 0).toFixed(4) }}</div>
            <div class="stat-label">总费用</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-item">
            <div class="stat-value">{{ modelStats.length }}</div>
            <div class="stat-label">模型数</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Tabs -->
    <el-tabs v-model="activeTab" @tab-change="onTabChange">
      <!-- 请求日志列表 (新增) -->
      <el-tab-pane label="请求日志" name="loglist">
        <el-form :inline="true" style="margin-bottom: 16px; flex-wrap: wrap;">
          <el-form-item label="平台">
            <el-select v-model="logFilter.platform" clearable placeholder="全部" style="width: 130px">
              <el-option v-for="p in platformOptions" :key="p" :label="p" :value="p" />
            </el-select>
          </el-form-item>
          <el-form-item label="模型">
            <el-select v-model="logFilter.model" clearable placeholder="全部" filterable style="width: 200px">
              <el-option v-for="m in availableModels" :key="m" :label="m" :value="m" />
            </el-select>
          </el-form-item>
          <el-form-item label="用户">
            <el-select v-model="logFilter.user_id" clearable placeholder="全部用户" filterable style="width: 150px">
              <el-option v-for="u in users" :key="u.id" :label="u.username" :value="u.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="logFilter.success" clearable placeholder="全部" style="width: 100px">
              <el-option label="成功" value="true" />
              <el-option label="失败" value="false" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态码">
            <el-input v-model="logFilter.status_code" clearable placeholder="如 200" style="width: 90px" />
          </el-form-item>
          <el-form-item label="耗时(ms)">
            <el-input v-model="logFilter.min_duration" placeholder="最小" style="width: 80px" />
            <span style="margin: 0 4px; color: #999">-</span>
            <el-input v-model="logFilter.max_duration" placeholder="最大" style="width: 80px" />
          </el-form-item>
          <el-form-item label="时间">
            <el-date-picker v-model="logFilter.dateRange" type="datetimerange" range-separator="至"
              start-placeholder="开始" end-placeholder="结束" style="width: 340px" :shortcuts="dateShortcuts" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="fetchLogList">查询</el-button>
            <el-button @click="resetLogFilter">重置</el-button>
          </el-form-item>
        </el-form>

        <el-table :data="logList" v-loading="loadingLogList" stripe @row-click="showLogDetail" style="cursor: pointer">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="platform" label="平台" width="90" />
          <el-table-column prop="model" label="模型" min-width="180" show-overflow-tooltip />
          <el-table-column prop="request_ip" label="IP" width="130" show-overflow-tooltip />
          <el-table-column label="状态码" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status_code < 300 ? 'success' : row.status_code < 500 ? 'warning' : 'danger'" size="small">
                {{ row.status_code }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Token" width="90">
            <template #default="{ row }">{{ formatTokens(row.total_tokens) }}</template>
          </el-table-column>
          <el-table-column label="费用" width="100">
            <template #default="{ row }">${{ (row.total_cost || 0).toFixed(4) }}</template>
          </el-table-column>
          <el-table-column label="耗时" width="90">
            <template #default="{ row }">{{ row.duration }}ms</template>
          </el-table-column>
          <el-table-column label="成功" width="70" align="center">
            <template #default="{ row }">
              <el-tag :type="row.success ? 'success' : 'danger'" size="small">{{ row.success ? '是' : '否' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="时间" min-width="160">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click.stop="showLogDetail(row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-wrap" v-if="logPagination.total > 0">
          <el-pagination v-model:current-page="logPagination.page" v-model:page-size="logPagination.pageSize"
            :total="logPagination.total" :page-sizes="[20, 50, 100]" layout="total, sizes, prev, pager, next"
            @change="fetchLogList" />
        </div>
      </el-tab-pane>

      <!-- 每日汇总 -->
      <el-tab-pane label="每日汇总" name="daily">
        <el-table :data="dailyStats" v-loading="loadingDaily" stripe>
          <el-table-column prop="date" label="日期" min-width="140" />
          <el-table-column prop="request_count" label="请求数" min-width="120" />
          <el-table-column prop="total_tokens" label="Token" min-width="140">
            <template #default="{ row }">
              {{ formatTokens(row.total_tokens) }}
            </template>
          </el-table-column>
          <el-table-column label="费用" min-width="140">
            <template #default="{ row }">
              ${{ row.total_cost?.toFixed(4) || '0' }}
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="dailyStats.length === 0 && !loadingDaily" description="暂无数据" />
      </el-tab-pane>

      <!-- 模型统计 -->
      <el-tab-pane label="模型统计" name="models">
        <el-table :data="modelStats" v-loading="loadingModels" stripe>
          <el-table-column prop="model" label="模型" min-width="240" />
          <el-table-column prop="request_count" label="请求数" min-width="120" />
          <el-table-column prop="total_tokens" label="Token" min-width="140">
            <template #default="{ row }">
              {{ formatTokens(row.total_tokens) }}
            </template>
          </el-table-column>
          <el-table-column label="费用" min-width="140">
            <template #default="{ row }">
              ${{ row.total_cost?.toFixed(4) || '0' }}
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="modelStats.length === 0 && !loadingModels" description="暂无数据" />
      </el-tab-pane>

      <!-- 用户详细记录 -->
      <el-tab-pane label="用户详细记录" name="records">
        <el-form :inline="true" style="margin-bottom: 16px">
          <el-form-item label="选择用户">
            <el-select v-model="selectedUserId" clearable placeholder="选择用户" @change="handleUserChange" filterable style="width: 160px">
              <el-option v-for="user in users" :key="user.id" :label="user.username" :value="user.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="时间范围">
            <el-date-picker
              v-model="dateRange"
              type="daterange"
              range-separator="至"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              value-format="YYYY-MM-DD"
              style="width: 240px"
              :shortcuts="dateShortcuts"
            />
          </el-form-item>
          <el-form-item label="模型">
            <el-select v-model="filterModel" clearable placeholder="全部模型" filterable style="width: 200px">
              <el-option v-for="m in availableModels" :key="m" :label="m" :value="m" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="fetchRecords" :disabled="!selectedUserId">查询</el-button>
            <el-button @click="resetFilters">重置</el-button>
          </el-form-item>
        </el-form>

        <el-alert v-if="!selectedUserId" type="info" :closable="false" style="margin-bottom: 16px">
          请选择用户查看详细记录
        </el-alert>

        <el-table v-if="selectedUserId" :data="records" v-loading="loadingRecords" stripe>
          <el-table-column prop="model" label="模型" min-width="180" show-overflow-tooltip />
          <el-table-column prop="request_ip" label="请求IP" min-width="120" show-overflow-tooltip />
          <el-table-column label="输入" min-width="80">
            <template #default="{ row }">
              {{ formatTokens(row.input_tokens) }}
            </template>
          </el-table-column>
          <el-table-column label="输出" min-width="80">
            <template #default="{ row }">
              {{ formatTokens(row.output_tokens) }}
            </template>
          </el-table-column>
          <el-table-column label="缓存创建" min-width="90">
            <template #default="{ row }">
              <span :class="{ 'cache-highlight': row.cache_creation_input_tokens > 0 }">
                {{ formatTokens(row.cache_creation_input_tokens) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="缓存读取" min-width="90">
            <template #default="{ row }">
              <span :class="{ 'cache-read-highlight': row.cache_read_input_tokens > 0 }">
                {{ formatTokens(row.cache_read_input_tokens) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="总Token" min-width="90">
            <template #default="{ row }">
              {{ formatTokens(row.total_tokens) }}
            </template>
          </el-table-column>
          <el-table-column label="费用" min-width="100">
            <template #default="{ row }">
              ${{ (row.total_cost || 0).toFixed(4) }}
            </template>
          </el-table-column>
          <el-table-column label="时间" min-width="160">
            <template #default="{ row }">
              {{ formatTime(row.timestamp || row.request_time) }}
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-wrap" v-if="selectedUserId && pagination.total > 0">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :page-sizes="[20, 50, 100]"
            layout="total, sizes, prev, pager, next"
            @change="fetchRecords"
          />
        </div>
        <el-empty v-if="selectedUserId && records.length === 0 && !loadingRecords" description="暂无记录" />
      </el-tab-pane>
    </el-tabs>

    <!-- 日志详情弹窗 -->
    <el-dialog v-model="detailVisible" title="请求日志详情" width="900px" top="5vh" destroy-on-close>
      <div v-loading="loadingDetail" v-if="detailData">
        <el-descriptions :column="3" border size="small" style="margin-bottom: 16px">
          <el-descriptions-item label="ID">{{ detailData.id }}</el-descriptions-item>
          <el-descriptions-item label="平台">{{ detailData.platform }}</el-descriptions-item>
          <el-descriptions-item label="模型">{{ detailData.model }}</el-descriptions-item>
          <el-descriptions-item label="请求IP">{{ detailData.request_ip }}</el-descriptions-item>
          <el-descriptions-item label="状态码">
            <el-tag :type="detailData.status_code < 300 ? 'success' : 'danger'" size="small">{{ detailData.status_code }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="耗时">{{ detailData.duration }}ms</el-descriptions-item>
          <el-descriptions-item label="输入Token">{{ detailData.input_tokens }}</el-descriptions-item>
          <el-descriptions-item label="输出Token">{{ detailData.output_tokens }}</el-descriptions-item>
          <el-descriptions-item label="总费用">${{ (detailData.total_cost || 0).toFixed(6) }}</el-descriptions-item>
          <el-descriptions-item label="User-Agent" :span="3">
            <span style="word-break: break-all; font-size: 12px">{{ detailData.user_agent || '-' }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="时间" :span="3">{{ formatTime(detailData.created_at) }}</el-descriptions-item>
          <el-descriptions-item v-if="detailData.error" label="错误信息" :span="3">
            <span style="color: var(--el-color-danger)">{{ detailData.error }}</span>
          </el-descriptions-item>
        </el-descriptions>

        <el-tabs v-model="detailTab">
          <el-tab-pane label="请求头" name="req_headers">
            <pre class="json-block">{{ formatJSON(detailData.request_headers) }}</pre>
          </el-tab-pane>
          <el-tab-pane label="请求体" name="req_body">
            <pre class="json-block">{{ formatJSON(detailData.request_body) }}</pre>
          </el-tab-pane>
          <el-tab-pane label="响应头" name="res_headers">
            <pre class="json-block">{{ formatJSON(detailData.response_headers) }}</pre>
          </el-tab-pane>
          <el-tab-pane label="响应体" name="res_body">
            <pre class="json-block">{{ formatJSON(detailData.response_body) }}</pre>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import api from '@/api'

const activeTab = ref('loglist')
const selectedUserId = ref(null)
const users = ref([])

// 筛选条件
const dateRange = ref(null)
const filterModel = ref('')

const dateShortcuts = [
  { text: '今天', value: () => { const d = new Date(); d.setHours(0,0,0,0); return [d, new Date()] } },
  { text: '最近7天', value: () => { const e = new Date(); const s = new Date(); s.setDate(s.getDate() - 7); s.setHours(0,0,0,0); return [s, e] } },
  { text: '最近30天', value: () => { const e = new Date(); const s = new Date(); s.setDate(s.getDate() - 30); s.setHours(0,0,0,0); return [s, e] } },
  { text: '本月', value: () => { const e = new Date(); const s = new Date(e.getFullYear(), e.getMonth(), 1); return [s, e] } },
]

const platformOptions = [
  'claude', 'openai', 'gemini',
  'deepseek', 'qwen', 'glm', 'moonshot', 'doubao',
  'baichuan', 'yi', 'minimax', 'stepfun', 'spark',
  'xai', 'mistral', 'cohere', 'siliconflow', 'custom'
]

// 可选模型列表
const availableModels = computed(() => {
  return modelStats.value.map(m => m.model).filter(Boolean)
})

// 汇总数据
const summary = reactive({
  total_requests: 0,
  total_tokens: 0,
  total_cost: 0
})

const dailyStats = ref([])
const loadingDaily = ref(false)
const modelStats = ref([])
const loadingModels = ref(false)
const records = ref([])
const loadingRecords = ref(false)
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })

// 日志列表 (新增)
const logList = ref([])
const loadingLogList = ref(false)
const logPagination = reactive({ page: 1, pageSize: 20, total: 0 })
const logFilter = reactive({
  platform: '',
  model: '',
  user_id: '',
  success: '',
  status_code: '',
  min_duration: '',
  max_duration: '',
  dateRange: null
})

// 详情弹窗
const detailVisible = ref(false)
const detailData = ref(null)
const loadingDetail = ref(false)
const detailTab = ref('req_body')

function formatTokens(tokens) {
  if (!tokens) return '0'
  if (tokens >= 1000000) return (tokens / 1000000).toFixed(1) + 'M'
  if (tokens >= 1000) return (tokens / 1000).toFixed(1) + 'K'
  return tokens
}

function formatTime(time) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}

function formatJSON(str) {
  if (!str) return '(空)'
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

async function fetchUsers() {
  try {
    const res = await api.getUsers({ page: 1, page_size: 1000 })
    users.value = res.data?.items || []
  } catch (e) {
    console.error('Failed to fetch users:', e)
  }
}

async function fetchLogList() {
  loadingLogList.value = true
  try {
    const params = { page: logPagination.page, page_size: logPagination.pageSize }
    if (logFilter.platform) params.platform = logFilter.platform
    if (logFilter.model) params.model = logFilter.model
    if (logFilter.user_id) params.user_id = logFilter.user_id
    if (logFilter.success) params.success = logFilter.success
    if (logFilter.status_code) params.status_code = logFilter.status_code
    if (logFilter.min_duration) params.min_duration = logFilter.min_duration
    if (logFilter.max_duration) params.max_duration = logFilter.max_duration
    if (logFilter.dateRange && logFilter.dateRange.length === 2) {
      params.start_time = logFilter.dateRange[0].toISOString()
      params.end_time = logFilter.dateRange[1].toISOString()
    }
    const res = await api.getRequestLogs(params)
    const data = res.data || {}
    logList.value = data.items || []
    logPagination.total = data.total || 0
  } catch (e) {
    console.error('Failed to fetch logs:', e)
  } finally {
    loadingLogList.value = false
  }
}

function resetLogFilter() {
  Object.assign(logFilter, { platform: '', model: '', user_id: '', success: '', status_code: '', min_duration: '', max_duration: '', dateRange: null })
  logPagination.page = 1
  fetchLogList()
}

async function showLogDetail(row) {
  detailVisible.value = true
  loadingDetail.value = true
  detailTab.value = 'req_body'
  try {
    const res = await api.getRequestLogDetail(row.id)
    detailData.value = res.data
  } catch (e) {
    console.error('Failed to fetch log detail:', e)
    detailData.value = row
  } finally {
    loadingDetail.value = false
  }
}

async function fetchAllSummary() {
  loadingDaily.value = true
  loadingModels.value = true

  try {
    const res = await api.getAllUsageSummary({})
    const data = res.data || {}

    summary.total_requests = data.total_requests || 0
    summary.total_tokens = data.total_tokens || 0
    summary.total_cost = data.total_cost || 0
    dailyStats.value = data.daily || []
    modelStats.value = data.models || []
  } catch (e) {
    console.error('Failed to fetch summary:', e)
  } finally {
    loadingDaily.value = false
    loadingModels.value = false
  }
}

function handleUserChange() {
  pagination.page = 1
  records.value = []
}

async function fetchRecords() {
  if (!selectedUserId.value) return
  loadingRecords.value = true
  try {
    const params = { page: pagination.page, page_size: pagination.pageSize }
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }
    if (filterModel.value) params.model = filterModel.value
    const res = await api.getUserUsageRecords(selectedUserId.value, params)
    const data = res.data || {}
    records.value = data.items || []
    pagination.total = data.total || records.value.length
  } catch (e) {
    console.error('Failed to fetch records:', e)
  } finally {
    loadingRecords.value = false
  }
}

function resetFilters() {
  dateRange.value = null
  filterModel.value = ''
  pagination.page = 1
  if (selectedUserId.value) fetchRecords()
}

function onTabChange(tab) {
  if (tab === 'loglist' && logList.value.length === 0) fetchLogList()
}

function refreshAll() {
  fetchAllSummary()
  if (activeTab.value === 'loglist') fetchLogList()
  if (selectedUserId.value) fetchRecords()
}

onMounted(() => {
  fetchUsers()
  fetchAllSummary()
  fetchLogList()
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  color: var(--pink-text);
  margin: 0;
}

.summary-cards {
  margin-bottom: 20px;
}

.stat-item {
  text-align: center;
  padding: 10px 0;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: var(--pink-accent, #c97b8b);
}

.stat-item.cost .stat-value {
  color: #67c23a;
}

.stat-label {
  font-size: 14px;
  color: #6b6573;
  margin-top: 8px;
}

.token-info {
  font-family: monospace;
  font-size: 12px;
}

.pagination-wrap {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.cache-highlight {
  color: #e6a23c;
  font-weight: bold;
}

.cache-read-highlight {
  color: #67c23a;
  font-weight: bold;
}

.json-block {
  background: var(--pink-bg, #fdf8f9);
  border: 1px solid var(--pink-border, #f0e0e6);
  border-radius: 6px;
  padding: 12px 16px;
  max-height: 400px;
  overflow: auto;
  font-family: 'Fira Code', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--pink-text, #4a3347);
}
</style>
