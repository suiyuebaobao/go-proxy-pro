<!--
 * 文件作用：账户管理页面，管理多平台API账户
 * 负责功能：
 *   - 多平台账户展示（Claude/OpenAI/Gemini）
 *   - 账户状态管理（启用/禁用/健康检测）
 *   - 用量统计展示（5H/7D进度条）
 *   - 账户CRUD操作
 *   - Token刷新和强制恢复
 * 重要程度：⭐⭐⭐⭐⭐ 核心（账户管理）
 * 依赖模块：element-plus, AccountForm组件, api
-->
<template>
  <div class="accounts-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h2>账户管理</h2>
        <p class="header-desc">管理 Claude、Gemini、OpenAI 等账户与代理配置</p>
      </div>
      <div class="header-actions">
        <el-button @click="loadAccounts">
          <i class="fa-solid fa-sync-alt" :class="{ 'fa-spin': loading }"></i>
          刷新
        </el-button>
        <el-button type="primary" @click="showFormDialog = true">
          <i class="fa-solid fa-plus"></i>
          添加账户
        </el-button>
      </div>
    </div>

    <!-- 平台统计卡片 -->
    <div class="platform-stats">
      <div
        v-for="platform in platformStats"
        :key="platform.key"
        class="stat-card"
        :class="{ active: filters.platform === platform.key }"
        @click="filterByPlatform(platform.key)"
      >
        <div class="stat-icon" :style="{ background: platform.gradient }">
          <i :class="platform.icon"></i>
        </div>
        <div class="stat-info">
          <h3>{{ platform.name }}</h3>
          <div class="stat-numbers">
            <span class="total">{{ platform.count }} 个账户</span>
            <span class="valid">
              <i class="fa-solid fa-circle valid-dot"></i>
              {{ platform.validCount }} 可用
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 筛选和搜索 -->
    <div class="filter-bar">
      <div class="filter-left">
        <el-select v-model="filters.status" clearable placeholder="状态筛选" @change="loadAccounts">
          <el-option label="正常" value="valid" />
          <el-option label="无效" value="invalid" />
          <el-option label="限流中" value="rate_limited" />
          <el-option label="Token过期" value="token_expired" />
          <el-option label="疑似封号" value="suspended" />
          <el-option label="已封号" value="banned" />
          <el-option label="已禁用" value="disabled" />
        </el-select>
        <el-input
          v-model="filters.search"
          placeholder="搜索账户名称..."
          clearable
          style="width: 200px"
          @input="handleSearch"
        >
          <template #prefix>
            <i class="fa-solid fa-search"></i>
          </template>
        </el-input>
      </div>
      <div class="filter-right">
        <el-tag v-if="filters.platform" closable @close="filters.platform = ''; loadAccounts()">
          {{ getPlatformName(filters.platform) }}
        </el-tag>
      </div>
    </div>

    <!-- 账户列表 -->
    <el-card class="accounts-table-card" shadow="never">
      <el-table
        :data="accounts"
        v-loading="loading"
        stripe
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="50" />

        <el-table-column label="#" width="60" align="center">
          <template #default="{ $index }">
            <span class="row-index">{{ (pagination.page - 1) * pagination.pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>

        <el-table-column label="账户" min-width="200">
          <template #default="{ row }">
            <div class="account-cell">
              <div class="account-avatar" :style="{ background: getTypeColor(row.type) }">
                <i :class="getTypeIcon(row.type)"></i>
              </div>
              <div class="account-info">
                <span class="account-name">{{ row.name }}</span>
                <span class="account-type">{{ getTypeLabel(row.type, row) }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="平台/类型" width="180">
          <template #default="{ row }">
            <div class="platform-badge" :class="getPlatformClass(row.type)">
              <i :class="getPlatformIcon(row.type)"></i>
              <span>{{ getPlatformLabel(row.type) }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="180">
          <template #default="{ row }">
            <div class="status-badge" :class="row.status">
              <span class="status-dot"></span>
              {{ getStatusLabel(row.status) }}
            </div>
            <!-- 限流倒计时 -->
            <div v-if="row.status === 'rate_limited' && row.rate_limit_reset_at" class="status-detail rate-limit-reset">
              <i class="fa-solid fa-clock"></i>
              {{ formatResetTime(row.rate_limit_reset_at) }}
            </div>
            <!-- 下次检测时间 -->
            <div v-if="row.next_health_check_at && ['rate_limited', 'suspended', 'banned', 'token_expired'].includes(row.status)" class="status-detail next-check">
              <i class="fa-solid fa-stethoscope"></i>
              下次检测: {{ formatNextCheck(row.next_health_check_at) }}
            </div>
            <!-- 疑似封号计数 -->
            <div v-if="row.status === 'suspended' && row.suspended_count > 0" class="status-detail suspended-count">
              <i class="fa-solid fa-triangle-exclamation"></i>
              连续失败 {{ row.suspended_count }} 次
            </div>
            <!-- 错误信息 -->
            <el-tooltip v-if="row.last_error && ['invalid', 'suspended', 'banned', 'token_expired'].includes(row.status)" :content="row.last_error" placement="top">
              <div class="status-detail error-hint">
                <i class="fa-solid fa-circle-info"></i>
                查看错误
              </div>
            </el-tooltip>
          </template>
        </el-table-column>

        <!-- 用量进度条 -->
        <el-table-column label="用量" width="180">
          <template #default="{ row }">
            <!-- Claude Official: 显示 5H/7D/7D-S 进度条 -->
            <div v-if="row.type === 'claude-official' && hasUsageData(row)" class="usage-bars">
              <!-- 5小时窗口 -->
              <div class="usage-bar-item" v-if="row.five_hour_utilization !== null && row.five_hour_utilization !== undefined">
                <div class="usage-bar-label">
                  <span class="label-text">5H</span>
                  <span class="label-value">{{ row.five_hour_utilization.toFixed(1) }}%</span>
                </div>
                <div class="usage-bar-track">
                  <div
                    class="usage-bar-fill"
                    :class="getUsageBarClass(row.five_hour_utilization)"
                    :style="{ width: Math.min(row.five_hour_utilization, 100) + '%' }"
                  ></div>
                </div>
              </div>
              <!-- 7天窗口 -->
              <div class="usage-bar-item" v-if="row.seven_day_utilization !== null && row.seven_day_utilization !== undefined">
                <div class="usage-bar-label">
                  <span class="label-text">7D</span>
                  <span class="label-value">{{ row.seven_day_utilization.toFixed(1) }}%</span>
                </div>
                <div class="usage-bar-track">
                  <div
                    class="usage-bar-fill"
                    :class="getUsageBarClass(row.seven_day_utilization)"
                    :style="{ width: Math.min(row.seven_day_utilization, 100) + '%' }"
                  ></div>
                </div>
              </div>
              <!-- 7天Sonnet窗口 -->
              <div class="usage-bar-item" v-if="row.seven_day_sonnet_utilization !== null && row.seven_day_sonnet_utilization !== undefined">
                <div class="usage-bar-label">
                  <span class="label-text">7D-S</span>
                  <span class="label-value">{{ row.seven_day_sonnet_utilization.toFixed(1) }}%</span>
                </div>
                <div class="usage-bar-track">
                  <div
                    class="usage-bar-fill"
                    :class="getUsageBarClass(row.seven_day_sonnet_utilization)"
                    :style="{ width: Math.min(row.seven_day_sonnet_utilization, 100) + '%' }"
                  ></div>
                </div>
              </div>
            </div>
            <!-- 其他类型: 显示预算进度条或今日统计 -->
            <div v-else-if="row.daily_budget > 0" class="usage-bars">
              <!-- 预算使用率进度条 -->
              <div class="usage-bar-item">
                <div class="usage-bar-label">
                  <span class="label-text">今日</span>
                  <span class="label-value">${{ formatCost(row.today_cost || 0) }} / ${{ formatCost(row.daily_budget) }}</span>
                </div>
                <div class="usage-bar-track">
                  <div
                    class="usage-bar-fill"
                    :class="getUsageBarClass(row.budget_utilization || 0)"
                    :style="{ width: Math.min(row.budget_utilization || 0, 100) + '%' }"
                  ></div>
                </div>
              </div>
              <!-- 今日请求统计 -->
              <div class="usage-stat-row">
                <span class="stat-label"><i class="fa-solid fa-coins"></i> {{ formatTokens(row.today_tokens) }}</span>
                <span class="stat-label"><i class="fa-solid fa-arrow-right-arrow-left"></i> {{ row.today_count }} 次</span>
              </div>
            </div>
            <!-- 无预算限制: 显示简单统计 -->
            <div v-else-if="row.today_tokens > 0 || row.today_count > 0" class="usage-stats">
              <div class="usage-stat-item">
                <i class="fa-solid fa-dollar-sign"></i>
                <span>${{ formatCost(row.today_cost || 0) }}</span>
              </div>
              <div class="usage-stat-item">
                <i class="fa-solid fa-coins"></i>
                <span>{{ formatTokens(row.today_tokens) }}</span>
              </div>
              <div class="usage-stat-item">
                <i class="fa-solid fa-arrow-right-arrow-left"></i>
                <span>{{ row.today_count }} 次</span>
              </div>
            </div>
            <span v-else class="no-usage">-</span>
          </template>
        </el-table-column>

        <el-table-column label="启用" width="80" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              size="small"
              @change="handleToggleEnabled(row)"
            />
          </template>
        </el-table-column>

        <el-table-column label="优先级" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.priority }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="并发" width="100" align="center">
          <template #default="{ row }">
            <div class="concurrency-cell">
              <span class="concurrency-current" :class="getConcurrencyClass(row)">
                {{ row.current_concurrency || 0 }}
              </span>
              <span class="concurrency-separator">/</span>
              <span class="concurrency-max">{{ row.max_concurrency || 5 }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="请求次数" width="100" align="right">
          <template #default="{ row }">
            <span class="request-count">{{ formatNumber(row.request_count) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="总费用" width="110" align="right">
          <template #default="{ row }">
            <span class="total-cost" v-if="row.total_cost > 0">
              ${{ formatCost(row.total_cost) }}
            </span>
            <span class="no-cost" v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="今日用量" width="140" align="right">
          <template #default="{ row }">
            <div class="today-usage">
              <span class="usage-tokens">{{ formatTokens(row.today_tokens) }}</span>
              <span class="usage-count">{{ row.today_count || 0 }} 次</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="最后使用" width="150">
          <template #default="{ row }">
            <span class="last-used" v-if="row.last_used_at">{{ formatRelativeTime(row.last_used_at) }}</span>
            <span class="no-used" v-else>-</span>
          </template>
        </el-table-column>

        <el-table-column label="代理" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.proxy?.enabled" size="small" type="warning">
              <i class="fa-solid fa-shield-halved"></i>
              {{ row.proxy.type }}
            </el-tag>
            <span v-else class="no-proxy">-</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleEdit(row)">
              <i class="fa-solid fa-edit"></i> 编辑
            </el-button>
            <!-- 健康检测按钮 -->
            <el-button
              v-if="canHealthCheck(row)"
              link
              type="info"
              size="small"
              :loading="healthCheckingIds.includes(row.id)"
              @click="handleHealthCheck(row)"
            >
              <i class="fa-solid fa-stethoscope"></i> 检测
            </el-button>
            <!-- 强制恢复按钮 -->
            <el-button
              v-if="canRecover(row)"
              link
              type="success"
              size="small"
              :loading="recoveringIds.includes(row.id)"
              @click="handleForceRecover(row)"
            >
              <i class="fa-solid fa-rotate"></i> 恢复
            </el-button>
            <!-- 刷新 Token 按钮 -->
            <el-button
              v-if="canRefreshToken(row)"
              link
              type="warning"
              size="small"
              :loading="refreshingIds.includes(row.id)"
              @click="handleRefreshToken(row)"
            >
              <i class="fa-solid fa-key"></i> 刷新Token
            </el-button>
            <el-popconfirm
              title="确定删除该账户吗？"
              confirm-button-text="删除"
              cancel-button-text="取消"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button link type="danger" size="small">
                  <i class="fa-solid fa-trash"></i> 删除
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="table-footer">
        <div class="selection-info" v-if="selectedAccounts.length > 0">
          已选择 {{ selectedAccounts.length }} 项
          <el-button link type="danger" @click="handleBatchDelete">批量删除</el-button>
        </div>
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @change="loadAccounts"
        />
      </div>
    </el-card>

    <!-- 添加/编辑弹窗 -->
    <AccountForm
      v-model="showFormDialog"
      :edit-data="editingAccount"
      @success="handleFormSuccess"
      @update:modelValue="handleDialogClose"
    />
  </div>
</template>

<script setup>
import { ensureFontAwesomeLoaded } from '@/utils/fontawesome'
ensureFontAwesomeLoaded()

import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import AccountForm from '@/components/AccountForm.vue'
import api from '@/api'

const loading = ref(false)
const accounts = ref([])
const selectedAccounts = ref([])
const showFormDialog = ref(false)
const editingAccount = ref(null)

// 操作加载状态
const healthCheckingIds = ref([])
const recoveringIds = ref([])
const refreshingIds = ref([])

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const filters = reactive({
  platform: '',
  status: '',
  search: ''
})

// 平台分组定义
const platformGroups = [
  {
    key: 'claude',
    name: 'Claude',
    icon: 'fa-solid fa-brain',
    gradient: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    types: ['claude-official', 'claude-console', 'bedrock']
  },
  {
    key: 'openai',
    name: 'OpenAI',
    icon: 'fa-solid fa-robot',
    gradient: 'linear-gradient(135deg, #11998e 0%, #38ef7d 100%)',
    types: ['openai', 'openai-responses', 'azure-openai']
  },
  {
    key: 'gemini',
    name: 'Gemini',
    icon: 'fa-brands fa-google',
    gradient: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
    types: ['gemini']
  }
]

// 子类型定义
const subtypeMap = {
  'claude-official': { label: 'Claude Official', icon: 'fa-solid fa-key', color: '#667eea', platform: 'Claude' },
  'claude-console': { label: 'Claude Console', icon: 'fa-solid fa-terminal', color: '#764ba2', platform: 'Claude' },
  'bedrock': { label: 'AWS Bedrock', icon: 'fa-brands fa-aws', color: '#ff9900', platform: 'Claude' },
  'openai': { label: 'OpenAI 三方 API', icon: 'fa-solid fa-bolt', color: '#11998e', platform: 'OpenAI' },
  'openai-responses': { label: 'ChatGPT 官方', icon: 'fa-solid fa-comments', color: '#38ef7d', platform: 'OpenAI' },
  'azure-openai': { label: 'Azure OpenAI', icon: 'fa-brands fa-microsoft', color: '#0078d4', platform: 'OpenAI' },
  'gemini': { label: 'Gemini', icon: 'fa-brands fa-google', color: '#4facfe', platform: 'Gemini' }
}

// 平台统计
const platformStats = computed(() => {
  return platformGroups.map(group => {
    const platformAccounts = accounts.value.filter(a => group.types.includes(a.type))
    return {
      ...group,
      count: platformAccounts.length,
      validCount: platformAccounts.filter(a => a.status === 'valid' && a.enabled).length
    }
  })
})

// 获取类型相关方法
function getTypeLabel(type, row = null) {
  const baseLabel = subtypeMap[type]?.label || type

  if (!row) return baseLabel

  // 对 claude-official 类型显示更详细的认证方式
  if (type === 'claude-official') {
    const hasToken = row.access_token
    const hasSessionKey = row.session_key
    if (hasToken && hasSessionKey) {
      return 'OAuth + SK'
    } else if (hasToken) {
      return 'OAuth'
    } else if (hasSessionKey) {
      return 'SessionKey'
    } else if (row.api_key) {
      return 'API Key'
    }
    return 'Claude Official'
  }

  // OpenAI 三方 API
  if (type === 'openai') {
    if (row.base_url) {
      // 尝试从 base_url 提取服务商名称
      try {
        const url = new URL(row.base_url)
        const host = url.hostname.replace('www.', '')
        // 常见的第三方服务
        if (host.includes('openrouter')) return 'OpenRouter'
        if (host.includes('together')) return 'Together AI'
        if (host.includes('groq')) return 'Groq'
        if (host.includes('deepseek')) return 'DeepSeek'
        if (host.includes('moonshot') || host.includes('kimi')) return 'Moonshot'
        if (host.includes('zhipu') || host.includes('bigmodel')) return 'ZhipuAI'
        if (host.includes('baichuan')) return 'Baichuan'
        if (host.includes('minimax')) return 'MiniMax'
        if (host.includes('yi.') || host.includes('lingyiwanwu')) return '零一万物'
        if (host.includes('siliconflow')) return 'SiliconFlow'
        // 返回简短域名
        return host.split('.')[0]
      } catch {
        return '三方 API'
      }
    }
    return row.api_key ? 'API Key' : 'OpenAI'
  }

  // ChatGPT 官方 (openai-responses)
  if (type === 'openai-responses') {
    if (row.access_token && row.refresh_token) {
      return 'ChatGPT OAuth'
    } else if (row.access_token) {
      return 'ChatGPT Token'
    }
    return 'ChatGPT'
  }

  // Azure OpenAI
  if (type === 'azure-openai') {
    if (row.azure_endpoint) {
      try {
        const url = new URL(row.azure_endpoint)
        // 提取资源名称，如 xxx.openai.azure.com -> xxx
        const parts = url.hostname.split('.')
        if (parts.length > 0) {
          return `Azure: ${parts[0]}`
        }
      } catch {}
    }
    return 'Azure OpenAI'
  }

  // Gemini
  if (type === 'gemini') {
    if (row.access_token) {
      return 'Gemini OAuth'
    }
    return 'Gemini'
  }

  if (type === 'gemini-api') {
    return 'Gemini API'
  }

  // Bedrock
  if (type === 'bedrock') {
    if (row.aws_region) {
      return `Bedrock: ${row.aws_region}`
    }
    return 'AWS Bedrock'
  }

  return baseLabel
}

function getTypeIcon(type) {
  return subtypeMap[type]?.icon || 'fa-solid fa-circle'
}

function getTypeColor(type) {
  return subtypeMap[type]?.color || '#999'
}

function getPlatformLabel(type) {
  return subtypeMap[type]?.platform || 'Unknown'
}

function getPlatformIcon(type) {
  const platform = subtypeMap[type]?.platform
  if (platform === 'Claude') return 'fa-solid fa-brain'
  if (platform === 'OpenAI') return 'fa-solid fa-robot'
  if (platform === 'Gemini') return 'fa-brands fa-google'
  return 'fa-solid fa-circle'
}

function getPlatformClass(type) {
  const platform = subtypeMap[type]?.platform?.toLowerCase()
  return platform || 'unknown'
}

function getPlatformName(key) {
  return platformGroups.find(g => g.key === key)?.name || key
}

function getStatusLabel(status) {
  const map = {
    valid: '正常',
    invalid: '无效',
    rate_limited: '限流中',
    overloaded: '过载',
    token_expired: 'Token过期',
    suspended: '疑似封号',
    banned: '已封号',
    disabled: '已禁用'
  }
  return map[status] || status
}

function getUsageStatusLabel(status) {
  const map = {
    allowed: '5H正常',
    allowed_warning: '5H接近限额',
    rejected: '5H已限流'
  }
  return map[status] || status
}

// 判断是否有用量数据
function hasUsageData(row) {
  return row.five_hour_utilization !== null && row.five_hour_utilization !== undefined ||
         row.seven_day_utilization !== null && row.seven_day_utilization !== undefined ||
         row.seven_day_sonnet_utilization !== null && row.seven_day_sonnet_utilization !== undefined
}

// 根据用量百分比获取进度条颜色类
function getUsageBarClass(utilization) {
  if (utilization >= 90) return 'danger'
  if (utilization >= 70) return 'warning'
  return 'normal'
}

// 获取并发状态颜色类
function getConcurrencyClass(row) {
  const current = row.current_concurrency || 0
  const max = row.max_concurrency || 5
  if (current >= max) return 'danger'
  if (current >= max * 0.8) return 'warning'
  return 'normal'
}

function formatNumber(num) {
  if (!num) return '0'
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

function formatTokens(tokens) {
  if (!tokens) return '0'
  if (tokens >= 1000000) return (tokens / 1000000).toFixed(1) + 'M'
  if (tokens >= 1000) return (tokens / 1000).toFixed(1) + 'K'
  return tokens.toString()
}

function formatCost(cost) {
  if (!cost) return '0.00'
  if (cost >= 1000) return (cost / 1000).toFixed(2) + 'K'
  if (cost >= 1) return cost.toFixed(2)
  if (cost >= 0.01) return cost.toFixed(3)
  return cost.toFixed(4)
}

function formatRelativeTime(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((now - date) / 1000) // 秒数差

  if (diff < 60) return '刚刚'
  if (diff < 3600) return Math.floor(diff / 60) + ' 分钟前'
  if (diff < 86400) return Math.floor(diff / 3600) + ' 小时前'
  if (diff < 604800) return Math.floor(diff / 86400) + ' 天前'

  // 超过7天显示具体日期
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

// 格式化限流恢复时间（显示倒计时）
function formatResetTime(dateStr) {
  if (!dateStr) return ''
  const resetTime = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((resetTime - now) / 1000) // 秒数差

  if (diff <= 0) return '即将恢复'
  if (diff < 60) return diff + ' 秒后恢复'
  if (diff < 3600) return Math.floor(diff / 60) + ' 分钟后恢复'
  if (diff < 86400) {
    const hours = Math.floor(diff / 3600)
    const mins = Math.floor((diff % 3600) / 60)
    return hours + '时' + mins + '分后恢复'
  }
  // 超过1天显示具体时间
  return resetTime.toLocaleString('zh-CN', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }) + ' 恢复'
}

// 格式化下次检测时间
function formatNextCheck(dateStr) {
  if (!dateStr) return ''
  const checkTime = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((checkTime - now) / 1000)

  if (diff <= 0) return '待检测'
  if (diff < 60) return diff + ' 秒后'
  if (diff < 3600) return Math.floor(diff / 60) + ' 分钟后'
  return Math.floor(diff / 3600) + ' 小时后'
}

// 判断是否可以执行健康检测
function canHealthCheck(row) {
  // OAuth 类型账号才能检测
  const oauthTypes = ['claude-official', 'openai-responses', 'gemini']
  return oauthTypes.includes(row.type)
}

// 判断是否可以恢复
function canRecover(row) {
  const recoverableStatuses = ['invalid', 'rate_limited', 'token_expired', 'suspended', 'banned', 'disabled']
  return recoverableStatuses.includes(row.status)
}

// 判断是否可以刷新 Token
function canRefreshToken(row) {
  // 只有有 session_key 的 claude-official 类型才能刷新
  return row.type === 'claude-official' && row.session_key
}

// 筛选和搜索
function filterByPlatform(key) {
  if (filters.platform === key) {
    filters.platform = ''
  } else {
    filters.platform = key
  }
  loadAccounts()
}

let searchTimer = null
function handleSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    loadAccounts()
  }, 300)
}

onUnmounted(() => {
  if (searchTimer) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
})

// 加载账户列表
async function loadAccounts() {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize,
      status: filters.status,
      search: filters.search
    }

    // 根据平台筛选类型
    if (filters.platform) {
      const group = platformGroups.find(g => g.key === filters.platform)
      if (group) {
        params.types = group.types.join(',')
      }
    }

    const res = await api.getAccounts(params)
    accounts.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (e) {
    console.error('Failed to load accounts:', e)
  } finally {
    loading.value = false
  }
}

// 选择处理
function handleSelectionChange(selection) {
  selectedAccounts.value = selection.map(item => item.id)
}

// 切换启用状态
async function handleToggleEnabled(row) {
  try {
    await api.updateAccount(row.id, { enabled: row.enabled })
    ElMessage.success('更新成功')
  } catch (e) {
    row.enabled = !row.enabled
    ElMessage.error('更新失败')
  }
}

// 编辑
function handleEdit(row) {
  editingAccount.value = { ...row }
  showFormDialog.value = true
}

// 删除
async function handleDelete(id) {
  try {
    await api.deleteAccount(id)
    ElMessage.success('删除成功')
    loadAccounts()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

// 恢复账户状态为正常
async function handleResetStatus(row) {
  try {
    await api.updateAccountStatus(row.id, { status: 'valid', last_error: '' })
    ElMessage.success('账户状态已恢复为正常')
    loadAccounts()
  } catch (e) {
    ElMessage.error('恢复失败')
  }
}

// 健康检测
async function handleHealthCheck(row) {
  healthCheckingIds.value.push(row.id)
  try {
    const res = await api.checkAccountHealth(row.id)
    const data = res.data || res
    if (data.healthy) {
      ElMessage.success(`[${row.name}] ${data.message}`)
    } else {
      ElMessage.warning(`[${row.name}] ${data.message}`)
    }
    loadAccounts()
  } catch (e) {
    ElMessage.error('检测失败')
  } finally {
    healthCheckingIds.value = healthCheckingIds.value.filter(id => id !== row.id)
  }
}

// 强制恢复
async function handleForceRecover(row) {
  recoveringIds.value.push(row.id)
  try {
    await api.recoverAccount(row.id)
    ElMessage.success(`[${row.name}] 账号已强制恢复`)
    loadAccounts()
  } catch (e) {
    ElMessage.error('恢复失败')
  } finally {
    recoveringIds.value = recoveringIds.value.filter(id => id !== row.id)
  }
}

// 刷新 Token
async function handleRefreshToken(row) {
  refreshingIds.value.push(row.id)
  try {
    await api.refreshAccountToken(row.id)
    ElMessage.success(`[${row.name}] Token 刷新成功`)
    loadAccounts()
  } catch (e) {
    ElMessage.error('Token 刷新失败')
  } finally {
    refreshingIds.value = refreshingIds.value.filter(id => id !== row.id)
  }
}

// 批量删除
async function handleBatchDelete() {
  if (selectedAccounts.value.length === 0) return

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedAccounts.value.length} 个账户吗？`,
      '批量删除',
      { type: 'warning' }
    )

    for (const id of selectedAccounts.value) {
      await api.deleteAccount(id)
    }
    ElMessage.success('批量删除成功')
    selectedAccounts.value = []
    loadAccounts()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('批量删除失败')
    }
  }
}

// 表单成功回调
function handleFormSuccess() {
  showFormDialog.value = false
  editingAccount.value = null
  loadAccounts()
}

// 弹窗关闭时清除编辑数据
function handleDialogClose(val) {
  if (!val) {
    // 弹窗关闭时，清除编辑数据
    editingAccount.value = null
  }
}

onMounted(() => {
  loadAccounts()
})
</script>

<style scoped>
.accounts-page {
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

/* 平台统计卡片 */
.platform-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: white;
  border-radius: 12px;
  border: 2px solid transparent;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
}

.stat-card.active {
  border-color: #3b82f6;
  background: linear-gradient(135deg, #f0f7ff 0%, #e8f4ff 100%);
}

.stat-icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 22px;
  flex-shrink: 0;
}

.stat-info h3 {
  margin: 0 0 6px;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.stat-numbers {
  display: flex;
  align-items: center;
  gap: 12px;
}

.stat-numbers .total {
  font-size: 13px;
  color: #6b7280;
}

.stat-numbers .valid {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #059669;
}

.valid-dot {
  font-size: 6px;
  color: #10b981;
}

/* 筛选栏 */
.filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.filter-left {
  display: flex;
  gap: 12px;
}

/* 账户表格 */
.accounts-table-card {
  border-radius: 12px;
}

.account-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.account-avatar {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 16px;
  flex-shrink: 0;
}

.account-info {
  display: flex;
  flex-direction: column;
}

.account-name {
  font-weight: 600;
  color: #1f2937;
}

.account-type {
  font-size: 12px;
  color: #6b7280;
}

.platform-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
}

.platform-badge.claude {
  background: #eef2ff;
  color: #667eea;
}

.platform-badge.openai {
  background: #ecfdf5;
  color: #059669;
}

.platform-badge.gemini {
  background: #eff6ff;
  color: #3b82f6;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.valid {
  background: #d1fae5;
  color: #059669;
}

.status-badge.valid .status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
}

.status-badge.invalid {
  background: #fee2e2;
  color: #dc2626;
}

.status-badge.invalid .status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #ef4444;
}

.status-badge.rate_limited {
  background: #fef3c7;
  color: #d97706;
}

.status-badge.rate_limited .status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f59e0b;
}

.status-badge.token_expired {
  background: #fef3c7;
  color: #b45309;
}

.status-badge.token_expired .status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f59e0b;
}

.status-badge.suspended {
  background: #fed7aa;
  color: #c2410c;
}

.status-badge.suspended .status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #ea580c;
}

.status-badge.banned {
  background: #fecaca;
  color: #991b1b;
}

.status-badge.banned .status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #b91c1c;
}

.status-badge.disabled {
  background: #e5e7eb;
  color: #6b7280;
}

.status-badge.disabled .status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #9ca3af;
}

.status-badge.overloaded {
  background: #ddd6fe;
  color: #7c3aed;
}

.status-badge.overloaded .status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #8b5cf6;
}

/* 状态详情 */
.status-detail {
  font-size: 11px;
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.status-detail i {
  font-size: 10px;
}

.status-detail.next-check {
  color: #6b7280;
}

.status-detail.suspended-count {
  color: #c2410c;
}

.status-detail.error-hint {
  color: #6b7280;
  cursor: pointer;
}

.status-detail.error-hint:hover {
  color: #3b82f6;
}

.rate-limit-reset {
  font-size: 11px;
  color: #d97706;
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.rate-limit-reset i {
  font-size: 10px;
}

.request-count {
  font-family: 'SF Mono', Monaco, monospace;
  color: #4b5563;
}

.total-cost {
  font-family: 'SF Mono', Monaco, monospace;
  font-weight: 600;
  color: #059669;
}

.no-cost {
  color: #d1d5db;
}

.no-proxy {
  color: #9ca3af;
}

.today-usage {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.usage-tokens {
  font-family: 'SF Mono', Monaco, monospace;
  font-weight: 600;
  color: #3b82f6;
}

.usage-count {
  font-size: 11px;
  color: #9ca3af;
}

.last-used {
  font-size: 13px;
  color: #6b7280;
}

.no-used {
  color: #d1d5db;
}

/* 5H 用量状态徽章 */
.usage-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
  margin-top: 4px;
}

.usage-status-badge.allowed {
  background: #d1fae5;
  color: #059669;
}

.usage-status-badge.allowed_warning {
  background: #fef3c7;
  color: #d97706;
}

.usage-status-badge.rejected {
  background: #fee2e2;
  color: #dc2626;
}

.usage-status-icon {
  display: flex;
  align-items: center;
}

.usage-status-icon i {
  font-size: 10px;
}

/* 用量进度条样式 */
.usage-bars {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.usage-bar-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.usage-bar-label {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  line-height: 1;
}

.usage-bar-label .label-text {
  color: #6b7280;
  font-weight: 500;
}

.usage-bar-label .label-value {
  font-family: 'SF Mono', Monaco, monospace;
  font-weight: 600;
  color: #374151;
}

.usage-bar-track {
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
}

.usage-bar-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.3s ease;
}

.usage-bar-fill.normal {
  background: linear-gradient(90deg, #10b981, #34d399);
}

.usage-bar-fill.warning {
  background: linear-gradient(90deg, #f59e0b, #fbbf24);
}

.usage-bar-fill.danger {
  background: linear-gradient(90deg, #ef4444, #f87171);
}

.no-usage {
  color: #d1d5db;
}

/* OpenAI/其他类型用量统计 */
.usage-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.usage-stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #374151;
}

.usage-stat-item i {
  width: 14px;
  color: #9ca3af;
  font-size: 11px;
}

.usage-stat-item span {
  font-family: 'SF Mono', Monaco, monospace;
  font-weight: 500;
}

.usage-stat-row {
  display: flex;
  gap: 12px;
  margin-top: 4px;
}

.usage-stat-row .stat-label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #6b7280;
}

.usage-stat-row .stat-label i {
  font-size: 10px;
  color: #9ca3af;
}

.table-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.selection-info {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #6b7280;
  font-size: 14px;
}

.row-index {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
  color: #9ca3af;
}

/* 并发列样式 */
.concurrency-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
}

.concurrency-current {
  font-weight: 600;
}

.concurrency-current.normal {
  color: #059669;
}

.concurrency-current.warning {
  color: #d97706;
}

.concurrency-current.danger {
  color: #dc2626;
}

.concurrency-separator {
  color: #9ca3af;
}

.concurrency-max {
  color: #6b7280;
}
</style>
