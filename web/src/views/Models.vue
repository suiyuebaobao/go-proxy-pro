<!--
 * 文件作用：模型管理页面，管理AI模型配置
 * 负责功能：
 *   - 模型列表和CRUD
 *   - 模型价格配置
 *   - 模型映射管理
 *   - 平台筛选和状态切换
 * 重要程度：⭐⭐⭐⭐ 重要（模型配置）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="models-page">
    <div class="page-header">
      <h2>模型管理</h2>
    </div>

    <!-- Tab 切换 -->
    <el-tabs v-model="activeTab" class="models-tabs">
      <el-tab-pane label="模型列表" name="models">
        <div class="tab-header">
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
      </el-tab-pane>

      <el-tab-pane label="模型映射" name="mappings">
        <div class="tab-header">
          <div class="mapping-info">
            <el-alert
              type="info"
              :closable="false"
              show-icon
              style="margin-bottom: 16px;"
            >
              模型映射允许将客户端请求的模型名自动转换为实际的模型名。例如: claude-3-5-sonnet → claude-sonnet-4-20250514
            </el-alert>
          </div>
          <div class="header-actions">
            <el-button @click="refreshMappingCache">
              <i class="fa-solid fa-sync"></i> 刷新缓存
            </el-button>
            <el-button type="primary" @click="showAddMappingDialog">
              <i class="fa-solid fa-plus"></i> 添加映射
            </el-button>
          </div>
        </div>

        <el-table :data="mappings" v-loading="mappingLoading" stripe>
          <el-table-column prop="source_model" label="源模型" min-width="200">
            <template #default="{ row }">
              <code class="model-code">{{ row.source_model }}</code>
            </template>
          </el-table-column>
          <el-table-column label="" width="60" align="center">
            <template #default>
              <i class="fa-solid fa-arrow-right" style="color: #909399;"></i>
            </template>
          </el-table-column>
          <el-table-column prop="target_model" label="目标模型" min-width="200">
            <template #default="{ row }">
              <code class="model-code">{{ row.target_model }}</code>
            </template>
          </el-table-column>
          <el-table-column prop="priority" label="优先级" width="80" align="center" />
          <el-table-column prop="description" label="描述" min-width="150">
            <template #default="{ row }">
              <span class="description">{{ row.description || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="enabled" label="状态" width="80">
            <template #default="{ row }">
              <el-switch v-model="row.enabled" @change="toggleMappingEnabled(row)" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="editMapping(row)">编辑</el-button>
              <el-button size="small" type="danger" @click="deleteMapping(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 缓存统计 -->
        <div class="cache-stats" v-if="mappingCacheStats">
          <el-divider />
          <div class="stats-header">
            <strong>当前生效的映射 ({{ mappingCacheStats.count }} 条)</strong>
          </div>
          <div class="stats-content" v-if="mappingCacheStats.mappings && Object.keys(mappingCacheStats.mappings).length > 0">
            <el-tag
              v-for="(target, source) in mappingCacheStats.mappings"
              :key="source"
              class="mapping-tag"
              type="info"
            >
              {{ source }} → {{ target }}
            </el-tag>
          </div>
          <div v-else class="no-mappings">暂无生效的映射</div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 添加/编辑模型对话框 -->
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
              <el-select v-model="form.platform" filterable allow-create placeholder="选择或输入平台" style="width: 100%">
                <el-option label="Claude" value="claude" />
                <el-option label="OpenAI" value="openai" />
                <el-option label="Gemini" value="gemini" />
              </el-select>
              <div class="form-tip">决定使用哪个 API 处理请求</div>
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

    <!-- 添加/编辑映射对话框 -->
    <el-dialog v-model="mappingDialogVisible" :title="isMappingEdit ? '编辑映射' : '添加映射'" width="500px">
      <el-form :model="mappingForm" :rules="mappingRules" ref="mappingFormRef" label-width="100px">
        <el-form-item label="源模型" prop="source_model">
          <el-select
            v-model="mappingForm.source_model"
            filterable
            allow-create
            placeholder="选择或输入客户端请求的模型名"
            style="width: 100%"
          >
            <el-option
              v-for="model in models"
              :key="model.id"
              :label="model.name"
              :value="model.name"
            >
              <span>{{ model.name }}</span>
              <span style="color: #909399; font-size: 12px; margin-left: 8px;">{{ model.display_name }}</span>
            </el-option>
          </el-select>
          <div class="form-tip">客户端请求时使用的模型名称（可输入自定义名称）</div>
        </el-form-item>
        <el-form-item label="目标模型" prop="target_model">
          <el-select
            v-model="mappingForm.target_model"
            filterable
            placeholder="选择实际转发的模型"
            style="width: 100%"
          >
            <el-option
              v-for="model in enabledModels"
              :key="model.id"
              :label="model.name"
              :value="model.name"
            >
              <span>{{ model.name }}</span>
              <span style="color: #909399; font-size: 12px; margin-left: 8px;">{{ model.display_name }}</span>
            </el-option>
          </el-select>
          <div class="form-tip">实际发送到上游的模型名称</div>
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="mappingForm.priority" :min="0" :max="100" style="width: 100%" />
          <div class="form-tip">数值越大优先级越高</div>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="mappingForm.description" type="textarea" :rows="2" placeholder="映射说明（可选）" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="mappingForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="mappingDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitMappingForm" :loading="mappingSubmitting">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ensureFontAwesomeLoaded } from '@/utils/fontawesome'
ensureFontAwesomeLoaded()

import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'

// ========== 公共状态 ==========
const activeTab = ref('models')

// ========== 模型列表状态 ==========
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

// 启用的模型列表（用于目标模型下拉框）
const enabledModels = computed(() => {
  return models.value.filter(m => m.enabled)
})

// ========== 模型映射状态 ==========
const mappingLoading = ref(false)
const mappings = ref([])
const mappingDialogVisible = ref(false)
const isMappingEdit = ref(false)
const mappingSubmitting = ref(false)
const mappingFormRef = ref()
const mappingCacheStats = ref(null)

const defaultMappingForm = {
  source_model: '',
  target_model: '',
  priority: 0,
  description: '',
  enabled: true
}

const mappingForm = reactive({ ...defaultMappingForm })

const mappingRules = {
  source_model: [{ required: true, message: '请输入源模型名称', trigger: 'blur' }],
  target_model: [{ required: true, message: '请输入目标模型名称', trigger: 'blur' }]
}

// ========== 生命周期 ==========
onMounted(() => {
  loadModels()
  loadMappings()
  loadMappingCacheStats()
})

watch(activeTab, (newTab) => {
  if (newTab === 'mappings') {
    loadMappings()
    loadMappingCacheStats()
  }
})

// ========== 模型列表方法 ==========
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

// ========== 模型映射方法 ==========
async function loadMappings() {
  mappingLoading.value = true
  try {
    const res = await api.getModelMappings()
    mappings.value = res.data?.mappings || []
  } catch (e) {
    ElMessage.error('加载映射列表失败')
  } finally {
    mappingLoading.value = false
  }
}

async function loadMappingCacheStats() {
  try {
    const res = await api.getModelMappingCacheStats()
    mappingCacheStats.value = res.data || null
  } catch (e) {
    console.error('加载缓存统计失败:', e)
  }
}

function showAddMappingDialog() {
  isMappingEdit.value = false
  Object.assign(mappingForm, { ...defaultMappingForm })
  mappingDialogVisible.value = true
}

function editMapping(row) {
  isMappingEdit.value = true
  Object.assign(mappingForm, { ...row })
  mappingDialogVisible.value = true
}

async function submitMappingForm() {
  const valid = await mappingFormRef.value?.validate().catch(() => false)
  if (!valid) return

  mappingSubmitting.value = true
  try {
    if (isMappingEdit.value) {
      await api.updateModelMapping(mappingForm.id, mappingForm)
      ElMessage.success('更新成功')
    } else {
      await api.createModelMapping(mappingForm)
      ElMessage.success('创建成功')
    }
    mappingDialogVisible.value = false
    loadMappings()
    loadMappingCacheStats()
  } catch (e) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    mappingSubmitting.value = false
  }
}

async function toggleMappingEnabled(row) {
  try {
    await api.toggleModelMapping(row.id)
    loadMappingCacheStats()
  } catch (e) {
    row.enabled = !row.enabled
    ElMessage.error('切换状态失败')
  }
}

async function deleteMapping(row) {
  try {
    await ElMessageBox.confirm(`确定删除映射 "${row.source_model}" 吗？`, '确认删除', {
      type: 'warning'
    })
    await api.deleteModelMapping(row.id)
    ElMessage.success('删除成功')
    loadMappings()
    loadMappingCacheStats()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

async function refreshMappingCache() {
  try {
    await api.refreshModelMappingCache()
    ElMessage.success('缓存已刷新')
    loadMappingCacheStats()
  } catch (e) {
    ElMessage.error('刷新缓存失败')
  }
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

.models-tabs {
  background: var(--el-bg-color);
  border-radius: 8px;
  padding: 16px;
}

.tab-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
}

.mapping-info {
  flex: 1;
  min-width: 300px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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

.model-code {
  background: var(--el-fill-color-light);
  padding: 2px 8px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 13px;
}

.description {
  color: #909399;
  font-size: 13px;
}

.cache-stats {
  margin-top: 20px;
}

.stats-header {
  margin-bottom: 12px;
  color: #606266;
}

.stats-content {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.mapping-tag {
  font-family: monospace;
}

.no-mappings {
  color: #909399;
  font-size: 14px;
}
</style>
