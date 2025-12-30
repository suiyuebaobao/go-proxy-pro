<!--
 * 文件作用：用户仪表盘页面
 * 负责功能：
 *   - 显示用户使用概览
 *   - 套餐余额和状态
 *   - API Key 统计
 *   - 最近使用趋势
 * 重要程度：⭐⭐⭐⭐ 重要（用户首页）
-->
<template>
  <div class="user-dashboard">
    <div class="welcome-section">
      <h1>欢迎回来，{{ userStore.user?.username || '用户' }}</h1>
      <p class="subtitle">这里是您的使用概览</p>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stat-cards">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon cost">
            <el-icon :size="28"><Money /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">${{ formatNumber(stats.today_cost, 4) }}</div>
            <div class="stat-label">今日消费</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon tokens">
            <el-icon :size="28"><Coin /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ formatLargeNumber(stats.today_tokens) }}</div>
            <div class="stat-label">今日 Token</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon requests">
            <el-icon :size="28"><Connection /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ formatNumber(stats.today_requests) }}</div>
            <div class="stat-label">今日请求</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon keys">
            <el-icon :size="28"><Key /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.api_key_count }}</div>
            <div class="stat-label">API Key 数量</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 套餐信息 -->
    <el-row :gutter="20" class="section-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="hover" class="info-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><Box /></el-icon> 我的套餐</span>
              <el-button type="primary" link @click="$router.push('/user/packages')">
                查看全部 <el-icon><ArrowRight /></el-icon>
              </el-button>
            </div>
          </template>
          <div v-if="packages.length > 0" class="package-list">
            <div v-for="pkg in packages.slice(0, 3)" :key="pkg.id" class="package-item">
              <div class="package-info">
                <div class="package-name">
                  <el-tag :type="pkg.type === 'subscription' ? 'primary' : 'success'" size="small">
                    {{ pkg.type === 'subscription' ? '订阅' : '额度' }}
                  </el-tag>
                  <span>{{ pkg.name }}</span>
                </div>
                <div class="package-status">
                  <el-tag v-if="pkg.status === 'active'" type="success" size="small">有效</el-tag>
                  <el-tag v-else-if="pkg.status === 'expired'" type="info" size="small">已过期</el-tag>
                  <el-tag v-else type="warning" size="small">{{ pkg.status }}</el-tag>
                </div>
              </div>
              <div class="package-quota" v-if="pkg.type === 'quota'">
                <el-progress
                  :percentage="getQuotaPercentage(pkg)"
                  :color="getQuotaColor(pkg)"
                  :stroke-width="8"
                />
                <div class="quota-text">
                  已用 ${{ pkg.quota_used?.toFixed(2) || '0' }} / ${{ pkg.quota_total?.toFixed(2) || '0' }}
                </div>
              </div>
              <div class="package-quota" v-else>
                <div class="subscription-info">
                  <span>日额度: ${{ pkg.daily_used?.toFixed(2) || '0' }} / ${{ pkg.daily_quota?.toFixed(2) || '无限' }}</span>
                  <span class="expire-date">到期: {{ formatDate(pkg.expire_time) }}</span>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无套餐" :image-size="80" />
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <el-card shadow="hover" class="info-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><Key /></el-icon> 我的 API Key</span>
              <el-button type="primary" link @click="$router.push('/user/api-keys')">
                管理 <el-icon><ArrowRight /></el-icon>
              </el-button>
            </div>
          </template>
          <div v-if="apiKeys.length > 0" class="api-key-list">
            <div v-for="key in apiKeys.slice(0, 3)" :key="key.id" class="api-key-item">
              <div class="key-info">
                <div class="key-name">{{ key.name }}</div>
                <div class="key-prefix">{{ key.key_prefix }}...</div>
              </div>
              <div class="key-status">
                <el-tag :type="key.status === 'active' ? 'success' : 'danger'" size="small">
                  {{ key.status === 'active' ? '正常' : '禁用' }}
                </el-tag>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无 API Key" :image-size="80">
            <el-button type="primary" @click="$router.push('/user/api-keys')">创建 API Key</el-button>
          </el-empty>
        </el-card>
      </el-col>
    </el-row>

    <!-- 累计统计 -->
    <el-row :gutter="20" class="section-row">
      <el-col :span="24">
        <el-card shadow="hover" class="info-card">
          <template #header>
            <div class="card-header">
              <span><el-icon><TrendCharts /></el-icon> 累计使用统计</span>
            </div>
          </template>
          <el-row :gutter="40">
            <el-col :xs="12" :sm="6">
              <div class="total-stat">
                <div class="total-value">${{ formatNumber(stats.total_cost, 2) }}</div>
                <div class="total-label">累计消费</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6">
              <div class="total-stat">
                <div class="total-value">{{ formatLargeNumber(stats.total_tokens) }}</div>
                <div class="total-label">累计 Token</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6">
              <div class="total-stat">
                <div class="total-value">{{ formatNumber(stats.total_requests) }}</div>
                <div class="total-label">累计请求</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6">
              <div class="total-stat">
                <div class="total-value">{{ stats.model_count }}</div>
                <div class="total-label">使用模型数</div>
              </div>
            </el-col>
          </el-row>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import api from '@/api'
