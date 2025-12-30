<!--
 * 文件作用：使用记录页面
 * 负责功能：
 *   - 显示用户的请求历史
 *   - 按日期/模型筛选
 *   - Token 和费用统计
 * 重要程度：⭐⭐⭐⭐ 重要（用户核心功能）
-->
<template>
  <div class="my-usage-records">
    <div class="page-header">
      <h2>使用记录</h2>
    </div>

    <!-- 筛选条件 -->
    <el-card shadow="hover" class="filter-card">
      <el-form :inline="true" :model="filters">
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="filters.dateRange"
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
          <el-select v-model="filters.model" placeholder="全部模型" clearable filterable style="width: 200px">
            <el-option v-for="m in models" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchRecords">
            <el-icon><Search /></el-icon> 查询
          </el-button>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 统计摘要 -->
    <el-row :gutter="16" class="summary-row">
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="summary-card">
          <div class="summary-value">{{ summary.request_count || 0 }}</div>
          <div class="summary-label">请求次数</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="summary-card">
          <div class="summary-value">{{ formatLargeNumber(summary.total_tokens) }}</div>
          <div class="summary-label">总 Token</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="summary-card">
          <div class="summary-value">${{ (summary.total_cost || 0).toFixed(4) }}</div>
          <div class="summary-label">总费用</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="summary-card">
          <div class="summary-value">{{ summary.model_count || 0 }}</div>
          <div class="summary-label">使用模型数</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 记录列表 -->
    <el-card shadow="hover">
      <el-table :data="records" v-loading="loading" stripe>
        <el-table-column label="时间" width="160">
          <template #default="{ row }">
            {{ formatTime(row.request_time || row.timestamp) }}
          </template>
        </el-table-column>
        <el-table-column prop="model" label="模型" min-width="180" show-overflow-tooltip />
        <el-table-column label="输入" width="90">
          <template #default="{ row }">
            {{ formatNumber(row.input_tokens) }}
          </template>
        </el-table-column>
        <el-table-column label="输出" width="90">
          <template #default="{ row }">
            {{ formatNumber(row.output_tokens) }}
          </template>
        </el-table-column>
        <el-table-column label="缓存创建" width="90">
          <template #default="{ row }">
            <span :class="{ 'cache-highlight': row.cache_creation_input_tokens > 0 }">
              {{ formatNumber(row.cache_creation_input_tokens) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="缓存读取" width="90">
          <template #default="{ row }">
            <span :class="{ 'cache-read-highlight': row.cache_read_input_tokens > 0 }">
              {{ formatNumber(row.cache_read_input_tokens) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="总Token" width="100">
          <template #default="{ row }">
            {{ formatNumber(row.total_tokens) }}
          </template>
        </el-table-column>
        <el-table-column label="费用" width="100">
          <template #default="{ row }">
            ${{ (row.total_cost || 0).toFixed(4) }}
          </template>
        </el-table-column>
        <el-table-column prop="request_ip" label="IP" width="120" show-overflow-tooltip />
      </el-table>

      <div class="pagination-wrap" v-if="pagination.total > 0">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @change="fetchRecords"
        />
      </div>

      <el-empty v-if="!loading && records.length === 0" description="暂无记录" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import { Search } from '@element-plus/icons-vue'

const loading = ref(false)
const records = ref([])
const models = ref([])

const filters = reactive({
  dateRange: null,
  model: ''
})

const summary = reactive({
  request_count: 0,
  total_tokens: 0,
  total_cost: 0,
  model_count: 0
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const dateShortcuts = [
  { text: '今天', value: () => { const d = new Date(); return [d, d] } },
  { text: '最近7天', value: () => { const e = new Date(); const s = new Date(); s.setDate(s.getDate() - 7); return [s, e] } },
  { text: '最近30天', value: () => { const e = new Date(); const s = new Date(); s.setDate(s.getDate() - 30); return [s, e] } },
  { text: '本月', value: () => { const e = new Date(); const s = new Date(e.getFullYear(), e.getMonth(), 1); return [s, e] } },
]

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

const formatNumber = (num) => {
  if (!num) return '0'
  return num.toLocaleString()
}

const formatLargeNumber = (num) => {
  if (!num) return '0'
  if (num >= 1000000) return (num / 1000000).toFixed(2) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toLocaleString()
}

const fetchRecords = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize
    }

    if (filters.dateRange && filters.dateRange.length === 2) {
      params.start_date = filters.dateRange[0]
      params.end_date = filters.dateRange[1]
    }

    if (filters.model) {
      params.model = filters.model
    }

    const res = await api.getUserUsageRecords(params)
    if (res.data) {
      records.value = res.data.items || []
      pagination.total = res.data.total || 0
    }
  } catch (e) {
    ElMessage.error('获取记录失败')
  } finally {
    loading.value = false
  }
}

const fetchSummary = async () => {
  try {
    const res = await api.getUserUsageSummary()
    if (res.data) {
      summary.total_cost = res.data.total?.total_cost || 0
      summary.total_tokens = res.data.total?.total_tokens || 0
      summary.request_count = res.data.total?.request_count || 0
      summary.model_count = res.data.model_count || 0
    }
  } catch (e) {
    console.error('Failed to fetch summary:', e)
  }
}

const fetchModels = async () => {
  try {
    const res = await api.getUserModelStats()
    if (res.data) {
      models.value = res.data.map(m => m.model).filter(Boolean)
    }
  } catch (e) {
    console.error('Failed to fetch models:', e)
  }
}

const resetFilters = () => {
  filters.dateRange = null
  filters.model = ''
  pagination.page = 1
  fetchRecords()
}

onMounted(() => {
  fetchRecords()
  fetchSummary()
  fetchModels()
})
</script>

<style scoped>
.my-usage-records {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  color: #303133;
}

.filter-card {
  margin-bottom: 20px;
}

.summary-row {
  margin-bottom: 20px;
}

.summary-card {
  text-align: center;
  padding: 10px 0;
}

.summary-card :deep(.el-card__body) {
  padding: 20px;
}

.summary-value {
  font-size: 24px;
  font-weight: bold;
  color: #409eff;
}

.summary-label {
  font-size: 14px;
  color: #909399;
  margin-top: 8px;
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
</style>
