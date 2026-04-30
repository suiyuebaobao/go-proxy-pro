<!--
 * 文件作用：系统日志页面，查看和管理系统日志
 * 负责功能：
 *   - 应用日志和服务器日志切换
 *   - 日志文件列表和筛选
 *   - 结构化日志解析展示
 *   - 实时日志追踪
 * 重要程度：⭐⭐ 辅助（运维日志）
 * 依赖模块：element-plus, api
-->
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
                <el-checkbox v-model="defaultExpanded" size="small" style="margin-left: 10px;">展开详情</el-checkbox>
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
            <div v-else-if="logEntries.length === 0 && logLines.length === 0" class="empty-tip">
              <el-empty description="没有日志内容" />
            </div>
            <!-- 结构化日志显示（应用日志） -->
            <div v-else-if="logSource === 'app' && logEntries.length > 0" class="log-entries" ref="logPreRef">
              <div
                v-for="(entry, index) in logEntries"
                :key="index"
                :class="['log-entry', getLevelClass(entry.level), { 'expanded': isExpanded(index) }]"
                @click="toggleExpand(index)"
              >
                <template v-if="entry.is_json">
                  <!-- 主信息行 -->
                  <div class="log-main-line">
                    <span class="log-time">{{ formatTime(entry.timestamp) }}</span>
                    <span :class="['log-level', getLevelClass(entry.level)]">{{ entry.level }}</span>
                    <span class="log-module" v-if="entry.module">[{{ entry.module }}]</span>
                    <span class="log-request-id" v-if="entry.request_id">{{ entry.request_id }}</span>
                    <span class="log-message">{{ entry.message }}</span>
                    <!-- 内联关键字段 -->
                    <span class="inline-fields" v-if="hasInlineFields(entry)">
                      <span v-if="entry.fields?.user_id" class="inline-field user-id">
                        <span class="field-icon">👤</span>{{ entry.fields.user_id }}
                      </span>
                      <span v-if="entry.fields?.account_id" class="inline-field account-id">
                        <span class="field-icon">🔑</span>{{ entry.fields.account_id }}
                      </span>
                      <span v-if="entry.fields?.api_key_id" class="inline-field api-key">
                        <span class="field-icon">🎫</span>{{ entry.fields.api_key_id }}
                      </span>
                      <span v-if="entry.fields?.model" class="inline-field model">
                        <span class="field-icon">🤖</span>{{ entry.fields.model }}
                      </span>
                      <span v-if="entry.fields?.client_ip" class="inline-field client-ip">
                        <span class="field-icon">🌐</span>{{ entry.fields.client_ip }}
                      </span>
                      <span v-if="entry.fields?.status" class="inline-field" :class="getStatusClass(entry.fields.status)">
                        {{ entry.fields.status }}
                      </span>
                      <span v-if="entry.fields?.latency !== undefined" class="inline-field latency">
                        {{ formatLatency(entry.fields.latency) }}
                      </span>
                    </span>
                    <span class="expand-indicator" v-if="hasFields(entry)">
                      {{ isExpanded(index) ? '▼' : '▶' }}
                    </span>
                  </div>
                  <!-- 展开的详细字段 (默认展开) -->
                  <div v-if="isExpanded(index) && hasFields(entry)" class="log-fields">
                    <div class="fields-grid">
                      <template v-for="(value, key) in getSortedFields(entry.fields)" :key="key">
                        <div class="log-field" :class="getFieldClass(key)">
                          <span class="field-key">{{ key }}:</span>
                          <span class="field-value" :class="getValueClass(key, value)">{{ formatFieldValue(key, value) }}</span>
                        </div>
                      </template>
                      <div v-if="entry.caller" class="log-field caller-field">
                        <span class="field-key">caller:</span>
                        <span class="field-value caller-value">{{ entry.caller }}</span>
                      </div>
                    </div>
                  </div>
                </template>
                <template v-else>
                  <span class="log-raw">{{ entry.raw }}</span>
                </template>
              </div>
            </div>
            <!-- 原始文本显示（服务器日志或fallback） -->
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
const logEntries = ref([]) // 解析后的日志条目
const logInfo = ref({})
const loadingContent = ref(false)
const loadingTail = ref(false)
const page = ref(1)
const pageSize = ref(200)
const searchKeyword = ref('')
const reverseOrder = ref(false)
const logPreRef = ref(null)
const expandedRows = ref([]) // 展开的行
const defaultExpanded = ref(true) // 默认展开

