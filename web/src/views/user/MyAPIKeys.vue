<!--
 * 文件作用：我的 API Key 页面
 * 负责功能：
 *   - 查看用户的 API Key 列表
 *   - 创建新的 API Key
 *   - 删除 API Key
 *   - 查看 API Key 使用情况
 * 重要程度：⭐⭐⭐⭐ 重要（用户核心功能）
-->
<template>
  <div class="my-api-keys">
    <div class="page-header">
      <h2>我的 API Key</h2>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon> 创建 API Key
      </el-button>
    </div>

    <!-- API Key 列表 -->
    <el-card shadow="hover">
      <el-table :data="apiKeys" v-loading="loading" stripe>
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column label="Key" min-width="180">
          <template #default="{ row }">
            <div class="key-display">
              <code>{{ row.key_prefix }}...{{ row.key_suffix || '' }}</code>
              <el-button
                v-if="row.key_full"
                type="primary"
                link
                size="small"
                @click="copyKey(row.key_full)"
              >
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
              {{ row.status === 'active' ? '正常' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="绑定套餐" min-width="120">
          <template #default="{ row }">
            <span v-if="row.user_package">{{ row.user_package.name }}</span>
            <span v-else class="text-muted">未绑定</span>
          </template>
        </el-table-column>
        <el-table-column label="使用统计" width="150">
          <template #default="{ row }">
            <div class="usage-stats">
              <div>请求: {{ row.request_count || 0 }}</div>
              <div>消费: ${{ (row.cost_used || 0).toFixed(4) }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="viewUsage(row)">
              <el-icon><DataAnalysis /></el-icon> 详情
            </el-button>
            <el-popconfirm
              title="确定要删除这个 API Key 吗？"
              @confirm="deleteKey(row.id)"
            >
              <template #reference>
                <el-button type="danger" link size="small">
                  <el-icon><Delete /></el-icon> 删除
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && apiKeys.length === 0" description="暂无 API Key">
        <el-button type="primary" @click="showCreateDialog = true">创建第一个 API Key</el-button>
      </el-empty>
    </el-card>

    <!-- 创建 API Key 对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建 API Key" width="500px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="createForm.name" placeholder="请输入名称，如：开发测试" />
        </el-form-item>
        <el-form-item label="绑定套餐">
          <el-select v-model="createForm.user_package_id" placeholder="选择套餐" clearable style="width: 100%">
            <el-option
              v-for="pkg in activePackages"
              :key="pkg.id"
              :label="`${pkg.name} (${pkg.type === 'subscription' ? '订阅' : '额度'})`"
              :value="pkg.id"
            />
          </el-select>
          <div class="form-tip">不绑定套餐则使用默认计费</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createKey">创建</el-button>
      </template>
    </el-dialog>

    <!-- 新创建的 Key 显示对话框 -->
    <el-dialog v-model="showNewKeyDialog" title="API Key 创建成功" width="500px" :close-on-click-modal="false">
      <el-alert type="warning" :closable="false" style="margin-bottom: 16px">
        <template #title>
          <strong>请立即复制并保存此 API Key，关闭后将无法再次查看完整内容！</strong>
        </template>
      </el-alert>
      <div class="new-key-display">
        <el-input v-model="newKeyValue" readonly>
          <template #append>
            <el-button @click="copyKey(newKeyValue)">
              <el-icon><CopyDocument /></el-icon> 复制
            </el-button>
          </template>
        </el-input>
      </div>
      <template #footer>
        <el-button type="primary" @click="showNewKeyDialog = false">我已保存</el-button>
      </template>
    </el-dialog>

    <!-- 使用详情对话框 -->
    <el-dialog v-model="showUsageDialog" :title="`${selectedKey?.name} - 使用详情`" width="600px">
      <el-descriptions :column="2" border v-if="selectedKey">
        <el-descriptions-item label="名称">{{ selectedKey.name }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="selectedKey.status === 'active' ? 'success' : 'danger'" size="small">
            {{ selectedKey.status === 'active' ? '正常' : '禁用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="Key 前缀">{{ selectedKey.key_prefix }}...</el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ formatTime(selectedKey.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="请求次数">{{ selectedKey.request_count || 0 }}</el-descriptions-item>
        <el-descriptions-item label="Token 使用">{{ formatLargeNumber(selectedKey.tokens_used) }}</el-descriptions-item>
        <el-descriptions-item label="累计消费" :span="2">
          <span class="cost-value">${{ (selectedKey.cost_used || 0).toFixed(4) }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="最后使用" :span="2">
          {{ selectedKey.last_used_at ? formatTime(selectedKey.last_used_at) : '从未使用' }}
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import { Plus, CopyDocument, DataAnalysis, Delete } from '@element-plus/icons-vue'

const loading = ref(false)
const creating = ref(false)
const apiKeys = ref([])
const activePackages = ref([])

const showCreateDialog = ref(false)
const showNewKeyDialog = ref(false)
const showUsageDialog = ref(false)
const newKeyValue = ref('')
const selectedKey = ref(null)

const createForm = ref({
  name: '',
  user_package_id: null
})

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

const formatLargeNumber = (num) => {
  if (!num) return '0'
  if (num >= 1000000) return (num / 1000000).toFixed(2) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toLocaleString()
}

const copyKey = async (key) => {
  try {
    await navigator.clipboard.writeText(key)
    ElMessage.success('已复制到剪贴板')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const fetchApiKeys = async () => {
  loading.value = true
  try {
    const res = await api.getApiKeys()
    apiKeys.value = res.data || []
  } catch (e) {
    ElMessage.error('获取 API Key 列表失败')
  } finally {
    loading.value = false
  }
}

const fetchPackages = async () => {
  try {
    const res = await api.getMyActivePackages()
    activePackages.value = res.data || []
  } catch (e) {
    console.error('Failed to fetch packages:', e)
  }
}

const createKey = async () => {
  if (!createForm.value.name) {
    ElMessage.warning('请输入名称')
    return
  }

  creating.value = true
  try {
    const res = await api.createApiKey(createForm.value)
    if (res.code === 0 && res.data) {
      newKeyValue.value = res.data.key_full || res.data.key
      showCreateDialog.value = false
      showNewKeyDialog.value = true
      createForm.value = { name: '', user_package_id: null }
      fetchApiKeys()
    } else {
      ElMessage.error(res.message || '创建失败')
    }
  } catch (e) {
    ElMessage.error('创建失败')
  } finally {
    creating.value = false
  }
}

const deleteKey = async (id) => {
  try {
    await api.deleteApiKey(id)
    ElMessage.success('删除成功')
    fetchApiKeys()
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

const viewUsage = (key) => {
  selectedKey.value = key
  showUsageDialog.value = true
}

onMounted(() => {
  fetchApiKeys()
  fetchPackages()
})
</script>

<style scoped>
.my-api-keys {
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  color: #303133;
}

.key-display {
  display: flex;
  align-items: center;
  gap: 8px;
}

.key-display code {
  font-family: 'Consolas', 'Monaco', monospace;
  background: #f5f7fa;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 13px;
}

.usage-stats {
  font-size: 12px;
  color: #606266;
  line-height: 1.6;
}

.text-muted {
  color: #909399;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.new-key-display {
  margin: 16px 0;
}

.cost-value {
  font-size: 18px;
  font-weight: bold;
  color: #409eff;
}
</style>
