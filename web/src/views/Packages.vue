<!--
 * 文件作用：套餐管理页面，管理订阅和额度套餐
 * 负责功能：
 *   - 套餐列表和CRUD
 *   - 订阅类型配置（日/周/月额度）
 *   - 额度类型配置
 *   - 模型限制配置
 * 重要程度：⭐⭐⭐⭐ 重要（套餐配置）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="packages-page">
    <div class="page-header">
      <h2>套餐管理</h2>
      <el-button type="primary" @click="showCreateDialog">
        <el-icon><Plus /></el-icon> 创建套餐
      </el-button>
    </div>

    <!-- 套餐列表 -->
    <el-card>
      <el-table :data="packages" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="名称" width="120" />
        <el-table-column prop="type" label="类型" width="80">
          <template #default="{ row }">
            <el-tag :type="typeTagMap[row.type]?.type || 'info'" size="small">
              {{ typeTagMap[row.type]?.label || row.type }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="price" label="价格" width="80">
          <template #default="{ row }">
            ${{ row.price.toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column prop="duration" label="有效期" width="80">
          <template #default="{ row }">
            {{ row.duration }}天
          </template>
        </el-table-column>
        <el-table-column label="额度/次数" min-width="200">
          <template #default="{ row }">
            <template v-if="row.type === 'subscription'">
              <div class="quota-info">
                <span v-if="row.daily_quota > 0">日: ${{ row.daily_quota }}</span>
                <span v-if="row.weekly_quota > 0">周: ${{ row.weekly_quota }}</span>
                <span v-if="row.monthly_quota > 0">月: ${{ row.monthly_quota }}</span>
                <span v-if="!row.daily_quota && !row.weekly_quota && !row.monthly_quota">无限制</span>
              </div>
            </template>
            <template v-else-if="row.type === 'quota'">
              总额度: ${{ row.quota_amount }}
            </template>
            <template v-else-if="row.type === 'count'">
              总次数: {{ row.count_amount }} 次
            </template>
          </template>
        </el-table-column>
        <el-table-column label="模型限制" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.allowed_models" size="small" type="warning">
              {{ row.allowed_models.split(',').length }}个模型
            </el-tag>
            <span v-else class="text-muted">全部</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除该套餐吗？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button type="danger" link>删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 创建/编辑套餐弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editMode ? '编辑套餐' : '创建套餐'"
      width="600"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="套餐名称" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="form.type" style="width: 100%" :disabled="editMode">
            <el-option label="订阅 (包月)" value="subscription" />
            <el-option label="额度 (按费用)" value="quota" />
            <el-option label="按次 (按请求数)" value="count" />
          </el-select>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="价格($)" prop="price">
              <el-input-number v-model="form.price" :min="0" :precision="2" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="有效期(天)" prop="duration">
              <el-input-number v-model="form.duration" :min="1" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 订阅类型的额度限制 -->
        <template v-if="form.type === 'subscription'">
          <el-divider content-position="left">周期额度限制 (0=不限)</el-divider>
          <el-form-item label="每日额度">
            <el-input v-model="form.daily_quota" placeholder="0" style="width: 120px" />
            <span style="margin-left: 8px; color: #6b6573;">美元</span>
          </el-form-item>
          <el-form-item label="每周额度">
            <el-input v-model="form.weekly_quota" placeholder="0" style="width: 120px" />
            <span style="margin-left: 8px; color: #6b6573;">美元</span>
          </el-form-item>
          <el-form-item label="每月额度">
            <el-input v-model="form.monthly_quota" placeholder="0" style="width: 120px" />
            <span style="margin-left: 8px; color: #6b6573;">美元</span>
          </el-form-item>
        </template>

        <!-- 额度类型的总额度 -->
        <template v-if="form.type === 'quota'">
          <el-form-item label="总额度" prop="quota_amount">
            <el-input v-model="form.quota_amount" placeholder="0" style="width: 120px" />
            <span style="margin-left: 8px; color: #6b6573;">美元</span>
          </el-form-item>
        </template>

        <!-- 按次类型的总次数 -->
        <template v-if="form.type === 'count'">
          <el-form-item label="总次数" prop="count_amount">
            <el-input-number v-model="form.count_amount" :min="1" :step="100" style="width: 200px" />
            <span style="margin-left: 8px; color: #6b6573;">次请求</span>
          </el-form-item>
          <div class="form-tip" style="margin: -8px 0 16px 100px; color: #909399; font-size: 12px;">每次 API 请求消耗 1 次，不按 Token 或费用计费</div>
        </template>

        <el-form-item label="允许的模型">
          <div class="platform-quick-select">
            <span class="quick-label">快捷选择：</span>
            <el-check-tag
              v-for="group in platformGroups"
              :key="group.platform"
              :checked="isPlatformFullySelected(group.platform)"
              @change="togglePlatform(group.platform)"
              class="platform-tag"
            >
              {{ group.label }} ({{ group.models.length }})
            </el-check-tag>
            <el-button link type="danger" size="small" @click="selectedModels = []" v-if="selectedModels.length">
              清空
            </el-button>
          </div>
          <el-select
            v-model="selectedModels"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            :max-collapse-tags="3"
            placeholder="留空表示全部模型"
            style="width: 100%"
          >
            <el-option-group
              v-for="group in platformGroups"
              :key="group.platform"
              :label="group.label"
            >
              <el-option
                v-for="model in group.models"
                :key="model.id"
                :label="model.display_name || model.name"
                :value="model.name"
              />
            </el-option-group>
          </el-select>
          <div class="form-tip">点击平台标签快速选中/取消该平台全部模型，或在下拉框中逐个选择</div>
        </el-form-item>

        <el-form-item label="状态" prop="status">
          <el-select v-model="form.status" style="width: 100%">
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="套餐描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ editMode ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '@/api'

const loading = ref(false)
const packages = ref([])
const modelList = ref([])

const dialogVisible = ref(false)
const editMode = ref(false)
const submitting = ref(false)
const formRef = ref()
const typeTagMap = {
  subscription: { type: 'primary', label: '订阅' },
  quota: { type: 'success', label: '额度' },
  count: { type: 'warning', label: '按次' },
}

const form = ref({
  id: 0,
  name: '',
  type: 'subscription',
  price: 0,
  duration: 30,
  daily_quota: 0,
  weekly_quota: 0,
  monthly_quota: 0,
  quota_amount: 0,
  count_amount: 0,
  allowed_models: '',
  status: 'active',
  description: ''
})

// selectedModels 是数组，和 form.allowed_models (逗号分隔字符串) 双向转换
const selectedModels = computed({
  get() {
    if (!form.value.allowed_models) return []
    return form.value.allowed_models.split(',').filter(m => m.trim())
  },
  set(val) {
    form.value.allowed_models = val.join(',')
  }
})

const platformLabelMap = {
  claude: 'Claude', openai: 'OpenAI', gemini: 'Gemini', deepseek: 'DeepSeek',
  qwen: '通义千问', glm: '智谱 GLM', moonshot: 'Kimi', doubao: '豆包',
  baichuan: '百川', yi: '零一万物', minimax: 'MiniMax', stepfun: '阶跃星辰',
  spark: '讯飞星火', siliconflow: '硅基流动', xai: 'xAI', mistral: 'Mistral',
  cohere: 'Cohere',
}

const platformGroups = computed(() => {
  const groups = {}
  for (const m of modelList.value) {
    const p = m.platform || 'other'
    if (!groups[p]) groups[p] = { platform: p, label: platformLabelMap[p] || p, models: [] }
    groups[p].models.push(m)
  }
  return Object.values(groups).sort((a, b) => a.models[0]?.sort_order - b.models[0]?.sort_order)
})

function isPlatformFullySelected(platform) {
  const group = platformGroups.value.find(g => g.platform === platform)
  if (!group || group.models.length === 0) return false
  return group.models.every(m => selectedModels.value.includes(m.name))
}

function togglePlatform(platform) {
  const group = platformGroups.value.find(g => g.platform === platform)
  if (!group) return
  const names = group.models.map(m => m.name)
  if (isPlatformFullySelected(platform)) {
    selectedModels.value = selectedModels.value.filter(n => !names.includes(n))
  } else {
    const current = new Set(selectedModels.value)
    names.forEach(n => current.add(n))
    selectedModels.value = [...current]
  }
}

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }]
}

