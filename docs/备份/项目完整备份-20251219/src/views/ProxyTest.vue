<template>
  <div class="proxy-test">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>API 代理测试</span>
        </div>
      </template>

      <el-form :model="form" label-width="120px">
        <!-- 客户端模式选择 -->
        <el-form-item label="客户端模式">
          <el-radio-group v-model="form.clientMode" @change="onClientModeChange">
            <el-radio-button value="http">
              <el-icon><Monitor /></el-icon>
              HTTP 直连
            </el-radio-button>
            <el-radio-button value="claude_code">
              <el-icon><Platform /></el-icon>
              Claude Code
            </el-radio-button>
            <el-radio-button value="sdk">
              <el-icon><Cpu /></el-icon>
              SDK 模式
            </el-radio-button>
            <el-radio-button value="curl">
              <el-icon><Document /></el-icon>
              cURL 命令
            </el-radio-button>
          </el-radio-group>
        </el-form-item>

        <!-- 客户端模式说明 -->
        <el-alert
          :title="clientModeDescription"
          type="info"
          :closable="false"
          style="margin-bottom: 20px"
          show-icon
        />

        <el-form-item label="接口格式">
          <el-radio-group v-model="form.format">
            <el-radio value="openai">OpenAI 兼容</el-radio>
            <el-radio value="claude">Claude 原生</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="模型">
          <el-select v-model="form.model" placeholder="选择模型" style="width: 300px">
            <el-option-group label="Claude 模型">
              <el-option value="claude-opus-4-5-20251101" label="Claude Opus 4.5" />
              <el-option value="claude-sonnet-4-20250514" label="Claude Sonnet 4" />
              <el-option value="claude-3-5-sonnet-20241022" label="Claude 3.5 Sonnet" />
              <el-option value="claude-3-opus-20240229" label="Claude 3 Opus" />
              <el-option value="claude-3-haiku-20240307" label="Claude 3 Haiku" />
            </el-option-group>
            <el-option-group label="OpenAI 模型">
              <el-option value="gpt-4o" label="GPT-4o" />
              <el-option value="gpt-4-turbo" label="GPT-4 Turbo" />
              <el-option value="gpt-3.5-turbo" label="GPT-3.5 Turbo" />
            </el-option-group>
            <el-option-group label="Gemini 模型">
              <el-option value="gemini-pro" label="Gemini Pro" />
              <el-option value="gemini-1.5-pro" label="Gemini 1.5 Pro" />
            </el-option-group>
          </el-select>
          <el-input
            v-model="form.customModel"
            placeholder="或输入自定义模型名"
            style="width: 200px; margin-left: 10px"
          />
        </el-form-item>

        <el-form-item label="账户类型">
          <el-select v-model="form.accountType" placeholder="自动选择" clearable style="width: 200px">
            <el-option value="" label="自动选择" />
            <el-option value="claude-official" label="Claude Official" />
            <el-option value="claude-console" label="Claude Console" />
            <el-option value="bedrock" label="AWS Bedrock" />
            <el-option value="ccr" label="Claude CCR" />
            <el-option value="openai" label="OpenAI" />
            <el-option value="azure-openai" label="Azure OpenAI" />
            <el-option value="gemini" label="Gemini" />
            <el-option value="gemini-api" label="Gemini API" />
          </el-select>
        </el-form-item>

        <el-form-item label="System Prompt">
          <el-input
            v-model="form.system"
            type="textarea"
            :rows="2"
            placeholder="可选的系统提示词"
          />
        </el-form-item>

        <el-form-item label="消息内容">
          <el-input
            v-model="form.message"
            type="textarea"
            :rows="4"
            placeholder="请输入要发送的消息"
          />
        </el-form-item>

        <el-form-item label="流式响应">
          <el-switch v-model="form.stream" />
        </el-form-item>

        <el-form-item label="参数设置">
          <el-col :span="8">
            <span style="margin-right: 10px">Max Tokens:</span>
            <el-input-number v-model="form.maxTokens" :min="1" :max="8192" />
          </el-col>
          <el-col :span="8">
            <span style="margin-right: 10px">Temperature:</span>
            <el-input-number v-model="form.temperature" :min="0" :max="2" :step="0.1" />
          </el-col>
          <el-col :span="8">
            <span style="margin-right: 10px">Top P:</span>
            <el-input-number v-model="form.topP" :min="0" :max="1" :step="0.1" />
          </el-col>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="sendRequest" :loading="loading">
            <el-icon><VideoPlay /></el-icon>
            发送请求
          </el-button>
          <el-button @click="clearResponse">
            <el-icon><Delete /></el-icon>
            清空响应
          </el-button>
          <el-button v-if="form.clientMode === 'curl'" @click="copyCurl" type="success">
            <el-icon><CopyDocument /></el-icon>
            复制 cURL
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 请求信息卡片 -->
    <el-card style="margin-top: 20px">
      <template #header>
        <div class="card-header">
          <span>请求信息</span>
          <el-tag :type="getClientModeTagType(form.clientMode)" size="small">
            {{ getClientModeLabel(form.clientMode) }}
          </el-tag>
        </div>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="请求地址">{{ requestInfo.url }}</el-descriptions-item>
        <el-descriptions-item label="请求方法">POST</el-descriptions-item>
      </el-descriptions>

      <!-- 请求头信息 -->
      <div style="margin-top: 15px">
        <strong>请求头:</strong>
        <pre class="code-block headers-block">{{ requestInfo.headers }}</pre>
      </div>

      <!-- 请求体 -->
      <div style="margin-top: 10px">
        <strong>请求体:</strong>
        <pre class="code-block">{{ requestInfo.body }}</pre>
      </div>

      <!-- cURL 命令 -->
      <div v-if="form.clientMode === 'curl'" style="margin-top: 15px">
        <strong>cURL 命令:</strong>
        <pre class="code-block curl-block">{{ curlCommand }}</pre>
      </div>

      <!-- SDK 代码示例 -->
      <div v-if="form.clientMode === 'sdk'" style="margin-top: 15px">
        <div style="display: flex; align-items: center; margin-bottom: 10px">
          <strong>SDK 代码示例:</strong>
          <el-radio-group v-model="sdkLanguage" style="margin-left: 15px" size="small">
            <el-radio-button value="python">Python</el-radio-button>
            <el-radio-button value="nodejs">Node.js</el-radio-button>
            <el-radio-button value="go">Go</el-radio-button>
          </el-radio-group>
        </div>
        <pre class="code-block sdk-block">{{ sdkCode }}</pre>
      </div>
    </el-card>

    <!-- 响应结果卡片 -->
    <el-card style="margin-top: 20px">
      <template #header>
        <div class="card-header">
          <span>响应结果</span>
          <div>
            <el-tag v-if="responseInfo.status" :type="responseInfo.status === 200 ? 'success' : 'danger'">
              {{ responseInfo.status }}
            </el-tag>
            <el-tag v-if="responseInfo.time" type="info" style="margin-left: 10px">
              {{ responseInfo.time }}ms
            </el-tag>
          </div>
        </div>
      </template>

      <div v-if="loading" class="loading-container">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span style="margin-left: 10px">请求中...</span>
      </div>

      <div v-else-if="form.stream && streamContent">
        <div class="stream-content">{{ streamContent }}</div>
      </div>

      <div v-else-if="responseInfo.data">
        <pre class="code-block">{{ responseInfo.data }}</pre>
      </div>

      <el-empty v-else description="暂无响应数据" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Loading, Monitor, Platform, Cpu, Document,
  VideoPlay, Delete, CopyDocument
} from '@element-plus/icons-vue'

