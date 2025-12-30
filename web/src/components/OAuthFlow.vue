<!--
 * 文件作用：OAuth授权流程组件，处理多平台授权
 * 负责功能：
 *   - Claude/Gemini/OpenAI OAuth流程
 *   - SessionKey自动授权（支持批量）
 *   - 授权链接生成和Code交换
 *   - 代理配置传递
 * 重要程度：⭐⭐⭐⭐ 重要（OAuth认证）
 * 依赖模块：element-plus, api
-->
<template>
  <div class="oauth-flow">
    <!-- Claude OAuth流程 -->
    <div v-if="isClaudePlatform" class="oauth-section claude">
      <div class="oauth-header">
        <div class="platform-icon claude">
          <i class="fa-solid fa-brain"></i>
        </div>
        <div class="platform-info">
          <h4>Claude 账户授权</h4>
        </div>
      </div>

      <!-- 授权方式选择 -->
      <div class="auth-method-select">
        <label class="method-label">选择授权方式</label>
        <el-radio-group v-model="authMethod" @change="onAuthMethodChange">
          <el-radio value="manual">手动授权</el-radio>
          <el-radio value="cookie">SessionKey 自动授权</el-radio>
        </el-radio-group>
      </div>

      <!-- SessionKey 自动授权 -->
      <div v-if="authMethod === 'cookie'" class="cookie-auth">
        <el-alert type="info" :closable="false" class="mb-4">
          <template #title>
            <i class="fa-solid fa-cookie"></i>
            使用 claude.ai 的 sessionKey 自动完成 OAuth 授权流程
          </template>
        </el-alert>

        <el-form label-position="top">
          <el-form-item>
            <template #label>
              <span class="label-with-badge">
                <i class="fa-solid fa-cookie text-blue-500"></i>
                sessionKey
                <el-tag v-if="parsedSessionKeyCount > 1" size="small" type="primary" class="ml-2">
                  {{ parsedSessionKeyCount }} 个
                </el-tag>
                <el-button link type="primary" class="ml-2" @click="showSessionKeyHelp = !showSessionKeyHelp">
                  <i class="fa-solid fa-question-circle"></i>
                </el-button>
              </span>
            </template>
            <el-input
              v-model="sessionKey"
              type="textarea"
              :rows="3"
              placeholder="每行一个 sessionKey，例如：&#10;sk-ant-sid01-xxxxx...&#10;sk-ant-sid01-yyyyy..."
            />
            <div v-if="parsedSessionKeyCount > 1" class="session-key-tip">
              <i class="fa-solid fa-info-circle"></i>
              将批量创建 {{ parsedSessionKeyCount }} 个账户
            </div>
          </el-form-item>
        </el-form>

        <!-- SessionKey 帮助说明 -->
        <el-collapse-transition>
          <div v-if="showSessionKeyHelp" class="help-section">
            <h5><i class="fa-solid fa-lightbulb"></i> 如何获取 sessionKey</h5>
            <ol>
              <li>在浏览器中登录 <strong>claude.ai</strong></li>
              <li>按 <kbd>F12</kbd> 打开开发者工具</li>
              <li>切换到 <strong>Application</strong>（应用）标签页</li>
              <li>在左侧找到 <strong>Cookies</strong> → <strong>https://claude.ai</strong></li>
              <li>找到键为 <strong>sessionKey</strong> 的那一行</li>
              <li>复制其 <strong>Value</strong>（值）列的内容</li>
            </ol>
            <p class="tip">
              <i class="fa-solid fa-info-circle"></i>
              sessionKey 通常以 <code>sk-ant-sid01-</code> 开头
            </p>
          </div>
        </el-collapse-transition>

        <!-- 错误信息 -->
        <el-alert v-if="cookieAuthError" type="error" :closable="false" class="mb-4">
          <template #title>
            <i class="fa-solid fa-exclamation-circle"></i>
            {{ cookieAuthError }}
          </template>
        </el-alert>

        <!-- 授权按钮 -->
        <el-button
          type="primary"
          size="large"
          class="w-full"
          :loading="cookieAuthLoading"
          :disabled="!sessionKey.trim()"
          @click="handleCookieAuth"
        >
          <template v-if="cookieAuthLoading && batchProgress.total > 1">
            正在授权 {{ batchProgress.current }}/{{ batchProgress.total }}...
          </template>
          <template v-else-if="cookieAuthLoading">正在授权...</template>
          <template v-else>
            <i class="fa-solid fa-magic"></i> 开始自动授权
          </template>
        </el-button>
      </div>

      <!-- 手动授权流程 -->
      <div v-else class="manual-auth">
        <p class="auth-desc">请按照以下步骤完成 Claude 账户的授权：</p>

        <div class="auth-steps">
          <!-- 步骤1: 生成授权链接 -->
          <div class="step-card">
            <div class="step-header">
              <div class="step-num">1</div>
              <div class="step-title">点击下方按钮生成授权链接</div>
            </div>
            <div class="step-content">
              <el-button v-if="!authUrl" type="primary" :loading="loading" @click="generateAuthUrl">
                <i v-if="!loading" class="fa-solid fa-link"></i>
                {{ loading ? '生成中...' : '生成授权链接' }}
              </el-button>
              <div v-else class="auth-url-container">
                <el-input v-model="authUrl" readonly>
                  <template #append>
                    <el-button @click="copyAuthUrl">
                      <i :class="copied ? 'fa-solid fa-check text-green-500' : 'fa-solid fa-copy'"></i>
                    </el-button>
                  </template>
                </el-input>
                <el-button link type="primary" class="mt-2" @click="regenerateAuthUrl">
                  <i class="fa-solid fa-sync-alt"></i> 重新生成
                </el-button>
              </div>
            </div>
          </div>

          <!-- 步骤2: 打开链接授权 -->
          <div class="step-card">
            <div class="step-header">
              <div class="step-num">2</div>
              <div class="step-title">在浏览器中打开链接并完成授权</div>
            </div>
            <div class="step-content">
              <p>请在新标签页中打开授权链接，登录您的 Claude 账户并授权。</p>
              <el-alert type="warning" :closable="false" class="mt-2">
                <template #title>
                  <i class="fa-solid fa-exclamation-triangle"></i>
                  <strong>注意：</strong>如果您设置了代理，请确保浏览器也使用相同的代理访问授权页面。
                </template>
              </el-alert>
            </div>
          </div>

          <!-- 步骤3: 输入授权码 -->
          <div class="step-card">
            <div class="step-header">
              <div class="step-num">3</div>
              <div class="step-title">输入 Authorization Code</div>
            </div>
            <div class="step-content">
              <p class="mb-3">授权完成后，页面会显示一个 <strong>Authorization Code</strong>，请将其复制并粘贴到下方输入框：</p>
              <el-input
                v-model="authCode"
                type="textarea"
                :rows="3"
                placeholder="粘贴从 Claude 页面获取的 Authorization Code..."
              />
              <p class="tip mt-2">
                <i class="fa-solid fa-info-circle"></i>
                请粘贴从 Claude 页面复制的 Authorization Code
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Gemini OAuth流程 -->
    <div v-else-if="platform === 'gemini'" class="oauth-section gemini">
      <div class="oauth-header">
        <div class="platform-icon gemini">
          <i class="fa-brands fa-google"></i>
        </div>
        <div class="platform-info">
          <h4>Gemini 账户授权</h4>
        </div>
      </div>

      <p class="auth-desc">请按照以下步骤完成 Gemini 账户的授权：</p>

      <div class="auth-steps">
        <div class="step-card">
          <div class="step-header">
            <div class="step-num">1</div>
            <div class="step-title">点击下方按钮生成授权链接</div>
          </div>
          <div class="step-content">
            <el-button v-if="!authUrl" type="primary" :loading="loading" @click="generateAuthUrl">
              <i v-if="!loading" class="fa-solid fa-link"></i>
              {{ loading ? '生成中...' : '生成授权链接' }}
            </el-button>
            <div v-else class="auth-url-container">
              <el-input v-model="authUrl" readonly>
                <template #append>
                  <el-button @click="copyAuthUrl">
                    <i :class="copied ? 'fa-solid fa-check text-green-500' : 'fa-solid fa-copy'"></i>
                  </el-button>
                </template>
              </el-input>
              <el-button link type="primary" class="mt-2" @click="regenerateAuthUrl">
                <i class="fa-solid fa-sync-alt"></i> 重新生成
              </el-button>
            </div>
          </div>
        </div>

        <div class="step-card">
          <div class="step-header">
            <div class="step-num">2</div>
            <div class="step-title">在浏览器中打开链接并完成授权</div>
          </div>
          <div class="step-content">
            <p>请在新标签页中打开授权链接，登录您的 Google 账户并授权。</p>
          </div>
        </div>

        <div class="step-card">
          <div class="step-header">
            <div class="step-num">3</div>
            <div class="step-title">输入 Authorization Code</div>
          </div>
          <div class="step-content">
            <el-input
              v-model="authCode"
              type="textarea"
              :rows="3"
              placeholder="粘贴从 Gemini 页面获取的 Authorization Code..."
            />
          </div>
        </div>
      </div>
    </div>

    <!-- OpenAI OAuth流程 (包括 openai-responses) -->
    <div v-else-if="platform === 'openai' || platform === 'openai-responses'" class="oauth-section openai">
      <div class="oauth-header">
        <div class="platform-icon openai">
          <i class="fa-solid fa-robot"></i>
        </div>
        <div class="platform-info">
          <h4>OpenAI 账户授权</h4>
        </div>
      </div>

      <p class="auth-desc">请按照以下步骤完成 OpenAI 账户的授权：</p>

      <div class="auth-steps">
        <div class="step-card">
          <div class="step-header">
            <div class="step-num">1</div>
            <div class="step-title">点击下方按钮生成授权链接</div>
          </div>
          <div class="step-content">
            <el-button v-if="!authUrl" type="primary" :loading="loading" @click="generateAuthUrl">
              <i v-if="!loading" class="fa-solid fa-link"></i>
              {{ loading ? '生成中...' : '生成授权链接' }}
            </el-button>
            <div v-else class="auth-url-container">
              <el-input v-model="authUrl" readonly>
                <template #append>
                  <el-button @click="copyAuthUrl">
                    <i :class="copied ? 'fa-solid fa-check text-green-500' : 'fa-solid fa-copy'"></i>
                  </el-button>
                </template>
              </el-input>
              <el-button link type="primary" class="mt-2" @click="regenerateAuthUrl">
                <i class="fa-solid fa-sync-alt"></i> 重新生成
              </el-button>
            </div>
          </div>
        </div>

        <div class="step-card">
          <div class="step-header">
            <div class="step-num">2</div>
            <div class="step-title">在浏览器中打开链接并完成授权</div>
          </div>
          <div class="step-content">
            <p>请在新标签页中打开授权链接，登录您的 OpenAI 账户并授权。</p>
            <el-alert type="info" :closable="false" class="mt-2">
              <template #title>
                <i class="fa-solid fa-clock"></i>
                <strong>重要提示：</strong>授权后页面可能会加载较长时间，请耐心等待。
              </template>
              当浏览器地址栏变为 <strong>http://localhost:1455/...</strong> 开头时，表示授权已完成。
            </el-alert>
          </div>
        </div>

        <div class="step-card">
          <div class="step-header">
            <div class="step-num">3</div>
            <div class="step-title">输入授权链接或 Code</div>
          </div>
          <div class="step-content">
            <el-input
              v-model="authCode"
              type="textarea"
              :rows="3"
              placeholder="方式1：复制完整的链接（http://localhost:1455/auth/callback?code=...）&#10;方式2：仅复制 code 参数的值&#10;系统会自动识别并提取所需信息"
            />
            <el-alert type="info" :closable="false" class="mt-2">
              <template #title>
                <i class="fa-solid fa-lightbulb"></i>
                <strong>提示：</strong>您可以直接复制整个链接或仅复制 code 参数值，系统会自动识别。
              </template>
            </el-alert>
          </div>
        </div>
      </div>
    </div>

    <!-- 操作按钮 -->
    <div class="oauth-actions">
      <el-button @click="$emit('back')">上一步</el-button>
      <el-button
        v-if="!(isClaudePlatform && authMethod === 'cookie')"
        type="primary"
        :disabled="!canExchange"
        :loading="exchanging"
        @click="exchangeCode"
      >
        {{ exchanging ? '验证中...' : '完成授权' }}
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { ensureFontAwesomeLoaded } from '@/utils/fontawesome'
ensureFontAwesomeLoaded()

