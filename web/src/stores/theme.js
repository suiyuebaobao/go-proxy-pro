/**
 * 文件作用：主题状态管理，支持多主题切换
 * 负责功能：
 *   - 6种精心设计的主题预设（樱花粉、海洋蓝、森林绿、薰衣草、落日橙、午夜暗）
 *   - 主题切换与localStorage持久化
 *   - CSS变量动态注入到document.documentElement
 *   - 暗色主题的特殊html class处理
 * 重要程度：⭐⭐⭐⭐ 重要（全局UI主题控制）
 * 依赖模块：pinia
 */
import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const THEMES = {
  sakura: {
    name: '樱花粉',
    color: '#c97b8b',
    dark: false,
    vars: {
      '--el-color-primary': '#c97b8b',
      '--el-color-primary-light-3': '#d9a0ac',
      '--el-color-primary-light-5': '#e4bdc5',
      '--el-color-primary-light-7': '#eed8dd',
      '--el-color-primary-light-8': '#f4e5e9',
      '--el-color-primary-light-9': '#faf2f4',
      '--el-color-primary-dark-2': '#b06b7a',
      '--el-bg-color': '#fdf8f9',
      '--el-bg-color-page': '#faf5f7',
      '--el-bg-color-overlay': '#ffffff',
      '--el-fill-color': '#f5e8ec',
      '--el-fill-color-light': '#fdf0f3',
      '--el-fill-color-lighter': '#fef7f9',
      '--el-fill-color-extra-light': '#fefafb',
      '--el-fill-color-blank': '#ffffff',
      '--el-border-color': '#f0dde2',
      '--el-border-color-light': '#f5e8ec',
      '--el-border-color-lighter': '#f9eff2',
      '--el-border-color-extra-light': '#fcf5f7',
      '--el-text-color-primary': '#3a3045',
      '--el-text-color-regular': '#5a4d63',
      '--el-text-color-secondary': '#6b6573',
      '--el-text-color-placeholder': '#9e92a5',
      '--el-text-color-disabled': '#8b7d92',
      '--el-box-shadow': '0 1px 4px rgba(180, 140, 155, 0.08)',
      '--el-box-shadow-light': '0 1px 3px rgba(180, 140, 155, 0.06)',
      '--pink-bg': '#fdf8f9',
      '--pink-surface': '#ffffff',
      '--pink-sidebar': '#ffffff',
      '--pink-border': '#f0dde2',
      '--pink-text': '#3a3045',
      '--pink-text-secondary': '#8b7d92',
      '--pink-accent': '#c97b8b',
      '--pink-accent-light': '#faf2f4',
      '--pink-accent-hover': '#b06b7a',
      '--theme-scrollbar': '#e4bdc5',
      '--theme-card-hover-border': '#e4bdc5',
      '--theme-table-row-hover': '#fef5f7',
    }
  },

  ocean: {
    name: '海洋蓝',
    color: '#4a90d9',
    dark: false,
    vars: {
      '--el-color-primary': '#4a90d9',
      '--el-color-primary-light-3': '#7bb0e4',
      '--el-color-primary-light-5': '#a5c8ee',
      '--el-color-primary-light-7': '#cedff6',
      '--el-color-primary-light-8': '#deeaf9',
      '--el-color-primary-light-9': '#eef4fc',
      '--el-color-primary-dark-2': '#3d79b8',
      '--el-bg-color': '#f5f8fc',
      '--el-bg-color-page': '#f0f4f9',
      '--el-bg-color-overlay': '#ffffff',
      '--el-fill-color': '#dce7f3',
      '--el-fill-color-light': '#eaf1fa',
      '--el-fill-color-lighter': '#f2f6fc',
      '--el-fill-color-extra-light': '#f8fafe',
      '--el-fill-color-blank': '#ffffff',
      '--el-border-color': '#d0ddef',
      '--el-border-color-light': '#dce7f3',
      '--el-border-color-lighter': '#e8eff7',
      '--el-border-color-extra-light': '#f2f7fc',
      '--el-text-color-primary': '#1f2d3d',
      '--el-text-color-regular': '#3d5167',
      '--el-text-color-secondary': '#5a6d82',
      '--el-text-color-placeholder': '#8896a5',
      '--el-text-color-disabled': '#96a3b0',
      '--el-box-shadow': '0 1px 4px rgba(74, 144, 217, 0.08)',
      '--el-box-shadow-light': '0 1px 3px rgba(74, 144, 217, 0.06)',
      '--pink-bg': '#f5f8fc',
      '--pink-surface': '#ffffff',
      '--pink-sidebar': '#ffffff',
      '--pink-border': '#d0ddef',
      '--pink-text': '#1f2d3d',
      '--pink-text-secondary': '#5a6d82',
      '--pink-accent': '#4a90d9',
      '--pink-accent-light': '#eef4fc',
      '--pink-accent-hover': '#3d79b8',
      '--theme-scrollbar': '#a5c8ee',
      '--theme-card-hover-border': '#7bb0e4',
      '--theme-table-row-hover': '#f2f7fc',
    }
  },

  forest: {
    name: '森林绿',
    color: '#5ba784',
    dark: false,
    vars: {
      '--el-color-primary': '#5ba784',
      '--el-color-primary-light-3': '#88c0a4',
      '--el-color-primary-light-5': '#add3c2',
      '--el-color-primary-light-7': '#d2e6da',
      '--el-color-primary-light-8': '#e0ede6',
      '--el-color-primary-light-9': '#eff6f2',
      '--el-color-primary-dark-2': '#4d8f71',
      '--el-bg-color': '#f5faf7',
      '--el-bg-color-page': '#f0f6f3',
      '--el-bg-color-overlay': '#ffffff',
      '--el-fill-color': '#d8e8df',
      '--el-fill-color-light': '#e8f2ec',
      '--el-fill-color-lighter': '#f2f8f5',
      '--el-fill-color-extra-light': '#f8fbf9',
      '--el-fill-color-blank': '#ffffff',
      '--el-border-color': '#cdddd5',
      '--el-border-color-light': '#dbe8e1',
      '--el-border-color-lighter': '#e8f0ec',
      '--el-border-color-extra-light': '#f2f7f4',
      '--el-text-color-primary': '#1f3029',
      '--el-text-color-regular': '#3d5547',
      '--el-text-color-secondary': '#5a7066',
      '--el-text-color-placeholder': '#889e92',
      '--el-text-color-disabled': '#96aaa0',
      '--el-box-shadow': '0 1px 4px rgba(91, 167, 132, 0.08)',
      '--el-box-shadow-light': '0 1px 3px rgba(91, 167, 132, 0.06)',
      '--pink-bg': '#f5faf7',
      '--pink-surface': '#ffffff',
      '--pink-sidebar': '#ffffff',
      '--pink-border': '#cdddd5',
      '--pink-text': '#1f3029',
      '--pink-text-secondary': '#5a7066',
      '--pink-accent': '#5ba784',
      '--pink-accent-light': '#eff6f2',
      '--pink-accent-hover': '#4d8f71',
      '--theme-scrollbar': '#add3c2',
      '--theme-card-hover-border': '#88c0a4',
      '--theme-table-row-hover': '#f2f8f5',
    }
  },

  lavender: {
    name: '薰衣草',
    color: '#8b7cc8',
    dark: false,
    vars: {
      '--el-color-primary': '#8b7cc8',
      '--el-color-primary-light-3': '#aca0d8',
      '--el-color-primary-light-5': '#c5bde3',
      '--el-color-primary-light-7': '#ddd8ee',
      '--el-color-primary-light-8': '#e9e5f3',
      '--el-color-primary-light-9': '#f4f2f9',
      '--el-color-primary-dark-2': '#7668af',
      '--el-bg-color': '#f8f6fd',
      '--el-bg-color-page': '#f4f1fa',
      '--el-bg-color-overlay': '#ffffff',
      '--el-fill-color': '#e0dbed',
      '--el-fill-color-light': '#ece9f5',
      '--el-fill-color-lighter': '#f4f2f9',
      '--el-fill-color-extra-light': '#f9f8fc',
      '--el-fill-color-blank': '#ffffff',
      '--el-border-color': '#d5cfeb',
      '--el-border-color-light': '#e0dcf0',
      '--el-border-color-lighter': '#ece9f5',
      '--el-border-color-extra-light': '#f4f2f9',
      '--el-text-color-primary': '#2d2940',
      '--el-text-color-regular': '#4d4666',
      '--el-text-color-secondary': '#6b6580',
      '--el-text-color-placeholder': '#9590a5',
      '--el-text-color-disabled': '#8a85a0',
      '--el-box-shadow': '0 1px 4px rgba(139, 124, 200, 0.08)',
      '--el-box-shadow-light': '0 1px 3px rgba(139, 124, 200, 0.06)',
      '--pink-bg': '#f8f6fd',
      '--pink-surface': '#ffffff',
      '--pink-sidebar': '#ffffff',
      '--pink-border': '#d5cfeb',
      '--pink-text': '#2d2940',
      '--pink-text-secondary': '#6b6580',
      '--pink-accent': '#8b7cc8',
      '--pink-accent-light': '#f4f2f9',
      '--pink-accent-hover': '#7668af',
      '--theme-scrollbar': '#c5bde3',
      '--theme-card-hover-border': '#aca0d8',
      '--theme-table-row-hover': '#f6f4fb',
    }
  },

  sunset: {
    name: '落日橙',
    color: '#d08b50',
    dark: false,
    vars: {
      '--el-color-primary': '#d08b50',
      '--el-color-primary-light-3': '#dda97e',
      '--el-color-primary-light-5': '#e8c5a8',
      '--el-color-primary-light-7': '#f2ddd0',
      '--el-color-primary-light-8': '#f7e9df',
      '--el-color-primary-light-9': '#fbf4ee',
      '--el-color-primary-dark-2': '#b47844',
      '--el-bg-color': '#fcf8f3',
      '--el-bg-color-page': '#f9f4ee',
      '--el-bg-color-overlay': '#ffffff',
      '--el-fill-color': '#eddfd0',
      '--el-fill-color-light': '#f5ebe0',
      '--el-fill-color-lighter': '#f9f3ec',
      '--el-fill-color-extra-light': '#fcf9f5',
      '--el-fill-color-blank': '#ffffff',
      '--el-border-color': '#e5d5c4',
      '--el-border-color-light': '#eddfd0',
      '--el-border-color-lighter': '#f3eae0',
      '--el-border-color-extra-light': '#f8f2eb',
      '--el-text-color-primary': '#3d3020',
      '--el-text-color-regular': '#665540',
      '--el-text-color-secondary': '#806d58',
      '--el-text-color-placeholder': '#a59580',
      '--el-text-color-disabled': '#baa890',
      '--el-box-shadow': '0 1px 4px rgba(208, 139, 80, 0.08)',
      '--el-box-shadow-light': '0 1px 3px rgba(208, 139, 80, 0.06)',
      '--pink-bg': '#fcf8f3',
      '--pink-surface': '#ffffff',
      '--pink-sidebar': '#ffffff',
      '--pink-border': '#e5d5c4',
      '--pink-text': '#3d3020',
      '--pink-text-secondary': '#806d58',
      '--pink-accent': '#d08b50',
      '--pink-accent-light': '#fbf4ee',
      '--pink-accent-hover': '#b47844',
      '--theme-scrollbar': '#e8c5a8',
      '--theme-card-hover-border': '#dda97e',
      '--theme-table-row-hover': '#faf5ef',
    }
  },

  midnight: {
    name: '午夜暗',
    color: '#6d9fd4',
    dark: true,
    vars: {
      '--el-color-primary': '#6d9fd4',
      '--el-color-primary-light-3': '#567ea8',
      '--el-color-primary-light-5': '#3f6080',
      '--el-color-primary-light-7': '#2e4a64',
      '--el-color-primary-light-8': '#263d54',
      '--el-color-primary-light-9': '#1e3245',
      '--el-color-primary-dark-2': '#85b1dd',
      '--el-bg-color': '#161922',
      '--el-bg-color-page': '#12141c',
      '--el-bg-color-overlay': '#1e2130',
      '--el-fill-color': '#252838',
      '--el-fill-color-light': '#1e2130',
      '--el-fill-color-lighter': '#1a1d2a',
      '--el-fill-color-extra-light': '#171a25',
      '--el-fill-color-blank': '#1e2130',
      '--el-border-color': '#2e3348',
      '--el-border-color-light': '#282d40',
      '--el-border-color-lighter': '#222638',
      '--el-border-color-extra-light': '#1e2232',
      '--el-text-color-primary': '#e0e4f0',
      '--el-text-color-regular': '#b8bdd0',
      '--el-text-color-secondary': '#8890a8',
      '--el-text-color-placeholder': '#5a6278',
      '--el-text-color-disabled': '#4a5268',
      '--el-box-shadow': '0 1px 6px rgba(0, 0, 0, 0.3)',
      '--el-box-shadow-light': '0 1px 4px rgba(0, 0, 0, 0.2)',
      '--el-mask-color': 'rgba(0, 0, 0, 0.6)',
      '--el-mask-color-extra-light': 'rgba(0, 0, 0, 0.4)',
      '--pink-bg': '#12141c',
      '--pink-surface': '#1e2130',
      '--pink-sidebar': '#181b28',
      '--pink-border': '#2e3348',
      '--pink-text': '#e0e4f0',
      '--pink-text-secondary': '#8890a8',
      '--pink-accent': '#6d9fd4',
      '--pink-accent-light': '#1e2a40',
      '--pink-accent-hover': '#85b1dd',
      '--theme-scrollbar': '#3a4060',
      '--theme-card-hover-border': '#3e4560',
      '--theme-table-row-hover': '#222638',
    }
  },
}

export const useThemeStore = defineStore('theme', () => {
  const currentTheme = ref(localStorage.getItem('app-theme') || 'sakura')

  function getThemes() {
    return Object.entries(THEMES).map(([key, t]) => ({
      key,
      name: t.name,
      color: t.color,
      dark: t.dark,
    }))
  }

  function getThemeMeta() {
    return THEMES[currentTheme.value] || THEMES.sakura
  }

  function applyTheme(themeKey) {
    const theme = THEMES[themeKey]
    if (!theme) return

    const root = document.documentElement

    Object.entries(theme.vars).forEach(([prop, value]) => {
      root.style.setProperty(prop, value)
    })

    if (theme.dark) {
      root.classList.add('theme-dark')
    } else {
      root.classList.remove('theme-dark')
    }

    currentTheme.value = themeKey
    localStorage.setItem('app-theme', themeKey)
  }

  function initTheme() {
    const saved = localStorage.getItem('app-theme')
    const key = (saved && THEMES[saved]) ? saved : 'sakura'
    applyTheme(key)
  }

  return { currentTheme, getThemes, getThemeMeta, applyTheme, initTheme }
})
