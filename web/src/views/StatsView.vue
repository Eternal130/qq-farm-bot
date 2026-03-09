<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, DataZoomComponent } from 'echarts/components'

use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent, DataZoomComponent])

import { dashboardApi, statsApi, type StatsResponse } from '@/api'
import {
  ElRow,
  ElCol,
  ElCard,
  ElSelect,
  ElOption,
  ElDatePicker,
  ElRadioGroup,
  ElRadioButton,
  ElTable,
  ElTableColumn,
  ElStatistic,
  ElTag,
  ElIcon,
  ElEmpty
} from 'element-plus'
import {
  Coin,
  TrendCharts
} from '@element-plus/icons-vue'


const opTypeNames: Record<string, string> = {
  harvest: '收获',
  plant: '种植',
  buy_seed: '购买种子',
  sell: '出售',
  weed: '除草',
  bug: '除虫',
  water: '浇水',
  fertilize: '施肥',
  steal: '偷菜',
  help_weed: '帮除草',
  help_bug: '帮除虫',
  help_water: '帮浇水',
  task_claim: '领取任务',
  fert_buy: '购买化肥',
  fert_open: '开启化肥',
  fert_use: '使用化肥',
  unlock_land: '解锁土地',
  upgrade_land: '升级土地'
}

interface AccountWithUptime {
  id: number
  name: string
  status: string
  uptime_seconds: number
  started_at: string | null
}

const selectedAccountId = ref<number | null>(null)
const dateRange = ref<[Date, Date] | null>(null)
const granularity = ref<string>('day')
const timePreset = ref<string>('')
const stats = ref<StatsResponse | null>(null)
const loading = ref(false)
const accounts = ref<AccountWithUptime[]>([])
let refreshInterval: number | null = null

const timePresets = [
  { label: '今天', value: 'today' },
  { label: '近7天', value: '7days' },
  { label: '近30天', value: '30days' },
  { label: '全部', value: 'all' }
] as const

