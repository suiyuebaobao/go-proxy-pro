<!--
 * 文件作用：系统首页，展示平台介绍、动态效果和完整对接文档
 * 负责功能：
 *   - 动态效果展示（粒子系统、星空、数据流、打字机等）
 *   - 服务端点展示（Claude/OpenAI/Gemini Base URL）
 *   - API 接口文档展示
 *   - 功能特性展示（多平台聚合、智能调度、计费系统、管理后台）
 *   - 快速对接教程（Claude Code、Cursor、Continue、Open WebUI）
 * 重要程度：⭐⭐⭐⭐⭐ 核心（系统门面 + 用户文档）
 * 依赖组件：Element Plus, Vue Router, FontAwesome
-->
<template>
  <div class="home-container" @mousemove="handleMouseMove">
    <!-- 鼠标跟随光晕 -->
    <div class="mouse-glow" :style="mouseGlowStyle"></div>

    <!-- 星空背景 -->
    <div class="stars-background">
      <div v-for="n in 100" :key="n" class="star" :style="getStarStyle(n)"></div>
    </div>

    <!-- 粒子背景 -->
    <div class="particles-background">
      <canvas ref="particlesCanvas"></canvas>
    </div>

    <!-- 网格背景 -->
    <div class="grid-background"></div>

    <!-- 数据流线条 -->
    <div class="data-streams">
      <div v-for="n in 5" :key="n" class="data-stream" :style="getDataStreamStyle(n)"></div>
    </div>

    <!-- 头部导航 -->
    <header class="home-header">
      <div class="header-content">
        <div class="logo">
          <div class="logo-icon">
            <i class="fa-solid fa-atom"></i>
          </div>
          <div class="logo-text">
            <h1>Go-AIProxy</h1>
            <p class="tagline">下一代 AI API 代理网关</p>
          </div>
        </div>
        <nav class="header-nav">
          <a href="#features" class="nav-link">
            <i class="fa-solid fa-microchip"></i> 核心能力
          </a>
          <a href="#api-docs" class="nav-link">
            <i class="fa-solid fa-code"></i> API 文档
          </a>
          <a href="#tutorials" class="nav-link">
            <i class="fa-solid fa-terminal"></i> 接入指南
          </a>
          <router-link to="/login" class="btn-login">
            <i class="fa-solid fa-power-off"></i> 控制台
          </router-link>
        </nav>
      </div>
    </header>

    <!-- Hero 区域 -->
    <section class="hero-section">
      <div class="hero-orb hero-orb-1"></div>
      <div class="hero-orb hero-orb-2"></div>
      <div class="hero-orb hero-orb-3"></div>

      <div class="hero-content">
        <div class="hero-badge" :class="{ 'animate': heroAnimated }">
          <span class="badge-dot"></span>
          <span class="badge-text">企业级 AI 基础设施</span>
        </div>

        <h2 class="hero-title">
          <span class="title-line">{{ typewriterText }}</span>
          <span class="title-line title-gradient">智能代理网关</span>
          <span class="cursor">|</span>
        </h2>

        <p class="hero-subtitle" :class="{ 'animate': heroAnimated }">
          融合 Claude · OpenAI · Gemini 多平台能力
          <br>
          构建企业级 AI 服务中台
        </p>

        <div class="server-display" :class="{ 'animate': heroAnimated }">
          <div class="server-label">
            <i class="fa-solid fa-satellite-dish"></i> 服务端点
          </div>
          <div class="server-endpoints">
            <div class="endpoint-item">
              <span class="endpoint-badge claude">Claude</span>
              <code>{{ baseUrl }}/claude</code>
              <button class="copy-btn-small" @click="copyText(baseUrl + '/claude')">
                <i class="fa-solid fa-copy"></i>
              </button>
            </div>
            <div class="endpoint-item">
              <span class="endpoint-badge openai">OpenAI</span>
              <code>{{ baseUrl }}/openai</code>
              <button class="copy-btn-small" @click="copyText(baseUrl + '/openai')">
                <i class="fa-solid fa-copy"></i>
              </button>
            </div>
            <div class="endpoint-item">
              <span class="endpoint-badge gemini">Gemini</span>
              <code>{{ baseUrl }}/gemini</code>
              <button class="copy-btn-small" @click="copyText(baseUrl + '/gemini')">
                <i class="fa-solid fa-copy"></i>
              </button>
            </div>
          </div>
        </div>

        <div class="hero-actions" :class="{ 'animate': heroAnimated }">
          <router-link to="/login" class="btn btn-primary">
            <i class="fa-solid fa-rocket"></i>
            <span>立即启动</span>
            <i class="fa-solid fa-arrow-right btn-arrow"></i>
          </router-link>
          <a href="#tutorials" class="btn btn-glow">
            <i class="fa-solid fa-book-open"></i>
            <span>快速接入</span>
          </a>
        </div>
      </div>

      <!-- 浮动数据卡片 -->
      <div class="floating-cards">
        <div class="float-card float-card-1" :class="{ 'animate': heroAnimated }">
          <div class="card-icon">
            <i class="fa-solid fa-bolt"></i>
          </div>
          <div class="card-text">
            <div class="card-value"><CountUp :end-val="99.9" :decimals="1" suffix="%"></CountUp></div>
            <div class="card-label">可用性</div>
          </div>
        </div>

        <div class="float-card float-card-2" :class="{ 'animate': heroAnimated }">
          <div class="card-icon">
            <i class="fa-solid fa-clock"></i>
          </div>
          <div class="card-text">
            <div class="card-value">&lt;<CountUp :end-val="50" suffix="ms"></CountUp></div>
            <div class="card-label">响应延迟</div>
          </div>
        </div>

        <div class="float-card float-card-3" :class="{ 'animate': heroAnimated }">
          <div class="card-icon">
            <i class="fa-solid fa-shield-halved"></i>
          </div>
          <div class="card-text">
            <div class="card-value">AES-256</div>
            <div class="card-label">加密级别</div>
          </div>
        </div>

        <div class="float-card float-card-4" :class="{ 'animate': heroAnimated }">
          <div class="card-icon">
            <i class="fa-solid fa-server"></i>
          </div>
          <div class="card-text">
            <div class="card-value"><CountUp :end-val="3" suffix="+"></CountUp></div>
            <div class="card-label">AI 平台</div>
          </div>
        </div>

        <div class="float-card float-card-5" :class="{ 'animate': heroAnimated }">
          <div class="card-icon">
            <i class="fa-solid fa-users"></i>
          </div>
          <div class="card-text">
            <div class="card-value"><CountUp :end-val="10000" suffix="+"></CountUp></div>
            <div class="card-label">服务用户</div>
          </div>
        </div>
      </div>
    </section>

    <!-- 功能特性 -->
    <section id="features" class="features-section">
      <div class="section-container">
        <div class="section-header">
          <div class="section-badge">
            <i class="fa-solid fa-star"></i>
            <span>核心能力</span>
          </div>
          <h2 class="section-title">
            赋能企业级 AI 应用
          </h2>
          <p class="section-subtitle">
            完整的 AI API 代理解决方案，支持多云多模型统一接入
          </p>
        </div>

        <div class="features-grid">
          <div
            v-for="(feature, index) in features"
            :key="index"
            class="feature-card"
            :style="{ '--delay': index * 0.1 + 's' }"
            @mouseenter="handleCardEnter($event, index)"
            @mouseleave="handleCardLeave($event, index)"
          >
            <div class="feature-glow"></div>
            <div class="feature-border"></div>
            <div class="feature-icon">
              <i :class="feature.icon"></i>
            </div>
            <h3 class="feature-title">{{ feature.title }}</h3>
            <p class="feature-desc">{{ feature.desc }}</p>
            <ul class="feature-list">
              <li v-for="(item, i) in feature.items" :key="i">
                <i class="fa-solid fa-check-circle"></i>
                {{ item }}
              </li>
            </ul>
          </div>
        </div>
      </div>
    </section>

    <!-- API 文档 -->
    <section id="api-docs" class="api-section">
      <div class="section-container">
        <div class="section-header">
          <div class="section-badge">
            <i class="fa-solid fa-code"></i>
            <span>API 接口</span>
          </div>
          <h2 class="section-title">
            标准化 API 文档
          </h2>
          <p class="section-subtitle">
            完全兼容官方 API 格式，零成本迁移
          </p>
        </div>

        <div class="api-list">
          <div class="api-card" v-for="(api, index) in apis" :key="index">
            <div class="api-header">
              <div class="api-platform">
                <i :class="api.icon"></i>
                <span>{{ api.name }}</span>
              </div>
              <div class="api-badges">
                <span v-for="badge in api.badges" :key="badge" class="api-badge" :class="badge.class">
                  {{ badge.text }}
                </span>
              </div>
            </div>

            <div class="api-body">
              <div class="api-endpoints">
                <div class="endpoint" v-for="(endpoint, i) in api.endpoints" :key="i">
                  <span class="method" :class="endpoint.method.toLowerCase()">{{ endpoint.method }}</span>
                  <code class="endpoint-path">{{ endpoint.path }}</code>
                </div>
              </div>

              <div class="api-code">
                <div class="code-header">
                  <span>{{ api.language }} 示例</span>
                  <button class="copy-btn-small" @click="copyText(api.code)">
                    <i class="fa-solid fa-copy"></i>
                  </button>
                </div>
                <pre><code>{{ api.code }}</code></pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- 接入教程 -->
    <section id="tutorials" class="tutorials-section">
      <div class="section-container">
        <div class="section-header">
          <div class="section-badge">
            <i class="fa-solid fa-graduation-cap"></i>
            <span>接入指南</span>
          </div>
          <h2 class="section-title">
            快速接入 Claude Code
          </h2>
          <p class="section-subtitle">
            官方命令行工具，三步完成配置
          </p>
        </div>

        <div class="tutorial-container">
          <!-- Claude Code 教程 -->
          <div class="tutorial-main">
            <div class="tutorial-card">
              <div class="tutorial-header">
                <h3><i class="fa-solid fa-laptop-code"></i> Claude Code 对接</h3>
                <span class="tutorial-badge">热门</span>
              </div>
              <div class="tutorial-content">
                <h4>什么是 Claude Code？</h4>
                <p>Claude Code 是 Anthropic 官方的 AI 编程助手，可以通过命令行直接与 Claude 对话。</p>
                <h4>对接步骤：</h4>
                <div class="steps-list">
                  <div class="step-item">
                    <div class="step-number">1</div>
                    <div class="step-content">
                      <h5>安装 Claude Code</h5>
                      <pre><code>npm install -g @anthropic-ai/claude-code</code></pre>
                    </div>
                  </div>
                  <div class="step-item">
                    <div class="step-number">2</div>
                    <div class="step-content">
                      <h5>配置环境变量</h5>
                      <p>在 ~/.bashrc 或 ~/.zshrc 中添加：</p>
                      <pre><code>export ANTHROPIC_BASE_URL="{{ baseUrl }}/claude"
