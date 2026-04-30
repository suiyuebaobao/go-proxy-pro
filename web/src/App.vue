<!--
 * 文件作用：Vue应用根组件，定义全局样式和路由出口
 * 负责功能：
 *   - 应用根模板
 *   - 初始化主题系统
 *   - 全局CSS重置和Element Plus组件覆盖
 *   - 暗色主题专用样式覆盖
 * 重要程度：⭐⭐⭐⭐ 重要（应用根组件）
 * 依赖模块：vue-router, theme store
-->
<template>
  <router-view />
</template>

<script setup>
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
themeStore.initTheme()
</script>

<style>
/* ===================== 基础重置 ===================== */
:root {
  --el-font-size-base: 14px;
  --el-font-size-small: 13px;
  --el-font-size-extra-small: 12px;
  --el-border-radius-base: 10px;
  --el-border-radius-small: 6px;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background-color: var(--pink-bg);
  color: var(--pink-text);
}

/* ===================== Element Plus 组件覆盖 ===================== */

/* Card */
.el-card {
  border: 1px solid var(--pink-border) !important;
  box-shadow: none !important;
  border-radius: 12px !important;
  background-color: var(--el-fill-color-blank) !important;
}

.el-card:hover {
  border-color: var(--theme-card-hover-border) !important;
}

/* Buttons */
.el-button--primary {
  background-color: var(--pink-accent) !important;
  border-color: var(--pink-accent) !important;
  color: #fff !important;
}

.el-button--primary:hover,
.el-button--primary:focus {
  background-color: var(--pink-accent-hover) !important;
  border-color: var(--pink-accent-hover) !important;
  color: #fff !important;
}

.el-button--success {
  background-color: #67c23a !important;
  border-color: #67c23a !important;
  color: #fff !important;
}

.el-button--success:hover,
.el-button--success:focus {
  background-color: #529b2e !important;
  border-color: #529b2e !important;
  color: #fff !important;
}

.el-button--warning {
  background-color: #e6a23c !important;
  border-color: #e6a23c !important;
  color: #fff !important;
}

.el-button--warning:hover,
.el-button--warning:focus {
  background-color: #cf9236 !important;
  border-color: #cf9236 !important;
  color: #fff !important;
}

.el-button--danger {
  background-color: #f56c6c !important;
  border-color: #f56c6c !important;
  color: #fff !important;
}

.el-button--danger:hover,
.el-button--danger:focus {
  background-color: #dd6161 !important;
  border-color: #dd6161 !important;
  color: #fff !important;
}

.el-button--info {
  background-color: #909399 !important;
  border-color: #909399 !important;
  color: #fff !important;
}

.el-button--info:hover,
.el-button--info:focus {
  background-color: #73767a !important;
  border-color: #73767a !important;
  color: #fff !important;
}

.el-button--default {
  background-color: var(--el-fill-color-blank) !important;
  border-color: var(--pink-border) !important;
  color: var(--pink-text) !important;
}

.el-button--default:hover,
.el-button--default:focus {
  background-color: var(--pink-accent-light) !important;
  border-color: var(--pink-accent) !important;
  color: var(--pink-accent) !important;
}

.el-button {
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.2px;
  border-radius: 8px !important;
  transition: all 0.2s ease;
}

.el-button--small {
  font-size: 12px;
  padding: 5px 11px;
  border-radius: 6px !important;
}

.el-button--large {
  font-size: 15px;
  border-radius: 10px !important;
}

.el-button.is-link {
  font-size: 13px;
  font-weight: 500;
  padding: 2px 4px;
}

.el-button + .el-button {
  margin-left: 8px;
}

/* Table */
.el-table {
  --el-table-border-color: var(--pink-border);
  --el-table-header-bg-color: var(--pink-accent-light);
  --el-table-header-text-color: var(--pink-text);
  --el-table-row-hover-bg-color: var(--theme-table-row-hover);
  --el-table-bg-color: var(--el-fill-color-blank);
  --el-table-tr-bg-color: var(--el-fill-color-blank);
  --el-table-text-color: var(--el-text-color-regular);
}

.el-table th.el-table__cell {
  font-weight: 600;
  font-size: 13px;
  letter-spacing: 0.3px;
}

/* Pagination */
.el-pagination .el-pager li.is-active {
  background-color: var(--pink-accent) !important;
  color: #fff !important;
}

/* Tags */
.el-tag--info {
  background-color: var(--pink-accent-light) !important;
  border-color: var(--pink-border) !important;
  color: var(--pink-text) !important;
}

/* Menu */
.el-menu-item.is-active {
  color: var(--pink-accent) !important;
}

/* Dialog */
.el-dialog {
  border-radius: 14px !important;
  background-color: var(--el-bg-color-overlay) !important;
}

.el-dialog__title {
  font-size: 16px;
  font-weight: 600;
  color: var(--pink-text);
}

.el-dialog__body {
  font-size: 14px;
}

/* Input */
.el-input__wrapper {
  border-radius: 8px !important;
}

/* Tabs */
.el-tabs__item.is-active {
  color: var(--pink-accent) !important;
}

.el-tabs__active-bar {
  background-color: var(--pink-accent) !important;
}

/* Switch */
.el-switch.is-checked .el-switch__core {
  background-color: var(--pink-accent) !important;
  border-color: var(--pink-accent) !important;
}

/* MessageBox */
.el-message-box {
  border-radius: 14px !important;
}

.el-message-box__title {
  font-size: 16px;
  font-weight: 600;
}

