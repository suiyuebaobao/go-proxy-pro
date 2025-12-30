<!--
 * 文件作用：系统设置页面，配置系统参数
 * 负责功能：
 *   - 安全配置（验证码、登录限制）
 *   - 记录配置（保留天数、价格倍率）
 *   - 账号健康检查配置
 *   - 分级检测策略配置
 * 重要程度：⭐⭐⭐⭐ 重要（系统配置）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="settings-page">
    <div class="page-header">
      <h2>系统设置</h2>
    </div>

    <el-row :gutter="20">
      <!-- 安全配置 -->
      <el-col :span="12">
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <span>安全配置</span>
            </div>
          </template>

          <el-form label-width="160px" v-loading="loading">
            <el-form-item label="启用登录验证码">
              <el-switch v-model="captchaEnabled" />
              <div class="form-tip">开启后登录和注册需要输入图片验证码</div>
            </el-form-item>

            <el-form-item label="验证码频率限制">
              <el-input-number
                v-model="configs.captcha_rate_limit"
                :min="1"
                :max="100"
              />
              <span class="unit">次/分钟</span>
              <div class="form-tip">每个 IP 每分钟最多获取验证码的次数</div>
            </el-form-item>

            <el-divider />

            <el-form-item label="启用登录频率限制">
              <el-switch v-model="loginRateLimitEnabled" />
              <div class="form-tip">防止暴力破解，限制登录尝试次数</div>
            </el-form-item>

            <el-form-item label="登录限制次数">
              <el-input-number
                v-model="configs.login_rate_limit_count"
                :min="1"
                :max="100"
                :disabled="!loginRateLimitEnabled"
              />
              <span class="unit">次</span>
              <div class="form-tip">时间窗口内允许的最大登录尝试次数</div>
            </el-form-item>

            <el-form-item label="登录限制窗口">
              <el-input-number
                v-model="configs.login_rate_limit_window"
                :min="1"
                :max="60"
                :disabled="!loginRateLimitEnabled"
              />
              <span class="unit">分钟</span>
              <div class="form-tip">频率限制的时间窗口</div>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 记录配置 -->
      <el-col :span="12">
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <span>记录配置</span>
            </div>
          </template>

          <el-form label-width="140px" v-loading="loading">
            <el-form-item label="记录保留天数">
              <el-input-number
                v-model="configs.record_retention_days"
                :min="1"
                :max="365"
              />
              <span class="unit">天</span>
              <div class="form-tip">使用记录的保留时间</div>
            </el-form-item>

            <el-form-item label="最大记录数">
              <el-input-number
                v-model="configs.record_max_count"
                :min="100"
                :max="10000"
                :step="100"
              />
              <span class="unit">条/用户</span>
              <div class="form-tip">每个用户保留的最大记录数</div>
            </el-form-item>

            <el-divider />

            <el-form-item label="全局价格倍率">
              <el-input-number
                v-model="configs.global_price_rate"
                :min="0"
                :max="10"
                :step="0.1"
                :precision="2"
              />
              <div class="form-tip">
                全局价格倍率（1=原价，0=免费，2=2倍）<br/>
                优先级：全局倍率 → 用户倍率（全局为1时使用用户倍率）
              </div>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <!-- 账号健康检查配置 -->
      <el-col :span="12">
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <span>账号健康检查</span>
              <div class="header-actions">
                <el-tag :type="healthCheckEnabled ? 'success' : 'info'" size="small">
                  {{ healthCheckEnabled ? '已启用' : '已禁用' }}
                </el-tag>
                <el-button
                  type="primary"
                  size="small"
                  @click="viewHealthCheckStatus"
                  :disabled="!healthCheckEnabled"
                  style="margin-left: 10px;"
                >
                  查看状态
                </el-button>
              </div>
            </div>
          </template>

          <el-form label-width="160px" v-loading="loading">
            <el-form-item label="启用健康检查">
              <el-switch v-model="healthCheckEnabled" />
              <div class="form-tip">定期检查 OAuth 账号的有效性（仅检查 Claude Official、OpenAI Responses、Gemini 类型）</div>
            </el-form-item>

            <el-form-item label="检查间隔">
              <el-input-number
                v-model="configs.account_health_check_interval"
                :min="1"
                :max="60"
                :disabled="!healthCheckEnabled"
              />
              <span class="unit">分钟</span>
              <div class="form-tip">正常账号的定期检查间隔</div>
            </el-form-item>

            <el-form-item label="连续错误阈值">
              <el-input-number
                v-model="configs.account_error_threshold"
                :min="1"
                :max="100"
                :disabled="!healthCheckEnabled"
              />
              <span class="unit">次</span>
              <div class="form-tip">连续检查失败达到此次数后标记为疑似封号</div>
            </el-form-item>

            <el-divider content-position="left">恢复策略</el-divider>

            <el-form-item label="自动恢复">
              <el-switch v-model="healthCheckAutoRecovery" :disabled="!healthCheckEnabled" />
              <div class="form-tip">检测通过时自动恢复账号为正常状态</div>
            </el-form-item>

            <el-form-item label="自动刷新 Token">
              <el-switch v-model="healthCheckAutoTokenRefresh" :disabled="!healthCheckEnabled" />
              <div class="form-tip">Token 过期时自动尝试刷新</div>
            </el-form-item>

            <el-divider content-position="left">OAuth 配置</el-divider>

            <el-form-item label="OAuth 自动重授权">
              <el-switch v-model="oauthAutoReauthorizeEnabled" :disabled="!healthCheckEnabled" />
              <div class="form-tip">OAuth Token 失效时，自动使用 SessionKey 重新获取 Token</div>
            </el-form-item>

            <el-form-item label="重授权冷却时间">
              <el-input-number
                v-model="configs.oauth_reauthorize_cooldown"
                :min="5"
                :max="1440"
                :disabled="!healthCheckEnabled || !oauthAutoReauthorizeEnabled"
              />
              <span class="unit">分钟</span>
              <div class="form-tip">重新授权失败后的冷却时间（默认 30 分钟）</div>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 分级检测策略 -->
      <el-col :span="12">
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <span>分级检测策略</span>
            </div>
          </template>

          <el-form label-width="160px" v-loading="loading">
            <el-divider content-position="left">限流账号探测</el-divider>

            <el-form-item label="启用主动探测">
              <el-switch v-model="rateLimitedProbeEnabled" :disabled="!healthCheckEnabled" />
              <div class="form-tip">限流账号不等待官方 reset 时间，主动探测恢复</div>
            </el-form-item>

            <el-form-item label="初始探测间隔">
              <el-input-number
                v-model="configs.rate_limited_probe_init_interval"
                :min="1"
                :max="60"
                :disabled="!healthCheckEnabled || !rateLimitedProbeEnabled"
              />
              <span class="unit">分钟</span>
              <div class="form-tip">限流后首次探测的等待时间</div>
            </el-form-item>

            <el-form-item label="最大探测间隔">
              <el-input-number
                v-model="configs.rate_limited_probe_max_interval"
                :min="10"
                :max="120"
                :disabled="!healthCheckEnabled || !rateLimitedProbeEnabled"
              />
              <span class="unit">分钟</span>
              <div class="form-tip">探测间隔递增的上限</div>
            </el-form-item>

            <el-form-item label="间隔递增因子">
              <el-input-number
                v-model="configs.rate_limited_probe_backoff"
                :min="1"
                :max="3"
                :step="0.1"
                :precision="1"
                :disabled="!healthCheckEnabled || !rateLimitedProbeEnabled"
              />
              <div class="form-tip">每次失败后间隔乘以此因子（如 1.5 表示每次增加 50%）</div>
            </el-form-item>

            <el-divider content-position="left">疑似封号检测</el-divider>

            <el-form-item label="探测间隔">
              <el-input-number
                v-model="configs.suspended_probe_interval"
                :min="1"
                :max="60"
                :disabled="!healthCheckEnabled"
              />
              <span class="unit">分钟</span>
              <div class="form-tip">疑似封号账号的探测间隔</div>
            </el-form-item>

            <el-form-item label="确认封号阈值">
              <el-input-number
                v-model="configs.suspended_confirm_threshold"
                :min="1"
                :max="10"
                :disabled="!healthCheckEnabled"
              />
              <span class="unit">次</span>
              <div class="form-tip">连续失败此次数后确认为封号状态</div>
            </el-form-item>

            <el-divider content-position="left">封号账号复活检测</el-divider>

            <el-form-item label="启用复活检测">
              <el-switch v-model="bannedProbeEnabled" :disabled="!healthCheckEnabled" />
              <div class="form-tip">定期检测已封号账号是否恢复</div>
            </el-form-item>

            <el-form-item label="复活探测间隔">
              <el-input-number
                v-model="configs.banned_probe_interval"
                :min="1"
                :max="24"
                :disabled="!healthCheckEnabled || !bannedProbeEnabled"
              />
              <span class="unit">小时</span>
              <div class="form-tip">封号账号的复活检测间隔</div>
            </el-form-item>

            <el-divider content-position="left">Token 刷新</el-divider>

            <el-form-item label="刷新冷却时间">
              <el-input-number
                v-model="configs.token_refresh_cooldown"
                :min="5"
                :max="120"
                :disabled="!healthCheckEnabled"
              />
              <span class="unit">分钟</span>
              <div class="form-tip">Token 刷新失败后的冷却时间</div>
            </el-form-item>

            <el-form-item label="最大重试次数">
              <el-input-number
                v-model="configs.token_refresh_max_retries"
                :min="1"
                :max="10"
                :disabled="!healthCheckEnabled"
              />
              <span class="unit">次</span>
              <div class="form-tip">Token 刷新的最大重试次数</div>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <!-- 所有配置项 -->
      <el-col :span="24">
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <span>配置项列表</span>
            </div>
          </template>

          <el-table :data="configList" stripe size="small" max-height="300">
            <el-table-column prop="key" label="配置项" width="260" />
            <el-table-column prop="value" label="当前值" width="100" />
            <el-table-column prop="desc" label="说明" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 健康检测状态弹窗 -->
    <el-dialog v-model="healthStatusVisible" title="健康检测服务状态" width="600px">
      <el-descriptions :column="2" border v-if="healthStatus">
        <el-descriptions-item label="服务状态">
          <el-tag :type="healthStatus.running ? 'success' : 'danger'">
            {{ healthStatus.running ? '运行中' : '已停止' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="检测间隔">
          {{ healthStatus.interval }} 分钟
        </el-descriptions-item>
        <el-descriptions-item label="上次检测">
          {{ healthStatus.last_check ? formatDate(healthStatus.last_check) : '暂无' }}
        </el-descriptions-item>
        <el-descriptions-item label="问题账号数">
          <el-tag type="warning">{{ healthStatus.problem_account_count || 0 }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="已检测账号数">
          {{ healthStatus.checked_count || 0 }}
        </el-descriptions-item>
        <el-descriptions-item label="失败账号数">
          <el-tag type="danger" v-if="healthStatus.failed_count > 0">{{ healthStatus.failed_count }}</el-tag>
          <span v-else>0</span>
        </el-descriptions-item>
        <el-descriptions-item label="错误阈值">
          {{ healthStatus.threshold }} 次
        </el-descriptions-item>
        <el-descriptions-item label="最后错误">
          <span class="error-text">{{ healthStatus.last_error || '无' }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <div style="margin-top: 20px; text-align: center;">
        <el-button type="primary" @click="triggerHealthCheck" :loading="healthChecking">
          手动触发检测
        </el-button>
        <el-button @click="refreshHealthStatus">刷新状态</el-button>
      </div>
    </el-dialog>

    <!-- 保存按钮 -->
    <div class="action-bar">
      <el-button type="primary" size="large" @click="saveConfigs" :loading="saving">
        保存配置
      </el-button>
      <el-button size="large" @click="loadConfigs">重置</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const saving = ref(false)
const healthStatusVisible = ref(false)
const healthChecking = ref(false)
const healthStatus = ref(null)

const configs = reactive({
  record_retention_days: 30,
  record_max_count: 1000,
  // 计费配置
  global_price_rate: 1,
  // 安全配置
  captcha_enabled: 'true',
  captcha_rate_limit: 10,
  login_rate_limit_enable: 'true',
  login_rate_limit_count: 3,
  login_rate_limit_window: 5,
  // 账号健康检查配置
  account_health_check_enabled: 'false',
  account_health_check_interval: 5,
  account_error_threshold: 5,
  // OAuth 自动重新授权配置
  oauth_auto_reauthorize_enabled: 'true',
  oauth_reauthorize_cooldown: 30,
  // 健康检测策略配置
  health_check_auto_recovery: 'true',
  health_check_auto_token_refresh: 'true',
  // 限流账号探测
  rate_limited_probe_enabled: 'true',
  rate_limited_probe_init_interval: 10,
  rate_limited_probe_max_interval: 30,
  rate_limited_probe_backoff: 1.5,
  // 疑似封号检测
  suspended_probe_interval: 5,
  suspended_confirm_threshold: 3,
  // 封号账号复活检测
  banned_probe_enabled: 'false',
  banned_probe_interval: 1,
  // Token 刷新
  token_refresh_cooldown: 30,
  token_refresh_max_retries: 3
})

const configList = ref([])

const captchaEnabled = computed({
  get: () => configs.captcha_enabled === 'true',
  set: (val) => { configs.captcha_enabled = val ? 'true' : 'false' }
})

const loginRateLimitEnabled = computed({
  get: () => configs.login_rate_limit_enable === 'true',
  set: (val) => { configs.login_rate_limit_enable = val ? 'true' : 'false' }
})

const healthCheckEnabled = computed({
  get: () => configs.account_health_check_enabled === 'true',
  set: (val) => { configs.account_health_check_enabled = val ? 'true' : 'false' }
})

const oauthAutoReauthorizeEnabled = computed({
  get: () => configs.oauth_auto_reauthorize_enabled === 'true',
  set: (val) => { configs.oauth_auto_reauthorize_enabled = val ? 'true' : 'false' }
})

const healthCheckAutoRecovery = computed({
  get: () => configs.health_check_auto_recovery === 'true',
  set: (val) => { configs.health_check_auto_recovery = val ? 'true' : 'false' }
})

const healthCheckAutoTokenRefresh = computed({
  get: () => configs.health_check_auto_token_refresh === 'true',
  set: (val) => { configs.health_check_auto_token_refresh = val ? 'true' : 'false' }
})

const rateLimitedProbeEnabled = computed({
  get: () => configs.rate_limited_probe_enabled === 'true',
  set: (val) => { configs.rate_limited_probe_enabled = val ? 'true' : 'false' }
})

const bannedProbeEnabled = computed({
  get: () => configs.banned_probe_enabled === 'true',
  set: (val) => { configs.banned_probe_enabled = val ? 'true' : 'false' }
})

function formatDate(str) {
  if (!str) return ''
  return new Date(str).toLocaleString('zh-CN')
}

async function loadConfigs() {
  loading.value = true
  try {
    const res = await api.getSystemConfigs()
    configList.value = res.items || []

    // 填充表单
    for (const cfg of configList.value) {
      if (cfg.key in configs) {
        if (cfg.type === 'int') {
          configs[cfg.key] = parseInt(cfg.value) || 0
        } else if (cfg.type === 'float') {
          configs[cfg.key] = parseFloat(cfg.value) || 0
        } else {
          configs[cfg.key] = cfg.value
        }
      }
    }
  } catch (e) {
    // handled
  } finally {
    loading.value = false
  }
}

async function saveConfigs() {
  saving.value = true
  try {
    const toSave = {
      record_retention_days: String(configs.record_retention_days),
      record_max_count: String(configs.record_max_count),
      // 计费配置
      global_price_rate: String(configs.global_price_rate),
      // 安全配置
      captcha_enabled: configs.captcha_enabled,
      captcha_rate_limit: String(configs.captcha_rate_limit),
      login_rate_limit_enable: configs.login_rate_limit_enable,
      login_rate_limit_count: String(configs.login_rate_limit_count),
      login_rate_limit_window: String(configs.login_rate_limit_window),
      // 账号健康检查配置
      account_health_check_enabled: configs.account_health_check_enabled,
      account_health_check_interval: String(configs.account_health_check_interval),
      account_error_threshold: String(configs.account_error_threshold),
      // OAuth 自动重新授权配置
      oauth_auto_reauthorize_enabled: configs.oauth_auto_reauthorize_enabled,
      oauth_reauthorize_cooldown: String(configs.oauth_reauthorize_cooldown),
      // 健康检测策略配置
      health_check_auto_recovery: configs.health_check_auto_recovery,
      health_check_auto_token_refresh: configs.health_check_auto_token_refresh,
      // 限流账号探测
      rate_limited_probe_enabled: configs.rate_limited_probe_enabled,
      rate_limited_probe_init_interval: String(configs.rate_limited_probe_init_interval),
      rate_limited_probe_max_interval: String(configs.rate_limited_probe_max_interval),
      rate_limited_probe_backoff: String(configs.rate_limited_probe_backoff),
      // 疑似封号检测
      suspended_probe_interval: String(configs.suspended_probe_interval),
      suspended_confirm_threshold: String(configs.suspended_confirm_threshold),
      // 封号账号复活检测
      banned_probe_enabled: configs.banned_probe_enabled,
      banned_probe_interval: String(configs.banned_probe_interval),
      // Token 刷新
      token_refresh_cooldown: String(configs.token_refresh_cooldown),
      token_refresh_max_retries: String(configs.token_refresh_max_retries)
    }
    await api.updateSystemConfigs(toSave)
    ElMessage.success('配置保存成功')

    // 刷新配置
    await loadConfigs()
  } catch (e) {
    // handled
  } finally {
    saving.value = false
  }
}

async function viewHealthCheckStatus() {
  healthStatusVisible.value = true
  await refreshHealthStatus()
}

async function refreshHealthStatus() {
  try {
    const res = await api.getHealthCheckStatus()
    healthStatus.value = res
  } catch (e) {
    // handled
  }
}

let healthStatusTimer = null
async function triggerHealthCheck() {
  healthChecking.value = true
  try {
    await api.triggerHealthCheck()
    ElMessage.success('健康检测已触发')
    // 延迟刷新状态
    if (healthStatusTimer) clearTimeout(healthStatusTimer)
    healthStatusTimer = setTimeout(refreshHealthStatus, 2000)
  } catch (e) {
    // handled
  } finally {
    healthChecking.value = false
  }
}
onUnmounted(() => {
  if (healthStatusTimer) {
    clearTimeout(healthStatusTimer)
    healthStatusTimer = null
  }
})

onMounted(() => {
  loadConfigs()
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

.config-card {
  height: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.unit {
  margin-left: 10px;
  color: #909399;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.text-muted {
  color: #c0c4cc;
}

.action-bar {
  margin-top: 30px;
  padding: 20px;
  background: #f5f7fa;
  border-radius: 4px;
  text-align: center;
}

.header-actions {
  display: flex;
  align-items: center;
}

.error-text {
  color: #f56c6c;
  font-size: 12px;
}
</style>