export ANTHROPIC_API_KEY="YOUR_API_KEY"</code></pre>
                    </div>
                  </div>
                  <div class="step-item">
                    <div class="step-number">3</div>
                    <div class="step-content">
                      <h5>使用 Claude Code</h5>
                      <pre><code>claude</code></pre>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Cursor 教程 -->
            <div class="tutorial-card">
              <div class="tutorial-header">
                <h3><i class="fa-solid fa-mouse-pointer"></i> Cursor IDE 对接</h3>
              </div>
              <div class="tutorial-content">
                <h4>配置步骤：</h4>
                <ol>
                  <li>打开 Cursor IDE 设置（Settings）</li>
                  <li>找到 API Configuration 部分</li>
                  <li>设置 Base URL 为：<code>{{ baseUrl }}/claude</code></li>
                  <li>输入你的 API Key</li>
                  <li>保存设置并重启 Cursor</li>
                </ol>
                <div class="code-example">
                  <pre><code>// Cursor 配置示例
{
  "apiKey": "YOUR_API_KEY",
  "baseURL": "{{ baseUrl }}/claude",
  "provider": "anthropic"
}</code></pre>
                </div>
              </div>
            </div>

            <!-- Continue 教程 -->
            <div class="tutorial-card">
              <div class="tutorial-header">
                <h3><i class="fa-solid fa-forward"></i> Continue Dev 对接</h3>
              </div>
              <div class="tutorial-content">
                <h4>VS Code 配置：</h4>
                <p>在 VS Code 的 settings.json 中添加：</p>
                <div class="code-example">
                  <pre><code>{
  "continue.anthropicApiKey": "YOUR_API_KEY",
  "continue.anthropicBaseUrl": "{{ baseUrl }}/claude"
}</code></pre>
                </div>
                <p>或者在 Continue 的配置文件 (~/.continue/config.json) 中：</p>
                <div class="code-example">
                  <pre><code>{
  "models": [{
    "title": "Claude Sonnet 4",
    "provider": "anthropic",
    "model": "claude-sonnet-4-20250514",
    "apiKey": "YOUR_API_KEY",
    "apiBase": "{{ baseUrl }}/claude"
  }]
}</code></pre>
                </div>
              </div>
            </div>

            <!-- Open WebUI 教程 -->
            <div class="tutorial-card">
              <div class="tutorial-header">
                <h3><i class="fa-solid fa-globe"></i> Open WebUI 对接</h3>
              </div>
              <div class="tutorial-content">
                <h4>配置步骤：</h4>
                <ol>
                  <li>登录 Open WebUI 管理后台</li>
                  <li>进入 Settings → Providers</li>
                  <li>添加 OpenAI 兼容提供商</li>
                </ol>
                <div class="code-example">
                  <pre><code>// 连接配置
