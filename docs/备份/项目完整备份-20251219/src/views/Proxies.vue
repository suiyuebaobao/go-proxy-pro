<template>
  <div class="proxies-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h2>代理管理</h2>
        <p class="header-desc">管理代理服务器配置，用于账户连接</p>
      </div>
      <div class="header-actions">
        <el-button @click="loadProxies">
          <i class="fa-solid fa-sync-alt" :class="{ 'fa-spin': loading }"></i>
          刷新
        </el-button>
        <el-button type="primary" @click="handleAdd">
          <i class="fa-solid fa-plus"></i>
          添加代理
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="filter-bar">
      <el-input
        v-model="keyword"
        placeholder="搜索代理名称、地址..."
        clearable
        style="width: 300px"
        @input="handleSearch"
      >
        <template #prefix>
          <i class="fa-solid fa-search"></i>
        </template>
      </el-input>
    </div>

    <!-- 代理列表 -->
    <el-card class="proxies-table-card" shadow="never">
      <el-table :data="proxies" v-loading="loading" stripe>
        <el-table-column label="代理名称" min-width="180">
          <template #default="{ row }">
            <div class="proxy-name-cell">
              <div class="proxy-icon" :class="row.type">
                <i class="fa-solid fa-shield-halved"></i>
              </div>
              <div class="proxy-info">
                <div class="proxy-name-row">
                  <span class="proxy-name">{{ row.name }}</span>
                  <el-tag v-if="row.is_default" type="warning" size="small" class="default-tag">
                    <i class="fa-solid fa-star"></i> 默认
                  </el-tag>
                </div>
                <span class="proxy-remark" v-if="row.remark">{{ row.remark }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="getTypeTagType(row.type)" size="small">
              {{ row.type.toUpperCase() }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="地址" min-width="200">
          <template #default="{ row }">
            <span class="proxy-address">{{ row.host }}:{{ row.port }}</span>
          </template>
        </el-table-column>

        <el-table-column label="连接状态" width="120" align="center">
          <template #default="{ row }">
            <div class="status-cell">
              <el-tag
                v-if="row.testStatus === 'testing'"
                type="info"
                size="small"
              >
                <i class="fa-solid fa-spinner fa-spin"></i> 测试中
              </el-tag>
              <el-tag
                v-else-if="row.testStatus === 'success'"
                type="success"
                size="small"
              >
                <i class="fa-solid fa-check"></i> {{ row.latency }}ms
              </el-tag>
              <el-tag
                v-else-if="row.testStatus === 'failed'"
                type="danger"
                size="small"
              >
                <i class="fa-solid fa-xmark"></i> 失败
              </el-tag>
              <span v-else class="no-test">未测试</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="启用" width="80" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              size="small"
              @change="handleToggleEnabled(row)"
            />
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button
              link
              type="warning"
              size="small"
              @click="handleSetDefault(row)"
              :disabled="row.is_default || !row.enabled"
            >
              <i class="fa-solid fa-star"></i> 设为默认
            </el-button>
            <el-button
              link
              type="success"
              size="small"
              @click="handleTestRow(row)"
              :loading="row.testStatus === 'testing'"
            >
              <i class="fa-solid fa-plug-circle-check" v-if="row.testStatus !== 'testing'"></i> 测试
            </el-button>
            <el-button link type="primary" size="small" @click="handleEdit(row)">
              <i class="fa-solid fa-edit"></i> 编辑
            </el-button>
            <el-popconfirm
              title="确定删除该代理吗？"
              confirm-button-text="删除"
              cancel-button-text="取消"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button link type="danger" size="small">
                  <i class="fa-solid fa-trash"></i> 删除
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="table-footer">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @change="loadProxies"
        />
      </div>
    </el-card>

    <!-- 添加/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑代理' : '添加代理'"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
        <!-- 快速导入 -->
        <el-form-item label="快速导入">
          <el-input
            v-model="proxyUrl"
            placeholder="粘贴代理URL自动解析，如: http://user:pass@host:port 或 socks5://host:port"
            clearable
            @input="parseProxyUrl"
          >
            <template #prefix>
              <i class="fa-solid fa-magic-wand-sparkles"></i>
            </template>
          </el-input>
          <div class="form-tip">支持格式: http://, https://, socks5:// 开头的代理地址</div>
        </el-form-item>

        <el-divider content-position="left">代理配置</el-divider>

        <el-form-item label="代理名称" prop="name">
          <el-input v-model="form.name" placeholder="为代理设置一个易识别的名称" />
        </el-form-item>

        <el-form-item label="代理类型" prop="type">
          <el-radio-group v-model="form.type">
            <el-radio-button value="http">HTTP</el-radio-button>
            <el-radio-button value="https">HTTPS</el-radio-button>
            <el-radio-button value="socks5">SOCKS5</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-row :gutter="16">
          <el-col :span="16">
            <el-form-item label="主机地址" prop="host">
              <el-input v-model="form.host" placeholder="例如: 127.0.0.1 或 proxy.example.com" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="端口" prop="port">
              <el-input-number v-model="form.port" :min="1" :max="65535" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">认证配置（可选）</el-divider>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="用户名">
              <el-input v-model="form.username" placeholder="代理认证用户名" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="密码">
              <el-input v-model="form.password" type="password" show-password placeholder="代理认证密码" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="备注信息（可选）" />
        </el-form-item>

        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ isEdit ? '保存修改' : '添加代理' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const proxies = ref([])
const keyword = ref('')
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref()
const proxyUrl = ref('')

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const defaultForm = {
  name: '',
  type: 'http',
  host: '',
  port: 7890,
  username: '',
  password: '',
  remark: '',
  enabled: true
}

const form = reactive({ ...defaultForm })

const rules = {
  name: [{ required: true, message: '请输入代理名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择代理类型', trigger: 'change' }],
  host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }]
}

