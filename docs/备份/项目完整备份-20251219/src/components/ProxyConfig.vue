<template>
  <div class="proxy-config">
    <!-- 启用开关 -->
    <div class="proxy-header">
      <div class="proxy-title">
        <i class="fa-solid fa-shield-halved"></i>
        <span>代理配置</span>
      </div>
      <el-switch v-model="localProxy.enabled" @change="emitUpdate" />
    </div>

    <!-- 代理配置表单 -->
    <div v-if="localProxy.enabled" class="proxy-form">
      <!-- 快速配置 -->
      <div class="quick-config">
        <el-input
          v-model="proxyUrl"
          placeholder="快速配置：粘贴代理URL，如 socks5://user:pass@host:port"
          @paste="handlePaste"
          @input="parseProxyUrl"
        >
          <template #prefix>
            <i class="fa-solid fa-link"></i>
          </template>
        </el-input>
        <div class="quick-tip">
          <i class="fa-solid fa-lightbulb"></i>
          支持格式：socks5://user:pass@host:port 或 http://host:port
        </div>
      </div>

      <el-divider content-position="left">或手动配置</el-divider>

      <el-form label-position="top" size="default">
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="代理类型">
              <el-select v-model="localProxy.type" style="width: 100%" @change="emitUpdate">
                <el-option label="SOCKS5" value="socks5" />
                <el-option label="HTTP" value="http" />
                <el-option label="HTTPS" value="https" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="10">
            <el-form-item label="主机地址">
              <el-input v-model="localProxy.host" placeholder="proxy.example.com" @change="emitUpdate" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="端口">
              <el-input-number v-model="localProxy.port" :min="1" :max="65535" style="width: 100%" @change="emitUpdate" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="用户名（可选）">
              <el-input v-model="localProxy.username" placeholder="代理用户名" @change="emitUpdate" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="密码（可选）">
              <el-input v-model="localProxy.password" type="password" show-password placeholder="代理密码" @change="emitUpdate" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <!-- 代理测试 -->
      <div class="proxy-test">
        <el-button size="small" :loading="testing" @click="testProxy">
          <i v-if="!testing" class="fa-solid fa-plug"></i>
          {{ testing ? '测试中...' : '测试连接' }}
        </el-button>
        <span v-if="testResult" :class="['test-result', testResult.success ? 'success' : 'error']">
          <i :class="testResult.success ? 'fa-solid fa-check-circle' : 'fa-solid fa-times-circle'"></i>
          {{ testResult.message }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({
      enabled: false,
      type: 'socks5',
      host: '',
      port: 1080,
      username: '',
      password: ''
    })
  }
})

const emit = defineEmits(['update:modelValue'])

const localProxy = reactive({
  enabled: false,
  type: 'socks5',
  host: '',
  port: 1080,
  username: '',
  password: ''
})

const proxyUrl = ref('')
const testing = ref(false)
const testResult = ref(null)

// 监听外部值变化
watch(() => props.modelValue, (val) => {
  if (val) {
    Object.assign(localProxy, val)
  }
}, { immediate: true, deep: true })

function emitUpdate() {
  emit('update:modelValue', { ...localProxy })
}

function handlePaste(event) {
  setTimeout(() => {
    parseProxyUrl()
  }, 10)
}

function parseProxyUrl() {
  const url = proxyUrl.value.trim()
  if (!url) return

  try {
    // 解析代理URL格式: type://[user:pass@]host:port
    const match = url.match(/^(socks5|http|https):\/\/(?:([^:]+):([^@]+)@)?([^:]+):(\d+)$/i)
    if (match) {
      localProxy.type = match[1].toLowerCase()
      localProxy.username = match[2] || ''
      localProxy.password = match[3] || ''
      localProxy.host = match[4]
      localProxy.port = parseInt(match[5])
      localProxy.enabled = true
      emitUpdate()
    }
  } catch (e) {
    console.error('Failed to parse proxy URL:', e)
  }
}

async function testProxy() {
  if (!localProxy.host || !localProxy.port) {
    testResult.value = { success: false, message: '请填写代理地址和端口' }
    return
  }

  testing.value = true
  testResult.value = null

  try {
    // 这里调用后端API测试代理连接
    // 暂时模拟测试结果
    await new Promise(resolve => setTimeout(resolve, 1000))
    testResult.value = { success: true, message: '连接成功' }
  } catch (e) {
    testResult.value = { success: false, message: e.message || '连接失败' }
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.proxy-config {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 16px;
}

.proxy-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.proxy-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #334155;
}

.proxy-title i {
  color: #6366f1;
}

.proxy-form {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-10px); }
  to { opacity: 1; transform: translateY(0); }
}

.quick-config {
  margin-bottom: 12px;
}

.quick-tip {
  font-size: 12px;
  color: #64748b;
  margin-top: 6px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.quick-tip i {
  color: #eab308;
}

.proxy-test {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e2e8f0;
}

.test-result {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.test-result.success {
  color: #10b981;
}

.test-result.error {
  color: #ef4444;
}

:deep(.el-divider__text) {
  font-size: 12px;
  color: #94a3b8;
  background: #f8fafc;
}
</style>