Base URL: {{ baseUrl }}/openai
API Key: YOUR_API_KEY

// 支持的模型
- gpt-4 (映射到 Claude)
- gpt-4-turbo (映射到 Claude)
- claude-sonnet-4-20250514</code></pre>
                </div>
              </div>
            </div>
          </div>

          <div class="tutorial-aside">
            <div class="info-card">
              <h4>
                <i class="fa-solid fa-lightbulb"></i>
                快速复制
              </h4>
              <div class="config-block">
                <pre><code>export ANTHROPIC_BASE_URL="{{ baseUrl }}/claude"
export ANTHROPIC_API_KEY="YOUR_API_KEY"</code></pre>
                <button class="copy-btn-full" @click="copyConfig">
                  <i class="fa-solid fa-copy"></i> 复制配置
                </button>
              </div>
            </div>

            <div class="info-card">
              <h4>
                <i class="fa-solid fa-circle-info"></i>
                支持的客户端
              </h4>
              <ul class="client-list">
                <li>
                  <i class="fa-solid fa-laptop-code"></i>
                  <span>Claude Code 官方</span>
                </li>
                <li>
                  <i class="fa-solid fa-mouse-pointer"></i>
                  <span>Cursor IDE</span>
                </li>
                <li>
                  <i class="fa-solid fa-forward"></i>
                  <span>Continue Dev</span>
                </li>
                <li>
                  <i class="fa-solid fa-globe"></i>
                  <span>Open WebUI</span>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- CTA 区域 -->
    <section class="cta-section">
      <div class="cta-bg"></div>
      <div class="cta-content">
        <h2>准备好开始了吗？</h2>
        <p>立即接入，构建你的 AI 应用</p>
        <router-link to="/login" class="btn btn-cta">
          <i class="fa-solid fa-rocket"></i>
          <span>立即启动</span>
        </router-link>
      </div>
    </section>

    <!-- 页脚 -->
    <footer class="home-footer">
      <div class="footer-content">
        <div class="footer-main">
          <div class="footer-brand">
            <div class="footer-logo">
              <i class="fa-solid fa-atom"></i>
              <span>Go-AIProxy</span>
            </div>
            <p>企业级 AI API 代理管理平台</p>
          </div>

          <div class="footer-links">
            <div class="link-group">
              <h4>平台</h4>
              <a href="#features">核心能力</a>
              <a href="#api-docs">API 文档</a>
              <a href="#tutorials">接入指南</a>
            </div>
            <div class="link-group">
              <h4>技术栈</h4>
              <span>Go 1.21+</span>
              <span>Vue 3.4+</span>
              <span>MySQL 8.0+</span>
            </div>
            <div class="link-group">
              <h4>联系我们</h4>
              <a href="https://github.com/suiyuebaobao/go-proxy-pro" target="_blank">GitHub</a>
              <a href="/login">控制台</a>
            </div>
          </div>
        </div>

        <div class="footer-qq">
          <div class="qq-qr">
            <img src="/qq-group.jpg" alt="QQ群二维码" />
            <p>扫码加入 QQ 交流群</p>
            <a href="https://qm.qq.com/q/iJ4bHLlMEa" target="_blank" class="qq-link-btn">
              <i class="fa-solid fa-link"></i>
              点击加入群聊【go-proxy-pro】
            </a>
          </div>
        </div>

        <div class="footer-bottom">
          <p>&copy; 2025 Go-AIProxy. All rights reserved.</p>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { ensureFontAwesomeLoaded } from '@/utils/fontawesome'
ensureFontAwesomeLoaded()

import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'

// CountUp 组件
const CountUp = {
  props: {
    endVal: { type: Number, required: true },
    decimals: { type: Number, default: 0 },
    suffix: { type: String, default: '' },
    duration: { type: Number, default: 2000 }
  },
  setup(props) {
    const display = ref('0')
    let startTime = null
    let animationFrame = null

    const animate = (timestamp) => {
      if (!startTime) startTime = timestamp
      const progress = Math.min((timestamp - startTime) / props.duration, 1)
      const easeOutQuart = 1 - Math.pow(1 - progress, 4)
      const current = props.endVal * easeOutQuart
      display.value = current.toFixed(props.decimals) + props.suffix

      if (progress < 1) {
        animationFrame = requestAnimationFrame(animate)
      }
    }

    onMounted(() => {
      animationFrame = requestAnimationFrame(animate)
    })

    onUnmounted(() => {
      if (animationFrame) {
        cancelAnimationFrame(animationFrame)
      }
    })

    return { display }
  },
  template: '<span>{{ display }}</span>'
}

const host = ref(window.location.host)
const protocol = window.location.protocol
const particlesCanvas = ref(null)
const mouseX = ref(0)
const mouseY = ref(0)
const heroAnimated = ref(false)
const typewriterText = ref('')
const fullText = '统一 AI API'

const baseUrl = computed(() => {
  return `${protocol}//${host.value}`
})

const mouseGlowStyle = computed(() => ({
  left: mouseX.value + 'px',
  top: mouseY.value + 'px'
}))

