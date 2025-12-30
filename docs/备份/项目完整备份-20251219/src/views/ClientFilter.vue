<template>
  <div class="client-filter-page">
    <!-- 主开关 -->
    <el-card class="main-card">
      <div class="main-switch">
        <div class="switch-info">
          <h2>客户端过滤</h2>
          <p class="description">只允许指定客户端访问 API</p>
        </div>
        <el-switch
          v-model="config.filter_enabled"
          size="large"
          active-text="已开启"
          inactive-text="已关闭"
          @change="saveConfig"
          :loading="configLoading"
        />
      </div>
    </el-card>

    <!-- 过滤关闭提示 -->
    <el-alert
      v-if="!config.filter_enabled"
      title="过滤功能已关闭，所有请求都将被允许"
      type="info"
      :closable="false"
      show-icon
      style="margin-bottom: 20px;"
    />

    <template v-if="config.filter_enabled">
      <!-- 允许的客户端 -->
      <el-card class="clients-card">
        <h3>允许的客户端</h3>
        <div class="client-list">
          <div
            v-for="ct in clientTypes"
            :key="ct.id"
            class="client-item"
            :class="{ enabled: ct.enabled }"
            @click="toggleClient(ct)"
          >
            <span class="client-icon">{{ ct.icon }}</span>
            <div class="client-info">
              <span class="client-name">{{ ct.name }}</span>
              <span class="client-desc">{{ ct.description }}</span>
            </div>
            <el-icon v-if="ct.enabled" class="check-icon"><Check /></el-icon>
          </div>
        </div>
      </el-card>

      <!-- 模式选择 -->
      <el-card class="mode-card">
        <h3>验证模式</h3>
        <el-radio-group v-model="config.filter_mode" @change="saveConfig" size="large">
          <el-radio-button value="simple">
            <div class="mode-option">
              <span class="mode-title">简单模式</span>
              <span class="mode-desc">宽松验证</span>
            </div>
          </el-radio-button>
          <el-radio-button value="strict">
            <div class="mode-option">
              <span class="mode-title">严格模式</span>
              <span class="mode-desc">完整验证</span>
            </div>
          </el-radio-button>
        </el-radio-group>

        <!-- 模式说明 -->
        <div class="mode-detail">
          <template v-if="config.filter_mode === 'simple'">
            <p><strong>简单模式</strong>：User-Agent 以 <code>claude-cli/版本号</code> 开头即可</p>
          </template>
          <template v-else>
            <p><strong>严格模式</strong>：User-Agent 必须完整格式 <code>claude-cli/版本 (external, cli)</code></p>
          </template>
        </div>
      </el-card>

      <!-- 验证测试 -->
      <el-card class="test-card">
        <el-collapse>
          <el-collapse-item title="验证测试">
            <el-form label-width="100px" size="default">
              <el-form-item label="User-Agent">
                <el-input
                  v-model="testData.user_agent"
                  placeholder='claude-cli/1.0.15 (external, cli)'
                />
              </el-form-item>
              <el-form-item label="请求头">
                <el-input
                  v-model="testData.headersJson"
                  type="textarea"
                  :rows="2"
                  placeholder='{"x-app": "claude-code", "anthropic-version": "2023-06-01"}'
                />
              </el-form-item>
              <el-form-item label="请求体">
                <el-input
                  v-model="testData.bodyJson"
                  type="textarea"
                  :rows="3"
                  placeholder='{"system": [{"text": "You are Claude Code..."}]}'
                />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="runTest" :loading="testing">
                  测试
                </el-button>
                <el-button @click="fillTestExample">填入示例</el-button>
              </el-form-item>
            </el-form>

            <div v-if="testResult" class="test-result">
              <el-alert
                :title="testResult.allowed ? '验证通过' : '验证失败'"
                :type="testResult.allowed ? 'success' : 'error'"
                show-icon
                :closable="false"
              >
                <template #default>
                  <p>客户端: {{ testResult.client_name || '未识别' }}</p>
                  <p v-if="testResult.details?.reason">原因: {{ testResult.details.reason }}</p>
                </template>
              </el-alert>
            </div>
          </el-collapse-item>
        </el-collapse>
      </el-card>
    </template>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import api from '@/api'

// 配置
const config = reactive({
  filter_enabled: false,
  filter_mode: 'simple',
})
const configLoading = ref(false)

// 客户端列表
const clientTypes = ref([])

// 测试
const testing = ref(false)
const testData = reactive({
  user_agent: '',
  headersJson: '',
  bodyJson: ''
})
const testResult = ref(null)