import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'

const props = defineProps({
  platform: {
    type: String,
    required: true
  },
  proxy: {
    type: Object,
    default: null
  },
  authMode: {
    type: String,
    default: 'oauth',  // 'oauth' | 'cookie'
    validator: (value) => ['oauth', 'cookie'].includes(value)
  }
})

const emit = defineEmits(['success', 'back'])

// 状态
const loading = ref(false)
const exchanging = ref(false)
const authUrl = ref('')
const authCode = ref('')
const copied = ref(false)
const sessionId = ref('')

// Cookie自动授权相关状态
const authMethod = ref('manual')
const sessionKey = ref('')
const cookieAuthLoading = ref(false)
const cookieAuthError = ref('')
const showSessionKeyHelp = ref(false)
const batchProgress = ref({ current: 0, total: 0 })

// 根据 authMode 属性设置初始授权方式
watch(() => props.authMode, (mode) => {
  if (mode === 'cookie') {
    authMethod.value = 'cookie'
  } else {
    authMethod.value = 'manual'
  }
}, { immediate: true })

// 判断是否是 Claude 平台
const isClaudePlatform = computed(() => {
  return props.platform === 'claude' || props.platform === 'claude-official'
})

// 解析后的 sessionKey 数量
const parsedSessionKeyCount = computed(() => {
  return sessionKey.value
    .split('\n')
    .map(s => s.trim())
    .filter(s => s.length > 0).length
})

