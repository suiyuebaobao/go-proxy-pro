<!--
 * 文件作用：管理员数据概览页面，展示系统关键指标的可视化图表
 * 负责功能：
 *   - 汇总卡片（今日消费、请求数、Token 用量、活跃模型数）
 *   - 每日消费趋势折线图
 *   - 模型使用分布饼图
 *   - 每日请求量柱状图
 *   - 小时级请求趋势折线图
 *   - Token 使用趋势面积图
 * 重要程度：⭐⭐⭐⭐ 重要（管理员首页入口）
 * 依赖组件：vue-echarts, Element Plus, api/index.js
-->
<template>
  <div class="overview-page">
    <div class="page-header">
      <h2>数据概览</h2>
      <div class="header-actions">
        <el-select v-model="dayRange" style="width: 120px" @change="refreshAll">
          <el-option :value="7" label="最近 7 天" />
          <el-option :value="14" label="最近 14 天" />
          <el-option :value="30" label="最近 30 天" />
          <el-option :value="90" label="最近 90 天" />
        </el-select>
        <el-button @click="refreshAll"><el-icon><Refresh /></el-icon> 刷新</el-button>
      </div>
    </div>

    <!-- 汇总卡片 -->
    <el-row :gutter="16" class="summary-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #667eea, #764ba2)">
            <el-icon :size="24"><Document /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ formatNum(totalSummary.total_requests) }}</div>
            <div class="stat-label">总请求数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #f093fb, #f5576c)">
            <el-icon :size="24"><Cpu /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ formatTokens(totalSummary.total_tokens) }}</div>
            <div class="stat-label">总 Token</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #43e97b, #38f9d7)">
            <el-icon :size="24"><Money /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">${{ (totalSummary.total_cost || 0).toFixed(2) }}</div>
            <div class="stat-label">总费用</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon" style="background: linear-gradient(135deg, #fa709a, #fee140)">
            <el-icon :size="24"><DataAnalysis /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ modelDistribution.length }}</div>
            <div class="stat-label">活跃模型</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 图表区域 -->
    <el-row :gutter="16" class="chart-row">
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header><span class="card-title">每日消费趋势</span></template>
          <div ref="costChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header><span class="card-title">模型使用分布</span></template>
          <div ref="pieChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="chart-row">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header><span class="card-title">每日请求量</span></template>
          <div ref="requestChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header><span class="card-title">24 小时请求趋势</span></template>
          <div ref="hourlyChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="chart-row">
      <el-col :span="24">
        <el-card shadow="hover">
          <template #header><span class="card-title">Token 使用趋势（输入 / 输出）</span></template>
          <div ref="tokenChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import * as echarts from 'echarts/core'
import { LineChart, BarChart, PieChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, TitleComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import api from '@/api'

echarts.use([LineChart, BarChart, PieChart, GridComponent, TooltipComponent, LegendComponent, TitleComponent, CanvasRenderer])

const dayRange = ref(30)

const totalSummary = ref({ total_requests: 0, total_tokens: 0, total_cost: 0 })
const dailyTrend = ref([])
const modelDistribution = ref([])
const hourlyTrend = ref([])

const costChartRef = ref(null)
const pieChartRef = ref(null)
const requestChartRef = ref(null)
const hourlyChartRef = ref(null)
const tokenChartRef = ref(null)

let charts = []

function getStyle(prop) {
  return getComputedStyle(document.documentElement).getPropertyValue(prop).trim()
}

function chartColors() {
  const accent = getStyle('--pink-accent') || '#c97b8b'
  return {
    accent,
    textColor: getStyle('--pink-text') || '#4a3347',
    borderColor: getStyle('--pink-border') || '#f0e0e6',
    bg: getStyle('--pink-bg') || '#fdf8f9',
    palette: [accent, '#5470c6', '#91cc75', '#fac858', '#ee6666', '#73c0de', '#3ba272', '#fc8452', '#9a60b4', '#ea7ccc']
  }
}

function formatNum(n) {
  if (!n) return '0'
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return n
}

function formatTokens(t) {
  if (!t) return '0'
  if (t >= 1000000000) return (t / 1000000000).toFixed(1) + 'B'
  if (t >= 1000000) return (t / 1000000).toFixed(1) + 'M'
  if (t >= 1000) return (t / 1000).toFixed(1) + 'K'
  return t
}

function initChart(el) {
  if (!el) return null
  const c = echarts.init(el)
  charts.push(c)
  return c
}

function renderCostChart() {
  const c = initChart(costChartRef.value)
  if (!c) return
  if (dailyTrend.value.length === 0) { c.setOption({ graphic: { type: 'text', left: 'center', top: 'middle', style: { text: '暂无数据', fontSize: 14, fill: '#999' } } }); return }
  const colors = chartColors()
  const dates = dailyTrend.value.map(d => d.date)
  const costs = dailyTrend.value.map(d => d.total_cost || 0)

  c.setOption({
    tooltip: { trigger: 'axis', formatter: params => `${params[0].name}<br/>费用: $${params[0].value.toFixed(4)}` },
    grid: { left: 60, right: 20, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: dates, axisLabel: { color: colors.textColor, fontSize: 11 }, axisLine: { lineStyle: { color: colors.borderColor } } },
    yAxis: { type: 'value', axisLabel: { color: colors.textColor, formatter: v => '$' + v.toFixed(2) }, splitLine: { lineStyle: { color: colors.borderColor } } },
    series: [{ type: 'line', data: costs, smooth: true, areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: colors.accent + '40' }, { offset: 1, color: colors.accent + '05' }]) }, lineStyle: { color: colors.accent, width: 2 }, itemStyle: { color: colors.accent } }]
  })
}