const form = ref({
  clientMode: 'http',
  format: 'openai',
  model: 'claude-3-5-sonnet-20241022',
  customModel: '',
  accountType: '',
  system: '',
  message: 'Hello! Please introduce yourself briefly.',
  stream: false,
  maxTokens: 1024,
  temperature: 0.7,
  topP: 1.0
})

const loading = ref(false)
const streamContent = ref('')
const sdkLanguage = ref('python')

const requestInfo = ref({
  url: '',
  headers: '',
  body: ''
})

const responseInfo = ref({
  status: null,
  time: null,
  data: null
})

// 客户端模式说明
const clientModeDescription = computed(() => {
  const descriptions = {
    http: 'HTTP 直连模式：标准 HTTP 请求，不添加额外的客户端标识头',
    claude_code: 'Claude Code 模式：模拟 Claude Code CLI 的请求头，包含 User-Agent 和 X-Client-Name',
    sdk: 'SDK 模式：模拟官方 SDK 的请求方式，生成对应语言的代码示例',
    curl: 'cURL 模式：生成可直接复制使用的 cURL 命令'
  }
  return descriptions[form.value.clientMode] || ''
})

// 计算实际使用的模型名
const actualModel = computed(() => {
  const model = form.value.customModel || form.value.model
  if (form.value.accountType) {
    return `${form.value.accountType},${model}`
  }
  return model
})

// 获取请求头
const getRequestHeaders = computed(() => {
  const headers = {
    'Content-Type': 'application/json'
  }

  if (form.value.format === 'claude') {
    headers['anthropic-version'] = '2023-06-01'
  }

  // 根据客户端模式添加不同的头
  switch (form.value.clientMode) {
    case 'claude_code':
      headers['User-Agent'] = 'claude-code/1.0.0'
      headers['X-Client-Name'] = 'claude_code'
      break
    case 'sdk':
      if (form.value.format === 'claude') {
        headers['User-Agent'] = 'anthropic-python/0.25.0'
        headers['X-Stainless-Lang'] = 'python'
        headers['X-Stainless-Package-Version'] = '0.25.0'
      } else {
        headers['User-Agent'] = 'OpenAI/v1 PythonBindings/1.0.0'
      }
      break
    case 'curl':
      // curl 默认 User-Agent
      headers['User-Agent'] = 'curl/8.0.0'
      break
    default:
      // HTTP 直连不添加特殊头
      break
  }

  return headers
})