// 计算是否可以交换code
const canExchange = computed(() => {
  return authUrl.value && authCode.value.trim()
})

// 监听授权码输入，自动提取URL中的code参数
watch(authCode, (newValue) => {
  if (!newValue || typeof newValue !== 'string') return
  const trimmedValue = newValue.trim()
  if (!trimmedValue) return

  const isUrl = trimmedValue.startsWith('http://') || trimmedValue.startsWith('https://')
  if (isUrl) {
    if (trimmedValue.startsWith('http://localhost:45462') || trimmedValue.startsWith('http://localhost:1455')) {
      try {
        const url = new URL(trimmedValue)
        const code = url.searchParams.get('code')
        if (code) {
          authCode.value = code
          ElMessage.success('成功提取授权码！')
        }
      } catch (e) {
        console.error('Failed to parse URL:', e)
      }
    }
  }
})

// 获取 OAuth API 使用的 platform（openai-responses 映射为 openai）
function getOAuthPlatform() {
  if (props.platform === 'openai-responses') {
    return 'openai'
  }
  return props.platform
}

// 生成授权URL
async function generateAuthUrl() {
  loading.value = true
  authUrl.value = ''
  authCode.value = ''
  sessionId.value = ''

  try {
    // 构建代理配置
    const proxyConfig = props.proxy?.enabled ? {
      type: props.proxy.type,
      host: props.proxy.host,
      port: parseInt(props.proxy.port),
      username: props.proxy.username || null,
      password: props.proxy.password || null
    } : null

    const res = await api.generateOAuthUrl(getOAuthPlatform(), proxyConfig)
    authUrl.value = res.data.auth_url
    sessionId.value = res.data.session_id

    ElMessage.success('授权链接已生成')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || e.message || '生成授权链接失败')
  } finally {
    loading.value = false
  }
}