// 加载代理列表
async function loadProxies() {
  loading.value = true
  try {
    const res = await api.getProxyConfigs({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: keyword.value
    })
    // 使用数据库中的测试状态
    proxies.value = (res.items || []).map(p => ({
      ...p,
      testStatus: p.test_status || null,
      latency: p.test_latency || null
    }))
    pagination.total = res.total || 0
  } catch (e) {
    console.error('Failed to load proxies:', e)
  } finally {
    loading.value = false
  }
}

// 搜索
let searchTimer = null
function handleSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    pagination.page = 1
    loadProxies()
  }, 300)
}

// 获取类型标签样式
function getTypeTagType(type) {
  const map = {
    http: '',
    https: 'success',
    socks5: 'warning'
  }
  return map[type] || ''
}

// 格式化时间
function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

// 解析代理URL
function parseProxyUrl() {
  const url = proxyUrl.value.trim()
  if (!url) return

  try {
    // 支持格式: protocol://user:pass@host:port 或 protocol://host:port
    const regex = /^(https?|socks5):\/\/(?:([^:@]+):([^@]+)@)?([^:\/]+):(\d+)\/?$/i
    const match = url.match(regex)

    if (match) {
      const [, protocol, username, password, host, port] = match
      form.type = protocol.toLowerCase()
      form.host = host
      form.port = parseInt(port, 10)
      form.username = username || ''
      form.password = password || ''

      // 自动生成名称（如果为空）
      if (!form.name) {
        form.name = `${protocol.toUpperCase()} - ${host}:${port}`
      }

      ElMessage.success('代理地址解析成功')
    } else {
      // 尝试简单格式: host:port
      const simpleMatch = url.match(/^([^:\/]+):(\d+)$/)
      if (simpleMatch) {
        form.host = simpleMatch[1]
        form.port = parseInt(simpleMatch[2], 10)
        if (!form.name) {
          form.name = `HTTP - ${form.host}:${form.port}`
        }
        ElMessage.success('代理地址解析成功')
      }
    }
  } catch (e) {
    console.error('解析代理URL失败:', e)
  }
}