const formatDateTime = (dateStr: string): string => {
  return new Date(dateStr).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatUptime = (seconds: number): string => {
  if (seconds <= 0) return '未运行'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (days > 0) parts.push(`${days}天`)
  if (hours > 0) parts.push(`${hours}小时`)
  if (mins > 0) parts.push(`${mins}分钟`)
  return parts.length > 0 ? parts.join(' ') : '不到1分钟'
}

const getSelectedAccount = computed(() => {
  return accounts.value.find(a => a.id === selectedAccountId.value)
})

const getSelectedAccountUptime = computed(() => {
  const acc = getSelectedAccount.value
  if (!acc) return null
  return {
    uptime: formatUptime(acc.uptime_seconds),
    startedAt: acc.started_at ? formatDateTime(acc.started_at) : null
  }
})

const uptime = computed(() => {
  const acc = getSelectedAccount.value
  if (!acc) return { uptime: '未运行', startedAt: '' }
  return {
    uptime: formatUptime(acc.uptime_seconds || 0),
    startedAt: acc.started_at ? formatDateTime(acc.started_at) : ''
  }
})

const fetchAccounts = async () => {
  try {
    const response = await dashboardApi.getStats()
    accounts.value = response.data.accounts.map(acc => ({
      id: acc.id,
      name: acc.name,
      status: acc.status,
      uptime_seconds: acc.uptime_seconds || 0,
      started_at: acc.started_at
    }))
    // Auto-select first running account
    const firstRunning = accounts.value.find(acc => acc.status === 'running')
    if (firstRunning && !selectedAccountId.value) {
      selectedAccountId.value = firstRunning.id
    }
  } catch (err) {
    console.error('Failed to fetch accounts:', err)
  }
}

const fetchStats = async () => {
  if (!selectedAccountId.value) return
  loading.value = true
  try {
    const params: Record<string, string> = { granularity: granularity.value }
    if (dateRange.value && dateRange.value[0] && dateRange.value[1]) {
      params.from = dateRange.value[0].toISOString()
      params.to = dateRange.value[1].toISOString()
    }
    const response = await statsApi.getStats(
      selectedAccountId.value,
      params.granularity,
      params.from,
      params.to
    )
    stats.value = response.data
  } catch (err) {
    console.error('Failed to fetch stats:', err)
  } finally {
    loading.value = false
  }
}

const handleTimePresetChange = (preset: string | number | boolean | undefined) => {
  if (preset === 'today') {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    dateRange.value = [today, new Date(today.getTime() + 24 * 3600 * 1000)]
  } else if (preset === '7days') {
    const endDate = new Date()
    const startDate = new Date()
    startDate.setDate(startDate.getDate() - 7)
    dateRange.value = [startDate, endDate]
  } else if (preset === '30days') {
    const endDate = new Date()
    const startDate = new Date()
    startDate.setDate(startDate.getDate() - 30)
    dateRange.value = [startDate, endDate]
  } else {
    dateRange.value = null
  }
  fetchStats()
}

const handleGranularityChange = (_value: string | number | boolean | undefined) => {
  fetchStats()
}

const getTotalOpCount = (opType: string): number => {
  return stats.value?.summary.op_counts[opType] || 0
}

const handleAccountChange = () => {
  if (selectedAccountId.value) {
    fetchStats()
  }
}

// Chart Options - Using CSS variable values via getComputedStyle
const getCSSVar = (name: string): string => {
  if (typeof window !== 'undefined') {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  }
  return ''
}

const goldTrendChartOption = computed(() => {
  if (!stats.value?.timeline) return {}
  
  const timeline = stats.value.timeline
  const periods = timeline.map(t => t.period)
  const goldIn = timeline.map(t => t.gold_in)
  const goldOut = timeline.map(t => t.gold_out)
  
  const textSecondary = getCSSVar('--text-secondary') || '#9CA3AF'
  const bgElevated = getCSSVar('--bg-elevated') || '#252540'
  const border = getCSSVar('--border') || '#2a2a3e'
  const textPrimary = getCSSVar('--text-primary') || '#E5E7EB'
  const bgCard = getCSSVar('--bg-card') || '#1a1a2e'
  const gold = getCSSVar('--gold') || '#EAB308'
  const danger = getCSSVar('--danger') || '#EF4444'
  const primary = getCSSVar('--primary') || '#3B82F6'
  
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
      formatter: (params: any[]) => {
        let result = `${params[0].axisValue}<br/>`
        params.forEach(item => {
          result += `${item.marker} ${item.seriesName}: ${item.value.toLocaleString()}<br/>`
        })
        return result
      }
    },
    legend: {
      data: ['金币收入', '金币支出'],
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
      data: periods,
      axisLabel: {
        color: textSecondary
      },
      axisLine: {
        lineStyle: { color: border }
      }
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        color: textSecondary,
        formatter: (value: number) => {
          if (value >= 10000) return `${(value / 10000).toFixed(1)}万`
          return value.toString()
        }
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
    dataZoom: [
      {
        type: 'slider',
        start: 0,
        end: 100,
        bottom: '5%',
        height: 20,
        borderColor: border,
        backgroundColor: bgCard,
        fillerColor: 'rgba(59, 130, 246, 0.2)',
        handleStyle: { color: primary },
        textStyle: { color: textSecondary }
      }
    ],
    series: [
      {
        name: '金币收入',
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
              { offset: 0, color: 'rgba(234, 179, 8, 0.25)' },
              { offset: 1, color: 'rgba(234, 179, 8, 0.02)' }
            ]
          }
        },
        lineStyle: {
          color: gold,
          width: 2
        },
        itemStyle: {
          color: gold
        },
        data: goldIn
      },
      {
        name: '金币支出',
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
        data: goldOut
      }
    ]
  }
})

