<template>
  <div class="settings-page">
    <div class="page-header">
      <h2>系统设置</h2>
    </div>

    <el-row :gutter="20">
      <!-- 会话配置 -->
      <el-col :span="12">
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <span>会话配置</span>
            </div>
          </template>

          <el-form label-width="140px" v-loading="loading">
            <el-form-item label="会话粘性 TTL">
              <el-input-number
                v-model="configs.session_ttl"
                :min="1"
                :max="1440"
                :step="5"
              />
              <span class="unit">分钟</span>
              <div class="form-tip">会话绑定的过期时间，超时后重新分配账户</div>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 同步配置 -->
      <el-col :span="12">
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <span>同步配置</span>
              <el-tag :type="syncStatus.running ? 'success' : 'info'" size="small">
                {{ syncStatus.running ? '运行中' : '已停止' }}
              </el-tag>
            </div>
          </template>

          <el-form label-width="140px" v-loading="loading">
            <el-form-item label="启用同步">
              <el-switch v-model="syncEnabled" />
              <div class="form-tip">将 Redis 使用记录同步到 MySQL</div>
            </el-form-item>

            <el-form-item label="同步间隔">
              <el-input-number
                v-model="configs.sync_interval"
                :min="1"
                :max="60"
                :disabled="!syncEnabled"
              />
              <span class="unit">分钟</span>
            </el-form-item>

            <el-form-item label="上次同步">
              <span v-if="syncStatus.last_sync">{{ formatDate(syncStatus.last_sync) }}</span>
              <span v-else class="text-muted">暂无</span>
            </el-form-item>

            <el-form-item label="同步记录数">
              <span>{{ syncStatus.synced_count || 0 }}</span>
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="triggerSync" :loading="syncing">
                立即同步
              </el-button>
              <el-button @click="refreshSyncStatus">刷新状态</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
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
              <span>缓存记录配置</span>
            </div>
          </template>

          <el-form label-width="140px" v-loading="loading">
            <el-form-item label="Redis 保留天数">
              <el-input-number
                v-model="configs.record_retention_days"
                :min="1"
                :max="365"
              />
              <span class="unit">天</span>
              <div class="form-tip">Redis 中使用记录的保留时间</div>
            </el-form-item>

            <el-form-item label="最大记录数">
              <el-input-number
                v-model="configs.record_max_count"
                :min="100"
                :max="10000"
                :step="100"
              />
              <span class="unit">条/用户</span>
              <div class="form-tip">每个用户在 Redis 中保留的最大记录数</div>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <!-- 所有配置项 -->
      <el-col :span="12">
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <span>配置项列表</span>
            </div>
          </template>

          <el-table :data="configList" stripe size="small" max-height="300">
            <el-table-column prop="key" label="配置项" width="180" />
            <el-table-column prop="value" label="当前值" width="100" />
            <el-table-column prop="desc" label="说明" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

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
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const saving = ref(false)
const syncing = ref(false)

const configs = reactive({
  session_ttl: 30,
  sync_enabled: 'true',
  sync_interval: 5,
  record_retention_days: 30,
  record_max_count: 1000,
  // 安全配置
  captcha_enabled: 'true',
  captcha_rate_limit: 10,
  login_rate_limit_enable: 'true',
  login_rate_limit_count: 3,
  login_rate_limit_window: 5
})

const syncStatus = reactive({
  enabled: false,
  running: false,
  last_sync: null,
  synced_count: 0,
  interval: 5
})

const configList = ref([])

const syncEnabled = computed({
  get: () => configs.sync_enabled === 'true',
  set: (val) => { configs.sync_enabled = val ? 'true' : 'false' }
})

const captchaEnabled = computed({
  get: () => configs.captcha_enabled === 'true',
  set: (val) => { configs.captcha_enabled = val ? 'true' : 'false' }
})

const loginRateLimitEnabled = computed({
  get: () => configs.login_rate_limit_enable === 'true',
  set: (val) => { configs.login_rate_limit_enable = val ? 'true' : 'false' }
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
        } else {
          configs[cfg.key] = cfg.value
        }
      }
    }

    // 加载同步状态
    await refreshSyncStatus()
  } catch (e) {
    // handled
  } finally {
    loading.value = false
  }
}

async function refreshSyncStatus() {
  try {
    const res = await api.getSyncStatus()
    Object.assign(syncStatus, res)
  } catch (e) {
    // handled
  }
}

async function saveConfigs() {
  saving.value = true
  try {
    const toSave = {
      session_ttl: String(configs.session_ttl),
      sync_enabled: configs.sync_enabled,
      sync_interval: String(configs.sync_interval),
      record_retention_days: String(configs.record_retention_days),
      record_max_count: String(configs.record_max_count),
      // 安全配置
      captcha_enabled: configs.captcha_enabled,
      captcha_rate_limit: String(configs.captcha_rate_limit),
      login_rate_limit_enable: configs.login_rate_limit_enable,
      login_rate_limit_count: String(configs.login_rate_limit_count),
      login_rate_limit_window: String(configs.login_rate_limit_window)
    }
    await api.updateSystemConfigs(toSave)
    ElMessage.success('配置保存成功')

    // 刷新状态
    await refreshSyncStatus()
    await loadConfigs()
  } catch (e) {
    // handled
  } finally {
    saving.value = false
  }
}

async function triggerSync() {
  syncing.value = true
  try {
    await api.triggerSync()
    ElMessage.success('同步任务已触发')

    // 延迟刷新状态
    setTimeout(refreshSyncStatus, 2000)
  } catch (e) {
    // handled
  } finally {
    syncing.value = false
  }
}

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
</style>