const features = ref([
  {
    icon: 'fa-solid fa-layer-group',
    title: '多平台聚合',
    desc: '统一接入 Claude、OpenAI、Gemini 等主流 AI 平台',
    items: ['Claude 全系列', 'OpenAI GPT 系列', 'Google Gemini', 'Azure OpenAI']
  },
  {
    icon: 'fa-solid fa-network-wired',
    title: '智能调度',
    desc: '多账户池管理与负载均衡，自动故障转移',
    items: ['加权轮询', '健康检查', '自动恢复', '并发控制']
  },
  {
    icon: 'fa-solid fa-gauge-high',
    title: '高性能',
    desc: '异步处理架构，毫秒级响应，支持海量并发',
    items: ['流式响应', '连接复用', '智能缓存', '性能监控']
  },
  {
    icon: 'fa-solid fa-shield-halved',
    title: '安全可靠',
    desc: '企业级安全控制，完善的审计日志',
    items: ['API Key 认证', 'IP 白名单', '请求加密', '操作审计']
  },
  {
    icon: 'fa-solid fa-chart-line',
    title: '数据洞察',
    desc: '实时统计 Token 消耗、费用分析、使用报表',
    items: ['用量统计', '费用计算', '趋势分析', '报表导出']
  },
  {
    icon: 'fa-solid fa-puzzle-piece',
    title: '灵活扩展',
    desc: '平台分离架构，模型映射，易于扩展',
    items: ['自定义路由', '模型映射', '费率配置', '套餐管理']
  }
])

const apis = ref([
  {
    name: 'Claude API',
    icon: 'fa-solid fa-brain',
    badges: [{ text: '推荐', class: 'recommended' }],
    endpoints: [
      { method: 'POST', path: '/claude/v1/messages' }
    ],
    language: 'Python',
    code: `import anthropic

client = anthropic.Anthropic(
    base_url="${baseUrl.value}/claude",
    api_key="YOUR_API_KEY"
)

message = client.messages.create(
    model="claude-sonnet-4-20250514",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello!"}]
)`
  },
  {
    name: 'OpenAI API',
    icon: 'fa-solid fa-circle-dot',
    badges: [{ text: '兼容', class: 'compatible' }],
    endpoints: [
      { method: 'POST', path: '/openai/v1/chat/completions' }
    ],
    language: 'Python',
    code: `from openai import OpenAI

client = OpenAI(
    base_url="${baseUrl.value}/openai",
    api_key="YOUR_API_KEY"
)

response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)`
  },
  {
    name: 'Gemini API',
    icon: 'fa-solid fa-gem',
    badges: [{ text: '最新', class: 'new' }],
    endpoints: [
      { method: 'POST', path: '/gemini/v1/chat' }
    ],
    language: 'cURL',
    code: `curl ${baseUrl.value}/gemini/v1/chat \\
  -H "x-api-key: YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gemini-2.5-flash",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`
  }
])

const handleMouseMove = (e) => {
  mouseX.value = e.clientX
  mouseY.value = e.clientY
}

const handleCardEnter = (e, index) => {
  const card = e.currentTarget
  const rect = card.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top

  card.style.setProperty('--mouse-x', `${x}px`)
  card.style.setProperty('--mouse-y', `${y}px`)
}

const handleCardLeave = (e, index) => {
  // 卡片离开时的处理
}

const getStarStyle = (n) => {
  const x = Math.random() * 100
  const y = Math.random() * 100
  const size = Math.random() * 2 + 1
  const delay = Math.random() * 3
  const duration = Math.random() * 2 + 2

  return {
    left: `${x}%`,
    top: `${y}%`,
    width: `${size}px`,
    height: `${size}px`,
    animationDelay: `${delay}s`,
    animationDuration: `${duration}s`
  }
}

const getDataStreamStyle = (n) => {
  const left = (n * 20) + Math.random() * 10
  const delay = n * 0.5
  const duration = 3 + Math.random() * 2

  return {
    left: `${left}%`,
    animationDelay: `${delay}s`,
    animationDuration: `${duration}s`
  }
}

const copyText = (text) => {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

const copyConfig = () => {
  const config = `export ANTHROPIC_BASE_URL="${baseUrl.value}/claude"
export ANTHROPIC_API_KEY="YOUR_API_KEY"`
  navigator.clipboard.writeText(config).then(() => {
    ElMessage.success('配置已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

const typeWriter = () => {
  let i = 0
  const type = () => {
    if (i < fullText.length) {
      typewriterText.value += fullText.charAt(i)
      i++
      setTimeout(type, 100)
    }
  }
  type()
}

// 粒子动画（性能优化版）
let animationId = null
const initParticles = () => {
  const canvas = particlesCanvas.value
  if (!canvas) return

  const ctx = canvas.getContext('2d')
  let width = canvas.width = window.innerWidth
  let height = canvas.height = window.innerHeight

  const particles = []
  const particleCount = 60  // 减少粒子数量：150 → 60

  class Particle {
    constructor() {
      this.reset()
    }

    reset() {
      this.x = Math.random() * width
      this.y = Math.random() * height
      this.size = Math.random() * 2 + 0.5
      this.speedX = (Math.random() - 0.5) * 0.5  // 降低速度
      this.speedY = (Math.random() - 0.5) * 0.5
      this.opacity = Math.random() * 0.5 + 0.2
    }

    update() {
      this.x += this.speedX
      this.y += this.speedY

      if (this.x < 0 || this.x > width) this.speedX *= -1
      if (this.y < 0 || this.y > height) this.speedY *= -1
    }

    draw() {
      ctx.beginPath()
      ctx.arc(this.x, this.y, this.size, 0, Math.PI * 2)
      ctx.fillStyle = `rgba(99, 102, 241, ${this.opacity})`
      ctx.fill()
    }
  }

  for (let i = 0; i < particleCount; i++) {
    particles.push(new Particle())
  }

  const connectionDistance = 100
  const connectionDistanceSquared = connectionDistance * connectionDistance

  const animate = () => {
    ctx.clearRect(0, 0, width, height)

    particles.forEach(particle => {
      particle.update()
      particle.draw()
    })

    // 绘制连线（优化：使用距离平方避免 sqrt）
    particles.forEach((p1, i) => {
      particles.slice(i + 1).forEach(p2 => {
        const dx = p1.x - p2.x
        const dy = p1.y - p2.y
        const distSquared = dx * dx + dy * dy

        if (distSquared < connectionDistanceSquared) {
          const distance = Math.sqrt(distSquared)
          ctx.beginPath()
          ctx.strokeStyle = `rgba(99, 102, 241, ${0.15 * (1 - distance / connectionDistance)})`
          ctx.lineWidth = 0.5
          ctx.moveTo(p1.x, p1.y)
          ctx.lineTo(p2.x, p2.y)
          ctx.stroke()
        }
      })
    })

    animationId = requestAnimationFrame(animate)
  }

  animate()

  const handleResize = () => {
    width = canvas.width = window.innerWidth
    height = canvas.height = window.innerHeight
  }

  window.addEventListener('resize', handleResize)
}

onMounted(() => {
  // 触发动画
  setTimeout(() => {
    heroAnimated.value = true
  }, 300)

  // 打字机效果
  setTimeout(() => {
    typeWriter()
  }, 800)

  // 初始化粒子
  initParticles()

  // 平滑滚动
  document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
      const href = this.getAttribute('href')
      if (href !== '#') {
        e.preventDefault()
        const target = document.querySelector(href)
        if (target) {
          target.scrollIntoView({ behavior: 'smooth', block: 'start' })
        }
      }
    })
  })
})

onUnmounted(() => {
  if (animationId) {
    cancelAnimationFrame(animationId)
  }
})
</script>

<style scoped>
/* 全局样式 */
.home-container {
  min-height: 100vh;
  background: #0a0e27;
  position: relative;
  overflow-x: hidden;
}

/* 鼠标跟随光晕 */
.mouse-glow {
  position: fixed;
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, rgba(99, 102, 241, 0.15) 0%, transparent 70%);
  border-radius: 50%;
  pointer-events: none;
  transform: translate(-50%, -50%);
  z-index: 1;
  transition: opacity 0.3s;
}

