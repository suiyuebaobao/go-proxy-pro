<!--
 * 文件作用：我的套餐页面
 * 负责功能：
 *   - 显示用户的所有套餐
 *   - 套餐详情和使用情况
 *   - 额度/订阅进度展示
 * 重要程度：⭐⭐⭐⭐ 重要（用户核心功能）
-->
<template>
  <div class="my-packages">
    <div class="page-header">
      <h2>我的套餐</h2>
      <el-button @click="fetchPackages" :loading="loading">
        <el-icon><Refresh /></el-icon> 刷新
      </el-button>
    </div>

    <!-- 套餐卡片列表 -->
    <div v-loading="loading" class="packages-grid">
      <el-card
        v-for="pkg in packages"
        :key="pkg.id"
        shadow="hover"
        :class="['package-card', { 'expired': pkg.status !== 'active' }]"
      >
        <template #header>
          <div class="card-header">
            <div class="package-title">
              <el-tag :type="pkg.type === 'subscription' ? 'primary' : 'success'" size="small">
                {{ pkg.type === 'subscription' ? '订阅套餐' : '额度套餐' }}
              </el-tag>
              <span class="package-name">{{ pkg.name }}</span>
            </div>
            <el-tag :type="getStatusType(pkg.status)" size="small">
              {{ getStatusText(pkg.status) }}
            </el-tag>
          </div>
        </template>

        <!-- 额度套餐 -->
        <div v-if="pkg.type === 'quota'" class="package-content">
          <div class="quota-section">
            <div class="quota-header">
              <span>额度使用</span>
              <span class="quota-percentage">{{ getQuotaPercentage(pkg).toFixed(1) }}%</span>
            </div>
            <el-progress
              :percentage="getQuotaPercentage(pkg)"
              :color="getProgressColor(getQuotaPercentage(pkg))"
              :stroke-width="12"
              :show-text="false"
            />
            <div class="quota-detail">
              <span>已用: ${{ (pkg.quota_used || 0).toFixed(2) }}</span>
              <span>总额: ${{ (pkg.quota_total || 0).toFixed(2) }}</span>
            </div>
            <div class="quota-remaining">
              剩余额度: <strong>${{ ((pkg.quota_total || 0) - (pkg.quota_used || 0)).toFixed(2) }}</strong>
            </div>
          </div>
        </div>

        <!-- 订阅套餐 -->
        <div v-else class="package-content">
          <div class="subscription-section">
            <!-- 日额度 -->
            <div class="limit-item" v-if="pkg.daily_quota > 0">
              <div class="limit-header">
                <span>日额度</span>
                <span>{{ (pkg.daily_used || 0).toFixed(2) }} / {{ pkg.daily_quota.toFixed(2) }}</span>
              </div>
              <el-progress
                :percentage="getUsagePercentage(pkg.daily_used, pkg.daily_quota)"
                :color="getProgressColor(getUsagePercentage(pkg.daily_used, pkg.daily_quota))"
                :stroke-width="8"
                :show-text="false"
              />
            </div>

            <!-- 周额度 -->
            <div class="limit-item" v-if="pkg.weekly_quota > 0">
              <div class="limit-header">
                <span>周额度</span>
                <span>{{ (pkg.weekly_used || 0).toFixed(2) }} / {{ pkg.weekly_quota.toFixed(2) }}</span>
              </div>
              <el-progress
                :percentage="getUsagePercentage(pkg.weekly_used, pkg.weekly_quota)"
                :color="getProgressColor(getUsagePercentage(pkg.weekly_used, pkg.weekly_quota))"
                :stroke-width="8"
                :show-text="false"
              />
            </div>

            <!-- 月额度 -->
            <div class="limit-item" v-if="pkg.monthly_quota > 0">
              <div class="limit-header">
                <span>月额度</span>
                <span>{{ (pkg.monthly_used || 0).toFixed(2) }} / {{ pkg.monthly_quota.toFixed(2) }}</span>
              </div>
              <el-progress
                :percentage="getUsagePercentage(pkg.monthly_used, pkg.monthly_quota)"
                :color="getProgressColor(getUsagePercentage(pkg.monthly_used, pkg.monthly_quota))"
                :stroke-width="8"
                :show-text="false"
              />
            </div>

            <div v-if="!pkg.daily_quota && !pkg.weekly_quota && !pkg.monthly_quota" class="no-limit">
              <el-icon><Check /></el-icon> 无额度限制
            </div>
          </div>
        </div>

        <!-- 套餐信息 -->
        <div class="package-footer">
          <div class="info-row">
            <el-icon><Calendar /></el-icon>
            <span>开始: {{ formatDate(pkg.start_time) }}</span>
          </div>
          <div class="info-row">
            <el-icon><Timer /></el-icon>
            <span>到期: {{ formatDate(pkg.expire_time) }}</span>
          </div>
          <div class="info-row" v-if="pkg.status === 'active'">
            <el-icon><Clock /></el-icon>
            <span>剩余: {{ getDaysRemaining(pkg.expire_time) }} 天</span>
          </div>
        </div>
      </el-card>
    </div>

    <el-empty v-if="!loading && packages.length === 0" description="暂无套餐" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import { Refresh, Calendar, Timer, Clock, Check } from '@element-plus/icons-vue'