// 切换日志来源
function handleSourceChange() {
  // 重置状态
  files.value = []
  categories.value = []
  selectedFile.value = ''
  logLines.value = []
  logEntries.value = []
  logInfo.value = {}
  filterCategory.value = ''
  filterDate.value = ''
  page.value = 1
  searchKeyword.value = ''
  reverseOrder.value = false
  expandedRows.value = []

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
  expandedRows.value = []
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
  expandedRows.value = []
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
    logEntries.value = res.data?.entries || []
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
  expandedRows.value = []
  try {
    const res = await api.tailSystemLog({
      file: selectedFile.value,
      source: logSource.value,
      lines: 200
    })
    logLines.value = res.data?.lines || []
    logEntries.value = res.data?.entries || []
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
      logEntries.value = []
      logInfo.value = {}
    }

    loadFiles()
  } catch (e) {
    if (e !== 'cancel') {
      console.error('Failed to delete file:', e)
    }
  }
}

// 获取日志级别样式类
function getLevelClass(level) {
  if (!level) return ''
  const l = level.toUpperCase()
  if (l === 'ERROR' || l === 'FATAL') return 'level-error'
  if (l === 'WARN' || l === 'WARNING') return 'level-warn'
  if (l === 'DEBUG') return 'level-debug'
  return 'level-info'
}