/* 星空背景 */
.stars-background {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
  pointer-events: none;
}

.star {
  position: absolute;
  background: white;
  border-radius: 50%;
  animation: twinkle ease-in-out infinite;
}

@keyframes twinkle {
  0%, 100% { opacity: 0.2; transform: scale(1); }
  50% { opacity: 1; transform: scale(1.2); }
}

/* 粒子背景 */
.particles-background {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
  pointer-events: none;
}

.particles-background canvas {
  width: 100%;
  height: 100%;
}

/* 网格背景 */
.grid-background {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-image:
    linear-gradient(rgba(99, 102, 241, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(99, 102, 241, 0.03) 1px, transparent 1px);
  background-size: 50px 50px;
  z-index: 0;
  pointer-events: none;
}

/* 数据流线条 */
.data-streams {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
  pointer-events: none;
}

.data-stream {
  position: absolute;
  width: 2px;
  height: 100px;
  background: linear-gradient(to bottom, transparent, rgba(99, 102, 241, 0.5), transparent);
  animation: dataFlow linear infinite;
}

@keyframes dataFlow {
  0% { transform: translateY(-100px); }
  100% { transform: translateY(100vh); }
}

/* 头部导航 */
.home-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 1000;
  background: rgba(10, 14, 39, 0.8);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid rgba(99, 102, 241, 0.1);
  animation: slideDown 0.6s ease-out;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-100%);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.header-content {
  max-width: 1400px;
  margin: 0 auto;
  padding: 1rem 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logo {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.logo-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 1.5rem;
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.5);
  animation: pulse 2s ease-in-out infinite, rotate 20s linear infinite;
}

@keyframes pulse {
  0%, 100% { box-shadow: 0 0 20px rgba(99, 102, 241, 0.5); }
  50% { box-shadow: 0 0 40px rgba(99, 102, 241, 0.8); }
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.logo-text h1 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 800;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.logo-text .tagline {
  margin: 0;
  font-size: 0.75rem;
  color: #64748b;
  letter-spacing: 0.5px;
}

.header-nav {
  display: flex;
  align-items: center;
  gap: 2rem;
}

.nav-link {
  color: #94a3b8;
  text-decoration: none;
  font-weight: 500;
  font-size: 0.938rem;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  position: relative;
}

.nav-link::before {
  content: '';
  position: absolute;
  bottom: -4px;
  left: 0;
  width: 0;
  height: 2px;
  background: linear-gradient(90deg, #6366f1, #a855f7);
  transition: width 0.3s;
}

.nav-link:hover {
  color: #e0e7ff;
}

.nav-link:hover::before {
  width: 100%;
}

.btn-login {
  padding: 0.625rem 1.5rem;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  color: white;
  border-radius: 8px;
  text-decoration: none;
  font-weight: 600;
  font-size: 0.938rem;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.4);
}

.btn-login:hover {
  transform: translateY(-2px);
  box-shadow: 0 0 30px rgba(99, 102, 241, 0.6);
}

/* Hero 区域 */
.hero-section {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8rem 2rem 4rem 2rem;
  z-index: 2;
}

.hero-orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.4;
  animation: float 20s ease-in-out infinite;
}

.hero-orb-1 {
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, #6366f1 0%, transparent 70%);
  top: 10%;
  left: 10%;
}

