<template>
  <div class="models-page">
    <div class="page-header">
      <h2>模型管理</h2>
      <div class="header-actions">
        <el-select v-model="filterPlatform" placeholder="筛选平台" clearable style="width: 150px; margin-right: 12px;">
          <el-option label="全部" value="" />
          <el-option label="Claude" value="claude" />
          <el-option label="OpenAI" value="openai" />
          <el-option label="Gemini" value="gemini" />
        </el-select>
        <el-button type="primary" @click="showAddDialog">
          <i class="fa-solid fa-plus"></i> 添加模型
        </el-button>
      </div>
    </div>

    <el-table :data="filteredModels" v-loading="loading" stripe>
      <el-table-column prop="name" label="模型名称" min-width="200">
        <template #default="{ row }">
          <div class="model-name">
            <span class="name">{{ row.name }}</span>
            <el-tag v-if="row.is_default" size="small" type="success">默认</el-tag>
          </div>
          <div class="display-name">{{ row.display_name }}</div>
        </template>
      </el-table-column>
      <el-table-column prop="platform" label="平台" width="100">
        <template #default="{ row }">
          <el-tag :type="getPlatformType(row.platform)">{{ row.platform }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="context_size" label="上下文" width="100">
        <template #default="{ row }">
          {{ formatNumber(row.context_size) }}
        </template>
      </el-table-column>
      <el-table-column label="价格 ($/1M)" width="150">
        <template #default="{ row }">
          <div class="price-info">
            <span>入: ${{ row.input_price }}</span>
            <span>出: ${{ row.output_price }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="状态" width="80">
        <template #default="{ row }">
          <el-switch v-model="row.enabled" @change="toggleEnabled(row)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="editModel(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="deleteModel(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑模型' : '添加模型'" width="600px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="模型名称" prop="name">
              <el-input v-model="form.name" placeholder="如: claude-3-5-sonnet" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="显示名称">
              <el-input v-model="form.display_name" placeholder="如: Claude 3.5 Sonnet" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="平台" prop="platform">
              <el-select v-model="form.platform" style="width: 100%">
                <el-option label="Claude" value="claude" />
                <el-option label="OpenAI" value="openai" />
                <el-option label="Gemini" value="gemini" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="提供商">
              <el-input v-model="form.provider" placeholder="如: anthropic" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="上下文长度">
              <el-input-number v-model="form.context_size" :min="0" :step="1000" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="最大输出">
              <el-input-number v-model="form.max_output" :min="0" :step="1000" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="输入价格">
              <el-input-number v-model="form.input_price" :min="0" :precision="4" :step="0.1" style="width: 100%" />
              <div class="form-tip">$/1M tokens</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="输出价格">
              <el-input-number v-model="form.output_price" :min="0" :precision="4" :step="0.1" style="width: 100%" />
              <div class="form-tip">$/1M tokens</div>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="缓存写入">
              <el-input-number v-model="form.cache_create_price" :min="0" :precision="4" :step="0.1" style="width: 100%" />
              <div class="form-tip">$/1M tokens (可选)</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="缓存读取">
              <el-input-number v-model="form.cache_read_price" :min="0" :precision="4" :step="0.1" style="width: 100%" />
              <div class="form-tip">$/1M tokens (可选)</div>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="别名">
          <el-input v-model="form.aliases" placeholder="多个别名用逗号分隔" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="排序">
              <el-input-number v-model="form.sort_order" :min="0" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="启用">
              <el-switch v-model="form.enabled" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="默认">
              <el-switch v-model="form.is_default" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const models = ref([])
const filterPlatform = ref('')
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref()

const defaultForm = {
  name: '',
  display_name: '',
  platform: 'claude',
  provider: '',
  description: '',
  category: 'chat',
  context_size: 200000,
  max_output: 8192,
  input_price: 0,
  output_price: 0,
  cache_create_price: 0,
  cache_read_price: 0,
  enabled: true,
  is_default: false,
  sort_order: 0,
  aliases: ''
}

const form = reactive({ ...defaultForm })

const rules = {
  name: [{ required: true, message: '请输入模型名称', trigger: 'blur' }],
  platform: [{ required: true, message: '请选择平台', trigger: 'change' }]
}

const filteredModels = computed(() => {
  if (!filterPlatform.value) return models.value
  return models.value.filter(m => m.platform === filterPlatform.value)
})

onMounted(() => {
  loadModels()
})

async function loadModels() {
  loading.value = true
  try {
    const res = await api.getModels()
    models.value = res.data || []
  } catch (e) {
    ElMessage.error('加载模型列表失败')
  } finally {
    loading.value = false
  }
}

function showAddDialog() {
  isEdit.value = false
  Object.assign(form, { ...defaultForm })
  dialogVisible.value = true
}

function editModel(row) {
  isEdit.value = true
  Object.assign(form, { ...row })
  dialogVisible.value = true
}

async function submitForm() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await api.updateModel(form.id, form)
      ElMessage.success('更新成功')
    } else {
      await api.createModel(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadModels()
  } catch (e) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function toggleEnabled(row) {
  try {
    await api.toggleModel(row.id)
  } catch (e) {
    row.enabled = !row.enabled
    ElMessage.error('切换状态失败')
  }
}

async function deleteModel(row) {
  try {
    await ElMessageBox.confirm(`确定删除模型 "${row.name}" 吗？`, '确认删除', {
      type: 'warning'
    })
    await api.deleteModel(row.id)
    ElMessage.success('删除成功')
    loadModels()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

function getPlatformType(platform) {
  const map = { claude: 'primary', openai: 'success', gemini: 'warning' }
  return map[platform] || 'info'
}

function formatNumber(num) {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(0) + 'K'
  return num
}
</script>

<style scoped>
.models-page {
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
  font-size: 20px;
}

.header-actions {
  display: flex;
  align-items: center;
}

.model-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.model-name .name {
  font-weight: 500;
  font-family: monospace;
}

.display-name {
  color: #909399;
  font-size: 12px;
  margin-top: 2px;
}

.price-info {
  display: flex;
  flex-direction: column;
  font-size: 12px;
  color: #606266;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
