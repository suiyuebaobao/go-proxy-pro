<!--
 * 文件作用：系统监控页面，展示系统运行状态
 * 负责功能：
 *   - 今日使用概览（消费、Token、请求、用户）
 *   - Token使用详情
 *   - 账号状态统计
 *   - 系统资源监控（CPU、内存、磁盘）
 *   - 缓存和MySQL状态
 * 重要程度：⭐⭐⭐⭐ 重要（运维监控）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="system-monitor">
    <div class="page-header">
      <h2>系统监控</h2>
      <el-button type="primary" :loading="loading" @click="fetchData">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <!-- 今日使用概览 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon cost-icon">
            <el-icon><Money /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">${{ formatNumber(data.today_usage?.total_cost || 0, 4) }}</div>
            <div class="stat-label">今日消费</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon token-icon">
            <el-icon><Coin /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ formatNumber(data.today_usage?.total_tokens || 0) }}</div>
            <div class="stat-label">今日 Token</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon request-icon">
            <el-icon><Connection /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ formatNumber(data.today_usage?.request_count || 0) }}</div>
            <div class="stat-label">今日请求</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon user-icon">
            <el-icon><User /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ data.users?.active || 0 }}</div>
            <div class="stat-label">今日活跃用户</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 总使用概览 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card total">
          <div class="stat-icon total-cost-icon">
            <el-icon><Money /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">${{ formatNumber(data.total_usage?.total_cost || 0, 2) }}</div>
            <div class="stat-label">总消费</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card total">
          <div class="stat-icon total-token-icon">
            <el-icon><Coin /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ formatLargeNumber(data.total_usage?.total_tokens || 0) }}</div>
            <div class="stat-label">总 Token</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card total">
          <div class="stat-icon total-request-icon">
            <el-icon><Connection /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ formatNumber(data.total_usage?.request_count || 0) }}</div>
            <div class="stat-label">总请求</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card total">
          <div class="stat-icon total-cache-icon">
            <el-icon><Files /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ formatLargeNumber(data.total_usage?.cache_creation_tokens || 0) }}</div>
            <div class="stat-label">总缓存创建</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Token 详情 -->
    <el-row :gutter="16" class="section-row">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>Token 使用详情</span>
            </div>
          </template>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="输入 Token">
              {{ formatNumber(data.today_usage?.input_tokens || 0) }}
            </el-descriptions-item>
            <el-descriptions-item label="输出 Token">
              {{ formatNumber(data.today_usage?.output_tokens || 0) }}
            </el-descriptions-item>
            <el-descriptions-item label="缓存创建 Token">
              {{ formatNumber(data.today_usage?.cache_creation_tokens || 0) }}
            </el-descriptions-item>
            <el-descriptions-item label="缓存读取 Token">
              {{ formatNumber(data.today_usage?.cache_read_tokens || 0) }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>用户统计</span>
            </div>
          </template>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="总用户数">
              {{ data.users?.total || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="今日活跃">
              {{ data.users?.active || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="今日新增">
              <el-tag v-if="data.users?.new_today > 0" type="success" size="small">
                +{{ data.users?.new_today || 0 }}
              </el-tag>
              <span v-else>0</span>
            </el-descriptions-item>
            <el-descriptions-item label="活跃率">
              {{ data.users?.total > 0 ? ((data.users?.active / data.users?.total) * 100).toFixed(1) : 0 }}%
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <!-- 账号状态 -->
    <el-row :gutter="16" class="section-row">
      <el-col :span="24">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>账号状态</span>
            </div>
          </template>
          <el-row :gutter="16">
            <el-col :span="6">
              <div class="account-stat">
                <div class="account-value">{{ data.accounts?.total || 0 }}</div>
                <div class="account-label">总账号</div>
              </div>
            </el-col>
            <el-col :span="6">
              <div class="account-stat success">
                <div class="account-value">{{ data.accounts?.active || 0 }}</div>
                <div class="account-label">正常可用</div>
              </div>
            </el-col>
            <el-col :span="6">
              <div class="account-stat warning">
                <div class="account-value">{{ data.accounts?.rate_limited || 0 }}</div>
                <div class="account-label">限流中</div>
              </div>
            </el-col>
            <el-col :span="6">
              <div class="account-stat danger">
                <div class="account-value">{{ data.accounts?.invalid || 0 }}</div>
                <div class="account-label">无效/禁用</div>
              </div>
            </el-col>
          </el-row>
        </el-card>
      </el-col>
    </el-row>

    <!-- 系统资源 -->
    <el-row :gutter="16" class="section-row">
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>CPU</span>
              <el-tag size="small">{{ data.system?.cpu_cores || 0 }} 核</el-tag>
            </div>
          </template>
          <el-progress
            :percentage="data.system?.cpu_usage || 0"
            :color="getProgressColor(data.system?.cpu_usage)"
            :stroke-width="20"
            :format="(p) => p.toFixed(1) + '%'"
          />
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>内存</span>
              <el-tag size="small">{{ formatBytes(data.system?.memory_total) }}</el-tag>
            </div>
          </template>
          <el-progress
            :percentage="data.system?.memory_usage || 0"
            :color="getProgressColor(data.system?.memory_usage)"
            :stroke-width="20"
            :format="(p) => p.toFixed(1) + '%'"
          />
          <div class="resource-detail">
            已用 {{ formatBytes(data.system?.memory_used) }} / 可用 {{ formatBytes(data.system?.memory_free) }}
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>磁盘</span>
              <el-tag size="small">{{ formatBytes(data.system?.disk_total) }}</el-tag>
            </div>
          </template>
          <el-progress
            :percentage="data.system?.disk_usage || 0"
            :color="getProgressColor(data.system?.disk_usage)"
            :stroke-width="20"
            :format="(p) => p.toFixed(1) + '%'"
          />
          <div class="resource-detail">
            已用 {{ formatBytes(data.system?.disk_used) }} / 可用 {{ formatBytes(data.system?.disk_free) }}
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 数据库状态 -->
    <el-row :gutter="16" class="section-row">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>内存缓存</span>
              <el-tag type="success" size="small">运行中</el-tag>
            </div>
          </template>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="会话绑定数">
              {{ formatNumber(data.cache?.session_count || 0) }}
            </el-descriptions-item>
            <el-descriptions-item label="账户并发数">
              {{ formatNumber(data.cache?.account_concurrency_count || 0) }}
            </el-descriptions-item>
            <el-descriptions-item label="用户并发数">
              {{ formatNumber(data.cache?.user_concurrency_count || 0) }}
            </el-descriptions-item>
            <el-descriptions-item label="不可用标记">
              {{ formatNumber(data.cache?.unavailable_count || 0) }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>MySQL</span>
              <el-tag :type="data.mysql?.connected ? 'success' : 'danger'" size="small">
                {{ data.mysql?.connected ? '已连接' : '未连接' }}
              </el-tag>
            </div>
          </template>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="表数量">
              {{ data.mysql?.table_count || 0 }}
            </el-descriptions-item>
            <el-descriptions-item label="数据大小">
              {{ formatBytes(data.mysql?.data_size) }}
            </el-descriptions-item>
            <el-descriptions-item label="索引大小">
              {{ formatBytes(data.mysql?.index_size) }}
            </el-descriptions-item>
            <el-descriptions-item label="总大小">
              {{ formatBytes(data.mysql?.total_size) }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <!-- 更新时间 -->
    <div class="update-time" v-if="data.updated_at">
      最后更新: {{ formatTime(data.updated_at) }}
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@/api'
import { ElMessage } from 'element-plus'
import { Refresh, Money, Coin, Connection, User, Files } from '@element-plus/icons-vue'

const loading = ref(false)
const data = ref({})

const fetchData = async () => {
  loading.value = true
  try {
    const res = await api.getMonitorData()
    if (res.code === 0) {
      data.value = res.data
    } else {
      ElMessage.error(res.message || '获取监控数据失败')
    }
  } catch (err) {
    ElMessage.error('获取监控数据失败')
  } finally {
    loading.value = false
  }
}

const formatNumber = (num, decimals = 0) => {
  if (num === undefined || num === null) return '0'
  if (decimals > 0) {
    return num.toFixed(decimals)
  }
  return num.toLocaleString()
}

const formatLargeNumber = (num) => {
  if (num === undefined || num === null || num === 0) return '0'
  if (num >= 1000000000) return (num / 1000000000).toFixed(2) + 'B'
  if (num >= 1000000) return (num / 1000000).toFixed(2) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toLocaleString()
}

const formatBytes = (bytes) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}

const formatTime = (time) => {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}

const getProgressColor = (percentage) => {
  if (percentage < 60) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.system-monitor {
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
  color: #303133;
}

.stat-row {
  margin-bottom: 16px;
}

.section-row {
  margin-bottom: 16px;
}

.stat-card {
  display: flex;
  align-items: center;
  padding: 10px;
}

.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  width: 100%;
  padding: 15px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 15px;
  font-size: 28px;
}

.cost-icon {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.token-icon {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: white;
}

.request-icon {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: white;
}

.user-icon {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  color: white;
}

/* 总使用卡片样式 */
.stat-card.total {
  border-left: 3px solid #409eff;
}

.total-cost-icon {
  background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
  color: white;
}

.total-token-icon {
  background: linear-gradient(135deg, #a55eea 0%, #8854d0 100%);
  color: white;
}

.total-request-icon {
  background: linear-gradient(135deg, #20bf6b 0%, #26de81 100%);
  color: white;
}

.total-cache-icon {
  background: linear-gradient(135deg, #fa8231 0%, #fd9644 100%);
  color: white;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
  line-height: 1.2;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 4px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.account-stat {
  text-align: center;
  padding: 20px;
  border-radius: 8px;
  background: #f5f7fa;
}

.account-stat.success {
  background: #f0f9eb;
}

.account-stat.success .account-value {
  color: #67c23a;
}

.account-stat.warning {
  background: #fdf6ec;
}

.account-stat.warning .account-value {
  color: #e6a23c;
}

.account-stat.danger {
  background: #fef0f0;
}

.account-stat.danger .account-value {
  color: #f56c6c;
}

.account-value {
  font-size: 32px;
  font-weight: bold;
  color: #303133;
}

.account-label {
  font-size: 14px;
  color: #909399;
  margin-top: 8px;
}

.resource-detail {
  margin-top: 10px;
  font-size: 12px;
  color: #909399;
  text-align: center;
}

.update-time {
  text-align: right;
  font-size: 12px;
  color: #909399;
  margin-top: 16px;
}
</style>
