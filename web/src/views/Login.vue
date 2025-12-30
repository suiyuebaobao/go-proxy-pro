<!--
 * 文件作用：用户登录页面，提供系统入口认证
 * 负责功能：
 *   - 用户名密码输入
 *   - 验证码获取和验证
 *   - 登录请求和状态管理
 *   - 登录成功后跳转
 * 重要程度：⭐⭐⭐⭐ 重要（系统入口）
 * 依赖模块：element-plus, vue-router, user store, api
-->
<template>
  <div class="login-container">
    <div class="login-card">
      <h2 class="login-title">Go-AIProxy</h2>
      <p class="login-subtitle">AI API 代理管理平台</p>
      <p class="welcome-text">欢迎各位</p>

      <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin">
        <el-form-item prop="username">
          <el-input
            v-model="form.username"
            placeholder="用户名"
            prefix-icon="User"
            size="large"
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            prefix-icon="Lock"
            size="large"
            show-password
          />
        </el-form-item>

        <el-form-item v-if="captchaEnabled" prop="captcha">
          <div class="captcha-row">
            <el-input
              v-model="form.captcha"
              placeholder="验证码"
              size="large"
              class="captcha-input"
              @keyup.enter="handleLogin"
            />
            <img
              v-if="captchaImage"
              :src="captchaImage"
              class="captcha-image"
              @click="refreshCaptcha"
              title="点击刷新验证码"
            />
            <div v-else class="captcha-placeholder" @click="refreshCaptcha">
              加载中...
            </div>
          </div>
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="login-btn"
            @click="handleLogin"
          >
            登录
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import api from '@/api'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref()
const loading = ref(false)
const captchaImage = ref('')
const captchaId = ref('')
const captchaEnabled = ref(true) // 默认启用

const form = reactive({
  username: '',
  password: '',
  captcha: ''
})

const rules = computed(() => {
  const baseRules = {
    username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
    password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
  }
  if (captchaEnabled.value) {
    baseRules.captcha = [{ required: true, message: '请输入验证码', trigger: 'blur' }]
  }
  return baseRules
})

async function refreshCaptcha() {
  try {
    const res = await api.getCaptcha()
    captchaEnabled.value = res.data.enabled !== false
    if (captchaEnabled.value) {
      captchaId.value = res.data.captcha_id
      captchaImage.value = res.data.image
    }
  } catch (e) {
    // 获取验证码失败时默认不启用验证码
    captchaEnabled.value = false
  }
}

async function handleLogin() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    const loginData = {
      username: form.username,
      password: form.password
    }
    if (captchaEnabled.value) {
      loginData.captcha_id = captchaId.value
      loginData.captcha = form.captcha
    }
    await userStore.login(loginData)
    ElMessage.success('登录成功')
    // 直接走 /dashboard 做角色分流，避免多余跳转带来的加载与卡顿
    router.push('/dashboard')
  } catch (e) {
    // 登录失败刷新验证码
    if (captchaEnabled.value) {
      refreshCaptcha()
      form.captcha = ''
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refreshCaptcha()
})
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.login-title {
  text-align: center;
  font-size: 28px;
  color: #333;
  margin-bottom: 8px;
}

.login-subtitle {
  text-align: center;
  color: #999;
  margin-bottom: 10px;
}

.welcome-text {
  text-align: center;
  color: #667eea;
  font-size: 14px;
  margin-bottom: 20px;
  font-weight: 500;
}

.login-btn {
  width: 100%;
}

.captcha-row {
  display: flex;
  gap: 10px;
  width: 100%;
}

.captcha-input {
  flex: 1;
}

.captcha-image {
  height: 40px;
  cursor: pointer;
  border-radius: 4px;
  border: 1px solid #dcdfe6;
}

.captcha-placeholder {
  height: 40px;
  width: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  border: 1px solid #dcdfe6;
  color: #999;
  font-size: 12px;
  cursor: pointer;
}
</style>