// 重新生成授权URL
function regenerateAuthUrl() {
  authUrl.value = ''
  authCode.value = ''
  sessionId.value = ''
  generateAuthUrl()
}

// 复制授权URL
async function copyAuthUrl() {
  if (!authUrl.value) {
    ElMessage.warning('请先生成授权链接')
    return
  }

  try {
    await navigator.clipboard.writeText(authUrl.value)
    copied.value = true
    ElMessage.success('链接已复制')
    setTimeout(() => { copied.value = false }, 2000)
  } catch (e) {
    // 降级方案
    const input = document.createElement('input')
    input.value = authUrl.value
    document.body.appendChild(input)
    input.select()
    document.execCommand('copy')
    document.body.removeChild(input)
    copied.value = true
    ElMessage.success('链接已复制')
    setTimeout(() => { copied.value = false }, 2000)
  }
}

// 交换授权码
async function exchangeCode() {
  if (!canExchange.value) return

  exchanging.value = true
  try {
    const proxyConfig = props.proxy?.enabled ? {
      type: props.proxy.type,
      host: props.proxy.host,
      port: parseInt(props.proxy.port),
      username: props.proxy.username || null,
      password: props.proxy.password || null
    } : null

    const res = await api.exchangeOAuthCode(getOAuthPlatform(), authCode.value, sessionId.value, proxyConfig)
    const tokenInfo = res.data

    emit('success', tokenInfo)
  } catch (e) {
    ElMessage.error(e.response?.data?.message || e.message || '授权失败，请检查授权码是否正确')
  } finally {
    exchanging.value = false
  }
}

