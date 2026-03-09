<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'

use([CanvasRenderer, LineChart, PieChart, GridComponent, TooltipComponent, LegendComponent])

import { dataSummaryApi, dashboardApi, type DataSummaryResponse } from '@/api'
import {
  ElRow,
  ElCol,
  ElCard,
  ElSelect,
  ElOption,
  ElTable,
  ElTableColumn,
  ElButton,
  ElTag,
  ElIcon,
  ElEmpty
} from 'element-plus'
import {
  Coin,
  Refresh,
  Sunny,
  Trophy
} from '@element-plus/icons-vue'

interface AccountOption {
  id: number
  name: string
  status: string
}

const selectedAccountId = ref<number | null>(null)
const accounts = ref<AccountOption[]>([])
const data = ref<DataSummaryResponse | null>(null)
const loading = ref(false)
let refreshInterval: number | null = null

const fetchAccounts = async () => {
  try {
    const response = await dashboardApi.getStats()
    accounts.value = response.data.accounts.map(acc => ({
      id: acc.id,
      name: acc.name,
      status: acc.status
    }))
    const firstRunning = accounts.value.find(acc => acc.status === 'running')
    if (firstRunning && !selectedAccountId.value) {
      selectedAccountId.value = firstRunning.id
      await fetchData()
    }
  } catch (err) {
    console.error('Failed to fetch accounts:', err)
  }
}

const fetchData = async () => {
  if (!selectedAccountId.value) return
  loading.value = true
  try {
    const response = await dataSummaryApi.get(selectedAccountId.value, 24, 7)
    data.value = response.data
  } catch (err) {
    console.error('Failed to fetch data summary:', err)
  } finally {
    loading.value = false
  }
}

const handleAccountChange = () => {
  if (selectedAccountId.value) {
    fetchData()
  }
}

const handleRefresh = () => {
  fetchData()
}

const formatGold = (value: number): string => {
  return value.toLocaleString()
}

const formatGoldShort = (value: number): string => {
  if (value >= 10000) {
    return `${(value / 10000).toFixed(1)}万`
  }
  return value.toString()
}

// Helper to get CSS variable values
const getCSSVar = (name: string): string => {
  if (typeof window !== 'undefined') {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  }
  return ''
}

// Line chart for gold trend
const trendChartOption = computed(() => {
  if (!data.value?.hourly_trend) return {}

  const hourlyTrend = data.value.hourly_trend
  const hours = hourlyTrend.map(t => t.hour)
  const harvestGold = hourlyTrend.map(t => t.harvest_gold)
  const stealGold = hourlyTrend.map(t => t.steal_gold)

  const textSecondary = getCSSVar('--text-secondary') || '#9CA3AF'
  const bgElevated = getCSSVar('--bg-elevated') || '#252540'
  const border = getCSSVar('--border') || '#2a2a3e'
  const textPrimary = getCSSVar('--text-primary') || '#E5E7EB'
  const success = getCSSVar('--success') || '#22C55E'
  const danger = getCSSVar('--danger') || '#EF4444'

  return {
    backgroundColor: 'transparent',
    textStyle: { color: textSecondary },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross'
      },
      backgroundColor: bgElevated,
      borderColor: border,
      textStyle: { color: textPrimary },
      formatter: (params: { axisValue: string; marker: string; seriesName: string; value: number }[]) => {
        let result = `${params[0].axisValue}<br/>`
        params.forEach(item => {
          result += `${item.marker} ${item.seriesName}: ${item.value.toLocaleString()} 金币<br/>`
        })
        return result
      }
    },
    legend: {
      data: ['自己收获', '偷菜收益'],
      bottom: 0,
      textStyle: { color: textSecondary }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      top: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: hours,
      axisLabel: {
        color: textSecondary,
        rotate: 45
      },
      axisLine: {
        lineStyle: { color: border }
      }
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        color: textSecondary,
        formatter: (value: number) => formatGoldShort(value)
      },
      splitLine: {
        lineStyle: {
          color: border
        }
      },
      axisLine: {
        lineStyle: { color: border }
      }
    },
    series: [
      {
        name: '自己收获',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(34, 197, 94, 0.25)' },
              { offset: 1, color: 'rgba(34, 197, 94, 0.02)' }
            ]
          }
        },
        lineStyle: {
          color: success,
          width: 2
        },
        itemStyle: {
          color: success
        },
        data: harvestGold
      },
      {
        name: '偷菜收益',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(239, 68, 68, 0.25)' },
              { offset: 1, color: 'rgba(239, 68, 68, 0.02)' }
            ]
          }
        },
        lineStyle: {
          color: danger,
          width: 2
        },
        itemStyle: {
          color: danger
        },
        data: stealGold
      }
    ]
  }
})

