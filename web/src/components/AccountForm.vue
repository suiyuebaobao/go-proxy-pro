<!--
 * 文件作用：账户表单组件，创建和编辑账户
 * 负责功能：
 *   - 多平台选择和配置
 *   - OAuth/SessionKey/API Key授权方式
 *   - 基本信息和代理配置
 *   - 模型限制和映射配置
 * 重要程度：⭐⭐⭐⭐ 重要（账户管理核心）
 * 依赖模块：element-plus, OAuthFlow组件, api
-->
<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑账户' : '添加账户'"
    width="800px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <!-- 步骤指示器 -->
    <div v-if="!isEdit && needsOAuth" class="steps-indicator">
      <div class="step" :class="{ active: step >= 1, done: step > 1 }">
        <div class="step-num">1</div>
        <span>基本信息</span>
      </div>
      <div class="step-line"></div>
      <div class="step" :class="{ active: step >= 2 }">
        <div class="step-num">2</div>
        <span>授权认证</span>
      </div>
    </div>

    <!-- 步骤1: 基本信息 -->
    <div v-if="step === 1 && !isEdit" class="form-step">
      <!-- 平台选择 -->
      <div class="form-section">
        <h4 class="section-title">选择平台</h4>
        <div class="platform-groups">
          <div
            v-for="group in platformGroups"
            :key="group.key"
            class="platform-group-card"
            :class="{ selected: platformGroup === group.key }"
            @click="selectPlatformGroup(group.key)"
          >
            <div class="card-icon" :style="{ background: group.gradient }">
              <i :class="group.icon"></i>
            </div>
            <div class="card-info">
              <h5>{{ group.name }}</h5>
              <p>{{ group.desc }}</p>
            </div>
            <div v-if="platformGroup === group.key" class="check-mark">
              <i class="fa-solid fa-check"></i>
            </div>
          </div>
        </div>

        <!-- 子平台选择（仅多子类型平台显示） -->
        <div v-if="currentSubplatforms.length > 0" class="subplatform-section">
          <p class="subplatform-label">选择具体接入方式：</p>
          <div class="subplatform-grid">
            <label
              v-for="sub in currentSubplatforms"
              :key="sub.value"
              class="subplatform-card"
              :class="{ selected: form.type === sub.value }"
            >
              <input v-model="form.type" type="radio" :value="sub.value" class="sr-only" />
              <div class="sub-icon" :style="{ background: sub.color }">
                <i :class="sub.icon"></i>
              </div>
              <div class="sub-info">
                <span class="sub-name">{{ sub.label }}</span>
                <span class="sub-desc">{{ sub.desc }}</span>
              </div>
              <div v-if="form.type === sub.value" class="check-mark small">
                <i class="fa-solid fa-check"></i>
              </div>
            </label>
          </div>
        </div>
      </div>

      <!-- 添加方式选择 -->
      <div v-if="showAddTypeSelection" class="form-section">
        <h4 class="section-title">添加方式</h4>
        <div class="add-type-cards">
          <!-- Claude 平台 -->
          <template v-if="form.type === 'claude-official'">
            <div
              class="add-type-card"
              :class="{ selected: form.addType === 'cookie' }"
              @click="form.addType = 'cookie'"
            >
              <div class="type-icon" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%)">
                <i class="fa-solid fa-cookie"></i>
              </div>
              <div class="type-content">
                <h5>SessionKey 授权</h5>
                <p>使用 Claude.ai 的 sessionKey 自动完成授权</p>
              </div>
              <div v-if="form.addType === 'cookie'" class="type-check">
                <i class="fa-solid fa-check"></i>
              </div>
            </div>
            <div
              class="add-type-card"
              :class="{ selected: form.addType === 'oauth' }"
              @click="form.addType = 'oauth'"
            >
              <div class="type-icon" style="background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%)">
                <i class="fa-solid fa-key"></i>
              </div>
              <div class="type-content">
                <h5>OAuth 手动授权</h5>
                <p>生成授权链接，手动完成授权流程</p>
              </div>
              <div v-if="form.addType === 'oauth'" class="type-check">
                <i class="fa-solid fa-check"></i>
              </div>
            </div>
          </template>

          <!-- OpenAI 平台 -->
          <!-- OpenAI 只用 API Key，不需要选择 -->

          <!-- ChatGPT 官方平台 -->
          <template v-else-if="form.type === 'openai-responses'">
            <div
              class="add-type-card"
              :class="{ selected: form.addType === 'oauth' }"
              @click="form.addType = 'oauth'"
            >
              <div class="type-icon" style="background: linear-gradient(135deg, #764ba2 0%, #667eea 100%)">
                <i class="fa-solid fa-user-shield"></i>
              </div>
              <div class="type-content">
                <h5>OAuth 授权</h5>
                <p>通过 OAuth 授权访问 ChatGPT 账户</p>
              </div>
              <div v-if="form.addType === 'oauth'" class="type-check">
                <i class="fa-solid fa-check"></i>
              </div>
            </div>
            <div
              class="add-type-card"
              :class="{ selected: form.addType === 'cookie' }"
              @click="form.addType = 'cookie'"
            >
              <div class="type-icon" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%)">
                <i class="fa-solid fa-cookie"></i>
              </div>
              <div class="type-content">
                <h5>SessionKey 授权</h5>
                <p>使用 ChatGPT 的 SessionKey 直接认证</p>
              </div>
              <div v-if="form.addType === 'cookie'" class="type-check">
                <i class="fa-solid fa-check"></i>
              </div>
            </div>
          </template>

          <!-- Gemini 平台 -->
          <template v-else-if="form.type === 'gemini'">
            <div
              class="add-type-card"
              :class="{ selected: form.addType === 'apikey' }"
              @click="form.addType = 'apikey'"
            >
              <div class="type-icon" style="background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)">
                <i class="fa-solid fa-key"></i>
              </div>
              <div class="type-content">
                <h5>API Key</h5>
                <p>使用 Gemini API Key 直接连接</p>
              </div>
              <div v-if="form.addType === 'apikey'" class="type-check">
                <i class="fa-solid fa-check"></i>
              </div>
            </div>
            <div
              class="add-type-card"
              :class="{ selected: form.addType === 'oauth' }"
              @click="form.addType = 'oauth'"
            >
              <div class="type-icon" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%)">
                <i class="fa-solid fa-user-shield"></i>
              </div>
              <div class="type-content">
                <h5>OAuth 授权</h5>
                <p>通过 Google OAuth 授权访问</p>
              </div>
              <div v-if="form.addType === 'oauth'" class="type-check">
                <i class="fa-solid fa-check"></i>
              </div>
            </div>
          </template>
        </div>
      </div>

      <!-- 基本信息 -->
      <div class="form-section">
        <h4 class="section-title">基本信息</h4>
        <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
          <el-form-item label="账户名称" prop="name">
            <el-input v-model="form.name" placeholder="为账户设置一个易识别的名称" />
          </el-form-item>

          <el-form-item label="描述（可选）">
            <el-input v-model="form.description" type="textarea" :rows="2" placeholder="账户用途说明..." />
          </el-form-item>

          <el-row :gutter="16">
            <el-col :span="6">
              <el-form-item label="账户类型">
                <el-select v-model="form.accountType" style="width: 100%">
                  <el-option label="共享账户" value="shared" />
                  <el-option label="专属账户" value="dedicated" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="优先级">
                <el-input-number v-model="form.priority" :min="1" :max="100" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="最大并发">
                <el-input-number v-model="form.max_concurrency" :min="1" :max="100" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="每日预算($)">
                <el-input-number v-model="form.daily_budget" :min="0" :max="10000" :precision="2" :step="1" style="width: 100%" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="20">
            <el-col :span="6">
              <el-form-item label="启用">
                <el-switch v-model="form.enabled" />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </div>

      <!-- 平台特定配置 -->
      <div v-if="showPlatformConfig" class="form-section">
        <h4 class="section-title">{{ getPlatformConfigTitle }}</h4>

        <!-- Claude Console 配置 -->
        <template v-if="form.type === 'claude-console'">
          <el-form :model="form" label-position="top">
            <el-form-item label="API URL" required>
              <el-input v-model="form.api_url" placeholder="例如：https://api.anthropic.com" />
            </el-form-item>
            <el-form-item label="API Key" required>
              <el-input v-model="form.api_key" type="password" show-password placeholder="sk-ant-api03-..." />
            </el-form-item>
          </el-form>
        </template>

        <!-- Bedrock 配置 -->
        <template v-if="form.type === 'bedrock'">
          <el-form :model="form" label-position="top">
            <el-form-item label="AWS Access Key ID" required>
              <el-input v-model="form.aws_access_key_id" placeholder="请输入 AWS Access Key ID" />
            </el-form-item>
            <el-form-item label="AWS Secret Access Key" required>
              <el-input v-model="form.aws_secret_access_key" type="password" show-password placeholder="请输入 AWS Secret Access Key" />
            </el-form-item>
            <el-form-item label="AWS Region" required>
              <el-select v-model="form.aws_region" placeholder="选择 AWS 区域" style="width: 100%">
                <el-option label="us-east-1 (美国东部)" value="us-east-1" />
                <el-option label="us-west-2 (美国西部)" value="us-west-2" />
                <el-option label="eu-west-1 (欧洲爱尔兰)" value="eu-west-1" />
                <el-option label="ap-northeast-1 (东京)" value="ap-northeast-1" />
                <el-option label="ap-southeast-1 (新加坡)" value="ap-southeast-1" />
              </el-select>
            </el-form-item>
          </el-form>
        </template>

        <!-- OpenAI 配置 -->
        <template v-if="form.type === 'openai' && form.addType === 'apikey'">
          <el-form :model="form" label-position="top">
            <el-form-item required>
              <template #label>
                <span class="label-with-icon">
                  <i class="fa-solid fa-key text-green-500"></i>
                  API Key
                </span>
              </template>
              <el-input
                v-model="form.api_key"
                type="password"
                show-password
                placeholder="sk-..."
              />
              <div class="input-tip">
                <i class="fa-solid fa-info-circle"></i>
                在 <a href="https://platform.openai.com/api-keys" target="_blank">OpenAI Platform</a> 获取 API Key
              </div>
            </el-form-item>
            <el-form-item label="组织 ID（可选）">
              <el-input v-model="form.organization_id" placeholder="org-..." />
            </el-form-item>
            <el-form-item label="API Base URL（可选）">
              <el-input v-model="form.api_url" placeholder="默认: https://api.openai.com/v1" />
              <div class="input-tip">
                <i class="fa-solid fa-info-circle"></i>
                如果使用代理或自定义端点，请填写完整 URL
              </div>
            </el-form-item>
          </el-form>
        </template>

        <!-- ChatGPT 官方 SessionKey 配置 -->
        <template v-if="form.type === 'openai-responses' && form.addType === 'cookie'">
          <el-form :model="form" label-position="top">
            <el-form-item required>
              <template #label>
                <span class="label-with-icon">
                  <i class="fa-solid fa-cookie text-orange-500"></i>
                  SessionKey (Access Token)
                </span>
              </template>
              <el-input
                v-model="form.session_key"
                type="password"
                show-password
                placeholder="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
              />
            </el-form-item>
          </el-form>
          <el-alert type="info" :closable="false" class="session-key-help">
            <template #title>
              <i class="fa-solid fa-lightbulb"></i> 如何获取 SessionKey
            </template>
            <ol class="help-steps">
              <li>在浏览器中登录 <a href="https://chatgpt.com" target="_blank">chatgpt.com</a></li>
              <li>按 <kbd>F12</kbd> 打开开发者工具，切换到 <strong>Network</strong> 标签</li>
              <li>刷新页面，找到任意 API 请求（如 <code>conversations</code>）</li>
              <li>在请求头中找到 <strong>Authorization: Bearer xxx</strong></li>
              <li>复制 <strong>Bearer</strong> 后面的完整 token</li>
            </ol>
            <p class="help-note">
              <i class="fa-solid fa-info-circle"></i>
              SessionKey 通常以 <code>eyJhbGciOiJSUzI1NiI</code> 开头，有效期约 7 天
            </p>
          </el-alert>
        </template>

        <!-- Azure OpenAI 配置 -->
        <template v-if="form.type === 'azure-openai'">
          <el-form :model="form" label-position="top">
            <el-form-item label="Azure Endpoint" required>
              <el-input v-model="form.azure_endpoint" placeholder="https://your-resource.openai.azure.com" />
            </el-form-item>
            <el-form-item label="API Key" required>
              <el-input v-model="form.api_key" type="password" show-password placeholder="Azure API Key" />
            </el-form-item>
            <el-form-item label="部署名称" required>
              <el-input v-model="form.azure_deployment_name" placeholder="gpt-4" />
            </el-form-item>
            <el-form-item label="API 版本">
              <el-input v-model="form.azure_api_version" placeholder="2024-02-01" />
            </el-form-item>
          </el-form>
        </template>

        <!-- Gemini 配置 -->
        <template v-if="(form.type === 'gemini' || form.type === 'gemini-api') && form.addType === 'apikey'">
          <el-form :model="form" label-position="top">
            <el-form-item required>
              <template #label>
                <span class="label-with-icon">
                  <i class="fa-solid fa-key text-blue-500"></i>
                  API Key
                </span>
              </template>
              <el-input
                v-model="form.api_key"
                type="password"
                show-password
                placeholder="Gemini API Key"
              />
              <div class="input-tip">
                <i class="fa-solid fa-info-circle"></i>
                在 <a href="https://aistudio.google.com/apikey" target="_blank">Google AI Studio</a> 获取 API Key
              </div>
            </el-form-item>
            <el-form-item label="API Base URL（可选）">
              <el-input v-model="form.api_url" placeholder="默认: https://generativelanguage.googleapis.com/v1beta" />
            </el-form-item>
          </el-form>
        </template>

        <!-- OpenAI 兼容平台通用配置（DeepSeek、通义千问、智谱、Kimi 等） -->
        <template v-if="openaiCompatibleTypes.includes(form.type)">
          <el-form :model="form" label-position="top">
            <el-form-item required>
              <template #label>
                <span class="label-with-icon">
                  <i class="fa-solid fa-key"></i>
                  API Key
                </span>
              </template>
              <el-input
                v-model="form.api_key"
                type="password"
                show-password
                placeholder="请输入 API Key"
              />
            </el-form-item>
            <el-form-item required>
              <template #label>
                <span class="label-with-icon">
                  <i class="fa-solid fa-link"></i>
                  API Base URL
                </span>
              </template>
              <el-input
                v-model="form.api_url"
                :placeholder="platformBaseURLHints[form.type] || 'https://api.example.com'"
              />
              <div class="input-tip">
                <i class="fa-solid fa-info-circle"></i>
                填写平台 API 的基础地址（不含 /v1/chat/completions 后缀）
              </div>
            </el-form-item>
          </el-form>
        </template>

        <!-- 手动输入 Token -->
        <template v-if="form.addType === 'manual'">
          <el-form :model="form" label-position="top">
            <el-form-item label="Access Token" required>
              <el-input v-model="form.access_token" type="textarea" :rows="2" placeholder="OAuth Access Token" />
            </el-form-item>
            <el-form-item label="Refresh Token（可选）">
              <el-input v-model="form.refresh_token" type="textarea" :rows="2" placeholder="OAuth Refresh Token（用于自动刷新）" />
            </el-form-item>
          </el-form>
        </template>
      </div>

      <!-- 模型配置（新建时也可选） -->
      <div class="form-section">
        <div class="section-title">
          <i class="fa-solid fa-robot"></i>
          模型配置（可选）
        </div>
        <el-form-item label="允许的模型">
          <el-select
            v-model="form.allowedModelsList"
            multiple
            filterable
            allow-create
            default-first-option
            :loading="loadingModels"
            placeholder="留空表示允许所有模型"
            style="width: 100%"
          >
            <el-option-group
              v-for="group in availableModels"
              :key="group.platform"
              :label="group.platform"
            >
              <el-option
                v-for="model in group.models"
                :key="model"
                :label="model"
                :value="model"
              />
            </el-option-group>
          </el-select>
          <div class="input-tip">
            <i class="fa-solid fa-info-circle"></i>
            留空表示允许所有模型，可手动输入自定义模型名
          </div>
        </el-form-item>
      </div>

      <!-- 代理配置 -->
      <div class="form-section">
        <h4 class="section-title">代理配置</h4>
        <el-form-item label="选择代理">
          <el-select
            v-model="form.proxy_id"
            clearable
            placeholder="选择代理（留空直连）"
            style="width: 100%"
            :loading="loadingProxies"
          >
            <el-option
              v-for="proxy in proxyList"
              :key="proxy.id"
              :label="`${proxy.name} (${proxy.type}://${proxy.host}:${proxy.port})`"
              :value="proxy.id"
            />
          </el-select>
          <div class="input-tip">
            <i class="fa-solid fa-info-circle"></i>
            留空表示直连，选择代理后将通过该代理访问目标服务
          </div>
        </el-form-item>
      </div>
    </div>

    <!-- 步骤2: OAuth 授权 -->
    <div v-if="step === 2" class="form-step">
      <OAuthFlow
        :platform="form.type"
        :proxy="selectedProxy"
        :auth-mode="form.addType"
        @success="handleOAuthSuccess"
        @back="step = 1"
      />
    </div>

    <!-- 编辑模式 -->
    <div v-if="isEdit" class="form-step">
      <div class="edit-type-banner">
        <div class="type-badge" :style="{ background: getTypeColor(form.type) }">
          <i :class="getTypeIcon(form.type)"></i>
        </div>
        <span>{{ getTypeLabel(form.type) }}</span>
      </div>

      <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
        <el-form-item label="账户名称" prop="name">
          <el-input v-model="form.name" placeholder="账户名称" />
        </el-form-item>

        <el-form-item label="描述（可选）">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="账户用途说明..." />
        </el-form-item>

        <el-row :gutter="16">
          <el-col :span="6">
            <el-form-item label="优先级">
              <el-input-number v-model="form.priority" :min="1" :max="100" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="权重">
              <el-input-number v-model="form.weight" :min="1" :max="1000" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="最大并发">
              <el-input-number v-model="form.max_concurrency" :min="1" :max="100" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="启用">
              <el-switch v-model="form.enabled" />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- API 配置 (claude-console / openai / gemini) -->
        <div v-if="showEditApiConfig" class="form-section">
          <h4 class="section-title">API 配置</h4>
          <el-form-item label="API URL">
            <el-input v-model="form.api_url" placeholder="API Base URL" />
            <div class="input-tip">
              <i class="fa-solid fa-info-circle"></i>
              留空使用默认地址
            </div>
          </el-form-item>
          <el-form-item label="API Key">
            <el-input v-model="form.api_key" type="password" show-password placeholder="API Key" />
            <div class="input-tip">
              <i class="fa-solid fa-info-circle"></i>
              留空保持原有 Key 不变
            </div>
          </el-form-item>
        </div>

        <!-- AWS Bedrock ���置 -->
        <div v-if="form.type === 'bedrock'" class="form-section">
          <h4 class="section-title">AWS Bedrock 配置</h4>
          <el-form-item label="AWS Access Key ID">
            <el-input v-model="form.aws_access_key_id" placeholder="请输入 AWS Access Key ID" />
          </el-form-item>
          <el-form-item label="AWS Secret Access Key">
            <el-input v-model="form.aws_secret_access_key" type="password" show-password placeholder="请输入 AWS Secret Access Key（留空保持不变）" />
          </el-form-item>
          <el-form-item label="AWS Region">
            <el-select v-model="form.aws_region" placeholder="选择 AWS 区域" style="width: 100%">
              <el-option label="us-east-1 (美国东部)" value="us-east-1" />
              <el-option label="us-west-2 (美国西部)" value="us-west-2" />
              <el-option label="eu-west-1 (欧洲爱尔兰)" value="eu-west-1" />
              <el-option label="ap-northeast-1 (东京)" value="ap-northeast-1" />
              <el-option label="ap-southeast-1 (新加坡)" value="ap-southeast-1" />
            </el-select>
          </el-form-item>
        </div>

        <!-- Azure OpenAI 配置 -->
        <div v-if="form.type === 'azure-openai'" class="form-section">
          <h4 class="section-title">Azure OpenAI 配置</h4>
          <el-form-item label="Azure Endpoint">
            <el-input v-model="form.azure_endpoint" placeholder="https://your-resource.openai.azure.com" />
          </el-form-item>
          <el-form-item label="API Key">
            <el-input v-model="form.api_key" type="password" show-password placeholder="Azure API Key（留空保持不变）" />
          </el-form-item>
          <el-form-item label="部署名称">
            <el-input v-model="form.azure_deployment_name" placeholder="gpt-4" />
          </el-form-item>
          <el-form-item label="API 版本">
            <el-input v-model="form.azure_api_version" placeholder="2024-02-01" />
          </el-form-item>
        </div>

        <!-- ChatGPT 官方配置 -->
        <div v-if="form.type === 'openai-responses'" class="form-section">
          <h4 class="section-title">ChatGPT 官方配置</h4>
          <el-form-item label="Base URL">
            <el-input v-model="form.api_url" placeholder="默认: https://chatgpt.com/backend-api/codex" />
            <div class="input-tip">
              <i class="fa-solid fa-info-circle"></i>
              留空使用默认地址
            </div>
          </el-form-item>
          <el-form-item label="SessionKey (Access Token)">
            <el-input v-model="form.session_key" type="password" show-password placeholder="eyJhbGciOiJSUzI1NiI...（留空保持不变）" />
            <div class="input-tip">
              <i class="fa-solid fa-info-circle"></i>
              留空保持原有 SessionKey 不变
            </div>
          </el-form-item>
          <el-form-item label="Organization ID（可选）">
            <el-input v-model="form.organization_id" placeholder="org-..." />
          </el-form-item>
        </div>

        <!-- 模型配置 -->
        <div class="form-section">
          <div class="section-title">
            <i class="fa-solid fa-robot"></i>
            模型配置
          </div>
          <el-form-item label="允许的模型">
            <el-select
              v-model="form.allowedModelsList"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="选择或输入允许使用的模型"
              style="width: 100%"
            >
              <el-option-group
                v-for="group in availableModels"
                :key="group.platform"
                :label="group.platform"
              >
                <el-option
                  v-for="model in group.models"
                  :key="model"
                  :label="model"
                  :value="model"
                />
              </el-option-group>
            </el-select>
            <div class="input-tip">
              <i class="fa-solid fa-info-circle"></i>
              留空表示允许所有模型，可手动输入自定义模型名
            </div>
          </el-form-item>
          <el-form-item label="模型映射（可选）">
            <el-select
              v-model="form.selectedMappingIds"
              multiple
              filterable
              :loading="loadingMappings"
              placeholder="选择要应用的模型映射规则"
              style="width: 100%"
            >
              <el-option
                v-for="mapping in globalMappings"
                :key="mapping.id"
                :label="`${mapping.source_model} → ${mapping.target_model}`"
                :value="mapping.id"
              >
                <div class="mapping-option">
                  <span class="source-model">{{ mapping.source_model }}</span>
                  <span class="mapping-arrow-inline">→</span>
                  <span class="target-model">{{ mapping.target_model }}</span>
                </div>
              </el-option>
            </el-select>
            <div class="input-tip">
              <i class="fa-solid fa-info-circle"></i>
              从全局模型映射中选择要应用的规则，将请求的模型名映射到实际使用的模型
            </div>
          </el-form-item>
        </div>

        <!-- 代理配置 -->
        <div class="form-section">
          <h4 class="section-title">代理配置</h4>
          <el-form-item label="选择代理">
            <el-select
              v-model="form.proxy_id"
              clearable
              placeholder="选择代理（留空直连）"
              style="width: 100%"
              :loading="loadingProxies"
            >
              <el-option
                v-for="proxy in proxyList"
                :key="proxy.id"
                :label="`${proxy.name} (${proxy.type}://${proxy.host}:${proxy.port})`"
                :value="proxy.id"
              />
            </el-select>
            <div class="input-tip">
              <i class="fa-solid fa-info-circle"></i>
              留空表示直连，选择代理后将通过该代理访问目标服务
            </div>
          </el-form-item>
        </div>
      </el-form>
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <template v-if="!isEdit && step === 1">
          <el-button
            type="primary"
            :disabled="!canProceed"
            @click="handleNext"
          >
            {{ needsOAuth ? '下一步' : '创建账户' }}
            <i v-if="needsOAuth" class="fa-solid fa-arrow-right ml-2"></i>
          </el-button>
        </template>
        <template v-if="isEdit">
          <el-button type="primary" :loading="submitting" @click="handleSubmit">
            保存修改
          </el-button>
        </template>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ensureFontAwesomeLoaded } from '@/utils/fontawesome'
