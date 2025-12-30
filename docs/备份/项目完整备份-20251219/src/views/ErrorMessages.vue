<template>
  <div class="error-messages-page">
    <div class="page-header">
      <h2>错误消息配置</h2>
      <div class="header-actions">
        <el-button type="success" @click="enableAll" :loading="enablingAll">全部启用</el-button>
        <el-button type="warning" @click="disableAll" :loading="disablingAll">全部禁用</el-button>
        <el-divider direction="vertical" />
        <el-button type="primary" @click="showCreateDialog">新增配置</el-button>
        <el-button @click="initDefaults" :loading="initializing">初始化默认配置</el-button>
        <el-button @click="refreshCache" :loading="refreshing">刷新缓存</el-button>
      </div>
    </div>

    <el-card>
      <el-table :data="messages" v-loading="loading" stripe>
        <el-table-column prop="code" label="状态码" width="100">
          <template #default="{ row }">
            <el-tag :type="getCodeType(row.code)">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error_type" label="错误类型" width="200" />
        <el-table-column prop="custom_message" label="自定义消息" min-width="200" show-overflow-tooltip />
        <el-table-column prop="description" label="说明" min-width="150" show-overflow-tooltip />
        <el-table-column prop="enabled" label="状态" width="100">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              @change="toggleEnabled(row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="showEditDialog(row)">编辑</el-button>
            <el-popconfirm title="确定删除该配置?" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button type="danger" link>删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑错误消息' : '新增错误消息'"
      width="500px"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="状态码" prop="code">
          <el-select v-model="form.code" :disabled="isEdit" style="width: 100%">
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
          <el-input v-model="form.error_type" :disabled="isEdit" placeholder="如: auth_failed" />
        </el-form-item>
        <el-form-item label="自定义消息" prop="custom_message">
          <el-input
            v-model="form.custom_message"
            type="textarea"
            :rows="3"
            placeholder="返回给用户的错误消息"
          />
        </el-form-item>
        <el-form-item label="说明" prop="description">
          <el-input v-model="form.description" placeholder="管理员备注（可选）" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const initializing = ref(false)
const refreshing = ref(false)
const submitting = ref(false)
const enablingAll = ref(false)
const disablingAll = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const messages = ref([])
const formRef = ref()

const form = reactive({
  id: null,
  code: 400,
  error_type: '',
  custom_message: '',
  description: '',
  enabled: true
})

const rules = {
  code: [{ required: true, message: '请选择状态码', trigger: 'change' }],
  error_type: [{ required: true, message: '请输入错误类型', trigger: 'blur' }],
  custom_message: [{ required: true, message: '请输入自定义消息', trigger: 'blur' }]
}

function getCodeType(code) {
  if (code >= 500) return 'danger'
  if (code >= 400) return 'warning'
  return 'info'
}

async function loadMessages() {
  loading.value = true
  try {
    const res = await api.getErrorMessages()
    messages.value = res.data || res || []
  } catch (e) {
    // handled
  } finally {
    loading.value = false
  }
}

function showCreateDialog() {
  isEdit.value = false
  Object.assign(form, {
    id: null,
    code: 400,
    error_type: '',
    custom_message: '',
    description: '',
    enabled: true
  })
  dialogVisible.value = true
}

function showEditDialog(row) {
  isEdit.value = true
  Object.assign(form, {
    id: row.id,
    code: row.code,
    error_type: row.error_type,
    custom_message: row.custom_message,
    description: row.description || '',
    enabled: row.enabled
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await api.updateErrorMessage(form.id, {
        custom_message: form.custom_message,
        enabled: form.enabled,
        description: form.description
      })
      ElMessage.success('更新成功')
    } else {
      await api.createErrorMessage({
        code: form.code,
        error_type: form.error_type,
        custom_message: form.custom_message,
        enabled: form.enabled,
        description: form.description
      })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadMessages()
  } catch (e) {
    // handled
  } finally {
    submitting.value = false
  }
}

async function toggleEnabled(row) {
  try {
    await api.toggleErrorMessage(row.id)
    ElMessage.success('状态已更新')
  } catch (e) {
    row.enabled = !row.enabled // 恢复状态
  }
}

async function handleDelete(id) {
  try {
    await api.deleteErrorMessage(id)
    ElMessage.success('删除成功')
    loadMessages()
  } catch (e) {
    // handled
  }
}

async function initDefaults() {
  initializing.value = true
  try {
    await api.initErrorMessages()
    ElMessage.success('默认配置已初始化')
    loadMessages()
  } catch (e) {
    // handled
  } finally {
    initializing.value = false
  }
}

async function refreshCache() {
  refreshing.value = true
  try {
    await api.refreshErrorMessages()
    ElMessage.success('缓存已刷新')
  } catch (e) {
    // handled
  } finally {
    refreshing.value = false
  }
}

async function enableAll() {
  enablingAll.value = true
  try {
    const res = await api.enableAllErrorMessages()
    const affected = res.data?.affected || res?.affected || 0
    ElMessage.success(`已启用所有配置（影响 ${affected} 条）`)
    loadMessages()
  } catch (e) {
    // handled
  } finally {
    enablingAll.value = false
  }
}

async function disableAll() {
  disablingAll.value = true
  try {
    const res = await api.disableAllErrorMessages()
    const affected = res.data?.affected || res?.affected || 0
    ElMessage.success(`已禁用所有配置（影响 ${affected} 条）`)
    loadMessages()
  } catch (e) {
    // handled
  } finally {
    disablingAll.value = false
  }
}

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

.header-actions {
  display: flex;
  gap: 10px;
}
</style>
