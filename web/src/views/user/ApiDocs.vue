<!--
 * 文件作用：API 接入文档页面，帮助用户了解如何使用平台 API
 * 负责功能：
 *   - 展示认证方式说明
 *   - 各平台端点和请求格式
 *   - 代码示例（cURL / Python / Node.js）
 *   - 错误码参考
 *   - 自动填充用户 API Key
 * 重要程度：⭐⭐⭐⭐ 重要（用户接入入口）
 * 依赖组件：Element Plus, api/index.js
-->
<template>
  <div class="api-docs">
    <div class="page-header">
      <h2>API 接入文档</h2>
      <p class="subtitle">了解如何将 AI 能力集成到你的应用中</p>
    </div>

    <!-- 快速开始 -->
    <el-card shadow="never" class="doc-section">
      <template #header>
        <div class="section-title">
          <el-icon><Lightning /></el-icon>
          <span>快速开始</span>
        </div>
      </template>

      <el-steps :active="3" align-center class="quick-steps">
        <el-step title="创建 API Key" description="在「我的 API Key」页面创建" />
        <el-step title="选择平台端点" description="根据目标模型选择对应端点" />
        <el-step title="发送请求" description="携带 API Key 调用接口" />
      </el-steps>

      <el-alert type="info" :closable="false" show-icon class="base-url-alert">
        <template #title>
          <span>基础 URL：<code class="code-inline">{{ baseURL }}</code></span>
        </template>
        <template #default>
          <span>所有接口路径均相对于此 URL。认证 Header：<code class="code-inline">x-api-key: your-key</code> 或 <code class="code-inline">Authorization: Bearer your-key</code></span>
        </template>
      </el-alert>
    </el-card>

    <!-- 支持的 AI 平台 -->
    <el-card shadow="never" class="doc-section">
      <template #header>
        <div class="section-title">
          <el-icon><Grid /></el-icon>
          <span>支持的 AI 平台</span>
        </div>
      </template>
      <p style="color: var(--el-text-color-secondary); margin: 0 0 16px 0;">本平台支持以下 AI 服务商，通过统一的 API Key 即可调用所有平台的模型：</p>
      <div class="platform-grid">
        <div v-for="p in allPlatforms" :key="p.name" class="platform-chip" :class="p.type">
          <span class="platform-name">{{ p.name }}</span>
          <span class="platform-models">{{ p.models }}</span>
        </div>
      </div>
    </el-card>

    <!-- 接口端点 -->
    <el-card shadow="never" class="doc-section">
      <template #header>
        <div class="section-title">
          <el-icon><Connection /></el-icon>
          <span>接口端点</span>
        </div>
      </template>

      <el-alert type="warning" :closable="false" show-icon style="margin-bottom: 16px">
        <template #title>所有 OpenAI 兼容平台（DeepSeek、通义千问、GLM、Kimi 等）均通过 <code class="code-inline">/openai/v1/chat/completions</code> 端点访问，系统根据模型名自动路由到对应平台。</template>
      </el-alert>

      <el-table :data="endpoints" stripe style="width: 100%">
        <el-table-column prop="platform" label="适用平台" width="200">
          <template #default="{ row }">
            <el-tag :type="row.tagType" size="small">{{ row.platform }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="method" label="方法" width="80">
          <template #default>
            <el-tag type="success" size="small" effect="dark">POST</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="endpoint" label="端点">
          <template #default="{ row }">
            <code class="code-inline endpoint-code">{{ row.endpoint }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="format" label="请求格式" width="220" />
      </el-table>
    </el-card>

    <!-- 接口详情标签页 -->
    <el-card shadow="never" class="doc-section">
      <template #header>
        <div class="section-title">
          <el-icon><Document /></el-icon>
          <span>接口详情与示例</span>
        </div>
      </template>

      <el-tabs v-model="activeTab" type="border-card">
        <!-- Claude -->
        <el-tab-pane label="Claude" name="claude">
          <div class="api-detail">
            <div class="endpoint-banner">
              <el-tag type="success" size="small" effect="dark">POST</el-tag>
              <code>/claude/v1/messages</code>
            </div>
            <p>与 <a href="https://docs.anthropic.com/en/api/messages" target="_blank">Claude Messages API</a> 完全兼容。</p>

            <h4>请求体</h4>
            <pre class="code-block"><code>{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 1024,
  "messages": [
    {"role": "user", "content": "你好，请介绍一下自己"}
  ],
  "stream": false
}</code></pre>

            <h4>cURL 示例</h4>
            <pre class="code-block"><code>curl -X POST {{ baseURL }}/claude/v1/messages \
  -H "x-api-key: {{ displayKey }}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'</code></pre>

            <h4>响应示例</h4>
            <pre class="code-block"><code>{
  "id": "msg_xxxx",
  "type": "message",
  "role": "assistant",
  "model": "claude-sonnet-4-20250514",
  "content": [{"type": "text", "text": "你好！我是 Claude..."}],
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 12,
    "output_tokens": 45
  }
}</code></pre>

            <el-alert type="info" :closable="false" show-icon style="margin-top: 12px">
              <template #title>会话粘性</template>
              添加 <code class="code-inline">x-session-id: your-session-id</code> 请求头可保证同一会话始终路由到同一后端账户。
            </el-alert>
          </div>
        </el-tab-pane>

        <!-- OpenAI -->
        <el-tab-pane label="OpenAI" name="openai">
          <div class="api-detail">
            <div class="endpoint-banner">
              <el-tag type="success" size="small" effect="dark">POST</el-tag>
              <code>/openai/v1/chat/completions</code>
            </div>
            <p>与 <a href="https://platform.openai.com/docs/api-reference/chat" target="_blank">OpenAI Chat Completions API</a> 完全兼容。</p>

            <h4>请求体</h4>
            <pre class="code-block"><code>{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"}
  ],
  "stream": false
}</code></pre>

            <h4>cURL 示例</h4>
            <pre class="code-block"><code>curl -X POST {{ baseURL }}/openai/v1/chat/completions \
  -H "Authorization: Bearer {{ displayKey }}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello"}]
  }'</code></pre>

            <h4>响应示例</h4>
            <pre class="code-block"><code>{
  "id": "chatcmpl-xxxx",
  "object": "chat.completion",
  "model": "gpt-4o",
  "choices": [{
    "index": 0,
    "message": {"role": "assistant", "content": "Hello! How can I help you?"},
    "finish_reason": "stop"
  }],
  "usage": {"prompt_tokens": 20, "completion_tokens": 9, "total_tokens": 29}
}</code></pre>
          </div>
        </el-tab-pane>

        <!-- 多平台（OpenAI 兼容） -->
        <el-tab-pane label="多平台" name="multiplatform">
          <div class="api-detail">
            <div class="endpoint-banner">
              <el-tag type="success" size="small" effect="dark">POST</el-tag>
              <code>/openai/v1/chat/completions</code>
              <el-tag type="warning" size="small" style="margin-left: 8px">同一端点</el-tag>
            </div>
            <el-alert type="info" :closable="false" show-icon style="margin-bottom: 12px">
              <template #title>与 OpenAI 共用同一端点</template>
              DeepSeek、通义千问、GLM、Kimi、豆包等平台均兼容 OpenAI 格式，通过同一个端点访问，系统根据模型名自动路由到对应平台。
            </el-alert>

            <h4>模型路由表</h4>
            <p>直接在 model 字段填写对应平台的模型名即可，系统自动识别：</p>
            <el-table :data="modelRoutes" stripe size="small" style="margin: 12px 0">
              <el-table-column prop="prefix" label="模型前缀 / 名称" width="200">
                <template #default="{ row }">
                  <code class="code-inline">{{ row.prefix }}</code>
                </template>
              </el-table-column>
              <el-table-column prop="platform" label="路由到平台" />
              <el-table-column prop="example" label="示例模型" />
            </el-table>
            <p>也可用 <code class="code-inline">平台,模型名</code> 格式显式指定，例如 <code class="code-inline">"model": "deepseek,deepseek-chat"</code></p>

            <h4>DeepSeek 示例</h4>
            <pre class="code-block"><code>curl -X POST {{ baseURL }}/openai/v1/chat/completions \
  -H "Authorization: Bearer {{ displayKey }}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-chat",
    "messages": [{"role": "user", "content": "你好"}]
  }'</code></pre>

            <h4>通义千问示例</h4>
            <pre class="code-block"><code>curl -X POST {{ baseURL }}/openai/v1/chat/completions \
  -H "Authorization: Bearer {{ displayKey }}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen-turbo",
    "messages": [{"role": "user", "content": "你好"}]
  }'</code></pre>
          </div>
        </el-tab-pane>

        <!-- Gemini -->
        <el-tab-pane label="Gemini" name="gemini">
          <div class="api-detail">
            <div class="endpoint-banner">
              <el-tag type="success" size="small" effect="dark">POST</el-tag>
              <code>/gemini/v1/chat</code>
            </div>

            <h4>请求体</h4>
            <pre class="code-block"><code>{
  "model": "gemini-2.5-pro",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": false
}</code></pre>

            <h4>cURL 示例</h4>
            <pre class="code-block"><code>curl -X POST {{ baseURL }}/gemini/v1/chat \
  -H "x-api-key: {{ displayKey }}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-pro",
    "messages": [{"role": "user", "content": "Hello"}]
  }'</code></pre>
          </div>
        </el-tab-pane>

        <!-- Responses API -->
        <el-tab-pane label="Codex CLI" name="codex">
          <div class="api-detail">
            <div class="endpoint-banner">
              <el-tag type="success" size="small" effect="dark">POST</el-tag>
              <code>/v1/responses</code>
            </div>
            <p>兼容 OpenAI Responses API，主要用于 <strong>Codex CLI</strong> 和 <strong>Claude Code</strong> 等编程工具。</p>
            <p>以下端点均支持：<code class="code-inline">/responses</code>、<code class="code-inline">/v1/responses</code>、<code class="code-inline">/openai/v1/responses</code></p>

            <h4>Codex CLI 配置</h4>
            <pre class="code-block"><code># 设置环境变量