ensureFontAwesomeLoaded()

import { ref, reactive, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import OAuthFlow from './OAuthFlow.vue'
import api from '@/api'

const props = defineProps({
  modelValue: Boolean,
  editData: Object
})

const emit = defineEmits(['update:modelValue', 'success'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const isEdit = computed(() => !!props.editData?.id)

const formRef = ref()
const step = ref(1)
const submitting = ref(false)
const platformGroup = ref('')

// 平台定义 - 每个平台独立卡片
const platformGroups = [
  { key: 'claude', name: 'Claude', desc: 'Anthropic AI', icon: 'fa-solid fa-brain', gradient: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' },
  { key: 'openai', name: 'OpenAI', desc: 'GPT 系列模型', icon: 'fa-solid fa-robot', gradient: 'linear-gradient(135deg, #11998e 0%, #38ef7d 100%)' },
  { key: 'gemini', name: 'Gemini', desc: 'Google AI', icon: 'fa-brands fa-google', gradient: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)' },
  { key: 'deepseek', name: 'DeepSeek', desc: '深度求索', icon: 'fa-solid fa-microchip', gradient: 'linear-gradient(135deg, #4a7fd4 0%, #2756a8 100%)' },
  { key: 'qwen', name: '通义千问', desc: '阿里云百炼', icon: 'fa-solid fa-cloud', gradient: 'linear-gradient(135deg, #6236ff 0%, #4a1fd0 100%)' },
  { key: 'glm', name: '智谱 GLM', desc: '智谱 AI', icon: 'fa-solid fa-atom', gradient: 'linear-gradient(135deg, #3366cc 0%, #224499 100%)' },
  { key: 'moonshot', name: 'Kimi', desc: '月之暗面', icon: 'fa-solid fa-moon', gradient: 'linear-gradient(135deg, #3a3a5c 0%, #1a1a2e 100%)' },
  { key: 'doubao', name: '豆包', desc: '字节火山引擎', icon: 'fa-solid fa-fire', gradient: 'linear-gradient(135deg, #ff6a3d 0%, #d44a1d 100%)' },
  { key: 'baichuan', name: '百川智能', desc: 'Baichuan', icon: 'fa-solid fa-mountain', gradient: 'linear-gradient(135deg, #2c7be5 0%, #1a5ab8 100%)' },
  { key: 'yi', name: '零一万物', desc: 'Yi 系列', icon: 'fa-solid fa-star', gradient: 'linear-gradient(135deg, #00b386 0%, #008060 100%)' },
  { key: 'minimax', name: 'MiniMax', desc: 'MiniMax AI', icon: 'fa-solid fa-wand-sparkles', gradient: 'linear-gradient(135deg, #7c4dff 0%, #5a2ed4 100%)' },
  { key: 'stepfun', name: '阶跃星辰', desc: 'Step 系列', icon: 'fa-solid fa-stairs', gradient: 'linear-gradient(135deg, #2196f3 0%, #1565c0 100%)' },
  { key: 'spark', name: '讯飞星火', desc: 'Spark 系列', icon: 'fa-solid fa-fire-flame-curved', gradient: 'linear-gradient(135deg, #e53935 0%, #b71c1c 100%)' },
  { key: 'siliconflow', name: '硅基流动', desc: 'SiliconFlow', icon: 'fa-solid fa-microchip', gradient: 'linear-gradient(135deg, #6366f1 0%, #4338ca 100%)' },
  { key: 'xai', name: 'xAI', desc: 'Grok 系列', icon: 'fa-solid fa-bolt-lightning', gradient: 'linear-gradient(135deg, #1da1f2 0%, #0d7bbf 100%)' },
  { key: 'mistral', name: 'Mistral', desc: 'Mistral AI', icon: 'fa-solid fa-wind', gradient: 'linear-gradient(135deg, #ff7000 0%, #cc5800 100%)' },
  { key: 'cohere', name: 'Cohere', desc: 'Command 系列', icon: 'fa-solid fa-circle-nodes', gradient: 'linear-gradient(135deg, #39594d 0%, #263d34 100%)' },
  { key: 'custom', name: '自定义', desc: 'OpenAI 兼容', icon: 'fa-solid fa-plug', gradient: 'linear-gradient(135deg, #8898aa 0%, #6c757d 100%)' },
]

// 子平台定义（仅多子类型的平台需要二级选择）
const subplatformMap = {
  claude: [
    { value: 'claude-official', label: 'Claude Official', desc: 'OAuth 认证', icon: 'fa-solid fa-key', color: '#667eea' },
    { value: 'claude-console', label: 'Claude Console', desc: 'API Key 认证', icon: 'fa-solid fa-terminal', color: '#764ba2' },
    { value: 'bedrock', label: 'AWS Bedrock', desc: 'AWS 托管服务', icon: 'fa-brands fa-aws', color: '#ff9900' }
  ],
  openai: [
    { value: 'openai', label: 'OpenAI API', desc: 'API Key 认证', icon: 'fa-solid fa-bolt', color: '#11998e' },
    { value: 'openai-responses', label: 'ChatGPT 官方', desc: 'OAuth / SessionKey', icon: 'fa-solid fa-comments', color: '#38ef7d' },
    { value: 'azure-openai', label: 'Azure OpenAI', desc: 'Azure 托管', icon: 'fa-brands fa-microsoft', color: '#0078d4' }
  ],
  gemini: [
    { value: 'gemini', label: 'Gemini OAuth', desc: 'Google OAuth', icon: 'fa-brands fa-google', color: '#4facfe' },
    { value: 'gemini-api', label: 'Gemini API', desc: 'API Key 认证', icon: 'fa-brands fa-google', color: '#3d8bd4' }
  ],
  custom: [
    { value: 'custom', label: '自定义兼容 API', desc: '任意 OpenAI 兼容接口', icon: 'fa-solid fa-plug', color: '#607d8b' },
    { value: 'droid', label: 'Droid', desc: '第三方 Droid', icon: 'fa-solid fa-android', color: '#78909c' }
  ]
}

const currentSubplatforms = computed(() => subplatformMap[platformGroup.value] || [])

// 可用模型列表 - 从后端获取
const allModels = ref([])
const loadingModels = ref(false)

// 代理列表
const proxyList = ref([])
const loadingProxies = ref(false)

// 全局模型映射列表
const globalMappings = ref([])
const loadingMappings = ref(false)

// 根据当前平台筛选模型
const availableModels = computed(() => {
  let groupKey = platformGroup.value
  if (!groupKey && form.type) {
    for (const [key, subs] of Object.entries(subplatformMap)) {
      if (subs.some(s => s.value === form.type)) {
        groupKey = key
        break
      }
    }
    if (!groupKey) groupKey = form.type
  }

  if (!groupKey) return []
  return allModels.value.filter(g => g.platform.toLowerCase() === groupKey)
})

// 根据 proxy_id 获取选中的代理对象
const selectedProxy = computed(() => {
  if (!form.proxy_id) return null
  const proxy = proxyList.value.find(p => p.id === form.proxy_id)
  if (!proxy) return null
  return {
    enabled: true,
    type: proxy.type,
    host: proxy.host,
    port: proxy.port,
    username: proxy.username || '',
    password: proxy.password || ''
  }
})

// 加载模型列表
async function loadModels() {
  loadingModels.value = true
  try {
    const res = await api.getModels({ enabled: true })
    const models = res.data || []
    // 按平台分组
    const grouped = {}
    models.forEach(m => {
      const platform = m.platform || 'other'
      if (!grouped[platform]) {
        grouped[platform] = { platform: platform.charAt(0).toUpperCase() + platform.slice(1), models: [] }
      }
      grouped[platform].models.push(m.name)
    })
    allModels.value = Object.values(grouped)
  } catch (e) {
    console.error('Failed to load models:', e)
  } finally {
    loadingModels.value = false
  }
}

// 加载代理列表
async function loadProxies() {
  loadingProxies.value = true
  try {
    const res = await api.getEnabledProxyConfigs()
    proxyList.value = res.items || []
  } catch (e) {
    console.error('Failed to load proxies:', e)
  } finally {
    loadingProxies.value = false
  }
}

// 加载全局模型映射列表
async function loadMappings() {
  loadingMappings.value = true
  try {
    const res = await api.getModelMappings()
    // 只获取启用的映射
    globalMappings.value = (res.data?.mappings || []).filter(m => m.enabled)
  } catch (e) {
    console.error('Failed to load model mappings:', e)
  } finally {
    loadingMappings.value = false
  }
}

const defaultForm = {
  name: '',
  type: '',
  description: '',
  enabled: true,
  priority: 50,
  weight: 100,
  max_concurrency: 5,
  accountType: 'shared',
  addType: 'oauth',
  api_key: '',
  api_url: '',
  access_token: '',
  refresh_token: '',
  session_key: '',
  organization_id: '',
  aws_access_key_id: '',
  aws_secret_access_key: '',
  aws_region: 'us-east-1',
  azure_endpoint: '',
  azure_deployment_name: '',
  azure_api_version: '2024-02-01',
  allowed_models: '',
  allowedModelsList: [],
  model_mapping: '',
  selectedMappingIds: [],  // 选中的全局模型映射 ID 列表
  proxy_id: null  // 代理 ID
}

const form = reactive({ ...defaultForm })

const rules = {
  name: [{ required: true, message: '请输入账户名称', trigger: 'blur' }]
}

// 是否显示添加方式选择
const showAddTypeSelection = computed(() => {
  const type = form.type
  // 只有这些类型需要选择添加方式（openai 只用 API Key，不需要选择）
  return ['claude-official', 'openai-responses', 'gemini'].includes(type)
})

// 是否需要 API Key（用于验证）
const needsApiKey = computed(() => {
  const type = form.type
  if (['claude-console', 'azure-openai'].includes(type)) return true
  if ((type === 'openai' || type === 'gemini') && form.addType === 'apikey') return true
  return false
})

// OpenAI 兼容的平台类型列表（需要 API Key + Base URL 配置）
const openaiCompatibleTypes = [
  'deepseek', 'qwen', 'glm', 'moonshot', 'doubao', 'baichuan',
  'yi', 'minimax', 'stepfun', 'spark', 'siliconflow', 'xai', 'mistral', 'cohere', 'custom'
]

// 各平台默认 Base URL 提示
const platformBaseURLHints = {
  'deepseek': 'https://api.deepseek.com',
  'qwen': 'https://dashscope.aliyuncs.com/compatible-mode',
  'glm': 'https://open.bigmodel.cn/api/paas',
  'moonshot': 'https://api.moonshot.cn',
  'doubao': 'https://ark.cn-beijing.volces.com/api',
  'baichuan': 'https://api.baichuan-ai.com',
  'yi': 'https://api.lingyiwanwu.com',
  'minimax': 'https://api.minimax.chat',
  'stepfun': 'https://api.stepfun.com',
  'spark': 'https://spark-api-open.xf-yun.com',
  'siliconflow': 'https://api.siliconflow.cn',
  'xai': 'https://api.x.ai',
  'mistral': 'https://api.mistral.ai',
  'cohere': 'https://api.cohere.com/compatibility',
  'custom': 'https://your-api-endpoint.com',
}

// 是否显示平台特定配置
const showPlatformConfig = computed(() => {
  const type = form.type
  if (['claude-console', 'bedrock', 'azure-openai'].includes(type)) return true
  if ((type === 'openai' || type === 'gemini' || type === 'gemini-api') && form.addType === 'apikey') return true
  if (type === 'openai-responses' && form.addType === 'cookie') return true
  if (openaiCompatibleTypes.includes(type)) return true
  return false
})

// 获取平台配置标题
const getPlatformConfigTitle = computed(() => {
  const typeLabels = {
    'claude-console': 'Claude Console 配置',
    'bedrock': 'AWS Bedrock 配置',
    'azure-openai': 'Azure OpenAI 配置',
    'openai': 'OpenAI API 配置',
    'openai-responses': 'ChatGPT 官方配置',
    'gemini': 'Gemini 配置',
    'gemini-api': 'Gemini API 配置',
  }
  if (form.addType === 'manual') return 'Token 配置'
  if (openaiCompatibleTypes.includes(form.type)) {
    const group = platformGroups.find(g => g.key === form.type)
    return (group?.name || form.type) + ' 配置'
  }
  return typeLabels[form.type] || '平台配置'
})

// 是否需要 OAuth
const needsOAuth = computed(() => {
  // openai-responses 的 cookie 模式直接输入 SessionKey，不需要 OAuth 流程
  if (form.type === 'openai-responses' && form.addType === 'cookie') {
    return false
  }
  return form.addType === 'oauth' || form.addType === 'cookie'
})

// 编辑模式下是否显示 API 配置（URL �� Key）
const showEditApiConfig = computed(() => {
  const type = form.type
  // 这些类型在编辑时可以修改 API URL 和 API Key
  return ['claude-console', 'openai', 'gemini'].includes(type)
})

// 是否可以继续
const canProceed = computed(() => {
  if (!form.type || !form.name) return false
  return true
})

// 监听编辑数据变化
watch(() => props.editData, (val) => {
  if (val) {
    Object.assign(form, { ...defaultForm, ...val })
    // 后端字段映射到前端字段
    if (val.base_url) {
      form.api_url = val.base_url
    }
    // 解析 allowed_models 为数组
    if (val.allowed_models) {
      try {
        form.allowedModelsList = val.allowed_models.split(',').map(s => s.trim()).filter(Boolean)
      } catch {
        form.allowedModelsList = []
      }
    } else {
      form.allowedModelsList = []
    }
    // 解析 model_mapping JSON，匹配全局映射 ID
    if (val.model_mapping) {
      try {
        const mappingObj = JSON.parse(val.model_mapping)
        // 需要等待 globalMappings 加载后再匹配
        // 这里先保存原始映射，在 loadMappings 后再匹配
        form._rawModelMapping = mappingObj
        form.selectedMappingIds = []
      } catch {
        form.selectedMappingIds = []
      }
    } else {
      form.selectedMappingIds = []
    }
    // 找到对应的平台组
    for (const [key, subs] of Object.entries(subplatformMap)) {
      if (subs.some(s => s.value === val.type)) {
        platformGroup.value = key
        break
      }
    }
  }
}, { immediate: true })

// 监听弹窗显示
watch(visible, async (val) => {
  if (val && !isEdit.value) {
    resetForm()
  }
  if (val) {
    // 加载模型列表、代理列表和映射列表
    loadModels()
    loadProxies()
    await loadMappings()

    // 如果是编辑模式且有原始映射数据，匹配全局映射 ID
    if (form._rawModelMapping && globalMappings.value.length > 0) {
      const matchedIds = []
      for (const [source, target] of Object.entries(form._rawModelMapping)) {
        const found = globalMappings.value.find(
          m => m.source_model === source && m.target_model === target
        )
        if (found) {
          matchedIds.push(found.id)
        }
      }
      form.selectedMappingIds = matchedIds
      delete form._rawModelMapping
    }
  }
})

function selectPlatformGroup(key) {
  platformGroup.value = key
  form.type = ''
  form.allowedModelsList = []

  // 单类型平台自动选择 type（无需二级选择）
  const subs = subplatformMap[key]
  if (!subs) {
    form.type = key
    form.addType = 'apikey'
    return
  }

  // 多子类型平台设置默认添加方式
  if (key === 'claude') {
    form.addType = 'cookie'
  } else {
    form.addType = 'apikey'
  }
}

// 监听子平台类型变化，设置对应的默认添加方式
watch(() => form.type, (newType) => {
  if (!newType) return
  if (newType === 'claude-official') {
    form.addType = 'cookie'
  } else if (newType === 'openai-responses') {
    form.addType = 'oauth'
  } else {
    form.addType = 'apikey'
  }
})

function resetForm() {
  Object.assign(form, { ...defaultForm })
  platformGroup.value = ''
  step.value = 1
}

function handleClose() {
  visible.value = false
  resetForm()
}

async function handleNext() {
  const valid = await formRef.value?.validate().catch((err) => {
    console.error('Form validation failed:', err)
    return false
  })
  if (!valid) {
    ElMessage.warning('请填写必要的信息')
    return
  }

  if (needsOAuth.value) {
    step.value = 2
  } else {
    await handleSubmit()
  }
}

async function handleOAuthSuccess(tokenInfo) {
  // 处理 OAuth 成功返回的 token
  if (Array.isArray(tokenInfo)) {
    // 批量创建
    for (const token of tokenInfo) {
      form.access_token = token.access_token
      form.refresh_token = token.refresh_token
      form.session_key = token.session_key
      await createAccount()
    }
  } else {
    form.access_token = tokenInfo.access_token
    form.refresh_token = tokenInfo.refresh_token
    await createAccount()
  }
}

async function createAccount() {
  submitting.value = true
  try {
    const submitData = buildSubmitData()
    await api.createAccount(submitData)
    ElMessage.success('创建成功')
    emit('success')
    handleClose()
  } catch (e) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const submitData = buildSubmitData()

    if (isEdit.value) {
      await api.updateAccount(form.id, submitData)
      ElMessage.success('更新成功')
    } else {
      await api.createAccount(submitData)
      ElMessage.success('创建成功')
    }
    emit('success')
    handleClose()
  } catch (e) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

function buildSubmitData() {
  const data = {
    name: form.name,
    type: form.type,
    description: form.description,
    enabled: form.enabled,
    priority: form.priority,
    weight: form.weight,
    max_concurrency: form.max_concurrency,
    account_type: form.accountType
  }

  // 根据类型添加特定字段
  if (form.api_key) data.api_key = form.api_key
  if (form.api_url) data.base_url = form.api_url  // 前端用 api_url，后端用 base_url
  if (form.access_token) data.access_token = form.access_token
  if (form.refresh_token) data.refresh_token = form.refresh_token
  if (form.session_key) data.session_key = form.session_key
  if (form.organization_id) data.organization_id = form.organization_id

  // AWS Bedrock
  if (form.type === 'bedrock') {
    data.aws_access_key = form.aws_access_key_id
    data.aws_secret_key = form.aws_secret_access_key
    data.aws_region = form.aws_region
  }

  // Azure OpenAI
  if (form.type === 'azure-openai') {
    data.azure_endpoint = form.azure_endpoint
    data.azure_deployment_name = form.azure_deployment_name
    data.azure_api_version = form.azure_api_version
  }

  // 代理配置 (使用代理 ID)
  if (form.proxy_id) {
    data.proxy_id = form.proxy_id
  } else if (isEdit.value && props.editData?.proxy_id) {
    // 编辑模式下，如果原来有代理但现在清空了，发送 clear_proxy 标记
    data.clear_proxy = true
  }

  // 模型配置
  if (form.allowedModelsList?.length > 0) {
    data.allowed_models = form.allowedModelsList.join(',')
  } else if (isEdit.value && props.editData?.allowed_models) {
    // 编辑模式下，如果原来有允许的模型但现在清空了，发送 clear_allowed_models 标记
    data.clear_allowed_models = true
  }
  // 模型映射：从选中的全局映射 ID 构建 JSON 对象
  if (form.selectedMappingIds?.length > 0) {
    const mappingObj = {}
    form.selectedMappingIds.forEach(id => {
      const mapping = globalMappings.value.find(m => m.id === id)
      if (mapping) {
        mappingObj[mapping.source_model] = mapping.target_model
      }
    })
    if (Object.keys(mappingObj).length > 0) {
      data.model_mapping = JSON.stringify(mappingObj)
    }
  } else if (isEdit.value && props.editData?.model_mapping) {
    // 编辑模式下，如果原来有模型映射但现在清空了，发送 clear_model_mapping 标记
    data.clear_model_mapping = true
  }

  return data
}

function getTypeLabel(type) {
  for (const subs of Object.values(subplatformMap)) {
    const found = subs.find(s => s.value === type)
    if (found) return found.label
  }
  return type
}

function getTypeIcon(type) {
  for (const subs of Object.values(subplatformMap)) {
    const found = subs.find(s => s.value === type)
    if (found) return found.icon
  }
  return 'fa-solid fa-circle'
}

function getTypeColor(type) {
  for (const subs of Object.values(subplatformMap)) {
    const found = subs.find(s => s.value === type)
    if (found) return found.color
  }
  return '#999'
}
</script>

<style scoped>
.steps-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e5e7eb;
}

.step {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #9ca3af;
}

.step.active {
  color: #3b82f6;
}

.step.done {
  color: #10b981;
}

.step-num {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
}

.step.active .step-num {
  background: #3b82f6;
  color: white;
}

.step.done .step-num {
  background: #10b981;
  color: white;
}

.step-line {
  width: 60px;
  height: 2px;
  background: #e5e7eb;
}

.form-step {
  max-height: 60vh;
  overflow-y: auto;
  padding-right: 8px;
}

.form-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: #374151;
  margin: 0 0 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e5e7eb;
}

.platform-groups {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}

.platform-group-card {
  position: relative;
  padding: 12px;
  border: 2px solid #e5e7eb;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: white;
  display: flex;
  align-items: center;
  gap: 10px;
}

.platform-group-card:hover {
  border-color: #3b82f6;
  box-shadow: 0 3px 10px rgba(59, 130, 246, 0.12);
}

.platform-group-card.selected {
  border-color: #3b82f6;
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
}

.card-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 16px;
  flex-shrink: 0;
}

.card-info h5 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: #1f2937;
  line-height: 1.3;
}