// Donut chart for crop breakdown
const cropChartOption = computed(() => {
  if (!data.value?.crop_breakdown) return {}

  const breakdown = [...data.value.crop_breakdown]
  breakdown.sort((a, b) => b.gold - a.gold)

  const top5 = breakdown.slice(0, 5)
  const others = breakdown.slice(5)
  const othersTotal = others.reduce((sum, item) => sum + item.gold, 0)

  const chartData: Array<{ name: string; value: number }> = top5.map(item => ({
    name: item.name,
    value: item.gold
  }))

  if (othersTotal > 0) {
    chartData.push({ name: '其他', value: othersTotal })
  }

  const primary = getCSSVar('--primary') || '#3B82F6'
  const success = getCSSVar('--success') || '#22C55E'
  const gold = getCSSVar('--gold') || '#EAB308'
  const primaryLight = getCSSVar('--primary-light') || '#60A5FA'
  const warning = getCSSVar('--warning') || '#FBBF24'

  const colors = [primary, success, '#14B8A6', gold, primaryLight, warning]

  const textSecondary = getCSSVar('--text-secondary') || '#9CA3AF'
  const bgElevated = getCSSVar('--bg-elevated') || '#252540'
  const border = getCSSVar('--border') || '#2a2a3e'
  const textPrimary = getCSSVar('--text-primary') || '#E5E7EB'
  const bgCard = getCSSVar('--bg-card') || '#1a1a2e'

  return {
    backgroundColor: 'transparent',
    textStyle: { color: textSecondary },
    tooltip: {
      trigger: 'item',
      backgroundColor: bgElevated,
      borderColor: border,
      textStyle: { color: textPrimary },
      formatter: (params: { name: string; value: number; percent: number }) => {
        return `${params.name}<br/>${formatGold(params.value)} 金币 (${params.percent}%)`
      }
    },
    legend: {
      orient: 'vertical',
      right: '5%',
      top: 'center',
      textStyle: { color: textSecondary }
    },
    color: colors,
    series: [
      {
        name: '作物收益',
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 6,
          borderColor: bgCard,
          borderWidth: 2
        },
        label: {
          show: false
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 14,
            fontWeight: 'bold',
            color: textPrimary
          }
        },
        labelLine: {
          show: false
        },
        data: chartData
      }
    ]
  }
})

// Reversed hourly trend for table (newest first)
const reversedHourlyTrend = computed(() => {
  if (!data.value?.hourly_trend) return []
  return [...data.value.hourly_trend].reverse()
})

