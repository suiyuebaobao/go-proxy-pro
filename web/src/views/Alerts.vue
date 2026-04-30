<!--
 * 文件作用：告警管理页面，配置告警规则和查看告警历史
 * 负责功能：
 *   - 告警规则列表与 CRUD 操作
 *   - 支持 Telegram / Webhook / 邮件三种通知渠道配置
 *   - 告警条件配置（账户封禁/限流/配额耗尽/系统资源超限）
 *   - 静默期设置
 *   - 测试发送功能
 *   - 告警历史日志查看
 * 重要程度：⭐⭐⭐⭐ 重要（运维监控入口）
 * 依赖组件：Element Plus, api/index.js
-->
<template>
  <div class="alerts-page">
    <div class="page-header">
      <h2>告警管理</h2>
      <el-button type="primary" @click="openCreateDialog"><el-icon><Plus /></el-icon> 新建规则</el-button>
    </div>

    <el-tabs v-model="activeTab">
      <!-- 告警规则 -->
      <el-tab-pane label="告警规则" name="rules">
        <el-table :data="rules" v-loading="loadingRules" stripe>
          <el-table-column prop="name" label="规则名称" min-width="150" />
          <el-table-column label="触发条件" min-width="140">
            <template #default="{ row }">
              <el-tag size="small">{{ conditionLabel(row.condition_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="通知渠道" width="120">
            <template #default="{ row }">
              <el-tag :type="channelTagType(row.channel_type)" size="small">{{ channelLabel(row.channel_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="静默期" width="100">
            <template #default="{ row }">{{ row.silence_minutes }} 分钟</template>
          </el-table-column>
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-switch v-model="row.enabled" @change="toggleRule(row)" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="testRule(row)">测试</el-button>
              <el-button link type="primary" size="small" @click="editRule(row)">编辑</el-button>
              <el-button link type="danger" size="small" @click="deleteRule(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="rules.length === 0 && !loadingRules" description="暂无告警规则" />
      </el-tab-pane>

      <!-- 告警历史 -->
      <el-tab-pane label="告警历史" name="logs">
        <el-table :data="alertLogs" v-loading="loadingLogs" stripe>
          <el-table-column prop="rule_name" label="规则" min-width="140" show-overflow-tooltip />
          <el-table-column label="类型" width="140">
            <template #default="{ row }">
              <el-tag size="small">{{ conditionLabel(row.alert_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="渠道" width="100">
            <template #default="{ row }">
              <el-tag :type="channelTagType(row.channel)" size="small">{{ channelLabel(row.channel) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="message" label="消息" min-width="250" show-overflow-tooltip />
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 'sent' ? 'success' : 'danger'" size="small">{{ row.status === 'sent' ? '成功' : '失败' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="时间" width="170">
            <template #default="{ row }">{{ new Date(row.created_at).toLocaleString('zh-CN') }}</template>
          </el-table-column>
        </el-table>
        <div class="pagination-wrap" v-if="logTotal > 0">
          <el-pagination v-model:current-page="logPage" v-model:page-size="logPageSize" :total="logTotal"
            :page-sizes="[20, 50, 100]" layout="total, sizes, prev, pager, next" @change="fetchLogs" />
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 创建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="editingRule ? '编辑告警规则' : '新建告警规则'" width="600px" destroy-on-close>
      <el-form :model="form" label-width="100px">
        <el-form-item label="规则名称" required>
          <el-input v-model="form.name" placeholder="如：CPU 使用率过高告警" />
        </el-form-item>
        <el-form-item label="触发条件" required>
          <el-select v-model="form.condition_type" style="width: 100%">
            <el-option v-for="c in conditionTypes" :key="c.value" :label="c.label" :value="c.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="needsThreshold" label="阈值(%)">
          <el-input-number v-model="thresholdValue" :min="1" :max="100" />
        </el-form-item>
        <el-form-item label="通知渠道" required>
          <el-select v-model="form.channel_type" style="width: 100%" @change="onChannelChange">
            <el-option label="Telegram" value="telegram" />
            <el-option label="Webhook" value="webhook" />
            <el-option label="邮件 (SMTP)" value="email" />
          </el-select>
        </el-form-item>

        <!-- Telegram Config -->
        <template v-if="form.channel_type === 'telegram'">
          <el-form-item label="Bot Token" required>
            <el-input v-model="channelForm.bot_token" placeholder="从 @BotFather 获取" />
          </el-form-item>
          <el-form-item label="Chat ID" required>
            <el-input v-model="channelForm.chat_id" placeholder="目标聊天 ID" />
          </el-form-item>
        </template>

        <!-- Webhook Config -->
        <template v-if="form.channel_type === 'webhook'">
          <el-form-item label="URL" required>
            <el-input v-model="channelForm.url" placeholder="https://..." />
          </el-form-item>
        </template>

        <!-- Email Config -->
        <template v-if="form.channel_type === 'email'">
          <el-form-item label="SMTP 服务器" required>
            <el-input v-model="channelForm.smtp_host" placeholder="smtp.example.com" />
          </el-form-item>
          <el-form-item label="端口" required>
            <el-input v-model="channelForm.smtp_port" placeholder="587" style="width: 120px" />
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="channelForm.username" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="channelForm.password" type="password" show-password />
          </el-form-item>
          <el-form-item label="收件人" required>
            <el-input v-model="channelForm.to" placeholder="逗号分隔多个邮箱" />
          </el-form-item>
        </template>

        <el-form-item label="静默期">
          <el-input-number v-model="form.silence_minutes" :min="1" :max="1440" />
          <span style="margin-left: 8px; color: var(--pink-text-secondary)">分钟</span>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRule" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '@/api'

const activeTab = ref('rules')
const rules = ref([])
const loadingRules = ref(false)

const alertLogs = ref([])
const loadingLogs = ref(false)
const logPage = ref(1)
const logPageSize = ref(20)
const logTotal = ref(0)

const dialogVisible = ref(false)
const editingRule = ref(null)
const saving = ref(false)

const conditionTypes = [
  { value: 'account_banned', label: '账户被封禁' },
  { value: 'rate_limited', label: '账户被限速' },
  { value: 'quota_exhausted', label: '用户配额/次数耗尽' },
  { value: 'cpu_high', label: 'CPU 使用率过高' },
  { value: 'memory_high', label: '内存使用率过高' },
  { value: 'disk_high', label: '磁盘使用率过高' },
  { value: 'error_spike', label: '错误请求激增' },
]

const form = reactive({
  name: '',
  condition_type: 'account_banned',
  channel_type: 'telegram',
  silence_minutes: 30,
  enabled: true
})

const channelForm = reactive({
  bot_token: '', chat_id: '',
  url: '',
  smtp_host: '', smtp_port: '587', username: '', password: '', from: '', to: ''
})

const thresholdValue = ref(90)

const needsThreshold = computed(() => ['cpu_high', 'memory_high', 'disk_high'].includes(form.condition_type))

function conditionLabel(type) {
  return conditionTypes.find(c => c.value === type)?.label || type
}

function channelLabel(type) {
  return { telegram: 'Telegram', webhook: 'Webhook', email: '邮件' }[type] || type
}

function channelTagType(type) {
  return { telegram: '', webhook: 'warning', email: 'success' }[type] || 'info'
}

function onChannelChange() {
  Object.assign(channelForm, { bot_token: '', chat_id: '', url: '', smtp_host: '', smtp_port: '587', username: '', password: '', from: '', to: '' })
}

function buildPayload() {
  const conditionValue = needsThreshold.value ? JSON.stringify({ threshold: thresholdValue.value }) : '{}'
  const channelConfig = JSON.stringify(
    form.channel_type === 'telegram' ? { bot_token: channelForm.bot_token, chat_id: channelForm.chat_id } :
    form.channel_type === 'webhook' ? { url: channelForm.url } :
    { smtp_host: channelForm.smtp_host, smtp_port: channelForm.smtp_port, username: channelForm.username, password: channelForm.password, from: channelForm.from, to: channelForm.to }
  )
  return { ...form, condition_value: conditionValue, channel_config: channelConfig }
}

function openCreateDialog() {
  editingRule.value = null
  Object.assign(form, { name: '', condition_type: 'account_banned', channel_type: 'telegram', silence_minutes: 30, enabled: true })
  onChannelChange()
  thresholdValue.value = 90
  dialogVisible.value = true
}

function editRule(rule) {
  editingRule.value = rule
  Object.assign(form, { name: rule.name, condition_type: rule.condition_type, channel_type: rule.channel_type, silence_minutes: rule.silence_minutes, enabled: rule.enabled })
  try {
    const cv = JSON.parse(rule.condition_value || '{}')
    if (cv.threshold) thresholdValue.value = cv.threshold
  } catch { /* ignore */ }
  try {
    const cc = JSON.parse(rule.channel_config || '{}')
    Object.assign(channelForm, { bot_token: '', chat_id: '', url: '', smtp_host: '', smtp_port: '587', username: '', password: '', from: '', to: '', ...cc })
  } catch { /* ignore */ }
  dialogVisible.value = true
}

async function saveRule() {
  if (!form.name || !form.condition_type || !form.channel_type) {
    ElMessage.warning('请填写必填字段')
    return
  }
  saving.value = true
  try {
    const payload = buildPayload()
    if (editingRule.value) {
      await api.updateAlertRule(editingRule.value.id, payload)
      ElMessage.success('更新成功')
    } else {
      await api.createAlertRule(payload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchRules()
  } catch (e) {
    ElMessage.error('保存失败: ' + (e.message || e))
  } finally {
    saving.value = false
  }
}

async function toggleRule(rule) {
  try {
    await api.updateAlertRule(rule.id, { ...rule })
  } catch (e) {
    rule.enabled = !rule.enabled
    ElMessage.error('操作失败')
  }
}

async function deleteRule(rule) {
  try {
    await ElMessageBox.confirm(`确定删除规则 "${rule.name}"？`, '删除确认', { type: 'warning' })
    await api.deleteAlertRule(rule.id)
    ElMessage.success('已删除')
    fetchRules()
  } catch { /* cancelled */ }
}

async function testRule(rule) {
  try {
    await api.testAlertRule(rule.id)
    ElMessage.success('测试发送成功')
  } catch (e) {
    ElMessage.error('测试发送失败: ' + (e.message || e))
  }
}

async function fetchRules() {
  loadingRules.value = true
  try {
    const res = await api.getAlertRules()
    rules.value = res.data?.items || []
  } catch (e) {
    console.error('Failed to fetch rules:', e)
  } finally {
    loadingRules.value = false
  }
}

async function fetchLogs() {
  loadingLogs.value = true
  try {
    const res = await api.getAlertLogs({ page: logPage.value, page_size: logPageSize.value })
    const data = res.data || {}
    alertLogs.value = data.items || []
    logTotal.value = data.total || 0
  } catch (e) {
    console.error('Failed to fetch alert logs:', e)
  } finally {
    loadingLogs.value = false
  }
}

onMounted(() => {
  fetchRules()
  fetchLogs()
})
</script>

<style scoped>
.alerts-page .page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.alerts-page .page-header h2 {
  color: var(--pink-text);
  margin: 0;
}

.pagination-wrap {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