function renderPieChart() {
  const c = initChart(pieChartRef.value)
  if (!c) return
  if (modelDistribution.value.length === 0) { c.setOption({ graphic: { type: 'text', left: 'center', top: 'middle', style: { text: '暂无数据', fontSize: 14, fill: '#999' } } }); return }
  const colors = chartColors()
  const data = modelDistribution.value.slice(0, 10).map(m => ({ name: m.model, value: m.request_count }))

  c.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { type: 'scroll', bottom: 0, textStyle: { color: colors.textColor, fontSize: 11 } },
    color: colors.palette,
    series: [{ type: 'pie', radius: ['35%', '65%'], center: ['50%', '45%'], data, label: { show: false }, emphasis: { label: { show: true, fontSize: 13 } } }]
  })
}

function renderRequestChart() {
  const c = initChart(requestChartRef.value)
  if (!c) return
  const colors = chartColors()
  const dates = dailyTrend.value.map(d => d.date)
  const counts = dailyTrend.value.map(d => d.request_count || 0)

  c.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 50, right: 20, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: dates, axisLabel: { color: colors.textColor, fontSize: 11 }, axisLine: { lineStyle: { color: colors.borderColor } } },
    yAxis: { type: 'value', axisLabel: { color: colors.textColor }, splitLine: { lineStyle: { color: colors.borderColor } } },
    series: [{ type: 'bar', data: counts, itemStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: '#5470c6' }, { offset: 1, color: '#5470c640' }]), borderRadius: [4, 4, 0, 0] } }]
  })
}

function renderHourlyChart() {
  const c = initChart(hourlyChartRef.value)
  if (!c) return
  const colors = chartColors()
  const hours = hourlyTrend.value.map(d => d.hour?.substring(11, 16) || d.hour)
  const counts = hourlyTrend.value.map(d => d.request_count || 0)

  c.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 50, right: 20, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: hours, axisLabel: { color: colors.textColor, fontSize: 10, rotate: 45 }, axisLine: { lineStyle: { color: colors.borderColor } } },
    yAxis: { type: 'value', axisLabel: { color: colors.textColor }, splitLine: { lineStyle: { color: colors.borderColor } } },
    series: [{ type: 'line', data: counts, smooth: true, areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: '#91cc7540' }, { offset: 1, color: '#91cc7505' }]) }, lineStyle: { color: '#91cc75', width: 2 }, itemStyle: { color: '#91cc75' } }]
  })
}

function renderTokenChart() {
  const c = initChart(tokenChartRef.value)
  if (!c) return
  const colors = chartColors()
  const dates = dailyTrend.value.map(d => d.date)

  c.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['总 Token'], textStyle: { color: colors.textColor } },
    grid: { left: 70, right: 20, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: dates, axisLabel: { color: colors.textColor, fontSize: 11 }, axisLine: { lineStyle: { color: colors.borderColor } } },
    yAxis: { type: 'value', axisLabel: { color: colors.textColor, formatter: v => { if (v >= 1000000) return (v / 1000000).toFixed(0) + 'M'; if (v >= 1000) return (v / 1000).toFixed(0) + 'K'; return v } }, splitLine: { lineStyle: { color: colors.borderColor } } },
    series: [
      { name: '总 Token', type: 'line', data: dailyTrend.value.map(d => d.total_tokens || 0), smooth: true, areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: '#ee666640' }, { offset: 1, color: '#ee666605' }]) }, lineStyle: { color: '#ee6666', width: 2 }, itemStyle: { color: '#ee6666' } }
    ]
  })
}

async function fetchData() {
  try {
    const [trendRes, distRes, hourlyRes] = await Promise.all([
      api.getDailyTrend({ days: dayRange.value }),
      api.getModelDistribution({ days: dayRange.value }),
      api.getHourlyTrend()
    ])

    const trendItems = trendRes.data?.items || []
    dailyTrend.value = trendItems.reverse()
    modelDistribution.value = distRes.data?.items || []
    hourlyTrend.value = hourlyRes.data?.items || []

    let totalReq = 0, totalTok = 0, totalCost = 0
    for (const d of trendItems) {
      totalReq += d.request_count || 0
      totalTok += d.total_tokens || 0
      totalCost += d.total_cost || 0
    }
    totalSummary.value = { total_requests: totalReq, total_tokens: totalTok, total_cost: totalCost }
  } catch (e) {
    console.error('Failed to fetch overview data:', e)
  }
}

function renderCharts() {
  charts.forEach(c => c.dispose())
  charts = []
  renderCostChart()
  renderPieChart()
  renderRequestChart()
  renderHourlyChart()
  renderTokenChart()
}

async function refreshAll() {
  await fetchData()
  await nextTick()
  renderCharts()
}

let resizeHandler
onMounted(async () => {
  await fetchData()
  await nextTick()
  renderCharts()
  resizeHandler = () => charts.forEach(c => c.resize())
  window.addEventListener('resize', resizeHandler)
})

onUnmounted(() => {
  charts.forEach(c => c.dispose())
  charts = []
  if (resizeHandler) window.removeEventListener('resize', resizeHandler)
})
</script>

<style scoped>
.overview-page {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  color: var(--pink-text);
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.summary-row {
  margin-bottom: 16px;
}

.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
}

.stat-icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--pink-text, #4a3347);
  line-height: 1.2;
}

.stat-label {
  font-size: 13px;
  color: var(--pink-text-secondary, #8b7d92);
  margin-top: 4px;
}

.chart-row {
  margin-bottom: 16px;
}

.chart-container {
  height: 320px;
  width: 100%;
}

.card-title {
  font-weight: 600;
  color: var(--pink-text, #4a3347);
}
</style>
