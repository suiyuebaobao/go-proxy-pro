/**
 * 文件作用：Vue应用入口，初始化Vue应用和全局插件
 * 负责功能：
 *   - Vue应用创建
 *   - Element Plus UI库注册
 *   - 路由和状态管理初始化
 *   - 全局图标注册
 * 重要程度：⭐⭐⭐⭐ 重要（前端入口）
 * 依赖模块：vue, pinia, element-plus, router
 */
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import {
  ArrowDown,
  ArrowRight,
  Box,
  Calendar,
  Check,
  Clock,
  Coin,
  Connection,
  CopyDocument,
  Cpu,
  DataAnalysis,
  Delete,
  Document,
  Expand,
  Files,
  Filter,
  Fold,
  Key,
  Money,
  Monitor,
  More,
  Notebook,
  Plus,
  Position,
  Refresh,
  Search,
  Setting,
  ShoppingBag,
  SwitchButton,
  Tickets,
  Timer,
  Tools,
  TrendCharts,
  User,
  VideoPlay,
  Warning
} from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'

const app = createApp(App)

// 注册常用图标（避免全量注册导致首屏 JS 解析/执行卡顿）
const icons = {
  ArrowDown,
  ArrowRight,
  Box,
  Calendar,
  Check,
  Clock,
  Coin,
  Connection,
  CopyDocument,
  Cpu,
  DataAnalysis,
  Delete,
  Document,
  Expand,
  Files,
  Filter,
  Fold,
  Key,
  Money,
  Monitor,
  More,
  Notebook,
  Plus,
  Position,
  Refresh,
  Search,
  Setting,
  ShoppingBag,
  SwitchButton,
  Tickets,
  Timer,
  Tools,
  TrendCharts,
  User,
  VideoPlay,
  Warning
}
Object.entries(icons).forEach(([name, component]) => {
  app.component(name, component)
})

app.use(createPinia())
app.use(router)
app.use(ElementPlus)

try {
  app.mount('#app')
  window.__APP_BOOTED__ = true
} catch (e) {
  // eslint-disable-next-line no-console
  console.error('前端启动失败：', e)
  window.__APP_BOOT_ERROR__ = String((e && e.message) || e)
}
