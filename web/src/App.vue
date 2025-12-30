<!--
 * 文件作用：Vue应用根组件，定义全局样式和路由出口
 * 负责功能：
 *   - 应用根模板
 *   - 全局CSS重置
 *   - 路由视图容器
 * 重要程度：⭐⭐⭐⭐ 重要（应用根组件）
 * 依赖模块：vue-router
-->
<template>
  <router-view />
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}
</style>
