<template>
  <div class="system-logs-page">
    <div class="page-header">
      <h2>系统日志</h2>
      <div class="header-actions">
        <el-button @click="loadFiles" :loading="loadingFiles">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>
    </div>

    <!-- Tab 切换 -->
    <el-tabs v-model="logSource" @tab-change="handleSourceChange" class="log-tabs">
      <el-tab-pane label="应用日志" name="app">
        <template #label>
          <span><el-icon><Files /></el-icon> 应用日志</span>
        </template>
      </el-tab-pane>
      <el-tab-pane label="服务器日志" name="server">
        <template #label>
          <span><el-icon><Monitor /></el-icon> 服务器日志</span>
        </template>
      </el-tab-pane>
    </el-tabs>

    <el-row :gutter="20">
      <!-- 左侧：文件列表 -->
      <el-col :span="8">
        <el-card class="file-list-card">
          <template #header>
            <div class="card-header">
              <span>{{ logSource === 'app' ? '应用日志文件' : '服务器日志文件' }}</span>
              <el-tag size="small">{{ files.length }} 个文件</el-tag>
            </div>
          </template>

          <!-- 分类筛选 -->
          <div class="filter-bar">
            <el-select v-model="filterCategory" placeholder="选择分类" clearable size="small" @change="loadFiles">
              <el-option
                v-for="cat in categories"
                :key="cat.name"
                :label="`${cat.label} (${cat.count})`"
                :value="cat.name"
              />
            </el-select>
            <el-date-picker
              v-if="logSource === 'app'"
              v-model="filterDate"
              type="date"
              placeholder="选择日期"
              format="YYYY-MM-DD"
              value-format="YYYY-MM-DD"
              size="small"
              clearable
              @change="loadFiles"
              style="width: 140px; margin-left: 10px;"
            />
          </div>

          <!-- 文件列表 -->
          <el-table
            :data="files"
            v-loading="loadingFiles"
            stripe
            size="small"
            highlight-current-row
            @row-click="selectFile"
            :row-class-name="getRowClassName"
            max-height="500"
          >
            <el-table-column prop="name" label="文件名" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="file-name">
                  <el-icon><Document /></el-icon>
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="size_human" label="大小" width="80" />
            <el-table-column label="操作" width="80">
              <template #default="{ row }">
                <el-dropdown @command="(cmd) => handleFileCommand(cmd, row)">
                  <el-button type="primary" link size="small">
                    <el-icon><More /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="view">查看</el-dropdown-item>
                      <el-dropdown-item command="tail">实时</el-dropdown-item>
                      <el-dropdown-item command="download">下载</el-dropdown-item>
                      <el-dropdown-item v-if="logSource === 'app'" command="delete" divided>删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <!-- 右侧：日志内容 -->
      <el-col :span="16">
        <el-card class="log-content-card">
          <template #header>
            <div class="card-header">
              <div class="file-info" v-if="selectedFile">
                <span class="filename">{{ selectedFile }}</span>
                <el-tag size="small" type="info" v-if="logInfo.size_human">{{ logInfo.size_human }}</el-tag>
                <el-tag size="small" type="success" v-if="logInfo.total_lines">{{ logInfo.total_lines }} 行</el-tag>
              </div>
              <span v-else>请选择日志文件</span>
              <div class="header-tools" v-if="selectedFile">
                <el-input
                  v-model="searchKeyword"
                  placeholder="搜索关键词"
                  size="small"
                  clearable
                  style="width: 150px; margin-right: 10px;"
                  @keyup.enter="loadLogContent"
                />
                <el-checkbox v-model="reverseOrder" size="small" @change="loadLogContent">倒序</el-checkbox>
                <el-button size="small" @click="loadLogContent" :loading="loadingContent" style="margin-left: 10px;">
                  查询
                </el-button>
                <el-button size="small" type="success" @click="tailLog" :loading="loadingTail">
                  <el-icon><VideoPlay /></el-icon> 实时
                </el-button>
              </div>
            </div>
          </template>

          <!-- 日志内容 -->
          <div class="log-content" v-loading="loadingContent || loadingTail">
            <div v-if="!selectedFile" class="empty-tip">
              <el-empty description="请从左侧选择要查看的日志文件" />
            </div>
            <div v-else-if="logLines.length === 0" class="empty-tip">
              <el-empty description="没有日志内容" />
            </div>
            <pre v-else class="log-pre" ref="logPreRef">{{ logLines.join('\n') }}</pre>
          </div>

          <!-- 分页 -->
          <div class="pagination-wrap" v-if="logInfo.total_pages > 1">
            <el-pagination
              v-model:current-page="page"
              :page-size="pageSize"
              :total="logInfo.total_lines"
              :page-sizes="[100, 200, 500, 1000]"
              layout="total, sizes, prev, pager, next"
              @current-change="loadLogContent"
              @size-change="handlePageSizeChange"
            />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Document, More, VideoPlay, Files, Monitor } from '@element-plus/icons-vue'
import api from '@/api'

// 日志来源
const logSource = ref('app') // app=应用日志, server=服务器日志

// 文件列表
const files = ref([])
const categories = ref([])
const loadingFiles = ref(false)
const filterCategory = ref('')
const filterDate = ref('')

// 日志内容
const selectedFile = ref('')
const logLines = ref([])
const logInfo = ref({})
const loadingContent = ref(false)
const loadingTail = ref(false)
const page = ref(1)
const pageSize = ref(200)
const searchKeyword = ref('')
const reverseOrder = ref(false)
const logPreRef = ref(null)