.card-info p {
  margin: 0;
  font-size: 11px;
  color: #6b7280;
  line-height: 1.2;
}

.check-mark {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #3b82f6;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
}

.check-mark.small {
  width: 20px;
  height: 20px;
  font-size: 10px;
}

.subplatform-section {
  margin-top: 20px;
  padding: 16px;
  background: #f9fafb;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
}

.subplatform-label {
  font-size: 13px;
  font-weight: 500;
  color: #4b5563;
  margin-bottom: 12px;
}

.subplatform-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}

.subplatform-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: white;
}

.subplatform-card:hover {
  border-color: #3b82f6;
}

.subplatform-card.selected {
  border-color: #3b82f6;
  background: #eff6ff;
}

.sub-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 14px;
  flex-shrink: 0;
}

.sub-info {
  display: flex;
  flex-direction: column;
}

.sub-name {
  font-size: 13px;
  font-weight: 600;
  color: #1f2937;
}

.sub-desc {
  font-size: 11px;
  color: #6b7280;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.radio-tip {
  font-size: 12px;
  color: #6b7280;
  margin-left: 4px;
}

.edit-type-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #f0f7ff;
  border-radius: 8px;
  margin-bottom: 20px;
}

.type-badge {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 14px;
}

.edit-type-banner span {
  font-weight: 500;
  color: #1f2937;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.ml-2 {
  margin-left: 8px;
}

