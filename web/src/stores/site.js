/**
 * 文件作用：站点信息状态管理，提供全局可响应式的站点名称
 * 负责功能：
 *   - 从后端获取站点名称
 *   - 提供 siteName 响应式变量供全局使用
 *   - 自动更新 document.title
 * 重要程度：⭐⭐⭐ 一般（UI 显示）
 * 依赖模块：pinia, api
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export const useSiteStore = defineStore('site', () => {
  const siteName = ref(document.title || 'Go Proxy Pro')
  let fetched = false

  async function fetchSiteName() {
    if (fetched) return
    try {
      const res = await api.getSiteInfo()
      if (res.site_name) {
        siteName.value = res.site_name
        document.title = res.site_name
      }
      fetched = true
    } catch {
      // ignore, use default
    }
  }

  function refresh() {
    fetched = false
    return fetchSiteName()
  }

  return { siteName, fetchSiteName, refresh }
})