const loading = ref(false)
const packages = ref([])

const formatDate = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleDateString('zh-CN')
}

const getStatusType = (status) => {
  switch (status) {
    case 'active': return 'success'
    case 'expired': return 'info'
    case 'exhausted': return 'warning'
    default: return 'info'
  }
}

const getStatusText = (status) => {
  switch (status) {
    case 'active': return '有效'
    case 'expired': return '已过期'
    case 'exhausted': return '已耗尽'
    default: return status
  }
}

const getQuotaPercentage = (pkg) => {
  if (!pkg.quota_total || pkg.quota_total === 0) return 0
  return Math.min(100, (pkg.quota_used / pkg.quota_total) * 100)
}

const getUsagePercentage = (used, total) => {
  if (!total || total === 0) return 0
  return Math.min(100, ((used || 0) / total) * 100)
}

const getProgressColor = (percentage) => {
  if (percentage < 60) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
}

const getDaysRemaining = (expireTime) => {
  if (!expireTime) return 0
  const now = new Date()
  const expire = new Date(expireTime)
  const diff = expire - now
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
}

const fetchPackages = async () => {
  loading.value = true
  try {
    const res = await api.getMyPackages()
    packages.value = res.data || []
  } catch (e) {
    ElMessage.error('获取套餐列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchPackages()
})
</script>

<style scoped>
.my-packages {
  max-width: 1200px;
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

.packages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 20px;
}

.package-card {
  border-radius: 12px;
  transition: transform 0.3s, box-shadow 0.3s;
}

.package-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.package-card.expired {
  opacity: 0.7;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.package-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.package-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.package-content {
  padding: 10px 0;
}

.quota-section, .subscription-section {
  padding: 10px 0;
}

.quota-header, .limit-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 14px;
  color: #606266;
}

.quota-percentage {
  font-weight: bold;
  color: #409eff;
}

.quota-detail {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  font-size: 13px;
  color: #909399;
}

.quota-remaining {
  margin-top: 16px;
  padding: 12px;
  background: #f0f9eb;
  border-radius: 8px;
  text-align: center;
  color: #67c23a;
  font-size: 16px;
}

.quota-remaining strong {
  font-size: 20px;
}

.limit-item {
  margin-bottom: 16px;
}

.limit-item:last-child {
  margin-bottom: 0;
}

.no-limit {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px;
  background: #f0f9eb;
  border-radius: 8px;
  color: #67c23a;
  font-size: 16px;
}

.package-footer {
  border-top: 1px solid #ebeef5;
  padding-top: 16px;
  margin-top: 16px;
}

.info-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
}

.info-row:last-child {
  margin-bottom: 0;
}

@media (max-width: 768px) {
  .packages-grid {
    grid-template-columns: 1fr;
  }
}
</style>