// 加载数据
async function loadData() {
  try {
    const [configRes, typesRes] = await Promise.all([
      api.getClientFilterConfig(),
      api.getClientTypes()
    ])
    Object.assign(config, configRes.data)
    if (!config.filter_mode) {
      config.filter_mode = 'simple'
    }
    clientTypes.value = typesRes.data || []
  } catch (e) {
    // handled
  }
}

// 保存配置
async function saveConfig() {
  configLoading.value = true
  try {
    await api.updateClientFilterConfig(config)
    ElMessage.success('配置已保存')
  } catch (e) {
    await loadData()
  } finally {
    configLoading.value = false
  }
}

// 切换客户端
async function toggleClient(ct) {
  ct.enabled = !ct.enabled
  try {
    await api.toggleClientType(ct.id)
    ElMessage.success(`${ct.name} 已${ct.enabled ? '启用' : '禁用'}`)
  } catch (e) {
    ct.enabled = !ct.enabled
  }
}

// 填入测试示例
function fillTestExample() {
  testData.user_agent = 'claude-cli/1.0.15 (external, cli)'
  testData.headersJson = JSON.stringify({
    'x-app': 'claude-code',
    'anthropic-version': '2023-06-01',
    'anthropic-beta': 'oauth-2025-04-20',
    'x-stainless-os': 'Linux'
  }, null, 2)
  testData.bodyJson = JSON.stringify({
    system: [{ text: 'You are Claude Code, Anthropic\'s official CLI for Claude.' }],
    metadata: {
      user_id: 'user_d98385411c93cd074b2cefd5c9831fe77f24a53e4ecdcd1f830bba586fe62cb9_account__session_17cf0fd3-d51b-4b59-977d-b899dafb3022'
    }
  }, null, 2)
}

// 测试
async function runTest() {
  testing.value = true
  testResult.value = null
  try {
    let headers = {}, body = {}
    try {
      if (testData.headersJson) headers = JSON.parse(testData.headersJson)
    } catch {
      ElMessage.error('请求头 JSON 格式错误')
      testing.value = false
      return
    }
    try {
      if (testData.bodyJson) body = JSON.parse(testData.bodyJson)
    } catch {
      ElMessage.error('请求体 JSON 格式错误')
      testing.value = false
      return
    }

    const res = await api.testClientFilter({
      user_agent: testData.user_agent,
      path: '/v1/messages',
      headers,
      body
    })
    testResult.value = res.data
  } catch (e) {
    // handled
  } finally {
    testing.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.client-filter-page {
  max-width: 800px;
  margin: 0 auto;
}

.main-card {
  margin-bottom: 20px;
}

.main-switch {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.switch-info h2 {
  margin: 0 0 5px 0;
  font-size: 18px;
  color: #303133;
}

.switch-info .description {
  margin: 0;
  font-size: 13px;
  color: #909399;
}

.clients-card, .mode-card, .test-card {
  margin-bottom: 20px;
}

.clients-card h3, .mode-card h3 {
  margin: 0 0 15px 0;
  font-size: 16px;
  color: #303133;
}

.client-list {
  display: flex;
  gap: 15px;
  flex-wrap: wrap;
}

.client-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 15px 20px;
  border: 2px solid #e4e7ed;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  min-width: 200px;
  position: relative;
}

.client-item:hover {
  border-color: #409eff;
}

.client-item.enabled {
  border-color: #67c23a;
  background: #f0f9eb;
}

.client-icon {
  font-size: 28px;
}

.client-info {
  display: flex;
  flex-direction: column;
}

.client-name {
  font-weight: 600;
  color: #303133;
}

.client-desc {
  font-size: 12px;
  color: #909399;
}

.check-icon {
  position: absolute;
  right: 10px;
  top: 10px;
  color: #67c23a;
  font-size: 18px;
}

.mode-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 5px 20px;
}

.mode-title {
  font-weight: 500;
}

.mode-desc {
  font-size: 11px;
  color: #909399;
  margin-top: 2px;
}

.mode-detail {
  margin-top: 15px;
  padding: 10px 15px;
  background: #f5f7fa;
  border-radius: 6px;
  font-size: 13px;
}

.mode-detail p {
  margin: 0;
}

.mode-detail code {
  background: #e6e8eb;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 12px;
}

.test-result {
  margin-top: 15px;
}

.test-result p {
  margin: 3px 0;
  font-size: 13px;
}
</style>
