<!--
 * 文件作用：API Key管理页面，管理全局API Key
 * 负责功能：
 *   - API Key列表展示
 *   - Key状态切换和删除
 *   - 使用日志查看
 *   - 费用统计
 * 重要程度：⭐⭐⭐⭐ 重要（密钥管理）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="apikeys-page">
    <div class="page-header">
      <h2>API Key 管理</h2>
    </div>

    <!-- API Key 列表 -->
    <el-card>
      <el-table :data="apiKeys" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column label="Key" min-width="280">
          <template #default="{ row }">
            <code class="key-full">{{ row.key_full || row.key_prefix }}</code>
          </template>
        </el-table-column>
        <el-table-column label="用户" width="100">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.user?.username || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" width="100" />
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row)" size="small">
              {{ getStatusLabel(row) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="rate_limit" label="限速" width="80">
          <template #default="{ row }">
            {{ row.rate_limit }}/分
          </template>
        </el-table-column>
        <el-table-column prop="request_count" label="请求数" width="80" />
        <el-table-column label="费用" width="90">
          <template #default="{ row }">
            ${{ (row.cost_used || 0).toFixed(4) }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="150">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="viewLogs(row)">日志</el-button>
            <el-button link :type="row.status === 'active' ? 'warning' : 'success'" size="small" @click="handleToggle(row)">
              {{ row.status === 'active' ? '禁用' : '启用' }}
            </el-button>
            <el-popconfirm title="确定删除此 API Key？" @confirm="handleDelete(row)">
              <template #reference>
                <el-button link type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @change="fetchAPIKeys"
        />
      </div>
    </el-card>

    <!-- 使用日志弹窗 -->
    <el-dialog v-model="logDialogVisible" :title="`${currentKey?.key_prefix} 使用日志`" width="1000px" top="5vh">
      <div class="log-header">
        <el-tag type="info">用户: {{ currentKey?.user?.username }}</el-tag>
        <el-tag type="success">请求数: {{ currentKey?.request_count || 0 }}</el-tag>
        <el-tag type="warning">费用: ${{ (currentKey?.cost_used || 0).toFixed(4) }}</el-tag>
      </div>
      <el-table :data="logs" v-loading="logLoading" stripe max-height="500" size="small">
        <el-table-column prop="timestamp" label="时间" width="170">
          <template #default="{ row }">
            {{ formatDate(row.timestamp) }}
          </template>
        </el-table-column>
        <el-table-column prop="model" label="模型" min-width="200" show-overflow-tooltip />
        <el-table-column label="输入 Token" width="100">
          <template #default="{ row }">
            {{ row.input_tokens || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="输出 Token" width="100">
          <template #default="{ row }">
            {{ row.output_tokens || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="缓存创建" width="90">
          <template #default="{ row }">
            {{ row.cache_creation_input_tokens || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="缓存读取" width="90">
          <template #default="{ row }">
            {{ row.cache_read_input_tokens || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="总 Token" width="90">
          <template #default="{ row }">
            {{ row.total_tokens || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="费用" width="100">
          <template #default="{ row }">
            ${{ (row.total_cost || 0).toFixed(6) }}
          </template>
        </el-table-column>
      </el-table>
      <div class="log-pagination">
        <el-pagination
          v-model:current-page="logPagination.page"
          v-model:page-size="logPagination.pageSize"
          :total="logPagination.total"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @change="fetchLogs"
        />
      </div>
      <template #footer>
        <el-button @click="logDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const apiKeys = ref([])
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })

// 日志相关
const logDialogVisible = ref(false)
const logLoading = ref(false)
const currentKey = ref(null)
const logs = ref([])
const logPagination = reactive({ page: 1, pageSize: 20, total: 0 })

function formatDate(str) {
  if (!str) return ''
  return new Date(str).toLocaleString('zh-CN')
}

// 判断 API Key 是否过期
function isExpired(row) {
  if (!row.expires_at) return false
  return new Date(row.expires_at) < new Date()
}

// 获取状态显示标签
function getStatusLabel(row) {
  if (row.status === 'disabled') return '禁用'
  if (isExpired(row)) return '已过期'
  return '正常'
}

// 获取状态标签类型
function getStatusType(row) {
  if (row.status === 'disabled') return 'danger'
  if (isExpired(row)) return 'warning'
  return 'success'
}

async function fetchAPIKeys() {
  loading.value = true
  try {
    const res = await api.adminGetAllAPIKeys({ page: pagination.page, page_size: pagination.pageSize })
    apiKeys.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (e) {
    // handled
  } finally {
    loading.value = false
  }
}

async function handleToggle(row) {
  try {
    await api.adminToggleUserAPIKey(row.user_id, row.id)
    ElMessage.success('状态已更新')
    fetchAPIKeys()
  } catch (e) {
    // handled
  }
}

async function handleDelete(row) {
  try {
    await api.adminDeleteUserAPIKey(row.user_id, row.id)
    ElMessage.success('删除成功')
    fetchAPIKeys()
  } catch (e) {
    // handled
  }
}

// 查看日志
function viewLogs(row) {
  currentKey.value = row
  logPagination.page = 1
  logDialogVisible.value = true
  fetchLogs()
}

async function fetchLogs() {
  if (!currentKey.value) return
  logLoading.value = true
  logs.value = []
  try {
    const res = await api.adminGetAPIKeyLogs(currentKey.value.id, {
      page: logPagination.page,
      page_size: logPagination.pageSize
    })
    logs.value = res.data.items || []
    logPagination.total = res.data.total || 0
  } catch (e) {
    // handled
  } finally {
    logLoading.value = false
  }
}

onMounted(() => {
  fetchAPIKeys()
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

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.key-full {
  background: #f3f4f6;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  color: #4b5563;
  word-break: break-all;
}

.log-header {
  margin-bottom: 16px;
  display: flex;
  gap: 12px;
}

.log-pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
