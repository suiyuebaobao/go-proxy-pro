/**
 * 文件作用：用户状态管理，管理登录状态和用户信息
 * 负责功能：
 *   - 用户登录/登出
 *   - Token存储管理
 *   - 用户信息获取
 *   - 登录状态判断
 * 重要程度：⭐⭐⭐⭐ 重要（认证状态核心）
 * 依赖模块：pinia, api
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))

  const isLoggedIn = computed(() => !!token.value)

  async function login(loginData) {
    const res = await api.login(loginData)
    token.value = res.data.token
    user.value = res.data.user
    localStorage.setItem('token', token.value)
    localStorage.setItem('user', JSON.stringify(user.value))
    return res
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  async function fetchProfile() {
    const res = await api.getProfile()
    user.value = res.data
    localStorage.setItem('user', JSON.stringify(user.value))
  }

  return { token, user, isLoggedIn, login, logout, fetchProfile }
})