const expTrendChartOption = computed(() => {
  if (!stats.value?.timeline) return {}
  
  const timeline = stats.value.timeline
  const periods = timeline.map(t => t.period)
  const expGained = timeline.map(t => t.exp_gained)
  
  const textSecondary = getCSSVar('--text-secondary') || '#9CA3AF'
  const bgElevated = getCSSVar('--bg-elevated') || '#252540'
  const border = getCSSVar('--border') || '#2a2a3e'
  const textPrimary = getCSSVar('--text-primary') || '#E5E7EB'
  const bgCard = getCSSVar('--bg-card') || '#1a1a2e'
  const success = getCSSVar('--success') || '#22C55E'
  const primary = getCSSVar('--primary') || '#3B82F6'
  
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
      formatter: (params: any) => {
        return `${params.axisValue}<br/>${params.marker} ${params.seriesName}: ${params.value.toLocaleString()}`
      }
    },
    legend: {
      data: ['经验获得'],
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
      data: periods,
      axisLabel: {
        color: textSecondary
      },
      axisLine: {
        lineStyle: { color: border }
      }
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        color: textSecondary,
        formatter: (value: number) => {
          if (value >= 10000) return `${(value / 10000).toFixed(1)}万`
          return value.toString()
        }
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
    dataZoom: [
      {
        type: 'slider',
        start: 0,
        end: 100,
        bottom: '5%',
        height: 20,
        borderColor: border,
        backgroundColor: bgCard,
        fillerColor: 'rgba(59, 130, 246, 0.2)',
        handleStyle: { color: primary },
        textStyle: { color: textSecondary }
      }
    ],
    series: [
      {
        name: '经验获得',
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
        data: expGained
      }
    ]
  }
})

const operationDistributionOption = computed(() => {
  if (!stats.value?.summary.op_counts) return {}
  
  const opCounts = stats.value.summary.op_counts
  const data: Array<{ name: string; value: number }> = []
  
  Object.keys(opTypeNames).forEach(opType => {
    const count = opCounts[opType] || 0
    if (count > 0) {
      data.push({
        name: opTypeNames[opType],
        value: count
      })
    }
  })
  
  // Sort by count descending
  data.sort((a, b) => b.value - a.value)
  
  const textSecondary = getCSSVar('--text-secondary') || '#9CA3AF'
  const bgElevated = getCSSVar('--bg-elevated') || '#252540'
  const border = getCSSVar('--border') || '#2a2a3e'
  const textPrimary = getCSSVar('--text-primary') || '#E5E7EB'
  const success = getCSSVar('--success') || '#22C55E'
  
  return {
    backgroundColor: 'transparent',
    textStyle: { color: textSecondary },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      },
      backgroundColor: bgElevated,
      borderColor: border,
      textStyle: { color: textPrimary },
      formatter: (params: any) => {
        const item = params[0]
        return `${item.name}<br/>${item.marker} 次数: ${item.value.toLocaleString()}`
      }
    },
    grid: {
      left: '3%',
      right: '8%',
      bottom: '3%',
      top: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'value',
      axisLabel: {
        color: textSecondary
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
    yAxis: {
      type: 'category',
      data: data.map(d => d.name).reverse(),
      axisLabel: {
        color: textSecondary
      },
      axisLine: {
        lineStyle: { color: border }
      }
    },
    series: [
      {
        name: '操作次数',
        type: 'bar',
        data: data.map(d => d.value).reverse(),
        barWidth: '60%',
        itemStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 1,
            y2: 0,
            colorStops: [
              { offset: 0, color: success },
              { offset: 1, color: '#4ADE80' }
            ]
          },
          borderRadius: [0, 4, 4, 0]
        }
      }
    ]
  }
})

