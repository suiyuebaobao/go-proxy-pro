<!--
 * 文件作用：使用统计页面，展示用户API使用情况
 * 负责功能：
 *   - 总览统计卡片（请求数、Token、费用）
 *   - 每日使用统计表格
 *   - 按模型分类统计
 *   - 日期范围筛选
 * 重要程度：⭐⭐⭐ 一般（统计展示）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="usage-page">
    <div class="page-header">
      <h2>使用统计</h2>
    </div>

    <!-- 总览统计卡片 -->
    <el-row :gutter="16" class="stats-cards">
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="总请求数" :value="summary.totalRequests">
            <template #suffix>次</template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="总 Token 消耗" :value="summary.totalTokens" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="今日消费" :value="summary.todayCost" :precision="4">
            <template #prefix>$</template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <el-statistic title="总消费" :value="summary.totalCost" :precision="4">
            <template #prefix>$</template>
          </el-statistic>
        </el-card>
      </el-col>
    </el-row>

    <!-- 每日使用统计 -->
    <el-card class="usage-card">
      <template #header>
        <div class="card-header">
          <span>每日使用统计</span>
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            :shortcuts="dateShortcuts"
            @change="fetchDailyUsage"
          />
        </div>
      </template>
      <el-table :data="dailyUsage" v-loading="dailyLoading" stripe>
        <el-table-column prop="date" label="日期" width="120" />
        <el-table-column prop="requests" label="请求数" width="100" />
        <el-table-column prop="input_tokens" label="输入 Token" width="120" />
        <el-table-column prop="output_tokens" label="输出 Token" width="120" />
        <el-table-column prop="total_tokens" label="总 Token" width="120" />
        <el-table-column prop="cost" label="费用 ($)" width="120">
          <template #default="{ row }">{{ formatCost(row.cost) }}</template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 按模型使用统计 -->
    <el-card class="usage-card">
      <template #header>
        <span>按模型统计</span>
      </template>
      <el-table :data="modelUsage" v-loading="modelLoading" stripe>
        <el-table-column prop="model" label="模型" min-width="200" />
        <el-table-column prop="requests" label="请求数" width="100" />
        <el-table-column prop="input_tokens" label="输入 Token" width="120" />
        <el-table-column prop="output_tokens" label="输出 Token" width="120" />
        <el-table-column prop="total_tokens" label="总 Token" width="120" />
        <el-table-column prop="cost" label="费用 ($)" width="120">
          <template #default="{ row }">{{ formatCost(row.cost) }}</template>
        </el-table-column>
        <el-table-column label="占比" width="100">
          <template #default="{ row }">
            <el-progress
              :percentage="getModelPercentage(row.cost)"
              :show-text="false"
              :stroke-width="8"
            />
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import api from '@/api'

const summary = reactive({
  totalRequests: 0,
  totalTokens: 0,
  todayCost: 0,
  totalCost: 0
})

const dailyUsage = ref([])
const dailyLoading = ref(false)
const modelUsage = ref([])
const modelLoading = ref(false)

// 日期范围默认最近7天
const today = new Date()
const weekAgo = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000)
const dateRange = ref([weekAgo, today])

const dateShortcuts = [
  {
    text: '最近一周',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setTime(start.getTime() - 3600 * 1000 * 24 * 7)
      return [start, end]
    }
  },
  {
    text: '最近一月',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setTime(start.getTime() - 3600 * 1000 * 24 * 30)
      return [start, end]
    }
  },
  {
    text: '最近三月',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setTime(start.getTime() - 3600 * 1000 * 24 * 90)
      return [start, end]
    }
  }
]

// 计算模型费用占比
const totalModelCost = computed(() => {
  return modelUsage.value.reduce((sum, m) => sum + (m.cost || 0), 0)
})

function getModelPercentage(cost) {
  if (!totalModelCost.value || !cost) return 0
  return Math.round((cost / totalModelCost.value) * 100)
}

function formatCost(cost) {
  if (cost === undefined || cost === null) return '0.0000'
  return cost.toFixed(4)
}

function formatDate(date) {
  return date.toISOString().split('T')[0]
}

async function fetchSummary() {
  try {
    const res = await api.getMyUsageSummary()
    if (res.data) {
      summary.totalRequests = res.data.total_requests || 0
      summary.totalTokens = res.data.total_tokens || 0
      summary.todayCost = res.data.today_cost || 0
      summary.totalCost = res.data.total_cost || 0
    }
  } catch (e) {
    // handled
  }
}

async function fetchDailyUsage() {
  if (!dateRange.value || dateRange.value.length !== 2) return

  dailyLoading.value = true
  try {
    const res = await api.getMyDailyUsage({
      start_date: formatDate(dateRange.value[0]),
      end_date: formatDate(dateRange.value[1])
    })
    dailyUsage.value = res.data || []
  } catch (e) {
    // handled
  } finally {
    dailyLoading.value = false
  }
}

async function fetchModelUsage() {
  modelLoading.value = true
  try {
    const res = await api.getMyModelUsage()
    modelUsage.value = res.data || []
  } catch (e) {
    // handled
  } finally {
    modelLoading.value = false
  }
}

onMounted(() => {
  fetchSummary()
  fetchDailyUsage()
  fetchModelUsage()
})
</script>

<style scoped>
.usage-page {
  padding: 20px;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  color: #333;
}

.stats-cards {
  margin-bottom: 20px;
}

.stats-cards .el-card {
  text-align: center;
}

.usage-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
