<template>
  <div class="load-page">
    <div class="page-header">
      <h2>账户负载</h2>
      <el-button @click="fetchLoadStats">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <!-- 时间范围选择 -->
    <el-card class="filter-card">
      <el-form :inline="true">
        <el-form-item label="时间范围">
          <el-select v-model="timeRange" @change="handleTimeRangeChange">
            <el-option label="最近1小时" value="1h" />
            <el-option label="最近24小时" value="24h" />
            <el-option label="最近7天" value="7d" />
            <el-option label="最近30天" value="30d" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="timeRange === 'custom'">
          <el-date-picker
            v-model="dateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            @change="handleDateChange"
          />
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 负载列表 -->
    <el-card>
      <el-table :data="loadStats" v-loading="loading" stripe>
        <el-table-column prop="account_id" label="ID" width="70" />
        <el-table-column prop="account_name" label="账户名称" min-width="150" />
        <el-table-column prop="platform" label="平台" width="100">
          <template #default="{ row }">
            <el-tag :type="getPlatformType(row.platform)" size="small">
              {{ row.platform }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="请求数" width="120">
          <template #default="{ row }">
            <div class="request-count">
              <span class="total">{{ row.request_count }}</span>
              <span class="detail">
                (<span class="success">{{ row.success_count }}</span> /
                <span class="error">{{ row.error_count }}</span>)
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="成功率" width="120">
          <template #default="{ row }">
            <el-progress
              :percentage="getSuccessRate(row)"
              :status="getProgressStatus(row)"
              :stroke-width="10"
            />
          </template>
        </el-table-column>
        <el-table-column prop="total_tokens" label="Token使用" width="120">
          <template #default="{ row }">
            {{ formatTokens(row.total_tokens) }}
          </template>
        </el-table-column>
        <el-table-column prop="avg_duration" label="平均耗时" width="100">
          <template #default="{ row }">
            {{ row.avg_duration }}ms
          </template>
        </el-table-column>
        <el-table-column label="最后使用" width="170">
          <template #default="{ row }">
            {{ formatTime(row.last_used_at) }}
          </template>
        </el-table-column>
        <el-table-column label="负载" width="150">
          <template #default="{ row }">
            <el-progress
              :percentage="getLoadPercentage(row)"
              :color="getLoadColor(row)"
            />
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import api from '@/api'

const loading = ref(false)
const loadStats = ref([])
const timeRange = ref('24h')
const dateRange = ref(null)
const startTime = ref('')
const endTime = ref('')

const maxRequests = computed(() => {
  return Math.max(...loadStats.value.map(s => s.request_count), 1)
})

function getPlatformType(platform) {
  const map = { claude: 'warning', openai: 'success', gemini: 'primary' }
  return map[platform] || 'info'
}

function formatTokens(tokens) {
  if (tokens >= 1000000) return (tokens / 1000000).toFixed(1) + 'M'
  if (tokens >= 1000) return (tokens / 1000).toFixed(1) + 'K'
  return tokens
}

function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

function getSuccessRate(row) {
  if (row.request_count === 0) return 100
  return Math.round((row.success_count / row.request_count) * 100)
}

function getProgressStatus(row) {
  const rate = getSuccessRate(row)
  if (rate >= 95) return 'success'
  if (rate >= 80) return 'warning'
  return 'exception'
}

function getLoadPercentage(row) {
  return Math.round((row.request_count / maxRequests.value) * 100)
}

function getLoadColor(row) {
  const percentage = getLoadPercentage(row)
  if (percentage > 80) return '#f56c6c'
  if (percentage > 50) return '#e6a23c'
  return '#67c23a'
}

function handleTimeRangeChange() {
  const now = new Date()
  switch (timeRange.value) {
    case '1h':
      startTime.value = new Date(now - 60 * 60 * 1000).toISOString()
      endTime.value = now.toISOString()
      fetchLoadStats()
      break
    case '24h':
      startTime.value = new Date(now - 24 * 60 * 60 * 1000).toISOString()
      endTime.value = now.toISOString()
      fetchLoadStats()
      break
    case '7d':
      startTime.value = new Date(now - 7 * 24 * 60 * 60 * 1000).toISOString()
      endTime.value = now.toISOString()
      fetchLoadStats()
      break
    case '30d':
      startTime.value = new Date(now - 30 * 24 * 60 * 60 * 1000).toISOString()
      endTime.value = now.toISOString()
      fetchLoadStats()
      break
    case 'custom':
      break
  }
}

function handleDateChange(val) {
  if (val && val.length === 2) {
    startTime.value = val[0].toISOString()
    endTime.value = val[1].toISOString()
    fetchLoadStats()
  }
}

async function fetchLoadStats() {
  loading.value = true
  try {
    const params = {}
    if (startTime.value) params.start_time = startTime.value
    if (endTime.value) params.end_time = endTime.value
    const res = await api.getAccountLoadStats(params)
    loadStats.value = res.data || []
  } catch (e) {
    console.error('Failed to fetch load stats:', e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  handleTimeRangeChange()
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

.filter-card {
  margin-bottom: 20px;
}

.filter-card :deep(.el-card__body) {
  padding-bottom: 2px;
}

.request-count .total {
  font-weight: bold;
  font-size: 16px;
}

.request-count .detail {
  font-size: 12px;
  color: #909399;
}

.request-count .success {
  color: #67c23a;
}

.request-count .error {
  color: #f56c6c;
}
</style>