onMounted(() => {
  fetchAccounts()
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
  // Auto-refresh every 30 seconds
  refreshInterval = window.setInterval(() => {
    if (selectedAccountId.value) {
      fetchStats()
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
  <div class="stats-view">
    <!-- Account Selector -->
    <ElCard shadow="never" class="filter-card">
      <template #header>
        <div class="card-header">
          <span class="header-title">操作统计</span>
        </div>
      </template>

      <ElRow :gutter="16" class="filter-row">
        <ElCol :xs="24" :sm="12" :md="8">
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
        </ElCol>

        <ElCol :xs="24" :sm="12" :md="8">
          <div class="time-preset-group">
            <ElRadioGroup v-model="timePreset" @change="handleTimePresetChange">
              <ElRadioButton
                v-for="preset in timePresets"
                :key="preset.value"
                :label="preset.label"
                :value="preset.value"
              />
            </ElRadioGroup>
          </div>
        </ElCol>

        <ElCol :xs="24" :sm="12" :md="8">
          <ElDatePicker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            @change="() => { timePreset = ''; fetchStats() }"
          />
        </ElCol>

        <ElCol :xs="24" :sm="12" :md="8">
          <div class="granularity-group">
            <span class="granularity-label">统计粒度</span>
            <ElRadioGroup v-model="granularity" @change="handleGranularityChange">
              <ElRadioButton label="按小时" value="hour" />
              <ElRadioButton label="按天" value="day" />
              <ElRadioButton label="按周" value="week" />
              <ElRadioButton label="全部" value="all" />
            </ElRadioGroup>
          </div>
        </ElCol>
      </ElRow>

      <!-- Bot Uptime Display -->
      <div class="uptime-section" v-if="selectedAccountId && getSelectedAccountUptime">
        <div class="uptime-label">Bot 运行时间</div>
        <div class="uptime-value">
          {{ uptime.uptime }}
          <template v-if="uptime.startedAt"> — 启动于 {{ uptime.startedAt }}</template>
        </div>
      </div>
    </ElCard>

    <!-- Summary Cards -->
    <ElRow :gutter="16" class="summary-row" v-if="stats">
      <ElCol :xs="24" :sm="8" :md="8">
        <ElCard shadow="never" class="summary-card">
          <div class="stat-content">
            <div class="stat-icon-wrapper stat-icon--gold">
              <ElIcon :size="24"><Coin /></ElIcon>
            </div>
            <ElStatistic title="总金币收入" :value="stats.summary.total_gold_in" />
          </div>
          <div class="stat-per-hour">每小时 {{ stats.summary.avg_gold_in_per_hour.toFixed(0) }}</div>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :sm="8" :md="8">
        <ElCard shadow="never" class="summary-card">
          <div class="stat-content">
            <div class="stat-icon-wrapper stat-icon--red">
              <ElIcon :size="24"><Coin /></ElIcon>
            </div>
            <ElStatistic title="总金币支出" :value="stats.summary.total_gold_out" />
          </div>
          <div class="stat-per-hour">每小时 {{ stats.summary.avg_gold_out_per_hour.toFixed(0) }}</div>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :sm="8" :md="8">
        <ElCard shadow="never" class="summary-card">
          <div class="stat-content">
            <div class="stat-icon-wrapper stat-icon--green">
              <ElIcon :size="24"><TrendCharts /></ElIcon>
            </div>
            <ElStatistic title="总经验获得" :value="stats.summary.total_exp" />
          </div>
          <div class="stat-per-hour">每小时 {{ stats.summary.avg_exp_per_hour.toFixed(0) }}</div>
        </ElCard>
      </ElCol>
    </ElRow>

    <!-- Gold Trend Chart -->
    <ElCard shadow="never" class="chart-card" v-if="stats && stats.timeline.length > 0">
      <template #header>
        <div class="card-header">
          <span class="header-title">金币趋势图</span>
        </div>
      </template>
      <VChart
        :option="goldTrendChartOption"
        autoresize
        style="height: 320px; width: 100%;"
      />
    </ElCard>

    <!-- Experience Trend Chart -->
    <ElCard shadow="never" class="chart-card" v-if="stats && stats.timeline.length > 0">
      <template #header>
        <div class="card-header">
          <span class="header-title">经验趋势图</span>
        </div>
      </template>
      <VChart
        :option="expTrendChartOption"
        autoresize
        style="height: 320px; width: 100%;"
      />
    </ElCard>

    <!-- Operation Distribution Chart -->
    <ElCard shadow="never" class="chart-card" v-if="stats && Object.keys(stats.summary.op_counts).some(k => stats!.summary.op_counts[k] > 0)">
      <template #header>
        <div class="card-header">
          <span class="header-title">操作分布图</span>
        </div>
      </template>
      <VChart
        :option="operationDistributionOption"
        autoresize
        style="height: 320px; width: 100%;"
      />
    </ElCard>

    <!-- Operation Counts Section -->
    <ElCard shadow="never" class="ops-card" v-if="stats">
      <template #header>
        <div class="card-header">
          <span class="header-title">操作次数</span>
        </div>
      </template>

      <ElRow :gutter="12">
        <ElCol
          v-for="opType in Object.keys(opTypeNames)"
          :key="opType"
          :xs="12"
          :sm="8"
          :md="6"
          :lg="4"
        >
          <div class="op-count-item">
            <span class="op-label">{{ opTypeNames[opType] }}</span>
            <span class="op-value">{{ getTotalOpCount(opType) }}</span>
          </div>
        </ElCol>
      </ElRow>
    </ElCard>

    <!-- Timeline Table -->
    <ElCard shadow="never" class="timeline-card" v-if="stats && stats.timeline.length > 0">
      <template #header>
        <div class="card-header">
          <span class="header-title">时间线</span>
        </div>
      </template>

      <ElTable :data="stats.timeline" stripe style="width: 100%">
        <ElTableColumn prop="period" label="时间" min-width="140" fixed />
        <ElTableColumn
          v-for="opType in Object.keys(opTypeNames)"
          :key="opType"
          :label="opTypeNames[opType]"
          width="80"
          align="center"
        >
          <template #default="{ row }">
            {{ row.op_counts[opType] || 0 }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="gold_in" label="金币收入" width="100" align="right">
          <template #default="{ row }">
            {{ row.gold_in.toLocaleString() }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="gold_out" label="金币支出" width="100" align="right">
          <template #default="{ row }">
            {{ row.gold_out.toLocaleString() }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="exp_gained" label="经验获得" width="100" align="right">
          <template #default="{ row }">
            {{ row.exp_gained.toLocaleString() }}
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>

    <ElEmpty v-if="!stats && !loading" description="暂无数据" class="empty-state" />
  </div>
</template>

<style scoped>
.stats-view {
  padding: 0;
}

/* Filter Card */
.filter-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
}

.filter-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.filter-card :deep(.el-card__body) {
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

 .filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: center;
}

.account-select {
  width: 200px;
}

.status-tag {
  margin-left: 8px;
  border-radius: var(--radius-sm);
}

.time-preset-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.granularity-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.granularity-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-right: 8px;
}

/* Uptime Section */
.uptime-section {
  margin-top: 16px;
  padding: 16px;
  background: var(--success-bg);
  border-radius: var(--radius-lg);
  border: 1px solid var(--success);
  opacity: 0.8;
  display: flex;
  align-items: center;
  gap: 16px;
}

.uptime-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--success);
}

.uptime-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

/* Summary Cards */
.summary-row {
  margin-top: 16px;
}

.summary-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
  margin-bottom: 16px;
}

 .summary-card :deep(.el-card__body) {
  padding: 20px 24px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon-wrapper {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon--green {
  background: var(--success-bg);
  color: var(--success);
}

.stat-icon--gold {
  background: var(--gold-bg);
  color: var(--gold);
}

.stat-icon--red {
  background: var(--danger-bg);
  color: var(--danger);
}

.stat-per-hour {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 8px;
}

:deep(.el-statistic__head) {
  color: var(--text-secondary);
  font-size: 13px;
}

:deep(.el-statistic__content) {
  color: var(--text-heading);
  font-weight: 600;
}

/* Operation Counts */
.ops-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
  margin-top: 16px;
}

.ops-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.ops-card :deep(.el-card__body) {
  padding: 24px;
}

.op-count-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--bg-elevated);
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  margin-bottom: 12px;
}

.op-label {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.op-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--success);
}

/* Timeline Card */
.timeline-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
  margin-top: 16px;
}

