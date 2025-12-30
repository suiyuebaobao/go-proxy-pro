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
    <el-tabs v-model="activeTab">
      <!-- 每日汇总 -->
      <el-tab-pane label="每日汇总" name="daily">
        <el-table :data="dailyStats" v-loading="loadingDaily" stripe>
          <el-table-column prop="date" label="日期" width="120" />
          <el-table-column prop="request_count" label="请求数" width="100" />
          <el-table-column prop="total_tokens" label="Token" width="120">
            <template #default="{ row }">
              {{ formatTokens(row.total_tokens) }}
            </template>
          </el-table-column>
          <el-table-column label="费用" width="120">
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
          <el-table-column prop="model" label="模型" min-width="200" />
          <el-table-column prop="request_count" label="请求数" width="100" />
          <el-table-column prop="total_tokens" label="Token" width="120">
            <template #default="{ row }">
              {{ formatTokens(row.total_tokens) }}
            </template>
          </el-table-column>
          <el-table-column label="费用" width="120">
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
          <el-form-item label="数据源">
            <el-select v-model="dataSource" style="width: 100px">
              <el-option label="自动" value="auto" />
              <el-option label="Redis" value="redis" />
              <el-option label="MySQL" value="mysql" />
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

        <el-alert v-if="selectedUserId && recordSource" :type="recordSource === 'redis' ? 'success' : 'warning'" :closable="false" style="margin-bottom: 16px">
          数据来源: {{ recordSource === 'redis' ? 'Redis（实时缓存）' : 'MySQL（历史记录）' }}
        </el-alert>

        <el-table v-if="selectedUserId" :data="records" v-loading="loadingRecords" stripe>
          <el-table-column prop="model" label="模型" min-width="180" show-overflow-tooltip />
          <el-table-column prop="request_ip" label="请求IP" width="130" show-overflow-tooltip />
          <el-table-column label="输入Token" width="100">
            <template #default="{ row }">
              {{ row.input_tokens || 0 }}
            </template>
          </el-table-column>
          <el-table-column label="输出Token" width="100">
            <template #default="{ row }">
              {{ row.output_tokens || 0 }}
            </template>
          </el-table-column>
          <el-table-column label="缓存Token" width="100">
            <template #default="{ row }">
              {{ (row.cache_creation_input_tokens || 0) + (row.cache_read_input_tokens || 0) }}
            </template>
          </el-table-column>
          <el-table-column prop="total_tokens" label="总Token" width="100" />
          <el-table-column label="费用" width="100">
            <template #default="{ row }">
              ${{ (row.total_cost || 0).toFixed(4) }}
            </template>
          </el-table-column>
          <el-table-column label="时间" width="170">
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
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import api from '@/api'

const activeTab = ref('daily')
const selectedUserId = ref(null)
const users = ref([])

// 筛选条件
const dateRange = ref(null)
const filterModel = ref('')
const dataSource = ref('auto')
const recordSource = ref('')

// 日期快捷选项
const dateShortcuts = [
  { text: '今天', value: () => { const d = new Date(); return [d, d] } },
  { text: '最近7天', value: () => { const e = new Date(); const s = new Date(); s.setDate(s.getDate() - 7); return [s, e] } },
  { text: '最近30天', value: () => { const e = new Date(); const s = new Date(); s.setDate(s.getDate() - 30); return [s, e] } },
  { text: '本月', value: () => { const e = new Date(); const s = new Date(e.getFullYear(), e.getMonth(), 1); return [s, e] } },
]

// 可选模型列表（从模型统计中提取）
const availableModels = computed(() => {
  return modelStats.value.map(m => m.model).filter(Boolean)
})

// 汇总数据
const summary = reactive({
  total_requests: 0,
  total_tokens: 0,
  total_cost: 0
})

// 每日汇总
const dailyStats = ref([])
const loadingDaily = ref(false)

// 模型统计
const modelStats = ref([])
const loadingModels = ref(false)

// 详细记录
const records = ref([])
const loadingRecords = ref(false)
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })

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

async function fetchUsers() {
  try {
    const res = await api.getUsers({ page: 1, page_size: 1000 })
    users.value = res.data?.items || []
  } catch (e) {
    console.error('Failed to fetch users:', e)
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
  recordSource.value = ''
}

async function fetchRecords() {
  if (!selectedUserId.value) return

  loadingRecords.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize,
      source: dataSource.value
    }

    // 日期范围筛选
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_date = dateRange.value[0]
      params.end_date = dateRange.value[1]
    }

    // 模型筛选
    if (filterModel.value) {
      params.model = filterModel.value
    }

    const res = await api.getUserUsageRecords(selectedUserId.value, params)
    const data = res.data || {}
    records.value = data.items || []
    pagination.total = data.total || records.value.length
    recordSource.value = data.source || ''
  } catch (e) {
    console.error('Failed to fetch records:', e)
  } finally {
    loadingRecords.value = false
  }
}

function resetFilters() {
  dateRange.value = null
  filterModel.value = ''
  dataSource.value = 'auto'
  pagination.page = 1
  if (selectedUserId.value) {
    fetchRecords()
  }
}

function refreshAll() {
  fetchAllSummary()
  if (selectedUserId.value) {
    fetchRecords()
  }
}

onMounted(() => {
  fetchUsers()
  fetchAllSummary()
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
  color: #333;
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
  color: #409eff;
}

.stat-item.cost .stat-value {
  color: #67c23a;
}

.stat-label {
  font-size: 14px;
  color: #909399;
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
</style>
