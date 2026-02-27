<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { dashboardApi, accountApi, type LandStatus } from '@/api'
import { 
  ElRow, 
  ElCol, 
  ElCard, 
  ElStatistic,
  ElButton,
  ElTag,
  ElIcon,
  ElEmpty,
  ElMessage
} from 'element-plus'
import { 
  User, 
  Coin, 
  VideoPlay,
  VideoPause,
  TrendCharts
} from '@element-plus/icons-vue'

interface BotCard {
  id: number
  name: string
  level: number
  gold: number
  exp: number
  status: string
  platform: string
  total_steal: number
  total_help: number
  friends_count: number
  total_lands: number
  unlocked_lands: number
  lands: LandStatus[]
  // Level up estimation
  exp_rate_per_hour: number
  next_level_exp: number
  exp_to_next_level: number
  hours_to_next_level: number
}

// Stats and bot cards state
const stats = ref({
  totalAccounts: 0,
  runningBots: 0,
  totalGold: 0
})
const botCards = ref<BotCard[]>([])
let refreshInterval: number | null = null

const fetchDashboard = async () => {
  try {
    const response = await dashboardApi.getStats()
    const data = response.data
    stats.value = {
      totalAccounts: data.total_accounts,
      runningBots: data.running_bots,
      totalGold: data.total_gold
    }
    botCards.value = data.accounts.map(acc => ({
      id: acc.id,
      name: acc.name,
      level: acc.level,
      gold: acc.gold,
      exp: acc.exp,
      status: acc.status,
      platform: acc.platform,
      total_steal: acc.total_steal,
      total_help: acc.total_help,
      friends_count: acc.friends_count,
      total_lands: acc.total_lands,
      unlocked_lands: acc.unlocked_lands,
      lands: acc.lands || [],
      // Level up estimation
      exp_rate_per_hour: acc.exp_rate_per_hour || 0,
      next_level_exp: acc.next_level_exp || 0,
      exp_to_next_level: acc.exp_to_next_level || 0,
      hours_to_next_level: acc.hours_to_next_level || 0
    }))
  } catch (error) {
    console.error('Failed to fetch dashboard:', error)
  }
}

const toggleBot = async (bot: BotCard) => {
  try {
    if (bot.status === 'running') {
      await accountApi.stop(bot.id)
      ElMessage.success(`已停止 ${bot.name}`)
    } else {
      await accountApi.start(bot.id)
      ElMessage.success(`已启动 ${bot.name}`)
    }
    // Refresh data
    await fetchDashboard()
  } catch (error: any) {
    const message = error.response?.data?.error || '操作失败'
    if (message.includes('no login code')) {
      ElMessage.warning('该账号尚未登录，请前往账号管理页面扫码登录')
    } else {
      ElMessage.error(message)
    }
  }
}
const getStatusType = (status: string): 'success' | 'info' | 'danger' => {
  if (status === 'running') return 'success'
  if (status === 'error') return 'danger'
  return 'info'
}

const getStatusText = (status: string): string => {
  if (status === 'running') return '运行中'
  if (status === 'error') return '错误'
  return '已停止'
}

const getPhaseType = (phase: string | undefined): 'success' | 'info' | 'warning' | 'danger' | 'primary' => {
  if (!phase) return 'info'
  if (phase === '成熟') return 'success'
  if (phase === '枯萎') return 'danger'
  if (phase === '开花') return 'warning'
  if (['发芽', '小叶', '大叶'].includes(phase)) return 'primary'
  return 'info'
}

const formatLevelUpTime = (hours: number): string => {
  if (hours <= 0) return '-'  
  if (hours < 1) {
    const mins = Math.round(hours * 60)
    return `${mins}分钟`
  }
  if (hours < 24) {
    const h = Math.floor(hours)
    const m = Math.round((hours - h) * 60)
    return m > 0 ? `${h}小时${m}分` : `${h}小时`
  }
  const days = Math.floor(hours / 24)
  const h = Math.round(hours % 24)
  return h > 0 ? `${days}天${h}小时` : `${days}天`
}