// 格式化时间
function formatTime(timestamp) {
  if (!timestamp) return ''
  // ISO8601 格式: 2025-12-23T10:30:45.123+08:00
  try {
    const date = new Date(timestamp)
    return date.toLocaleString('zh-CN', {
      hour12: false,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  } catch {
    return timestamp
  }
}

// 格式化值
function formatValue(value) {
  if (typeof value === 'object') {
    return JSON.stringify(value)
  }
  return String(value)
}

// 检查是否有额外字段
function hasFields(entry) {
  return (entry.fields && Object.keys(entry.fields).length > 0) || entry.caller
}

// 检查是否有内联字段
function hasInlineFields(entry) {
  if (!entry.fields) return false
  const inlineKeys = ['user_id', 'account_id', 'api_key_id', 'model', 'client_ip', 'status', 'latency']
  return inlineKeys.some(key => entry.fields[key] !== undefined)
}

// 判断是否展开
function isExpanded(index) {
  // 如果默认展开，则未在 expandedRows 中的都是展开的
  // 如果默认收起，则在 expandedRows 中的才是展开的
  if (defaultExpanded.value) {
    return !expandedRows.value.includes(index)
  }
  return expandedRows.value.includes(index)
}

// 切换展开
function toggleExpand(index) {
  const idx = expandedRows.value.indexOf(index)
  if (idx > -1) {
    expandedRows.value.splice(idx, 1)
  } else {
    expandedRows.value.push(index)
  }
}

// 获取状态码样式类
function getStatusClass(status) {
  if (status >= 500) return 'status-error'
  if (status >= 400) return 'status-warn'
  if (status >= 200 && status < 300) return 'status-success'
  return 'status-info'
}

// 格式化延迟
function formatLatency(latency) {
  if (latency === undefined || latency === null) return ''
  if (latency >= 1000) {
    return (latency / 1000).toFixed(2) + 's'
  }
  return latency + 'ms'
}

// 字段排序（重要字段在前）
function getSortedFields(fields) {
  if (!fields) return {}
  const priority = [
    'user_id', 'api_key_id', 'account_id', 'account_name', 'model',
    'client_ip', 'method', 'path', 'status', 'latency',
    'input_tokens', 'output_tokens', 'cache_creation_tokens', 'cache_read_tokens', 'total_tokens',
    'input_cost', 'output_cost', 'cache_create_cost', 'cache_read_cost', 'total_cost',
    'price_rate', 'package_id', 'package_type',
    'request_size', 'response_size', 'host', 'protocol', 'user_agent', 'content_type',
    'attempts', 'max_retries', 'exec_duration', 'total_duration',
    'session_id', 'error'
  ]

  const sorted = {}
  // 先按优先级添加
  for (const key of priority) {
    if (fields[key] !== undefined) {
      sorted[key] = fields[key]
    }
  }
  // 再添加其他字段
  for (const key of Object.keys(fields)) {
    if (sorted[key] === undefined) {
      sorted[key] = fields[key]
    }
  }
  return sorted
}

// 获取字段样式类
function getFieldClass(key) {
  if (['user_id', 'api_key_id', 'account_id'].includes(key)) return 'field-id'
  if (['input_tokens', 'output_tokens', 'total_tokens', 'cache_creation_tokens', 'cache_read_tokens'].includes(key)) return 'field-token'
  if (['input_cost', 'output_cost', 'total_cost', 'cache_create_cost', 'cache_read_cost'].includes(key)) return 'field-cost'
  if (['latency', 'exec_duration', 'total_duration'].includes(key)) return 'field-duration'
  if (key === 'error') return 'field-error'
  if (key === 'client_ip') return 'field-ip'
  if (key === 'model') return 'field-model'
  return ''
}

// 获取值样式类
function getValueClass(key, value) {
  if (key === 'status') {
    if (value >= 500) return 'value-error'
    if (value >= 400) return 'value-warn'
    if (value >= 200 && value < 300) return 'value-success'
  }
  if (key === 'error') return 'value-error'
  return ''
}

// 格式化字段值
function formatFieldValue(key, value) {
  if (value === null || value === undefined) return '-'

  // 费用格式化
  if (['input_cost', 'output_cost', 'total_cost', 'cache_create_cost', 'cache_read_cost'].includes(key)) {
    return '$' + Number(value).toFixed(6)
  }

  // 时长格式化
  if (['latency', 'exec_duration', 'total_duration'].includes(key)) {
    if (typeof value === 'string' && value.includes('ms')) return value
    if (typeof value === 'string' && value.includes('s')) return value
    if (typeof value === 'number') {
      if (value >= 1000) return (value / 1000).toFixed(2) + 's'
      return value + 'ms'
    }
    return value
  }

  // Token 格式化
  if (['input_tokens', 'output_tokens', 'total_tokens', 'cache_creation_tokens', 'cache_read_tokens'].includes(key)) {
    return Number(value).toLocaleString()
  }

  // 对象格式化
  if (typeof value === 'object') {
    return JSON.stringify(value, null, 2)
  }

  return String(value)
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
  color: var(--pink-text);
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
  color: #6b6573;
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
  color: var(--pink-accent, #c97b8b);
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

/* 结构化日志样式 */
.log-entries {
  background: #1e1e1e;
  border-radius: 4px;
  padding: 10px;
  min-height: 400px;
  max-height: calc(100vh - 390px);
  overflow: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
}

.log-entry {
  padding: 4px 8px;
  border-radius: 3px;
  margin-bottom: 2px;
  cursor: pointer;
  transition: background-color 0.2s;
  line-height: 1.6;
}

.log-entry:hover {
  background: rgba(255, 255, 255, 0.08);
}

.log-entry.expanded {
  background: rgba(255, 255, 255, 0.03);
}

.log-main-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}

.log-time {
  color: #6a9955;
  margin-right: 8px;
}

.log-level {
  font-weight: bold;
  padding: 1px 4px;
  border-radius: 2px;
  margin-right: 8px;
  font-size: 11px;
}

.level-error .log-level,
.log-level.level-error {
  background: #5a1d1d;
  color: #f48771;
}

.level-warn .log-level,
.log-level.level-warn {
  background: #5a4a1d;
  color: #cca700;
}

.level-info .log-level,
.log-level.level-info {
  background: #1d3a5a;
  color: #4fc1ff;
}

.level-debug .log-level,
.log-level.level-debug {
  background: #2d2d2d;
  color: #888;
}

.log-module {
  color: #c586c0;
  margin-right: 8px;
}

.log-request-id {
  color: #ce9178;
  margin-right: 8px;
  font-size: 11px;
  opacity: 0.8;
}

.log-message {
  color: #d4d4d4;
}

.log-raw {
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 内联字段样式 */
.inline-fields {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-left: 10px;
}

.inline-field {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  background: rgba(255, 255, 255, 0.1);
}

.inline-field .field-icon {
  margin-right: 3px;
  font-size: 10px;
}

.inline-field.user-id {
  background: rgba(79, 193, 255, 0.2);
  color: #4fc1ff;
}

.inline-field.account-id {
  background: rgba(197, 134, 192, 0.2);
  color: #c586c0;
}

.inline-field.api-key {
  background: rgba(78, 201, 176, 0.2);
  color: #4ec9b0;
}

.inline-field.model {
  background: rgba(220, 220, 170, 0.2);
  color: #dcdcaa;
}

.inline-field.client-ip {
  background: rgba(156, 220, 254, 0.2);
  color: #9cdcfe;
}

.inline-field.latency {
  background: rgba(206, 145, 120, 0.2);
  color: #ce9178;
}

.inline-field.status-success {
  background: rgba(106, 153, 85, 0.3);
  color: #6a9955;
}

.inline-field.status-warn {
  background: rgba(204, 167, 0, 0.3);
  color: #cca700;
}

.inline-field.status-error {
  background: rgba(244, 135, 113, 0.3);
  color: #f48771;
}

.expand-indicator {
  margin-left: auto;
  color: #666;
  font-size: 10px;
  padding: 2px 5px;
}

.log-fields {
  margin-top: 8px;
  padding: 10px 12px;
  background: rgba(0, 0, 0, 0.4);
  border-radius: 4px;
  border-left: 3px solid #4fc1ff;
}

.fields-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 4px 20px;
}

.log-field {
  display: flex;
  align-items: flex-start;
  padding: 2px 0;
}

.field-key {
  color: #9cdcfe;
  margin-right: 8px;
  min-width: 120px;
  font-weight: 500;
}

.field-value {
  color: #ce9178;
  word-break: break-all;
}

/* 字段类型样式 */
.field-id .field-key { color: #4fc1ff; }
.field-token .field-key { color: #b5cea8; }
.field-token .field-value { color: #b5cea8; font-weight: bold; }
.field-cost .field-key { color: #dcdcaa; }
.field-cost .field-value { color: #dcdcaa; font-weight: bold; }
.field-duration .field-key { color: #ce9178; }
.field-duration .field-value { color: #ce9178; }
.field-error .field-key { color: #f48771; }
.field-error .field-value { color: #f48771; }
.field-ip .field-key { color: #9cdcfe; }
.field-model .field-key { color: #c586c0; }
.field-model .field-value { color: #c586c0; }

.caller-field {
  grid-column: 1 / -1;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  padding-top: 6px;
  margin-top: 4px;
}

.caller-value {
  color: #888 !important;
  font-size: 11px;
}

/* 值样式 */
.value-success { color: #6a9955 !important; }
.value-warn { color: #cca700 !important; }
.value-error { color: #f48771 !important; }

/* 按日志级别设置整行背景 */
.log-entry.level-error {
  background: rgba(244, 135, 113, 0.1);
  border-left: 3px solid #f48771;
}

.log-entry.level-warn {
  background: rgba(204, 167, 0, 0.1);
  border-left: 3px solid #cca700;
}

/* 原始文本样式 */
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