// Cookie自动授权处理（支持批量）
async function handleCookieAuth() {
  const sessionKeys = sessionKey.value
    .split('\n')
    .map(s => s.trim())
    .filter(s => s.length > 0)

  if (sessionKeys.length === 0) {
    cookieAuthError.value = '请输入至少一个 sessionKey'
    return
  }

  cookieAuthLoading.value = true
  cookieAuthError.value = ''
  batchProgress.value = { current: 0, total: sessionKeys.length }

  const proxyConfig = props.proxy?.enabled ? {
    type: props.proxy.type,
    host: props.proxy.host,
    port: parseInt(props.proxy.port),
    username: props.proxy.username || null,
    password: props.proxy.password || null
  } : null

  const results = []
  const errors = []

  for (let i = 0; i < sessionKeys.length; i++) {
    batchProgress.value.current = i + 1
    try {
      const res = await api.oauthByCookie(props.platform, sessionKeys[i], proxyConfig)
      results.push({
        ...res.data,
        session_key: sessionKeys[i]
      })
    } catch (e) {
      errors.push({
        index: i + 1,
        key: sessionKeys[i].substring(0, 20) + '...',
        error: e.response?.data?.message || e.message
      })
    }
  }

  batchProgress.value = { current: 0, total: 0 }
  cookieAuthLoading.value = false

  if (results.length > 0) {
    emit('success', results)
  }

  if (errors.length > 0 && results.length === 0) {
    cookieAuthError.value = '全部授权失败，请检查 sessionKey 是否有效'
  } else if (errors.length > 0) {
    cookieAuthError.value = `${errors.length} 个授权失败`
  }
}

// 切换授权方式时重置状态
function onAuthMethodChange() {
  sessionKey.value = ''
  cookieAuthError.value = ''
  cookieAuthLoading.value = false
  batchProgress.value = { current: 0, total: 0 }
  authUrl.value = ''
  authCode.value = ''
  sessionId.value = ''
}
</script>

<style scoped>
.oauth-flow {
  padding: 16px 0;
}

.oauth-section {
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #e2e8f0;
}

.oauth-section.claude {
  border-color: #c7d2fe;
  background: linear-gradient(135deg, #eef2ff 0%, #e0e7ff 100%);
}

.oauth-section.gemini {
  border-color: #bbf7d0;
  background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
}

.oauth-section.openai {
  border-color: #fed7aa;
  background: linear-gradient(135deg, #fff7ed 0%, #ffedd5 100%);
}

.oauth-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
}

.platform-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 18px;
}

.platform-icon.claude {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.platform-icon.gemini {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.platform-icon.openai {
  background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
}

.platform-info h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.auth-method-select {
  margin-bottom: 20px;
  padding: 16px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.method-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
  margin-bottom: 12px;
}

.cookie-auth {
  padding: 16px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.label-with-badge {
  display: flex;
  align-items: center;
  gap: 8px;
}

.session-key-tip {
  font-size: 12px;
  color: #3b82f6;
  margin-top: 8px;
}

.help-section {
  margin-top: 16px;
  padding: 16px;
  background: #fffbeb;
  border: 1px solid #fcd34d;
  border-radius: 8px;
}

.help-section h5 {
  margin: 0 0 12px;
  color: #92400e;
  font-size: 14px;
  font-weight: 600;
}

.help-section ol {
  margin: 0;
  padding-left: 20px;
  color: #78350f;
  font-size: 13px;
  line-height: 1.8;
}

.help-section .tip {
  margin-top: 12px;
  font-size: 12px;
  color: #d97706;
}

.help-section code {
  background: #fef3c7;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}

.manual-auth {
  padding: 16px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.auth-desc {
  color: #4b5563;
  font-size: 14px;
  margin-bottom: 20px;
}

.auth-steps {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.step-card {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
}

.step-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.step-num {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #3b82f6;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
}

.step-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.step-content {
  padding-left: 36px;
}

.step-content p {
  font-size: 13px;
  color: #6b7280;
  margin: 0;
}

.auth-url-container {
  margin-top: 8px;
}

.tip {
  font-size: 12px;
  color: #6b7280;
}

.oauth-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #e5e7eb;
}

.w-full {
  width: 100%;
}

.mb-4 {
  margin-bottom: 16px;
}

.mb-3 {
  margin-bottom: 12px;
}

.mt-2 {
  margin-top: 8px;
}

.ml-2 {
  margin-left: 8px;
}

kbd {
  background: #e5e7eb;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
}
</style>