.hero-orb-2 {
  width: 300px;
  height: 300px;
  background: radial-gradient(circle, #a855f7 0%, transparent 70%);
  top: 50%;
  right: 10%;
  animation-delay: -5s;
}

.hero-orb-3 {
  width: 350px;
  height: 350px;
  background: radial-gradient(circle, #ec4899 0%, transparent 70%);
  bottom: 10%;
  left: 30%;
  animation-delay: -10s;
}

@keyframes float {
  0%, 100% { transform: translate(0, 0); }
  25% { transform: translate(50px, -50px); }
  50% { transform: translate(-30px, 30px); }
  75% { transform: translate(-50px, -20px); }
}

.hero-content {
  position: relative;
  max-width: 900px;
  text-align: center;
  z-index: 2;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 1.25rem;
  background: rgba(99, 102, 241, 0.1);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 100px;
  margin-bottom: 2.5rem;
  backdrop-filter: blur(10px);
  opacity: 0;
  transform: translateY(20px);
  transition: all 0.6s ease-out;
}

.hero-badge.animate {
  opacity: 1;
  transform: translateY(0);
}

.badge-dot {
  width: 8px;
  height: 8px;
  background: #22c55e;
  border-radius: 50%;
  animation: blink 2s ease-in-out infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.badge-text {
  color: #c7d2fe;
  font-size: 0.875rem;
  font-weight: 500;
}

.hero-title {
  font-size: 4rem;
  font-weight: 900;
  line-height: 1.1;
  margin-bottom: 2rem;
}

.title-line {
  display: block;
  color: white;
}

.title-gradient {
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 50%, #ec4899 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.cursor {
  display: inline-block;
  color: #6366f1;
  animation: blinkCursor 1s step-end infinite;
}

@keyframes blinkCursor {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0; }
}

.hero-subtitle {
  font-size: 1.25rem;
  color: #94a3b8;
  line-height: 1.8;
  margin-bottom: 3rem;
  opacity: 0;
  transform: translateY(20px);
  transition: all 0.6s ease-out 0.2s;
}

.hero-subtitle.animate {
  opacity: 1;
  transform: translateY(0);
}

.server-display {
  max-width: 600px;
  margin: 0 auto 3rem auto;
  background: rgba(30, 41, 59, 0.5);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 16px;
  padding: 1.5rem;
  backdrop-filter: blur(10px);
  opacity: 0;
  transform: translateY(20px);
  transition: all 0.6s ease-out 0.4s;
}

.server-display.animate {
  opacity: 1;
  transform: translateY(0);
}

.server-label {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  color: #64748b;
  font-size: 0.875rem;
  margin-bottom: 1rem;
}

.server-endpoints {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.endpoint-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  background: rgba(15, 23, 42, 0.8);
  border-radius: 8px;
  border: 1px solid rgba(99, 102, 241, 0.2);
  transition: all 0.3s;
}

.endpoint-item:hover {
  border-color: rgba(99, 102, 241, 0.4);
  background: rgba(15, 23, 42, 0.9);
}

.endpoint-badge {
  padding: 0.25rem 0.625rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.025em;
  flex-shrink: 0;
}

.endpoint-badge.claude {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: white;
}

.endpoint-badge.openai {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
  color: white;
}

.endpoint-badge.gemini {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: white;
}

.endpoint-item code {
  flex: 1;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  color: #22d3ee;
  word-break: break-all;
  text-align: left;
}

.copy-btn-small {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  border: none;
  border-radius: 6px;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s;
  flex-shrink: 0;
}

.copy-btn-small:hover {
  transform: scale(1.1);
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.6);
}

.copy-btn {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  border: none;
  border-radius: 6px;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s;
  flex-shrink: 0;
}

.copy-btn:hover {
  transform: scale(1.1);
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.6);
}

.hero-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
  opacity: 0;
  transform: translateY(20px);
  transition: all 0.6s ease-out 0.6s;
}

.hero-actions.animate {
  opacity: 1;
  transform: translateY(0);
}

.btn {
  padding: 1rem 2rem;
  border-radius: 12px;
  text-decoration: none;
  font-weight: 600;
  font-size: 1rem;
  transition: all 0.3s;
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  border: none;
  cursor: pointer;
}

.btn-primary {
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  color: white;
  box-shadow: 0 0 30px rgba(99, 102, 241, 0.5);
}

.btn-primary:hover {
  transform: translateY(-3px);
  box-shadow: 0 0 50px rgba(99, 102, 241, 0.7);
}

.btn-glow {
  background: transparent;
  color: white;
  border: 2px solid rgba(99, 102, 241, 0.5);
}

.btn-glow:hover {
  background: rgba(99, 102, 241, 0.1);
  border-color: #6366f1;
  box-shadow: 0 0 30px rgba(99, 102, 241, 0.4);
}

.btn-arrow {
  transition: transform 0.3s;
}

.btn-primary:hover .btn-arrow {
  transform: translateX(5px);
}

/* 浮动卡片 */
.floating-cards {
  position: absolute;
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
  pointer-events: none;
  z-index: 1;
}

.float-card {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.5rem;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 12px;
  backdrop-filter: blur(10px);
  opacity: 0;
  transform: translateY(30px);
  transition: all 0.6s ease-out;
}

.float-card.animate {
  opacity: 1;
  transform: translateY(0);
}

.float-card-1 {
  top: 20%;
  right: 5%;
  animation-delay: 0s;
  transition-delay: 0.8s;
  animation: floatCard 10s ease-in-out infinite;
}

.float-card-2 {
  top: 50%;
  left: 5%;
  animation-delay: -3s;
  transition-delay: 1s;
  animation: floatCard 10s ease-in-out infinite;
}

.float-card-3 {
  bottom: 20%;
  right: 8%;
  animation-delay: -6s;
  transition-delay: 1.2s;
  animation: floatCard 10s ease-in-out infinite;
}

.float-card-4 {
  top: 25%;
  left: 5%;
  animation-delay: -2s;
  transition-delay: 1.6s;
  animation: floatCard 12s ease-in-out infinite;
}

.float-card-5 {
  bottom: 15%;
  left: 12%;
  animation-delay: -4s;
  transition-delay: 2s;
  animation: floatCard 14s ease-in-out infinite;
}

@keyframes floatCard {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-20px); }
}

.card-icon {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 1rem;
}

.card-value {
  font-size: 1.25rem;
  font-weight: 700;
  color: #22d3ee;
}

.card-label {
  font-size: 0.75rem;
  color: #64748b;
}

/* 通用区块 */
.section-container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 6rem 2rem;
  position: relative;
  z-index: 2;
}

.section-header {
  text-align: center;
  margin-bottom: 4rem;
}

.section-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: rgba(99, 102, 241, 0.1);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 100px;
  color: #a5b4fc;
  font-size: 0.875rem;
  font-weight: 500;
  margin-bottom: 1.5rem;
}

.section-title {
  font-size: 3rem;
  font-weight: 800;
  color: white;
  margin-bottom: 1rem;
}

.section-subtitle {
  font-size: 1.125rem;
  color: #64748b;
  max-width: 600px;
  margin: 0 auto;
}

/* 功能特性 */
.features-section {
  position: relative;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: 2rem;
}

.feature-card {
  position: relative;
  padding: 2rem;
  background: rgba(30, 41, 59, 0.5);
  border-radius: 20px;
  backdrop-filter: blur(10px);
  border: 1px solid rgba(99, 102, 241, 0.2);
  transition: all 0.4s;
  overflow: hidden;
  animation: fadeInUp 0.6s ease-out backwards;
  animation-delay: var(--delay);
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.feature-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: radial-gradient(
    600px circle at var(--mouse-x) var(--mouse-y),
    rgba(99, 102, 241, 0.15),
    transparent 40%
  );
  opacity: 0;
  transition: opacity 0.4s;
  pointer-events: none;
}