onMounted(() => {
  fetchDashboard()
  // Auto refresh every 5 seconds
  refreshInterval = window.setInterval(fetchDashboard, 5000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <div class="dashboard">
    <!-- Stats Row -->
    <ElRow :gutter="20" class="stats-row">
      <ElCol :xs="24" :sm="8" :md="8">
        <ElCard class="stat-card" shadow="hover">
          <div class="stat-content">
            <ElIcon class="stat-icon" :size="32" color="#409eff"><User /></ElIcon>
            <ElStatistic title="账号总数" :value="stats.totalAccounts" />
          </div>
        </ElCard>
      </ElCol>
      
      <ElCol :xs="24" :sm="8" :md="8">
        <ElCard class="stat-card" shadow="hover">
          <div class="stat-content">
            <ElIcon class="stat-icon" :size="32" color="#67c23a"><VideoPlay /></ElIcon>
            <ElStatistic title="运行中" :value="stats.runningBots" />
          </div>
        </ElCard>
      </ElCol>
      
      <ElCol :xs="24" :sm="8" :md="8">
        <ElCard class="stat-card" shadow="hover">
          <div class="stat-content">
            <ElIcon class="stat-icon" :size="32" color="#e6a23c"><Coin /></ElIcon>
            <ElStatistic title="金币总额" :value="stats.totalGold" :formatter="(val) => val.toLocaleString()" />
          </div>
        </ElCard>
      </ElCol>
    </ElRow>

    <!-- Bot Cards -->
    <ElCard class="bots-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span><ElIcon><TrendCharts /></ElIcon> 账号状态</span>
        </div>
      </template>
      
      <ElEmpty v-if="botCards.length === 0" description="暂无账号，请先添加账号" />
      
      <ElRow v-else :gutter="16">
        <ElCol 
          v-for="bot in botCards" 
          :key="bot.id" 
          :xs="24" 
          :sm="12" 
          :md="8" 
          :lg="6"
          class="bot-col"
        >
          <ElCard class="bot-card" :body-style="{ padding: '16px' }" shadow="hover">
            <div class="bot-header">
              <div class="bot-name">
                <span class="platform-tag">{{ bot.platform.toUpperCase() }}</span>
                {{ bot.name }}
              </div>
              <ElTag :type="getStatusType(bot.status)" size="small">
                {{ getStatusText(bot.status) }}
              </ElTag>
            </div>
            
            <div class="bot-stats">
              <div class="stat-item">
                <span class="label">等级</span>
                <span class="value">Lv.{{ bot.level }}</span>
              </div>
              <div class="stat-item">
                <span class="label">金币</span>
                <span class="value gold">{{ bot.gold.toLocaleString() }}</span>
              </div>
              <div class="stat-item">
                <span class="label">经验</span>
                <span class="value">{{ bot.exp.toLocaleString() }}</span>
              </div>
            </div>
            
            <div class="bot-stats bot-stats-extra">
              <div class="stat-item">
                <span class="label">偷菜</span>
                <span class="value steal">{{ bot.total_steal }}</span>
              </div>
              <div class="stat-item">
                <span class="label">帮助</span>
                <span class="value help">{{ bot.total_help }}</span>
              </div>
              <div class="stat-item">
                <span class="label">好友</span>
                <span class="value">{{ bot.friends_count }}</span>
              </div>
            </div>
            
            <div class="level-up-info" v-if="bot.status === 'running' && bot.exp_to_next_level > 0">
              <span class="level-up-label">升级</span>
              <span class="level-up-value" v-if="bot.hours_to_next_level > 0">
                预计 {{ formatLevelUpTime(bot.hours_to_next_level) }}
              </span>
              <span class="level-up-value" v-else>
                计算中...
              </span>
            </div>
            
            <div class="land-overview">
              <span class="land-label">土地</span>
              <span class="land-value">{{ bot.unlocked_lands }}/{{ bot.total_lands }} 已解锁</span>
            </div>
            
            <div class="land-grid" v-if="bot.lands && bot.lands.filter(l => l.unlocked).length > 0">
              <div 
                class="land-cell" 
                v-for="land in bot.lands.filter(l => l.unlocked)" 
                :key="land.id"
              >
                <div class="land-cell-header">
                  <span class="land-id">#{{ land.id }}</span>
                  <span class="land-level">Lv.{{ land.level }}</span>
                </div>
                <div class="land-crop">{{ land.crop_name || '空地' }}</div>
                <ElTag 
                  v-if="land.phase" 
                  :type="getPhaseType(land.phase)" 
                  size="small"
                  class="phase-tag"
                >
                  {{ land.phase }}
                </ElTag>
                <ElTag v-else type="info" size="small" class="phase-tag">空地</ElTag>
              </div>
            </div>
            
            <ElButton
              :type="bot.status === 'running' ? 'danger' : 'success'"
              size="small"
              class="control-btn"
              @click="toggleBot(bot)"
            >
              <ElIcon>
                <VideoPause v-if="bot.status === 'running'" />
                <VideoPlay v-else />
              </ElIcon>
            </ElButton>
          </ElCard>
        </ElCol>
      </ElRow>
    </ElCard>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 0;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  border-radius: 8px;
}

.stat-card :deep(.el-card__body) {
  padding: 20px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  flex-shrink: 0;
}

.bots-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 500;
}

.bot-col {
  margin-bottom: 16px;
}

.bot-card {
  border-radius: 8px;
  border: 1px solid #e4e7ed;
  transition: all 0.3s ease;
}

.bot-card:hover {
  border-color: #409eff;
}

.bot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.bot-name {
  font-weight: 500;
  color: #303133;
  display: flex;
  align-items: center;
  gap: 8px;
}

.platform-tag {
  font-size: 10px;
  padding: 2px 6px;
  background-color: #f0f2f5;
  border-radius: 4px;
  color: #909399;
}

.bot-stats {
  display: flex;
  justify-content: space-between;
  margin-bottom: 16px;
}

.stat-item {
  text-align: center;
}

.stat-item .label {
  display: block;
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
}

.stat-item .value {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.stat-item .value.gold {
  color: #e6a23c;
}


.stat-item .value.steal {
  color: #909399;
}

.stat-item .value.help {
  color: #67c23a;
}

.bot-stats-extra {
  margin-bottom: 12px;
}

.bot-stats-extra .stat-item .value {
  font-size: 14px;
}

.level-up-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: linear-gradient(135deg, #fef0f0 0%, #fde2e2 100%);
  border-radius: 6px;
}

.level-up-label {
  font-size: 12px;
  color: #f56c6c;
  font-weight: 500;
}

.level-up-value {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.land-overview {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: linear-gradient(135deg, #f0f9eb 0%, #e1f3d8 100%);
  border-radius: 6px;
}

.land-label {
  font-size: 12px;
  color: #67c23a;
  font-weight: 500;
}

.land-value {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.land-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
  margin-bottom: 12px;
  max-height: 180px;
  overflow-y: auto;
  padding: 2px;
}

.land-cell {
  background: #fafafa;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 6px;
  font-size: 11px;
  text-align: center;
  transition: all 0.2s ease;
}

.land-cell:hover {
  border-color: #409eff;
  background: #f0f7ff;
}

.land-cell-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.land-id {
  color: #909399;
  font-weight: 500;
}

.land-level {
  color: #409eff;
  font-weight: 600;
  font-size: 10px;
}

.land-crop {
  color: #303133;
  font-weight: 500;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.phase-tag {
  font-size: 10px !important;
  padding: 0 4px !important;
  height: 18px !important;
  line-height: 16px !important;
}

.control-btn {
  width: 100%;
}
</style>