.timeline-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.timeline-card :deep(.el-card__body) {
  padding: 0;
}

.timeline-card :deep(.el-table) {
  --el-table-border-color: var(--border);
  --el-table-bg-color: var(--bg-card);
  --el-table-header-bg-color: var(--bg-elevated);
  --el-table-header-text-color: var(--text-secondary);
  --el-table-text-color: var(--text-primary);
  --el-table-row-hover-bg-color: var(--bg-elevated);
  --el-table-tr-bg-color: var(--bg-card);
}

.timeline-card :deep(.el-table__header th) {
  font-weight: 600;
  font-size: 13px;
  padding: 12px 0;
  background: var(--bg-elevated);
}

.timeline-card :deep(.el-table__body td) {
  padding: 10px 8px;
}

.timeline-card :deep(.el-table__row--striped) {
  background: var(--bg-hover);
}

/* Empty State */
.empty-state {
  padding: 48px;
}

:deep(.el-empty__description) {
  color: var(--text-muted) !important;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .filter-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .account-select {
    width: 100%;
  }

  .time-preset-group {
    margin-top: 12px;
  }

  .granularity-group {
    margin-top: 12px;
  }
}

/* Chart Cards */
.chart-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
  margin-top: 16px;
}

.chart-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.chart-card :deep(.el-card__body) {
  padding: 24px;
}

</style>