.feature-card:hover::before {
  opacity: 1;
}

.feature-card:hover {
  transform: translateY(-10px);
  border-color: rgba(99, 102, 241, 0.5);
  box-shadow: 0 20px 60px rgba(99, 102, 241, 0.2);
}

.feature-glow {
  position: absolute;
  width: 200px;
  height: 200px;
  background: radial-gradient(circle, rgba(99, 102, 241, 0.3) 0%, transparent 70%);
  border-radius: 50%;
  top: -100px;
  right: -100px;
  filter: blur(40px);
  opacity: 0;
  transition: opacity 0.4s;
}

.feature-card:hover .feature-glow {
  opacity: 1;
}

.feature-border {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  border-radius: 20px;
  padding: 1px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.5), rgba(168, 85, 247, 0.5), rgba(236, 72, 153, 0.5));
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  opacity: 0;
  transition: opacity 0.4s;
}

.feature-card:hover .feature-border {
  opacity: 1;
}

.feature-icon {
  width: 60px;
  height: 60px;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 1.5rem;
  margin-bottom: 1.5rem;
  position: relative;
  z-index: 1;
  transition: transform 0.4s;
}

.feature-card:hover .feature-icon {
  transform: scale(1.1) rotate(5deg);
}

.feature-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: white;
  margin-bottom: 0.75rem;
  position: relative;
  z-index: 1;
}

.feature-desc {
  color: #94a3b8;
  line-height: 1.7;
  margin-bottom: 1.5rem;
  position: relative;
  z-index: 1;
}

.feature-list {
  list-style: none;
  padding: 0;
  margin: 0;
  position: relative;
  z-index: 1;
}

.feature-list li {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0;
  color: #94a3b8;
  font-size: 0.938rem;
}

.feature-list li i {
  color: #22d3ee;
  font-size: 0.875rem;
}

/* API 文档 */
.api-section {
  position: relative;
}