// 生成 cURL 命令
const curlCommand = computed(() => {
  const url = `http://localhost:8080${requestInfo.value.url}`
  const headers = getRequestHeaders.value
  const body = JSON.parse(requestInfo.value.body || '{}')

  let cmd = `curl -X POST '${url}'`

  for (const [key, value] of Object.entries(headers)) {
    cmd += ` \\\n  -H '${key}: ${value}'`
  }

  cmd += ` \\\n  -d '${JSON.stringify(body)}'`

  return cmd
})

// 生成 SDK 代码
const sdkCode = computed(() => {
  const model = actualModel.value
  const message = form.value.message
  const maxTokens = form.value.maxTokens

  if (form.value.format === 'claude') {
    // Claude SDK 代码
    switch (sdkLanguage.value) {
      case 'python':
        return `import anthropic

client = anthropic.Anthropic(
    base_url="http://localhost:8080/api",
    api_key="your-api-key"
)

message = client.messages.create(
    model="${model}",
    max_tokens=${maxTokens},
    messages=[
        {"role": "user", "content": "${message}"}
    ]${form.value.system ? `,\n    system="${form.value.system}"` : ''}
)

print(message.content[0].text)`

      case 'nodejs':
        return `import Anthropic from '@anthropic-ai/sdk';

const client = new Anthropic({
  baseURL: 'http://localhost:8080/api',
  apiKey: 'your-api-key'
});

const message = await client.messages.create({
  model: '${model}',
  max_tokens: ${maxTokens},
  messages: [
    { role: 'user', content: '${message}' }
  ]${form.value.system ? `,\n  system: '${form.value.system}'` : ''}
});

console.log(message.content[0].text);`

      case 'go':
        return `package main

import (
    "context"
    "fmt"
    "github.com/anthropics/anthropic-sdk-go"
)

func main() {
    client := anthropic.NewClient(
        anthropic.WithBaseURL("http://localhost:8080/api"),
        anthropic.WithAPIKey("your-api-key"),
    )

    message, err := client.Messages.Create(context.Background(), anthropic.MessageCreateParams{
        Model:     anthropic.F("${model}"),
        MaxTokens: anthropic.F(int64(${maxTokens})),
        Messages: anthropic.F([]anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock("${message}")),
        }),
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(message.Content[0].Text)
}`
    }
  } else {
    // OpenAI SDK 代码
    switch (sdkLanguage.value) {
      case 'python':
        return `from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-api-key"
)

response = client.chat.completions.create(
    model="${model}",
    max_tokens=${maxTokens},
    messages=[
        ${form.value.system ? `{"role": "system", "content": "${form.value.system}"},\n        ` : ''}{"role": "user", "content": "${message}"}
    ]
)

print(response.choices[0].message.content)`

      case 'nodejs':
        return `import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'your-api-key'
});

const response = await client.chat.completions.create({
  model: '${model}',
  max_tokens: ${maxTokens},
  messages: [
    ${form.value.system ? `{ role: 'system', content: '${form.value.system}' },\n    ` : ''}{ role: 'user', content: '${message}' }
  ]
});

console.log(response.choices[0].message.content);`

      case 'go':
        return `package main

import (
    "context"
    "fmt"
    "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("your-api-key")
    config.BaseURL = "http://localhost:8080/v1"
    client := openai.NewClientWithConfig(config)

    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model:     "${model}",
            MaxTokens: ${maxTokens},
            Messages: []openai.ChatCompletionMessage{
                ${form.value.system ? `{Role: openai.ChatMessageRoleSystem, Content: "${form.value.system}"},\n                ` : ''}{Role: openai.ChatMessageRoleUser, Content: "${message}"},
            },
        },
    )
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}`
    }
  }
  return ''
})

// 监听表单变化，更新请求信息
watch([form, sdkLanguage], () => {
  updateRequestInfo()
}, { deep: true })

function onClientModeChange() {
  updateRequestInfo()
}