export OPENAI_API_KEY="{{ displayKey }}"
export OPENAI_BASE_URL="{{ baseURL }}"

# 使用 Codex CLI
codex "explain this code"</code></pre>

            <h4>Claude Code 配置</h4>
            <pre class="code-block"><code># 设置环境变量
export ANTHROPIC_API_KEY="{{ displayKey }}"
export ANTHROPIC_BASE_URL="{{ baseURL }}"

# Claude Code 会自动使用 /claude/v1/messages 端点</code></pre>
          </div>
        </el-tab-pane>

        <!-- SDK 示例 -->
        <el-tab-pane label="SDK 示例" name="sdk">
          <div class="api-detail">
            <h4>Python - OpenAI SDK</h4>
            <pre class="code-block"><code>from openai import OpenAI

client = OpenAI(
    api_key="{{ displayKey }}",
    base_url="{{ baseURL }}/openai/v1"
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello"}]
)

print(response.choices[0].message.content)</code></pre>

            <h4>Python - Anthropic SDK</h4>
            <pre class="code-block"><code>import anthropic

client = anthropic.Anthropic(
    api_key="{{ displayKey }}",
    base_url="{{ baseURL }}/claude"
)

message = client.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}]
)

print(message.content[0].text)</code></pre>

            <h4>Node.js - OpenAI SDK</h4>
            <pre class="code-block"><code>import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: '{{ displayKey }}',
  baseURL: '{{ baseURL }}/openai/v1',
});