.api-list {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.api-card {
  background: rgba(30, 41, 59, 0.5);
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: 20px;
  overflow: hidden;
  transition: all 0.3s;
}

.api-card:hover {
  border-color: rgba(99, 102, 241, 0.4);
  box-shadow: 0 10px 40px rgba(99, 102, 241, 0.2);
}

.api-header {
  padding: 1.5rem 2rem;
  background: rgba(15, 23, 42, 0.8);
  border-bottom: 1px solid rgba(99, 102, 241, 0.1);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.api-platform {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  color: white;
  font-size: 1.25rem;
  font-weight: 700;
}

.api-platform i {
  color: #a855f7;
}

.api-badges {
  display: flex;
  gap: 0.5rem;
}

.api-badge {
  padding: 0.375rem 0.75rem;
  border-radius: 100px;
  font-size: 0.75rem;
  font-weight: 600;
}

.api-badge.recommended {
  background: rgba(99, 102, 241, 0.2);
  color: #a5b4fc;
  border: 1px solid rgba(99, 102, 241, 0.3);
}

.api-badge.compatible {
  background: rgba(34, 211, 238, 0.2);
  color: #67e8f9;
  border: 1px solid rgba(34, 211, 238, 0.3);
}

.api-badge.new {
  background: rgba(168, 85, 247, 0.2);
  color: #c4b5fd;
  border: 1px solid rgba(168, 85, 247, 0.3);
}

.api-body {
  padding: 2rem;
}

.api-endpoints {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 2rem;
}

.endpoint {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: rgba(15, 23, 42, 0.6);
  border-radius: 8px;
  border: 1px solid rgba(99, 102, 241, 0.1);
}

.method {
  padding: 0.375rem 0.75rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
}

.method.post {
  background: rgba(34, 197, 94, 0.2);
  color: #86efac;
}

.method.get {
  background: rgba(59, 130, 246, 0.2);
  color: #93c5fd;
}

.endpoint-path {
  color: #22d3ee;
  font-family: 'Courier New', monospace;
  font-size: 0.938rem;
}

.api-code {
  background: rgba(15, 23, 42, 0.8);
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid rgba(99, 102, 241, 0.1);
}

.code-header {
  padding: 0.75rem 1rem;
  background: rgba(30, 41, 59, 0.6);
  border-bottom: 1px solid rgba(99, 102, 241, 0.1);
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #94a3b8;
  font-size: 0.875rem;
}

.copy-btn-small {
  width: 28px;
  height: 28px;
  background: rgba(99, 102, 241, 0.2);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 4px;
  color: #a5b4fc;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.copy-btn-small:hover {
  background: rgba(99, 102, 241, 0.4);
}

.api-code pre {
  padding: 1.5rem;
  margin: 0;
  overflow-x: auto;
  font-size: 0.875rem;
  line-height: 1.6;
}

.api-code code {
  font-family: 'Courier New', monospace;
  color: #e2e8f0;
}

/* 教程区块 */
.tutorials-section {
  position: relative;
}

.tutorial-container {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 3rem;
}

.tutorial-main {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.tutorial-card {
  background: rgba(30, 41, 59, 0.5);
  border-radius: 20px;
  border: 1px solid rgba(99, 102, 241, 0.2);
  overflow: hidden;
  transition: all 0.3s;
}

.tutorial-card:hover {
  border-color: rgba(99, 102, 241, 0.4);
  box-shadow: 0 10px 40px rgba(99, 102, 241, 0.2);
}

.tutorial-header {
  padding: 1.5rem 2rem;
  background: rgba(15, 23, 42, 0.8);
  border-bottom: 1px solid rgba(99, 102, 241, 0.1);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tutorial-header h3 {
  margin: 0;
  color: white;
  font-size: 1.25rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.tutorial-header h3 i {
  color: #a855f7;
}

.tutorial-badge {
  padding: 0.375rem 0.75rem;
  background: #ef4444;
  border-radius: 100px;
  font-size: 0.813rem;
  font-weight: 600;
  color: white;
}

.tutorial-content {
  padding: 2rem;
}

.tutorial-content h4 {
  margin-top: 0;
  margin-bottom: 1rem;
  color: white;
  font-size: 1.125rem;
}

.tutorial-content p {
  color: #94a3b8;
  line-height: 1.7;
  margin-bottom: 1rem;
}

.tutorial-content ol {
  margin: 1rem 0;
  padding-left: 1.5rem;
}

.tutorial-content li {
  color: #94a3b8;
  line-height: 1.7;
  margin-bottom: 0.5rem;
}

.steps-list {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  margin: 2rem 0;
}

.step-item {
  display: flex;
  gap: 1.5rem;
  align-items: flex-start;
}

.step-item .step-number {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 700;
  flex-shrink: 0;
}

.step-item .step-content {
  flex: 1;
}

.step-item .step-content h5 {
  margin: 0 0 0.75rem 0;
  color: white;
  font-size: 1rem;
}

.step-item .step-content pre {
  background: rgba(15, 23, 42, 0.8);
  color: #22d3ee;
  padding: 1rem;
  border-radius: 8px;
  overflow-x: auto;
  font-size: 0.875rem;
  line-height: 1.6;
}

.code-example {
  margin-top: 1rem;
}

.code-example pre {
  background: rgba(15, 23, 42, 0.8);
  color: #e2e8f0;
  padding: 1rem;
  border-radius: 8px;
  overflow-x: auto;
  font-size: 0.875rem;
  line-height: 1.6;
}

.code-example code {
  font-family: 'Courier New', monospace;
}

.tutorial-aside {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.info-card {
  padding: 1.5rem;
  background: rgba(30, 41, 59, 0.5);
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: 16px;
  backdrop-filter: blur(10px);
}

.info-card h4 {
  margin: 0 0 1rem 0;
  color: white;
  font-size: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.info-card h4 i {
  color: #a855f7;
}

.config-block {
  position: relative;
}

.config-block pre {
  padding: 1rem;
  margin: 0 0 1rem 0;
  background: rgba(15, 23, 42, 0.8);
  border-radius: 8px;
  overflow-x: auto;
}

.config-block code {
  font-family: 'Courier New', monospace;
  font-size: 0.813rem;
  color: #22d3ee;
}

.copy-btn-full {
  width: 100%;
  padding: 0.75rem;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  border: none;
  border-radius: 8px;
  color: white;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.copy-btn-full:hover {
  transform: translateY(-2px);
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.5);
}

.client-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.client-list li {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 0;
  color: #94a3b8;
  border-bottom: 1px solid rgba(99, 102, 241, 0.1);
}

.client-list li:last-child {
  border-bottom: none;
}

.client-list li i {
  color: #a855f7;
}

/* CTA 区域 */
.cta-section {
  position: relative;
  padding: 8rem 2rem;
  text-align: center;
  z-index: 2;
}

.cta-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.1) 0%, rgba(168, 85, 247, 0.1) 50%, rgba(236, 72, 153, 0.1) 100%);
}

.cta-content {
  position: relative;
  max-width: 600px;
  margin: 0 auto;
}

.cta-content h2 {
  font-size: 3rem;
  font-weight: 800;
  color: white;
  margin-bottom: 1rem;
}

.cta-content p {
  font-size: 1.25rem;
  color: #94a3b8;
  margin-bottom: 2.5rem;
}

.btn-cta {
  padding: 1.25rem 3rem;
  font-size: 1.125rem;
  background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
  color: white;
  box-shadow: 0 0 40px rgba(99, 102, 241, 0.6);
}

.btn-cta:hover {
  transform: translateY(-3px);
  box-shadow: 0 0 60px rgba(99, 102, 241, 0.8);
}

/* 页脚 */
.home-footer {
  background: rgba(15, 23, 42, 0.8);
  border-top: 1px solid rgba(99, 102, 241, 0.1);
  padding: 4rem 2rem 2rem 2rem;
  position: relative;
  z-index: 2;
}

.footer-content {
  max-width: 1400px;
  margin: 0 auto;
}

.footer-main {
  display: grid;
  grid-template-columns: 2fr 3fr;
  gap: 4rem;
  margin-bottom: 3rem;
}

.footer-brand {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.footer-logo {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 1.5rem;
  font-weight: 800;
  color: white;
}

.footer-logo i {
  color: #a855f7;
}

.footer-brand p {
  color: #64748b;
  font-size: 0.938rem;
}

.footer-links {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 2rem;
}

.link-group h4 {
  margin: 0 0 1rem 0;
  color: white;
  font-size: 1rem;
}

.link-group a,
.link-group span {
  display: block;
  color: #64748b;
  text-decoration: none;
  margin-bottom: 0.5rem;
  font-size: 0.938rem;
  transition: color 0.2s;
}

.link-group a:hover {
  color: #a855f7;
}

.footer-bottom {
  padding-top: 2rem;
  border-top: 1px solid rgba(99, 102, 241, 0.1);
  text-align: center;
}

.footer-bottom p {
  margin: 0;
  color: #64748b;
  font-size: 0.875rem;
}

.footer-qq {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 2rem 0;
  border-top: 1px solid rgba(99, 102, 241, 0.1);
}

.qq-qr {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 1.5rem;
  background: rgba(99, 102, 241, 0.05);
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: 16px;
  transition: all 0.3s ease;
}

.qq-qr:hover {
  background: rgba(99, 102, 241, 0.1);
  border-color: rgba(99, 102, 241, 0.3);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(99, 102, 241, 0.15);
}

.qq-qr img {
  width: 180px;
  height: 180px;
  border-radius: 12px;
  object-fit: cover;
}

.qq-qr p {
  margin: 0;
  color: #a855f7;
  font-size: 0.938rem;
  font-weight: 500;
}

.qq-link-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.75rem;
  padding: 0.5rem 1rem;
  background: linear-gradient(135deg, #a855f7 0%, #6366f1 100%);
  color: white;
  text-decoration: none;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.3s ease;
}

.qq-link-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(168, 85, 247, 0.4);
}

.qq-link-btn i {
  font-size: 0.75rem;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .tutorial-container {
    grid-template-columns: 1fr;
  }

  .tutorial-aside {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
  }
}

@media (max-width: 768px) {
  .hero-title {
    font-size: 2.5rem;
  }

  .hero-subtitle {
    font-size: 1rem;
  }

  .section-title {
    font-size: 2rem;
  }

  .features-grid {
    grid-template-columns: 1fr;
  }

  .header-nav {
    gap: 1rem;
  }

  .nav-link {
    display: none;
  }

  .float-card {
    display: none;
  }

  .footer-main {
    grid-template-columns: 1fr;
  }

  .footer-links {
    grid-template-columns: 1fr;
  }

  .tutorial-aside {
    grid-template-columns: 1fr;
  }
}
</style>
