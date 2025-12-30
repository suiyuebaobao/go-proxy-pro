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
            <el-tag :type="row.type === 'subscription' ? 'primary' : 'success'" size="small">
              {{ row.type === 'subscription' ? '订阅' : '额度' }}
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
        <el-table-column label="额度限制" min-width="200">
          <template #default="{ row }">
            <template v-if="row.type === 'subscription'">
              <div class="quota-info">
                <span v-if="row.daily_quota > 0">日: ${{ row.daily_quota }}</span>
                <span v-if="row.weekly_quota > 0">周: ${{ row.weekly_quota }}</span>
                <span v-if="row.monthly_quota > 0">月: ${{ row.monthly_quota }}</span>
                <span v-if="!row.daily_quota && !row.weekly_quota && !row.monthly_quota">无限制</span>
              </div>
            </template>
            <template v-else>
              总额度: ${{ row.quota_amount }}
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
            <el-option label="额度" value="quota" />
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
            <span style="margin-left: 8px; color: #909399;">美元</span>
          </el-form-item>
          <el-form-item label="每周额度">
            <el-input v-model="form.weekly_quota" placeholder="0" style="width: 120px" />
            <span style="margin-left: 8px; color: #909399;">美元</span>
          </el-form-item>
          <el-form-item label="每月额度">
            <el-input v-model="form.monthly_quota" placeholder="0" style="width: 120px" />
            <span style="margin-left: 8px; color: #909399;">美元</span>
          </el-form-item>
        </template>

        <!-- 额度类型的总额度 -->
        <template v-if="form.type === 'quota'">
          <el-form-item label="总额度" prop="quota_amount">
            <el-input v-model="form.quota_amount" placeholder="0" style="width: 120px" />
            <span style="margin-left: 8px; color: #909399;">美元</span>
          </el-form-item>
        </template>

        <el-form-item label="允许的模型">
          <el-select
            v-model="selectedModels"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="留空表示全部模型"
            style="width: 100%"
          >
            <el-option
              v-for="model in modelList"
              :key="model.id"
              :label="model.name"
              :value="model.name"
            />
          </el-select>
          <div class="form-tip">限制该套餐可使用的模型列表，不选则允许全部模型</div>
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
    // 确保数值字段是数字类型
    const data = {
      ...form.value,
      price: parseFloat(form.value.price) || 0,
      duration: parseInt(form.value.duration) || 30,
      daily_quota: parseFloat(form.value.daily_quota) || 0,
      weekly_quota: parseFloat(form.value.weekly_quota) || 0,
      monthly_quota: parseFloat(form.value.monthly_quota) || 0,
      quota_amount: parseFloat(form.value.quota_amount) || 0
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
  color: #333;
  margin: 0;
}

.quota-info {
  display: flex;
  gap: 12px;
  font-size: 12px;
}

.quota-info span {
  background: #f0f2f5;
  padding: 2px 8px;
  border-radius: 4px;
}

.text-muted {
  color: #909399;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.form-tip-inline {
  margin-left: 8px;
  font-size: 12px;
  color: #909399;
}
</style>