const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Hello' }],
});

console.log(response.choices[0].message.content);</code></pre>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 错误码参考 -->
    <el-card shadow="never" class="doc-section">
      <template #header>
        <div class="section-title">
          <el-icon><Warning /></el-icon>
          <span>错误码参考</span>
        </div>
      </template>

      <el-table :data="errorCodes" stripe>
        <el-table-column prop="code" label="HTTP 状态码" width="140">
          <template #default="{ row }">
            <el-tag :type="row.code >= 500 ? 'danger' : row.code >= 400 ? 'warning' : 'info'" size="small">{{ row.code }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="desc" label="说明" width="250" />
        <el-table-column prop="action" label="建议处理" />
      </el-table>
    </el-card>

    <!-- 请求头参考 -->
    <el-card shadow="never" class="doc-section">
      <template #header>
        <div class="section-title">
          <el-icon><List /></el-icon>
          <span>请求头参考</span>
        </div>
      </template>

      <el-table :data="headers" stripe>
        <el-table-column prop="name" label="Header" width="220">
          <template #default="{ row }">
            <code class="code-inline">{{ row.name }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="required" label="必选" width="80">
          <template #default="{ row }">
            <el-tag :type="row.required ? 'danger' : 'info'" size="small">{{ row.required ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="desc" label="说明" />
      </el-table>
    </el-card>

    <!-- 注意事项 -->
    <el-card shadow="never" class="doc-section">
      <template #header>
        <div class="section-title">
          <el-icon><InfoFilled /></el-icon>
          <span>注意事项</span>
        </div>
      </template>
      <ul class="notes-list">
        <li><strong>模型可用性</strong>：可用的模型取决于管理员配置的后端账户池，并非所有模型名称都保证可用</li>
        <li><strong>Token 用量</strong>：返回的 Token 数量可能经过倍率调整，反映的是计费 Token</li>
        <li><strong>请求超时</strong>：流式请求默认无超时，非流式请求建议设置 60-120 秒超时</li>
        <li><strong>智能重试</strong>：系统内置自动重试（默认 3 次），一个账户失败会自动切换到其他账户</li>
        <li><strong>并发限制</strong>：每个用户和 API Key 可能配置了并发上限，超出返回 429</li>
        <li><strong>速率限制</strong>：配置了 RPM/RPD 后，响应头会包含 <code class="code-inline">X-RateLimit-Limit</code> 和 <code class="code-inline">X-RateLimit-Remaining</code></li>
      </ul>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@/api'
import {
  Lightning, Connection, Document, Warning, List, InfoFilled, Grid
} from '@element-plus/icons-vue'

const activeTab = ref('claude')
const baseURL = ref(window.location.origin)
const firstKey = ref('')

const displayKey = ref('your-api-key')

onMounted(async () => {
  try {
    const res = await api.getApiKeys()
    const keys = res?.data?.items || res?.data || []
    if (Array.isArray(keys) && keys.length > 0) {
      firstKey.value = keys[0].key || ''
      if (firstKey.value) {
        displayKey.value = firstKey.value.substring(0, 8) + '...'
      }
    }
  } catch {
    // ignore
  }
})

const allPlatforms = [
  { name: 'Claude (Anthropic)', models: 'claude-sonnet-4, claude-opus-4 等', type: 'claude' },
  { name: 'OpenAI', models: 'gpt-4o, gpt-4.1, o3 等', type: 'openai' },
  { name: 'Google Gemini', models: 'gemini-2.5-pro, gemini-2.5-flash 等', type: 'gemini' },
  { name: 'DeepSeek', models: 'deepseek-chat, deepseek-reasoner 等', type: 'compat' },
  { name: '通义千问 (Qwen)', models: 'qwen-turbo, qwen-max 等', type: 'compat' },
  { name: '智谱 GLM', models: 'glm-4, glm-4-flash 等', type: 'compat' },
  { name: 'Kimi (月之暗面)', models: 'moonshot-v1-8k 等', type: 'compat' },
  { name: '豆包 (字节)', models: 'doubao-pro, doubao-lite 等', type: 'compat' },
  { name: '零一万物', models: 'yi-large, yi-medium 等', type: 'compat' },
  { name: '阶跃星辰', models: 'step-1-8k, step-2-16k 等', type: 'compat' },
  { name: '讯飞星火', models: 'spark-lite, spark-pro 等', type: 'compat' },
  { name: 'xAI (Grok)', models: 'grok-2, grok-3 等', type: 'compat' },
  { name: 'Mistral', models: 'mistral-large 等', type: 'compat' },
  { name: 'Cohere', models: 'command-r-plus 等', type: 'compat' },
  { name: 'SiliconFlow', models: '硅基流动托管模型', type: 'compat' },
  { name: '自定义 API', models: '任意 OpenAI 兼容接口', type: 'custom' },
]

const endpoints = [
  { platform: 'Claude', endpoint: '/claude/v1/messages', format: 'Claude Messages API', tagType: '' },
  { platform: 'OpenAI + 所有兼容平台', endpoint: '/openai/v1/chat/completions', format: 'OpenAI Chat Completions', tagType: 'success' },
  { platform: 'Gemini', endpoint: '/gemini/v1/chat', format: 'Gemini Chat', tagType: 'warning' },
  { platform: 'Codex CLI / Claude Code', endpoint: '/v1/responses', format: 'OpenAI Responses API', tagType: 'info' },
]

const modelRoutes = [
  { prefix: 'deepseek-*', platform: 'DeepSeek', example: 'deepseek-chat, deepseek-reasoner' },
  { prefix: 'qwen-*', platform: '通义千问 (Qwen)', example: 'qwen-turbo, qwen-plus, qwen-max' },
  { prefix: 'glm-*', platform: '智谱 GLM', example: 'glm-4, glm-4-flash' },
  { prefix: 'moonshot-*', platform: 'Kimi (月之暗面)', example: 'moonshot-v1-8k' },
  { prefix: 'doubao-*', platform: '豆包 (字节)', example: 'doubao-pro-4k' },
  { prefix: 'yi-*', platform: '零一万物', example: 'yi-large, yi-medium' },
  { prefix: 'Baichuan*', platform: '百川', example: 'Baichuan2-Turbo' },
  { prefix: 'minimax-*', platform: 'MiniMax', example: 'minimax-abab6.5' },
  { prefix: 'step-*', platform: '阶跃星辰', example: 'step-1-8k' },
  { prefix: 'spark-*', platform: '讯飞星火', example: 'spark-v3.5' },
  { prefix: 'grok-*', platform: 'xAI Grok', example: 'grok-2' },
  { prefix: 'mistral-*', platform: 'Mistral', example: 'mistral-large-latest' },
  { prefix: 'command-*', platform: 'Cohere', example: 'command-r-plus' },
]

const errorCodes = [
  { code: 401, desc: 'API Key 无效/缺失/过期', action: '检查 API Key 是否正确，是否已过期' },
  { code: 402, desc: '配额/次数耗尽', action: '联系管理员充值、升级套餐或增加请求次数' },
  { code: 403, desc: '权限不足 / IP 被禁止', action: '检查 API Key 权限和 IP 白名单设置' },
  { code: 429, desc: '请求速率超限', action: '降低请求频率，参考响应头 Retry-After' },
  { code: 502, desc: '上游 API 错误', action: '稍后重试，或尝试更换模型' },
  { code: 503, desc: '无可用账户', action: '联系管理员检查后端账户池' },
]

const headers = [
  { name: 'x-api-key', required: true, desc: 'API Key 认证（与 Authorization 二选一）' },
  { name: 'Authorization', required: true, desc: 'Bearer Token 认证，格式: Bearer your-key' },
  { name: 'Content-Type', required: true, desc: '必须为 application/json' },
  { name: 'x-session-id', required: false, desc: '会话绑定，同一 session 路由到同一后端账户（Claude Code 自动发送）' },
]
</script>

<style scoped>
.api-docs {
  padding: 20px;
  max-width: 1200px;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h2 {
  margin: 0 0 8px 0;
  color: var(--pink-text, #3a3045);
  font-size: 24px;
}

.subtitle {
  color: var(--el-text-color-secondary);
  margin: 0;
  font-size: 14px;
}

.doc-section {
  margin-bottom: 20px;
}

.doc-section :deep(.el-card__header) {
  padding: 14px 20px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: var(--pink-text, #3a3045);
}

.section-title .el-icon {
  color: var(--pink-accent, #c97b8b);
}

.quick-steps {
  margin: 16px 0 20px;
}

.base-url-alert {
  margin-top: 8px;
}

.code-inline {
  background: var(--pink-accent-light, #faf2f4);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  color: var(--pink-accent, #c97b8b);
  word-break: break-all;
}

.endpoint-code {
  font-size: 13px;
}

.api-detail {
  padding: 8px 0;
}

.api-detail h4 {
  margin: 20px 0 10px;
  color: var(--pink-text, #3a3045);
  font-size: 15px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--pink-border, #f0dde2);
}

.api-detail p {
  color: var(--el-text-color-regular);
  line-height: 1.7;
  margin: 8px 0;
}

.api-detail a {
  color: var(--pink-accent, #c97b8b);
  text-decoration: none;
}

.api-detail a:hover {
  text-decoration: underline;
}

.endpoint-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: var(--pink-accent-light, #faf2f4);
  border-radius: 8px;
  margin-bottom: 12px;
}

.endpoint-banner code {
  font-size: 15px;
  font-weight: 600;
  color: var(--pink-text, #3a3045);
  font-family: 'Courier New', Courier, monospace;
}

.code-block {
  background: #1e1e2e;
  color: #cdd6f4;
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.6;
  font-family: 'Courier New', Courier, monospace;
  margin: 8px 0;
}

.code-block code {
  color: inherit;
  background: none;
  padding: 0;
}

.notes-list {
  padding-left: 20px;
  color: var(--el-text-color-regular);
  line-height: 2;
}

.notes-list li {
  margin-bottom: 4px;
}

.notes-list strong {
  color: var(--pink-text, #3a3045);
}

.platform-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.platform-chip {
  display: flex;
  flex-direction: column;
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid var(--pink-border, #f0dde2);
  transition: all 0.2s;
}

.platform-chip:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.08);
}

.platform-chip.claude {
  background: linear-gradient(135deg, #fdf2f4 0%, #fce8ec 100%);
  border-color: #e8b4bc;
}

.platform-chip.openai {
  background: linear-gradient(135deg, #f0f7f0 0%, #e5f0e5 100%);
  border-color: #b4d4b4;
}

.platform-chip.gemini {
  background: linear-gradient(135deg, #f5f3f0 0%, #ede8e0 100%);
  border-color: #d4c8b4;
}

.platform-chip.compat {
  background: linear-gradient(135deg, #f0f4fa 0%, #e5edf8 100%);
  border-color: #b4c8e8;
}

.platform-chip.custom {
  background: linear-gradient(135deg, #f5f0fa 0%, #ede5f5 100%);
  border-color: #c8b4e0;
}

.platform-name {
  font-weight: 600;
  font-size: 14px;
  color: var(--pink-text, #3a3045);
  margin-bottom: 4px;
}

.platform-models {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