// 测试代理连通性
async function testProxy(proxyData) {
  try {
    const res = await api.testProxyConnectivity({
      id: proxyData.id || 0,  // 传递 ID 以保存测试结果
      type: proxyData.type,
      host: proxyData.host,
      port: proxyData.port,
      username: proxyData.username || '',
      password: proxyData.password || ''
    })
    return res
  } catch (e) {
    return { success: false, error: e.message || '测试请求失败' }
  }
}

// 测试列表中的代理
async function handleTestRow(row) {
  row.testStatus = 'testing'
  row.latency = null

  const res = await testProxy(row)

  if (res.success) {
    row.testStatus = 'success'
    row.latency = res.latency
  } else {
    row.testStatus = 'failed'
    ElMessage.error(res.error || '代理连接失败')
  }
}

// 添加
function handleAdd() {
  Object.assign(form, { ...defaultForm })
  proxyUrl.value = ''
  isEdit.value = false
  dialogVisible.value = true
}

// 编辑
function handleEdit(row) {
  Object.assign(form, { ...row })
  proxyUrl.value = ''
  isEdit.value = true
  dialogVisible.value = true
}

// 切换启用状态
async function handleToggleEnabled(row) {
  try {
    await api.toggleProxyConfig(row.id)
    ElMessage.success('更新成功')
    // 如果禁用了默认代理，需要清除默认状态
    if (!row.enabled && row.is_default) {
      row.is_default = false
    }
  } catch (e) {
    row.enabled = !row.enabled
    ElMessage.error('更新失败')
  }
}

// 设置默认代理
async function handleSetDefault(row) {
  try {
    await api.setDefaultProxyConfig(row.id)
    // 更新本地状态
    proxies.value.forEach(p => {
      p.is_default = p.id === row.id
    })
    ElMessage.success(`已将 "${row.name}" 设为默认代理，用于OAuth认证`)
  } catch (e) {
    ElMessage.error(e.message || '设置失败')
  }
}

// 删除
async function handleDelete(id) {
  try {
    await api.deleteProxyConfig(id)
    ElMessage.success('删除成功')
    loadProxies()
  } catch (e) {
    ElMessage.error(e.message || '删除失败')
  }
}

// 提交表单
async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true

  try {
    // 先测试代理连通性
    ElMessage.info('正在测试代理连通性...')
    const testRes = await testProxy(form)

    if (!testRes.success) {
      ElMessage.warning(`代理连接测试失败: ${testRes.error}，仍将保存配置`)
    } else {
      ElMessage.success(`代理连接测试成功，延迟: ${testRes.latency}ms`)
    }

    // 保存代理配置
    if (isEdit.value) {
      await api.updateProxyConfig(form.id, form)
      ElMessage.success('更新成功')
    } else {
      await api.createProxyConfig(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadProxies()
  } catch (e) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadProxies()
})
</script>

<style scoped>
.proxies-page {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left h2 {
  margin: 0 0 4px;
  font-size: 24px;
  font-weight: 700;
  color: #1f2937;
}

.header-desc {
  margin: 0;
  font-size: 14px;
  color: #6b7280;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.filter-bar {
  margin-bottom: 16px;
}

.proxies-table-card {
  border-radius: 12px;
}

.proxy-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.proxy-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 14px;
  flex-shrink: 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.proxy-icon.https {
  background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
}

.proxy-icon.socks5 {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.proxy-info {
  display: flex;
  flex-direction: column;
}

.proxy-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.proxy-name {
  font-weight: 600;
  color: #1f2937;
}

.default-tag {
  font-size: 10px;
  padding: 0 4px;
}

.proxy-remark {
  font-size: 12px;
  color: #6b7280;
}

.proxy-address {
  font-family: 'SF Mono', Monaco, monospace;
  color: #4b5563;
}

.status-cell {
  display: flex;
  justify-content: center;
}

.no-test {
  color: #9ca3af;
  font-size: 12px;
}

.table-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
