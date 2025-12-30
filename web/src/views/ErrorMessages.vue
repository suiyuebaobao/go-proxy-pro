<!--
 * 文件作用：错误配置页面，管理错误消息和规则
 * 负责功能：
 *   - 错误消息CRUD和状态管理
 *   - 错误规则CRUD和优先级配置
 *   - 缓存刷新和批量操作
 *   - 自动账户禁用/限流规则
 * 重要程度：⭐⭐⭐ 一般（错误处理配置）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="error-config-page">
    <div class="page-header">
      <h2>错误配置</h2>
    </div>

    <el-tabs v-model="activeTab" type="border-card">
      <!-- Tab 1: 错误消息配置 -->
      <el-tab-pane label="错误消息" name="messages">
        <div class="tab-header">
          <el-button type="success" @click="enableAllMessages" :loading="enablingAll" size="small">全部启用</el-button>
          <el-button type="warning" @click="disableAllMessages" :loading="disablingAll" size="small">全部禁用</el-button>
          <el-divider direction="vertical" />
          <el-button type="primary" @click="showCreateMessageDialog" size="small">新增配置</el-button>
          <el-button @click="initMessageDefaults" :loading="initializingMessages" size="small">初始化默认</el-button>
          <el-button @click="refreshMessageCache" :loading="refreshingMessages" size="small">刷新缓存</el-button>
        </div>

        <el-table :data="messages" v-loading="loadingMessages" stripe style="margin-top: 15px">
          <el-table-column prop="code" label="状态码" width="100">
            <template #default="{ row }">
              <el-tag :type="getCodeType(row.code)">{{ row.code }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="error_type" label="错误类型" width="200" />
          <el-table-column prop="original_message" label="原始信息" min-width="150" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="muted-text">{{ row.original_message || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="custom_message" label="自定义消息" min-width="200" show-overflow-tooltip />
          <el-table-column prop="description" label="说明" min-width="150" show-overflow-tooltip />
          <el-table-column prop="enabled" label="状态" width="80">
            <template #default="{ row }">
              <el-switch v-model="row.enabled" @change="toggleMessageEnabled(row)" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="showEditMessageDialog(row)">编辑</el-button>
              <el-popconfirm title="确定删除该配置?" @confirm="deleteMessage(row.id)">
                <template #reference>
                  <el-button type="danger" link>删除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- Tab 2: 错误规则配置 -->
      <el-tab-pane label="错误规则" name="rules">
        <div class="tab-header">
          <el-button type="success" @click="enableAllRules" :loading="enablingAllRules" size="small">全部启用</el-button>
          <el-button type="warning" @click="disableAllRules" :loading="disablingAllRules" size="small">全部禁用</el-button>
          <el-divider direction="vertical" />
          <el-button type="primary" @click="showCreateRuleDialog" size="small">新增规则</el-button>
          <el-button @click="resetRulesToDefault" :loading="resettingRules" size="small">重置为默认</el-button>
          <el-button @click="refreshRuleCache" :loading="refreshingRules" size="small">刷新缓存</el-button>
        </div>

        <!-- 简化说明 -->
        <div class="rule-guide">
          <div class="guide-title">规则说明</div>
          <div class="guide-content">
            <div class="guide-item">
              <el-tag type="danger" size="small">禁用账户</el-tag>
              <span>账户被封号、认证失效时，自动禁用该账户</span>
            </div>
            <div class="guide-item">
              <el-tag type="warning" size="small">临时限流</el-tag>
              <span>遇到限流或临时错误，切换其他账户重试（1小时后恢复）</span>
            </div>
            <div class="guide-item">
              <el-tag type="info" size="small">过载</el-tag>
              <span>服务过载，临时切换账户</span>
            </div>
          </div>
        </div>

        <el-table :data="rules" v-loading="loadingRules" stripe>
          <el-table-column prop="target_status" label="处理方式" width="130">
            <template #default="{ row }">
              <el-tag :type="getTargetStatusType(row.target_status)" effect="dark">
                {{ getTargetStatusLabel(row.target_status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="匹配条件" min-width="280">
            <template #default="{ row }">
              <div class="match-condition">
                <span v-if="row.http_status_code > 0" class="condition-item">
                  <el-tag size="small" :type="getCodeType(row.http_status_code)">HTTP {{ row.http_status_code }}</el-tag>
                </span>
                <span v-if="row.http_status_code > 0 && row.keyword" class="condition-sep">+</span>
                <span v-if="row.keyword" class="condition-item">
                  <code class="keyword-code">{{ row.keyword }}</code>
                </span>
                <span v-if="!row.http_status_code && !row.keyword" class="muted-text">匹配所有错误</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="说明" min-width="180" show-overflow-tooltip />
          <el-table-column prop="priority" label="优先级" width="80" sortable>
            <template #default="{ row }">
              <span class="priority-num">{{ row.priority }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="enabled" label="启用" width="70">
            <template #default="{ row }">
              <el-switch v-model="row.enabled" @change="toggleRuleEnabled(row)" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link size="small" @click="showEditRuleDialog(row)">编辑</el-button>
              <el-popconfirm title="确定删除?" @confirm="deleteRule(row.id)">
                <template #reference>
                  <el-button type="danger" link size="small">删除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-container">
          <el-pagination
            v-model:current-page="rulePagination.page"
            v-model:page-size="rulePagination.pageSize"
            :total="rulePagination.total"
            :page-sizes="[20, 50, 100]"
            layout="total, sizes, prev, pager, next"
            @size-change="loadRules"
            @current-change="loadRules"
          />
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 错误消息对话框 -->
    <el-dialog v-model="messageDialogVisible" :title="isEditMessage ? '编辑错误消息' : '新增错误消息'" width="500px">
      <el-form ref="messageFormRef" :model="messageForm" :rules="messageFormRules" label-width="100px">
        <el-form-item label="状态码" prop="code">
          <el-select v-model="messageForm.code" :disabled="isEditMessage" style="width: 100%">
            <el-option :value="400" label="400 - Bad Request" />
            <el-option :value="401" label="401 - Unauthorized" />
            <el-option :value="403" label="403 - Forbidden" />
            <el-option :value="429" label="429 - Too Many Requests" />
            <el-option :value="500" label="500 - Internal Server Error" />
            <el-option :value="502" label="502 - Bad Gateway" />
            <el-option :value="503" label="503 - Service Unavailable" />
          </el-select>
        </el-form-item>
        <el-form-item label="错误类型" prop="error_type">
          <el-input v-model="messageForm.error_type" :disabled="isEditMessage" placeholder="如: auth_failed" />
        </el-form-item>
        <el-form-item label="自定义消息" prop="custom_message">
          <el-input v-model="messageForm.custom_message" type="textarea" :rows="3" placeholder="返回给用户的错误消息" />
        </el-form-item>
        <el-form-item label="说明" prop="description">
          <el-input v-model="messageForm.description" placeholder="管理员备注（可选）" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="messageForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="messageDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitMessage" :loading="submittingMessage">确定</el-button>
      </template>
    </el-dialog>

    <!-- 错误规则对话框 -->
    <el-dialog v-model="ruleDialogVisible" :title="isEditRule ? '编辑规则' : '新增规则'" width="550px">
      <el-form ref="ruleFormRef" :model="ruleForm" :rules="ruleFormRules" label-width="110px">
        <el-form-item label="HTTP状态码" prop="http_status_code">
          <el-input-number v-model="ruleForm.http_status_code" :min="0" :max="599" :controls="false" placeholder="0" style="width: 120px" />
          <span class="form-tip">0 表示匹配任意状态码</span>
        </el-form-item>
        <el-form-item label="错误关键词" prop="keyword">
          <el-input v-model="ruleForm.keyword" placeholder="匹配错误消息中的关键词（不区分大小写，留空表示任意）" />
        </el-form-item>
        <el-form-item label="目标状态" prop="target_status">
          <el-select v-model="ruleForm.target_status" placeholder="匹配后将账户标记为" style="width: 100%">
            <el-option value="valid" label="valid - 不修改（仅记录错误）" />
            <el-option value="invalid" label="invalid - 无效（禁用账户）" />
            <el-option value="rate_limited" label="rate_limited - 限流（1小时后恢复）" />
            <el-option value="overloaded" label="overloaded - 过载（临时不可用）" />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级" prop="priority">
          <el-input-number v-model="ruleForm.priority" :min="0" :max="1000" />
          <span class="form-tip">数值越大优先级越高</span>
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="ruleForm.description" placeholder="规则说明（可选）" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRule" :loading="submittingRule">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'

const activeTab = ref('messages')

// ==================== 错误消息相关 ====================
const loadingMessages = ref(false)
const initializingMessages = ref(false)
const refreshingMessages = ref(false)
const submittingMessage = ref(false)
const enablingAll = ref(false)
const disablingAll = ref(false)
const messageDialogVisible = ref(false)
const isEditMessage = ref(false)
const messages = ref([])
const messageFormRef = ref()

const messageForm = reactive({
  id: null,
  code: 400,
  error_type: '',
  custom_message: '',
  description: '',
  enabled: true
})

const messageFormRules = {
  code: [{ required: true, message: '请选择状态码', trigger: 'change' }],
  error_type: [{ required: true, message: '请输入错误类型', trigger: 'blur' }],
  custom_message: [{ required: true, message: '请输入自定义消息', trigger: 'blur' }]
}

async function loadMessages() {
  loadingMessages.value = true
  try {
    const res = await api.getErrorMessages()
    messages.value = res.data || res || []
  } catch (e) {}
  finally { loadingMessages.value = false }
}

function showCreateMessageDialog() {
  isEditMessage.value = false
  Object.assign(messageForm, { id: null, code: 400, error_type: '', custom_message: '', description: '', enabled: true })
  messageDialogVisible.value = true
}

function showEditMessageDialog(row) {
  isEditMessage.value = true
  Object.assign(messageForm, { id: row.id, code: row.code, error_type: row.error_type, custom_message: row.custom_message, description: row.description || '', enabled: row.enabled })
  messageDialogVisible.value = true
}

async function submitMessage() {
  const valid = await messageFormRef.value.validate().catch(() => false)
  if (!valid) return
  submittingMessage.value = true
  try {
    if (isEditMessage.value) {
      await api.updateErrorMessage(messageForm.id, { custom_message: messageForm.custom_message, enabled: messageForm.enabled, description: messageForm.description })
      ElMessage.success('更新成功')
    } else {
      await api.createErrorMessage({ code: messageForm.code, error_type: messageForm.error_type, custom_message: messageForm.custom_message, enabled: messageForm.enabled, description: messageForm.description })
      ElMessage.success('创建成功')
    }
    messageDialogVisible.value = false
    loadMessages()
  } catch (e) {}
  finally { submittingMessage.value = false }
}

async function toggleMessageEnabled(row) {
  try { await api.toggleErrorMessage(row.id); ElMessage.success('状态已更新') }
  catch (e) { row.enabled = !row.enabled }
}

async function deleteMessage(id) {
  try { await api.deleteErrorMessage(id); ElMessage.success('删除成功'); loadMessages() }
  catch (e) {}
}

async function initMessageDefaults() {
  initializingMessages.value = true
  try { await api.initErrorMessages(); ElMessage.success('默认配置已初始化'); loadMessages() }
  catch (e) {}
  finally { initializingMessages.value = false }
}

async function refreshMessageCache() {
  refreshingMessages.value = true
  try { await api.refreshErrorMessages(); ElMessage.success('缓存已刷新') }
  catch (e) {}
  finally { refreshingMessages.value = false }
}

async function enableAllMessages() {
  enablingAll.value = true
  try {
    const res = await api.enableAllErrorMessages()
    ElMessage.success(`已启用所有配置（影响 ${res.data?.affected || 0} 条）`)
    loadMessages()
  } catch (e) {}
  finally { enablingAll.value = false }
}

async function disableAllMessages() {
  disablingAll.value = true
  try {
    const res = await api.disableAllErrorMessages()
    ElMessage.success(`已禁用所有配置（影响 ${res.data?.affected || 0} 条）`)
    loadMessages()
  } catch (e) {}
  finally { disablingAll.value = false }
}

// ==================== 错误规则相关 ====================
const loadingRules = ref(false)
const resettingRules = ref(false)
const refreshingRules = ref(false)
const submittingRule = ref(false)
const enablingAllRules = ref(false)
const disablingAllRules = ref(false)
const ruleDialogVisible = ref(false)
const isEditRule = ref(false)
const rules = ref([])
const ruleFormRef = ref()

const rulePagination = reactive({ page: 1, pageSize: 50, total: 0 })

const ruleForm = reactive({
  id: null,
  http_status_code: 0,
  keyword: '',
  target_status: 'valid',
  priority: 50,
  description: '',
  enabled: true
})

const ruleFormRules = {
  target_status: [{ required: true, message: '请选择目标状态', trigger: 'change' }]
}

async function loadRules() {
  loadingRules.value = true
  try {
    const res = await api.getErrorRules({ page: rulePagination.page, page_size: rulePagination.pageSize })
    rules.value = res.data?.items || []
    rulePagination.total = res.data?.total || 0
  } catch (e) {}
  finally { loadingRules.value = false }
}

function showCreateRuleDialog() {
  isEditRule.value = false
  Object.assign(ruleForm, { id: null, http_status_code: 0, keyword: '', target_status: 'valid', priority: 50, description: '', enabled: true })
  ruleDialogVisible.value = true
}

function showEditRuleDialog(row) {
  isEditRule.value = true
  Object.assign(ruleForm, { id: row.id, http_status_code: row.http_status_code, keyword: row.keyword, target_status: row.target_status, priority: row.priority, description: row.description || '', enabled: row.enabled })
  ruleDialogVisible.value = true
}

async function submitRule() {
  const valid = await ruleFormRef.value.validate().catch(() => false)
  if (!valid) return
  submittingRule.value = true
  try {
    const data = { http_status_code: ruleForm.http_status_code || 0, keyword: ruleForm.keyword || '', target_status: ruleForm.target_status, priority: ruleForm.priority, description: ruleForm.description, enabled: ruleForm.enabled }
    if (isEditRule.value) {
      await api.updateErrorRule(ruleForm.id, data)
      ElMessage.success('更新成功')
    } else {
      await api.createErrorRule(data)
      ElMessage.success('创建成功')
    }
    ruleDialogVisible.value = false
    loadRules()
  } catch (e) {}
  finally { submittingRule.value = false }
}

async function toggleRuleEnabled(row) {
  try { await api.updateErrorRule(row.id, { enabled: row.enabled }); ElMessage.success('状态已更新') }
  catch (e) { row.enabled = !row.enabled }
}

async function deleteRule(id) {
  try { await api.deleteErrorRule(id); ElMessage.success('删除成功'); loadRules() }
  catch (e) {}
}

async function resetRulesToDefault() {
  try {
    await ElMessageBox.confirm('此操作将删除所有现有规则并恢复为默认规则，是否继续？', '确认重置', { type: 'warning' })
    resettingRules.value = true
    await api.resetErrorRules()
    ElMessage.success('已重置为默认规则')
    loadRules()
  } catch (e) {}
  finally { resettingRules.value = false }
}

async function refreshRuleCache() {
  refreshingRules.value = true
  try {
    const res = await api.refreshErrorRulesCache()
    ElMessage.success(`缓存已刷新，共 ${res.data?.rule_count || 0} 条规则`)
  } catch (e) {}
  finally { refreshingRules.value = false }
}

async function enableAllRules() {
  enablingAllRules.value = true
  try {
    const res = await api.enableAllErrorRules()
    ElMessage.success(`已启用所有规则（影响 ${res.data?.affected || 0} 条）`)
    loadRules()
  } catch (e) {}
  finally { enablingAllRules.value = false }
}

async function disableAllRules() {
  disablingAllRules.value = true
  try {
    const res = await api.disableAllErrorRules()
    ElMessage.success(`已禁用所有规则（影响 ${res.data?.affected || 0} 条）`)
    loadRules()
  } catch (e) {}
  finally { disablingAllRules.value = false }
}

// ==================== 通用方法 ====================
function getCodeType(code) {
  if (code >= 500) return 'danger'
  if (code === 429) return 'warning'
  if (code >= 400) return 'info'
  return ''
}

function getTargetStatusType(status) {
  switch (status) {
    case 'valid': return 'success'
    case 'invalid': return 'danger'
    case 'rate_limited': return 'warning'
    case 'overloaded': return 'info'
    default: return ''
  }
}

function getTargetStatusLabel(status) {
  switch (status) {
    case 'valid': return '忽略'
    case 'invalid': return '禁用账户'
    case 'rate_limited': return '临时限流'
    case 'overloaded': return '过载'
    default: return status
  }
}

// 切换 Tab 时加载数据
watch(activeTab, (val) => {
  if (val === 'messages' && messages.value.length === 0) loadMessages()
  if (val === 'rules' && rules.value.length === 0) loadRules()
})

onMounted(() => {
  loadMessages()
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

.tab-header {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.muted-text {
  color: #909399;
  font-size: 13px;
}

.keyword-code {
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 12px;
  color: #e6a23c;
}

.pagination-container {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.form-tip {
  margin-left: 10px;
  color: #909399;
  font-size: 12px;
}

.rule-guide {
  background: linear-gradient(135deg, #f0f9ff 0%, #e8f4fd 100%);
  border: 1px solid #d4e8f7;
  border-radius: 8px;
  padding: 15px 20px;
  margin: 15px 0;
}

.guide-title {
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
  font-size: 14px;
}

.guide-content {
  display: flex;
  gap: 30px;
  flex-wrap: wrap;
}

.guide-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #606266;
}

.match-condition {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.condition-sep {
  color: #909399;
  font-size: 12px;
}

.priority-num {
  color: #909399;
  font-size: 13px;
}
</style>
