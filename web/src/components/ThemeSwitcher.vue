<!--
 * 文件作用：主题切换器组件，提供多主题选择界面
 * 负责功能：
 *   - 展示所有可用主题的色彩预览
 *   - 切换主题并持久化选择
 *   - 当前主题高亮指示
 * 重要程度：⭐⭐⭐ 一般（UI辅助组件）
 * 依赖组件：element-plus, theme store
-->
<template>
  <el-popover
    placement="bottom"
    :width="240"
    trigger="click"
    :show-arrow="true"
    popper-class="theme-popover"
  >
    <template #reference>
      <div class="theme-trigger" title="切换主题">
        <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="5" />
          <line x1="12" y1="1" x2="12" y2="3" />
          <line x1="12" y1="21" x2="12" y2="23" />
          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
          <line x1="1" y1="12" x2="3" y2="12" />
          <line x1="21" y1="12" x2="23" y2="12" />
          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
        </svg>
      </div>
    </template>

    <div class="theme-panel">
      <div class="theme-panel-title">选择主题</div>
      <div class="theme-grid">
        <div
          v-for="theme in themes"
          :key="theme.key"
          class="theme-item"
          :class="{ active: themeStore.currentTheme === theme.key }"
          @click="themeStore.applyTheme(theme.key)"
        >
          <div class="theme-swatch" :style="swatchStyle(theme)">
            <svg
              v-if="themeStore.currentTheme === theme.key"
              viewBox="0 0 24 24"
              width="14"
              height="14"
              fill="none"
              stroke="#fff"
              stroke-width="3"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
          </div>
          <span class="theme-name">{{ theme.name }}</span>
        </div>
      </div>
    </div>
  </el-popover>
</template>

<script setup>
import { computed } from 'vue'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
const themes = computed(() => themeStore.getThemes())

function swatchStyle(theme) {
  if (theme.dark) {
    return {
      background: `linear-gradient(135deg, #1e2130 50%, ${theme.color} 50%)`,
      border: '2px solid #3a4060',
    }
  }
  return {
    background: theme.color,
  }
}
</script>

<style scoped>
.theme-trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  cursor: pointer;
  color: var(--pink-text-secondary, #8b7d92);
  transition: all 0.2s;
}

.theme-trigger:hover {
  background: var(--pink-accent-light, #faf2f4);
  color: var(--pink-accent, #c97b8b);
}

.theme-panel {
  padding: 4px 0;
}

.theme-panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--pink-text, #3a3045);
  margin-bottom: 12px;
  padding: 0 4px;
}

.theme-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.theme-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 8px 4px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.theme-item:hover {
  background: var(--pink-accent-light, #faf2f4);
}

.theme-item.active {
  background: var(--pink-accent-light, #faf2f4);
}

.theme-swatch {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.2s, box-shadow 0.2s;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.12);
}

.theme-item:hover .theme-swatch {
  transform: scale(1.1);
}

.theme-item.active .theme-swatch {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.theme-name {
  font-size: 11px;
  color: var(--pink-text-secondary, #8b7d92);
  font-weight: 500;
  white-space: nowrap;
}

.theme-item.active .theme-name {
  color: var(--pink-accent, #c97b8b);
  font-weight: 600;
}
</style>