// 切换日志来源
function handleSourceChange() {
  // 重置状态
  files.value = []
  categories.value = []
  selectedFile.value = ''
  logLines.value = []
  logInfo.value = {}
  filterCategory.value = ''
  filterDate.value = ''
  page.value = 1
  searchKeyword.value = ''
  reverseOrder.value = false

  loadFiles()
}

// 加载文件列表
async function loadFiles() {
  loadingFiles.value = true
  try {
    const params = { source: logSource.value }
    if (filterCategory.value) params.category = filterCategory.value
    if (filterDate.value && logSource.value === 'app') params.date = filterDate.value

    const res = await api.getSystemLogFiles(params)
    files.value = res.data?.files || []
    categories.value = res.data?.categories || []
  } catch (e) {
    console.error('Failed to load files:', e)
  } finally {
    loadingFiles.value = false
  }
}

// 选择文件
function selectFile(row) {
  selectedFile.value = row.name
  page.value = 1
  searchKeyword.value = ''
  loadLogContent()
}

// 获取行样式
function getRowClassName({ row }) {
  return row.name === selectedFile.value ? 'selected-row' : ''
}

// 加载日志内容
async function loadLogContent() {
  if (!selectedFile.value) return

  loadingContent.value = true
  try {
    const res = await api.readSystemLog({
      file: selectedFile.value,
      source: logSource.value,
      page: page.value,
      page_size: pageSize.value,
      keyword: searchKeyword.value,
      reverse: reverseOrder.value ? 'true' : ''
    })
    logLines.value = res.data?.lines || []
    logInfo.value = {
      size_human: res.data?.size_human,
      total_lines: res.data?.total_lines,
      total_pages: res.data?.total_pages
    }

    // 滚动到顶部
    nextTick(() => {
      if (logPreRef.value) {
        logPreRef.value.scrollTop = 0
      }
    })
  } catch (e) {
    console.error('Failed to load log content:', e)
    ElMessage.error('加载日志失败: ' + (e.message || '未知错误'))
  } finally {
    loadingContent.value = false
  }
}

// 查看实时日志
async function tailLog() {
  if (!selectedFile.value) return

  loadingTail.value = true
  try {
    const res = await api.tailSystemLog({
      file: selectedFile.value,
      source: logSource.value,
      lines: 200
    })
    logLines.value = res.data?.lines || []
    logInfo.value = {
      size_human: res.data?.size_human,
      total_lines: res.data?.count,
      total_pages: 1
    }
    page.value = 1

    // 滚动到底部
    nextTick(() => {
      if (logPreRef.value) {
        logPreRef.value.scrollTop = logPreRef.value.scrollHeight
      }
    })

    ElMessage.success('已加载最新 ' + (res.data?.count || 0) + ' 行日志')
  } catch (e) {
    console.error('Failed to tail log:', e)
    ElMessage.error('加载实时日志失败: ' + (e.message || '未知错误'))
  } finally {
    loadingTail.value = false
  }
}

// 处理页大小变化
function handlePageSizeChange(size) {
  pageSize.value = size
  page.value = 1
  loadLogContent()
}

// 处理文件操作命令
function handleFileCommand(cmd, row) {
  switch (cmd) {
    case 'view':
      selectFile(row)
      break
    case 'tail':
      selectedFile.value = row.name
      tailLog()
      break
    case 'download':
      window.open(api.downloadSystemLog(row.name, logSource.value), '_blank')
      break
    case 'delete':
      if (logSource.value === 'app') {
        confirmDelete(row.name)
      } else {
        ElMessage.warning('服务器日志不允许删除')
      }
      break
  }
}

// 确认删除
async function confirmDelete(filename) {
  try {
    await ElMessageBox.confirm(
      `确定要删除日志文件 "${filename}" 吗？此操作不可恢复。`,
      '删除确认',
      { type: 'warning' }
    )

    await api.deleteSystemLog(filename, logSource.value)
    ElMessage.success('删除成功')

    if (selectedFile.value === filename) {
      selectedFile.value = ''
      logLines.value = []
      logInfo.value = {}
    }

    loadFiles()
  } catch (e) {
    if (e !== 'cancel') {
      console.error('Failed to delete file:', e)
    }
  }
}

onMounted(() => {
  loadFiles()
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

.log-tabs {
  margin-bottom: 15px;
}

.log-tabs :deep(.el-tabs__item) {
  font-size: 14px;
}

.log-tabs :deep(.el-tabs__item .el-icon) {
  margin-right: 5px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-bar {
  margin-bottom: 15px;
  display: flex;
  align-items: center;
}

.file-list-card {
  height: calc(100vh - 200px);
}

.file-name {
  display: flex;
  align-items: center;
  gap: 5px;
}

.file-name .el-icon {
  color: #909399;
}

.log-content-card {
  height: calc(100vh - 200px);
  display: flex;
  flex-direction: column;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.filename {
  font-weight: bold;
  color: #409eff;
}

.header-tools {
  display: flex;
  align-items: center;
}

.log-content {
  flex: 1;
  overflow: auto;
  min-height: 400px;
  max-height: calc(100vh - 390px);
}

.log-pre {
  margin: 0;
  padding: 15px;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  overflow: auto;
  min-height: 400px;
  max-height: calc(100vh - 390px);
  border-radius: 4px;
}

.empty-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.pagination-wrap {
  margin-top: 15px;
  display: flex;
  justify-content: flex-end;
}

:deep(.selected-row) {
  background-color: #ecf5ff !important;
}

:deep(.el-card__body) {
  padding: 15px;
}
</style>