/* 添加方式卡片 */
.add-type-cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.add-type-card {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px;
  border: 2px solid #e5e7eb;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: white;
}

.add-type-card:hover {
  border-color: #3b82f6;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.1);
}

.add-type-card.selected {
  border-color: #3b82f6;
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
}

.type-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 16px;
  flex-shrink: 0;
}

.type-content {
  flex: 1;
}

.type-content h5 {
  margin: 0 0 4px;
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.type-content p {
  margin: 0;
  font-size: 12px;
  color: #6b7280;
  line-height: 1.4;
}

.type-check {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #3b82f6;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
}

/* 表单输入提示 */
.label-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.label-with-icon i {
  font-size: 14px;
}

.input-tip {
  font-size: 12px;
  color: #6b7280;
  margin-top: 6px;
}

.input-tip i {
  margin-right: 4px;
}

.input-tip a {
  color: #3b82f6;
  text-decoration: none;
}

.input-tip a:hover {
  text-decoration: underline;
}

.text-green-500 {
  color: #22c55e;
}

.text-blue-500 {
  color: #3b82f6;
}

.text-orange-500 {
  color: #f97316;
}

/* SessionKey 帮助说明 */
.session-key-help {
  margin-top: 16px;
}

.session-key-help .help-steps {
  margin: 12px 0 8px;
  padding-left: 20px;
  font-size: 13px;
  line-height: 1.8;
  color: #374151;
}

.session-key-help .help-steps li {
  margin-bottom: 4px;
}

.session-key-help .help-steps a {
  color: #3b82f6;
  text-decoration: none;
}

.session-key-help .help-steps a:hover {
  text-decoration: underline;
}

.session-key-help .help-steps kbd {
  background: #e5e7eb;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
}

.session-key-help .help-steps code {
  background: #dbeafe;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  color: #1e40af;
}

.session-key-help .help-note {
  margin-top: 8px;
  font-size: 12px;
  color: #6b7280;
}

.session-key-help .help-note code {
  background: #f3f4f6;
  padding: 2px 4px;
  border-radius: 3px;
  font-family: monospace;
}

/* 模型映射选项样式 */
.mapping-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.source-model {
  color: #3b82f6;
  font-weight: 500;
}

.mapping-arrow-inline {
  color: #9ca3af;
}

.target-model {
  color: #10b981;
  font-weight: 500;
}
</style>