async function fetchPackages() {
  loading.value = true
  try {
    const res = await api.getPackages()
    packages.value = res.data || []
  } catch (e) {
    // handled
  } finally {
    loading.value = false
  }
}

async function fetchModels() {
  try {
    const res = await api.getModels()
    modelList.value = (res.data || []).filter(m => m.enabled)
  } catch (e) {
    // handled
  }
}

function showCreateDialog() {
  editMode.value = false
  form.value = {
    id: 0,
    name: '',
    type: 'subscription',
    price: 0,
    duration: 30,
    daily_quota: 0,
    weekly_quota: 0,
    monthly_quota: 0,
    quota_amount: 0,
    count_amount: 0,
    allowed_models: '',
    status: 'active',
    description: ''
  }
  dialogVisible.value = true
}

function handleEdit(row) {
  editMode.value = true
  form.value = { ...row }
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const data = {
      ...form.value,
      price: parseFloat(form.value.price) || 0,
      duration: parseInt(form.value.duration) || 30,
      daily_quota: parseFloat(form.value.daily_quota) || 0,
      weekly_quota: parseFloat(form.value.weekly_quota) || 0,
      monthly_quota: parseFloat(form.value.monthly_quota) || 0,
      quota_amount: parseFloat(form.value.quota_amount) || 0,
      count_amount: parseInt(form.value.count_amount) || 0
    }
    if (editMode.value) {
      await api.updatePackage(form.value.id, data)
      ElMessage.success('更新成功')
    } else {
      await api.createPackage(data)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchPackages()
  } catch (e) {
    // handled
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id) {
  try {
    await api.deletePackage(id)
    ElMessage.success('删除成功')
    fetchPackages()
  } catch (e) {
    // handled
  }
}

onMounted(() => {
  fetchPackages()
  fetchModels()
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
  color: var(--pink-text);
  margin: 0;
}

.quota-info {
  display: flex;
  gap: 12px;
  font-size: 12px;
}

.quota-info span {
  background: var(--pink-accent-light, #faf2f4);
  padding: 2px 8px;
  border-radius: 6px;
}

.text-muted {
  color: #6b6573;
}

.form-tip {
  font-size: 12px;
  color: #6b6573;
  margin-top: 4px;
}

.form-tip-inline {
  margin-left: 8px;
  font-size: 12px;
  color: #6b6573;
}

.platform-quick-select {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}

.quick-label {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
}

.platform-tag {
  cursor: pointer;
  font-size: 12px;
}
</style>