onMounted(() => {
  fetchAccounts()
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
  refreshInterval = window.setInterval(() => {
    if (selectedAccountId.value) {
      fetchData()
    }
  }, 30000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <div class="data-summary-view">
    <!-- Page Header -->
    <ElCard shadow="never" class="header-card">
      <div class="page-header">
        <div class="header-left">
          <h1 class="page-title">数据汇总</h1>
          <p class="page-subtitle">查看农场数据，过去24小时您的农场经营明细</p>
        </div>
        <div class="header-right">
          <ElSelect
            v-model="selectedAccountId"
            placeholder="选择账号"
            class="account-select"
            @change="handleAccountChange"
          >
            <ElOption
              v-for="account in accounts"
              :key="account.id"
              :label="account.name"
              :value="account.id"
            >
              <span>{{ account.name }}</span>
              <ElTag
                :type="account.status === 'running' ? 'success' : 'info'"
                size="small"
                class="status-tag"
              >
                {{ account.status === 'running' ? '运行中' : '已停止' }}
              </ElTag>
            </ElOption>
          </ElSelect>
          <ElButton type="primary" :icon="Refresh" @click="handleRefresh" :loading="loading">
            刷新数据
          </ElButton>
        </div>
      </div>
    </ElCard>

    <!-- Summary Cards -->
    <div class="summary-cards" v-if="data">
      <div class="summary-card card-green">
        <div class="card-icon">
          <ElIcon :size="24"><Sunny /></ElIcon>
        </div>
        <div class="card-content">
          <div class="card-label">累计收获数量</div>
          <div class="card-value">{{ formatGold(data.summary.total_harvest_count) }}</div>
        </div>
      </div>
      <div class="summary-card card-gold">
        <div class="card-icon">
          <ElIcon :size="24"><Coin /></ElIcon>
        </div>
        <div class="card-content">
          <div class="card-label">收获总收益</div>
          <div class="card-value">{{ formatGold(data.summary.total_harvest_gold) }}</div>
          <div class="card-unit">金币</div>
        </div>
      </div>
      <div class="summary-card card-red">
        <div class="card-icon">
          <ElIcon :size="24"><Coin /></ElIcon>
        </div>
        <div class="card-content">
          <div class="card-label">累计偷菜数量</div>
          <div class="card-value">{{ formatGold(data.summary.total_steal_count) }}</div>
        </div>
      </div>
      <div class="summary-card card-gold">
        <div class="card-icon">
          <ElIcon :size="24"><Trophy /></ElIcon>
        </div>
        <div class="card-content">
          <div class="card-label">偷菜总收益</div>
          <div class="card-value">{{ formatGold(data.summary.total_steal_gold) }}</div>
          <div class="card-unit">金币</div>
        </div>
      </div>
    </div>

    <!-- Charts Row -->
    <ElRow :gutter="16" class="charts-row" v-if="data">
      <ElCol :xs="24" :md="14">
        <ElCard shadow="never" class="chart-card">
          <template #header>
            <div class="card-header">
              <span class="header-title">收益趋势图</span>
            </div>
          </template>
          <VChart
            :option="trendChartOption"
            autoresize
            style="height: 350px; width: 100%;"
          />
        </ElCard>
      </ElCol>
      <ElCol :xs="24" :md="10">
        <ElCard shadow="never" class="chart-card">
          <template #header>
            <div class="card-header">
              <span class="header-title">作物收益占比</span>
            </div>
          </template>
          <VChart
            :option="cropChartOption"
            autoresize
            style="height: 350px; width: 100%;"
          />
        </ElCard>
      </ElCol>
    </ElRow>

    <!-- Tables Row -->
    <ElRow :gutter="16" class="tables-row" v-if="data">
      <ElCol :xs="24" :md="12">
        <ElCard shadow="never" class="table-card">
          <template #header>
            <div class="card-header">
              <span class="header-title">偷菜排行</span>
            </div>
          </template>
          <ElTable :data="data.steal_ranking" stripe style="width: 100%" max-height="300">
            <ElTableColumn label="排名" width="70" align="center">
              <template #default="{ $index }">
                <span class="rank-number">{{ $index + 1 }}</span>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="friend_name" label="好友" min-width="100" />
            <ElTableColumn prop="steal_count" label="偷取(个)" width="100" align="right">
              <template #default="{ row }">
                {{ formatGold(row.steal_count) }}
              </template>
            </ElTableColumn>
            <ElTableColumn prop="steal_gold" label="偷取价值" width="100" align="right">
              <template #default="{ row }">
                {{ formatGold(row.steal_gold) }}
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </ElCol>
      <ElCol :xs="24" :md="12">
        <ElCard shadow="never" class="table-card">
          <template #header>
            <div class="card-header">
              <span class="header-title">分时详细记录</span>
            </div>
          </template>
          <ElTable :data="reversedHourlyTrend" stripe style="width: 100%" max-height="300">
            <ElTableColumn prop="hour" label="时间" width="100" />
            <ElTableColumn prop="harvest_count" label="收获数" width="80" align="right">
              <template #default="{ row }">
                {{ formatGold(row.harvest_count) }}
              </template>
            </ElTableColumn>
            <ElTableColumn prop="harvest_gold" label="收获收益" width="90" align="right">
              <template #default="{ row }">
                {{ formatGold(row.harvest_gold) }}
              </template>
            </ElTableColumn>
            <ElTableColumn prop="steal_count" label="偷菜数" width="80" align="right">
              <template #default="{ row }">
                {{ formatGold(row.steal_count) }}
              </template>
            </ElTableColumn>
            <ElTableColumn prop="steal_gold" label="偷菜收益" width="90" align="right">
              <template #default="{ row }">
                {{ formatGold(row.steal_gold) }}
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </ElCol>
    </ElRow>

    <!-- Daily Summary Table -->
    <ElCard shadow="never" class="daily-table-card" v-if="data && data.daily_summary.length > 0">
      <template #header>
        <div class="card-header">
          <span class="header-title">每日数据汇总（最近7天）</span>
        </div>
      </template>
      <ElTable :data="data.daily_summary" stripe style="width: 100%">
        <ElTableColumn prop="date" label="日期" width="120" />
        <ElTableColumn prop="harvest_count" label="收获数量" width="120" align="right">
          <template #default="{ row }">
            {{ formatGold(row.harvest_count) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="harvest_gold" label="收获收益" width="120" align="right">
          <template #default="{ row }">
            {{ formatGold(row.harvest_gold) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="steal_count" label="偷取数量" width="120" align="right">
          <template #default="{ row }">
            {{ formatGold(row.steal_count) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="steal_gold" label="偷菜收益" width="120" align="right">
          <template #default="{ row }">
            {{ formatGold(row.steal_gold) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="total_gold" label="当日总计收益" min-width="120" align="right">
          <template #default="{ row }">
            <span class="total-gold">{{ formatGold(row.total_gold) }}</span>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>

    <ElEmpty v-if="!data && !loading" description="暂无数据，请先选择账号" class="empty-state" />
  </div>
</template>

<style scoped>
.data-summary-view {
  padding: 0;
}

/* Header Card */
.header-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
}

.header-card :deep(.el-card__body) {
  padding: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
}

.header-left {
  flex: 1;
  min-width: 200px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-heading);
  margin: 0 0 8px 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.account-select {
  width: 200px;
}

.status-tag {
  margin-left: 8px;
  border-radius: var(--radius-sm);
}

/* Summary Cards Grid */
.summary-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-top: 16px;
}

.summary-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
}

.card-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.card-green .card-icon {
  background: var(--success-bg);
  color: var(--success);
}

.card-green .card-value {
  color: var(--success);
}

.card-gold .card-icon {
  background: var(--gold-bg);
  color: var(--gold);
}

.card-gold .card-value {
  color: var(--gold);
}

.card-red .card-icon {
  background: var(--danger-bg);
  color: var(--danger);
}

.card-red .card-value {
  color: var(--danger);
}

.card-content {
  flex: 1;
}

.card-label {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.card-value {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.2;
}

.card-unit {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 4px;
}

/* Charts Row */
.charts-row {
  margin-top: 16px;
}

.chart-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
  margin-bottom: 16px;
}

.chart-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.chart-card :deep(.el-card__body) {
  padding: 24px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-heading);
}

/* Tables Row */
.tables-row {
  margin-top: 0;
}

.table-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
  margin-bottom: 16px;
}

.table-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.table-card :deep(.el-card__body) {
  padding: 0;
}

.table-card :deep(.el-table) {
  --el-table-border-color: var(--border);
  --el-table-bg-color: var(--bg-card);
  --el-table-header-bg-color: var(--bg-elevated);
  --el-table-header-text-color: var(--text-secondary);
  --el-table-text-color: var(--text-primary);
  --el-table-row-hover-bg-color: var(--bg-elevated);
  --el-table-tr-bg-color: var(--bg-card);
}

.table-card :deep(.el-table__header th) {
  font-weight: 600;
  font-size: 13px;
  padding: 12px 0;
  background: var(--bg-elevated);
}

.table-card :deep(.el-table__body td) {
  padding: 10px 8px;
}

.table-card :deep(.el-table__row--striped) {
  background: var(--bg-hover);
}

.rank-number {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  font-weight: 600;
  font-size: 13px;
}

/* Daily Table Card */
.daily-table-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
  margin-top: 0;
}

.daily-table-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.daily-table-card :deep(.el-card__body) {
  padding: 0;
}

.daily-table-card :deep(.el-table) {
  --el-table-border-color: var(--border);
  --el-table-bg-color: var(--bg-card);
  --el-table-header-bg-color: var(--bg-elevated);
  --el-table-header-text-color: var(--text-secondary);
  --el-table-text-color: var(--text-primary);
  --el-table-row-hover-bg-color: var(--bg-elevated);
  --el-table-tr-bg-color: var(--bg-card);
}

.daily-table-card :deep(.el-table__header th) {
  font-weight: 600;
  font-size: 13px;
  padding: 12px 0;
  background: var(--bg-elevated);
}

.daily-table-card :deep(.el-table__body td) {
  padding: 10px 8px;
}

.daily-table-card :deep(.el-table__row--striped) {
  background: var(--bg-hover);
}

.total-gold {
  color: var(--gold);
  font-weight: 600;
}

/* Empty State */
.empty-state {
  padding: 48px;
  margin-top: 24px;
}

:deep(.el-empty__description) {
  color: var(--text-muted) !important;
}

/* Mobile Responsive */
@media (max-width: 1200px) {
  .summary-cards {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .header-right {
    width: 100%;
    flex-direction: column;
  }

  .account-select {
    width: 100%;
  }

  .header-right .el-button {
    width: 100%;
  }

  .summary-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .charts-row .el-col,
  .tables-row .el-col {
    margin-bottom: 16px;
  }
}

@media (max-width: 480px) {
  .summary-cards {
    grid-template-columns: 1fr;
  }

  .summary-card {
    padding: 16px;
  }

  .card-value {
    font-size: 24px;
  }
}
</style>