function updateRequestInfo() {
  const model = actualModel.value
  const headers = getRequestHeaders.value

  // 格式化请求头显示
  requestInfo.value.headers = Object.entries(headers)
    .map(([k, v]) => `${k}: ${v}`)
    .join('\n')

  if (form.value.format === 'openai') {
    requestInfo.value.url = '/v1/chat/completions'
    const messages = []
    if (form.value.system) {
      messages.push({ role: 'system', content: form.value.system })
    }
    messages.push({ role: 'user', content: form.value.message })

    requestInfo.value.body = JSON.stringify({
      model: model,
      messages: messages,
      max_tokens: form.value.maxTokens,
      temperature: form.value.temperature,
      top_p: form.value.topP,
      stream: form.value.stream
    }, null, 2)
  } else {
    requestInfo.value.url = '/api/v1/messages'
    const bodyObj = {
      model: model,
      messages: [{ role: 'user', content: form.value.message }],
      max_tokens: form.value.maxTokens,
      temperature: form.value.temperature,
      top_p: form.value.topP,
      stream: form.value.stream
    }
    if (form.value.system) {
      bodyObj.system = form.value.system
    }
    requestInfo.value.body = JSON.stringify(bodyObj, null, 2)
  }
}

function getClientModeLabel(mode) {
  const labels = {
    http: 'HTTP',
    claude_code: 'Claude Code',
    sdk: 'SDK',
    curl: 'cURL'
  }
  return labels[mode] || mode
}

function getClientModeTagType(mode) {
  const types = {
    http: 'info',
    claude_code: 'success',
    sdk: 'warning',
    curl: ''
  }
  return types[mode] || 'info'
}

async function sendRequest() {
  loading.value = true
  streamContent.value = ''
  responseInfo.value = { status: null, time: null, data: null }

  const startTime = Date.now()

  try {
    // 使用管理员测试接口
    const token = localStorage.getItem('token')
    const testBody = {
      model: actualModel.value,
      message: form.value.message,
      system: form.value.system,
      max_tokens: form.value.maxTokens,
      account_type: form.value.accountType,
      client_mode: form.value.clientMode
    }

    const response = await fetch('/api/admin/proxy/test', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(testBody)
    })

    responseInfo.value.status = response.status
    responseInfo.value.time = Date.now() - startTime

    const data = await response.json()
    responseInfo.value.data = JSON.stringify(data, null, 2)

  } catch (error) {
    responseInfo.value.status = 'Error'
    responseInfo.value.data = error.message
    responseInfo.value.time = Date.now() - startTime
  } finally {
    loading.value = false
  }
}

async function sendNonStreamRequest(body, headers, startTime) {
  const response = await fetch(requestInfo.value.url, {
    method: 'POST',
    headers: headers,
    body: JSON.stringify(body)
  })

  responseInfo.value.status = response.status
  responseInfo.value.time = Date.now() - startTime

  const data = await response.json()
  responseInfo.value.data = JSON.stringify(data, null, 2)
}

async function sendStreamRequest(body, headers, startTime) {
  const response = await fetch(requestInfo.value.url, {
    method: 'POST',
    headers: headers,
    body: JSON.stringify(body)
  })

  responseInfo.value.status = response.status

  const reader = response.body.getReader()
  const decoder = new TextDecoder()

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    const chunk = decoder.decode(value, { stream: true })
    const lines = chunk.split('\n')

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = line.substring(6)
        if (data === '[DONE]') {
          responseInfo.value.time = Date.now() - startTime
          return
        }

        try {
          const parsed = JSON.parse(data)
          if (form.value.format === 'openai') {
            // OpenAI 格式
            if (parsed.choices && parsed.choices[0]?.delta?.content) {
              streamContent.value += parsed.choices[0].delta.content
            }
          } else {
            // Claude 格式
            if (parsed.delta?.text) {
              streamContent.value += parsed.delta.text
            } else if (parsed.content_block?.text) {
              streamContent.value += parsed.content_block.text
            }
          }
        } catch (e) {
          // 忽略解析错误
        }
      }
    }
  }

  responseInfo.value.time = Date.now() - startTime
}

function clearResponse() {
  streamContent.value = ''
  responseInfo.value = { status: null, time: null, data: null }
}

async function copyCurl() {
  try {
    await navigator.clipboard.writeText(curlCommand.value)
    ElMessage.success('cURL 命令已复制到剪贴板')
  } catch (err) {
    ElMessage.error('复制失败')
  }
}

// 初始化请求信息
updateRequestInfo()
</script>

<style scoped>
.proxy-test {
  padding: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.code-block {
  background-color: #f5f7fa;
  padding: 15px;
  border-radius: 4px;
  overflow-x: auto;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}

.headers-block {
  background-color: #fef0f0;
  font-size: 12px;
}

.curl-block {
  background-color: #f0f9eb;
}

.sdk-block {
  background-color: #ecf5ff;
}

.stream-content {
  background-color: #f5f7fa;
  padding: 15px;
  border-radius: 4px;
  min-height: 100px;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: inherit;
  line-height: 1.6;
}

.loading-container {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: #409eff;
}

:deep(.el-radio-button__inner) {
  display: flex;
  align-items: center;
  gap: 5px;
}
</style>