import {
  Money, Coin, Connection, Key, Box, ArrowRight, TrendCharts
} from '@element-plus/icons-vue'

const userStore = useUserStore()

const stats = reactive({
  today_cost: 0,
  today_tokens: 0,
  today_requests: 0,
  api_key_count: 0,
  total_cost: 0,
  total_tokens: 0,
  total_requests: 0,
  model_count: 0
})

const packages = ref([])
const apiKeys = ref([])

const formatNumber = (num, decimals = 0) => {
  if (num === undefined || num === null) return '0'
  if (decimals > 0) return num.toFixed(decimals)
  return num.toLocaleString()
}

const formatLargeNumber = (num) => {
  if (!num) return '0'
  if (num >= 1000000) return (num / 1000000).toFixed(2) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toLocaleString()
}

const formatDate = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleDateString('zh-CN')
}

const getQuotaPercentage = (pkg) => {
  if (!pkg.quota_total || pkg.quota_total === 0) return 0
  return Math.min(100, (pkg.quota_used / pkg.quota_total) * 100)
}

const getQuotaColor = (pkg) => {
  const percentage = getQuotaPercentage(pkg)
  if (percentage < 60) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
}

const fetchStats = async () => {
  try {
    // 获取使用统计
    const summaryRes = await api.getUserUsageSummary()
    if (summaryRes.data) {
      stats.today_cost = summaryRes.data.today?.total_cost || 0
      stats.today_tokens = summaryRes.data.today?.total_tokens || 0
      stats.today_requests = summaryRes.data.today?.request_count || 0
      stats.total_cost = summaryRes.data.total?.total_cost || 0
      stats.total_tokens = summaryRes.data.total?.total_tokens || 0
      stats.total_requests = summaryRes.data.total?.request_count || 0
      stats.model_count = summaryRes.data.model_count || 0
    }
  } catch (e) {
    console.error('Failed to fetch stats:', e)
  }
}

const fetchPackages = async () => {
  try {
    const res = await api.getMyActivePackages()
    packages.value = res.data || []
  } catch (e) {
    console.error('Failed to fetch packages:', e)
  }
}

const fetchApiKeys = async () => {
  try {
    const res = await api.getApiKeys()
    apiKeys.value = res.data || []
    stats.api_key_count = apiKeys.value.length
  } catch (e) {
    console.error('Failed to fetch API keys:', e)
  }
}

onMounted(() => {
  fetchStats()
  fetchPackages()
  fetchApiKeys()
})
</script>

<style scoped>
.user-dashboard {
  max-width: 1400px;
  margin: 0 auto;
}

.welcome-section {
  margin-bottom: 24px;
}

.welcome-section h1 {
  margin: 0;
  font-size: 28px;
  color: #303133;
}

.subtitle {
  margin: 8px 0 0;
  color: #909399;
}

.stat-cards {
  margin-bottom: 20px;
}

.stat-card {
  display: flex;
  align-items: center;
  padding: 20px;
}

.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  width: 100%;
  padding: 20px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  color: white;
}

.stat-icon.cost {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-icon.tokens {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.stat-icon.requests {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.stat-icon.keys {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 4px;
}

.section-row {
  margin-bottom: 20px;
}

.info-card :deep(.el-card__header) {
  padding: 16px 20px;
  border-bottom: 1px solid #ebeef5;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 16px;
  font-weight: 500;
}

.card-header span {
  display: flex;
  align-items: center;
  gap: 8px;
}

.package-list, .api-key-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.package-item, .api-key-item {
  padding: 12px;
  background: #f5f7fa;
  border-radius: 8px;
}

.package-info, .key-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.package-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.package-quota {
  margin-top: 8px;
}

.quota-text {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  text-align: right;
}

.subscription-info {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #606266;
}

.expire-date {
  color: #909399;
}

.key-name {
  font-weight: 500;
}

.key-prefix {
  font-size: 12px;
  color: #909399;
  font-family: monospace;
}

.key-info {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.api-key-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.total-stat {
  text-align: center;
  padding: 20px 0;
}

.total-value {
  font-size: 28px;
  font-weight: bold;
  color: #409eff;
}

.total-label {
  font-size: 14px;
  color: #909399;
  margin-top: 8px;
}

@media (max-width: 768px) {
  .stat-card {
    margin-bottom: 12px;
  }

  .total-stat {
    padding: 12px 0;
  }

  .total-value {
    font-size: 20px;
  }
}
</style>