.el-message-box__message {
  font-size: 14px;
}

/* Form */
.el-form-item__label {
  font-size: 13px;
  font-weight: 500;
  color: var(--pink-text);
}

/* Tag */
.el-tag {
  font-size: 12px;
  border-radius: 6px;
}

/* Input/Select font size */
.el-select .el-input__inner,
.el-input__inner {
  font-size: 13px;
}

/* Descriptions */
.el-descriptions__label {
  font-size: 13px;
  font-weight: 500;
}

.el-descriptions__content {
  font-size: 13px;
}

/* Popconfirm */
.el-popconfirm__main {
  font-size: 13px;
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-thumb {
  background: var(--theme-scrollbar);
  border-radius: 3px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

/* ===================== 暗色主题覆盖 ===================== */

html.theme-dark .el-card {
  background-color: var(--pink-surface) !important;
}

html.theme-dark .el-dialog {
  background-color: var(--el-bg-color-overlay) !important;
}

html.theme-dark .el-message-box {
  background-color: var(--el-bg-color-overlay) !important;
}

html.theme-dark .el-drawer {
  background-color: var(--el-bg-color-overlay) !important;
}

html.theme-dark .el-dropdown-menu {
  background-color: var(--el-bg-color-overlay) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-popper.is-light {
  background: var(--el-bg-color-overlay) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-popper.is-light .el-popper__arrow::before {
  background: var(--el-bg-color-overlay) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-input__wrapper {
  background-color: var(--el-fill-color-blank) !important;
  box-shadow: 0 0 0 1px var(--pink-border) inset !important;
}

html.theme-dark .el-input__wrapper:hover {
  box-shadow: 0 0 0 1px var(--pink-accent) inset !important;
}

html.theme-dark .el-input__wrapper.is-focus {
  box-shadow: 0 0 0 1px var(--pink-accent) inset !important;
}

html.theme-dark .el-textarea__inner {
  background-color: var(--el-fill-color-blank) !important;
  box-shadow: 0 0 0 1px var(--pink-border) inset !important;
}

html.theme-dark .el-select-dropdown {
  background-color: var(--el-bg-color-overlay) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-select-dropdown__item.is-hovering {
  background-color: var(--pink-accent-light) !important;
}

html.theme-dark .el-table {
  --el-table-bg-color: var(--pink-surface);
  --el-table-tr-bg-color: var(--pink-surface);
  --el-table-header-bg-color: var(--el-fill-color);
}

html.theme-dark .el-table th.el-table__cell {
  background-color: var(--el-fill-color) !important;
}

html.theme-dark .el-table td.el-table__cell {
  background-color: var(--pink-surface) !important;
}

html.theme-dark .el-pagination button,
html.theme-dark .el-pagination .el-pager li {
  background-color: var(--pink-surface) !important;
  color: var(--pink-text) !important;
}

html.theme-dark .el-pagination .el-pager li.is-active {
  background-color: var(--pink-accent) !important;
  color: #fff !important;
}

html.theme-dark .el-descriptions {
  --el-descriptions-item-bordered-label-background: var(--el-fill-color);
}

html.theme-dark .el-tag {
  background-color: var(--el-fill-color) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-tag--info {
  background-color: var(--el-fill-color) !important;
}

html.theme-dark .el-button--default {
  background-color: var(--pink-surface) !important;
  border-color: var(--pink-border) !important;
  color: var(--pink-text) !important;
}

html.theme-dark .el-button--default:hover,
html.theme-dark .el-button--default:focus {
  background-color: var(--pink-accent-light) !important;
  border-color: var(--pink-accent) !important;
  color: var(--pink-accent) !important;
}

html.theme-dark .el-form-item__label {
  color: var(--pink-text) !important;
}

html.theme-dark .el-tabs__nav-wrap::after {
  background-color: var(--pink-border) !important;
}

html.theme-dark .el-divider {
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-loading-mask {
  background-color: rgba(18, 20, 28, 0.8) !important;
}

html.theme-dark .el-empty__description p {
  color: var(--pink-text-secondary) !important;
}

html.theme-dark .el-popover.el-popper {
  background: var(--el-bg-color-overlay) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-popover.el-popper .el-popper__arrow::before {
  background: var(--el-bg-color-overlay) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-collapse-item__header,
html.theme-dark .el-collapse-item__wrap {
  background-color: var(--pink-surface) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-date-editor .el-input__wrapper,
html.theme-dark .el-date-range-picker {
  background-color: var(--el-fill-color-blank) !important;
}

html.theme-dark .el-picker-panel {
  background-color: var(--el-bg-color-overlay) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-date-table td.current:not(.disabled) .el-date-table-cell__text {
  background-color: var(--pink-accent) !important;
}

html.theme-dark .el-notification {
  background-color: var(--el-bg-color-overlay) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-message {
  background-color: var(--el-bg-color-overlay) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-checkbox__inner {
  background-color: var(--el-fill-color-blank) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-checkbox__input.is-checked .el-checkbox__inner {
  background-color: var(--pink-accent) !important;
  border-color: var(--pink-accent) !important;
}

html.theme-dark .el-radio__inner {
  background-color: var(--el-fill-color-blank) !important;
  border-color: var(--pink-border) !important;
}

html.theme-dark .el-radio__input.is-checked .el-radio__inner {
  background-color: var(--pink-accent) !important;
  border-color: var(--pink-accent) !important;
}

html.theme-dark .el-progress__text {
  color: var(--pink-text) !important;
}
</style>
