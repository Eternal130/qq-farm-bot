<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { dashboardApi, accountApi, getErrorMessage, type LandStatus } from '@/api'
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
  } catch {
    // silently fail - dashboard shows empty state
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
  } catch (error: unknown) {
    const message = getErrorMessage(error, '操作失败')
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

const getLandLevelName = (level: number): string => {
  const names: Record<number, string> = { 1: '黄土', 2: '红土', 3: '黑土', 4: '金土' }
  return names[level] || `Lv.${level}`
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
        <ElCard class="stat-card stat-card--accounts" shadow="never">
          <div class="stat-content">
            <div class="stat-icon-wrapper stat-icon--green">
              <ElIcon :size="24"><User /></ElIcon>
            </div>
            <ElStatistic title="账号总数" :value="stats.totalAccounts" />
          </div>
        </ElCard>
      </ElCol>
      
      <ElCol :xs="24" :sm="8" :md="8">
        <ElCard class="stat-card stat-card--running" shadow="never">
          <div class="stat-content">
            <div class="stat-icon-wrapper stat-icon--emerald">
              <ElIcon :size="24"><VideoPlay /></ElIcon>
            </div>
            <ElStatistic title="运行中" :value="stats.runningBots" />
          </div>
        </ElCard>
      </ElCol>
      
      <ElCol :xs="24" :sm="8" :md="8">
        <ElCard class="stat-card stat-card--gold" shadow="never">
          <div class="stat-content">
            <div class="stat-icon-wrapper stat-icon--gold">
              <ElIcon :size="24"><Coin /></ElIcon>
            </div>
            <ElStatistic title="金币总额" :value="stats.totalGold" :formatter="(val) => val.toLocaleString()" />
          </div>
        </ElCard>
      </ElCol>
    </ElRow>

    <!-- Bot Cards -->
    <ElCard class="bots-card" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <ElIcon class="header-icon"><TrendCharts /></ElIcon>
            <span>账号状态</span>
          </div>
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
                <span class="platform-tag" :class="bot.platform === 'qq' ? 'platform-qq' : 'platform-wx'">
                  {{ bot.platform.toUpperCase() }}
                </span>
                <span class="bot-name-text">{{ bot.name }}</span>
              </div>
              <ElTag :type="getStatusType(bot.status)" size="small" class="status-tag">
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
            
            <div class="bot-stats bot-stats--secondary">
              <div class="stat-item">
                <span class="label">偷菜</span>
                <span class="value">{{ bot.total_steal }}</span>
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
                  <span class="land-level">{{ getLandLevelName(land.level) }}</span>
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
              {{ bot.status === 'running' ? '停止运行' : '启动运行' }}
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
  margin-bottom: 24px;
}

/* Stat Cards */
.stat-card {
  border-radius: 16px;
  border: none;
  margin-bottom: 16px;
  position: relative;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(21, 128, 61, 0.06), 0 4px 16px rgba(21, 128, 61, 0.04);
}

.stat-card::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  border-radius: 4px 0 0 4px;
}

.stat-card--accounts::before {
  background: #15803D;
}

.stat-card--running::before {
  background: #10B981;
}

.stat-card--gold::before {
  background: #CA8A04;
}

.stat-card :deep(.el-card__body) {
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
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon--green {
  background: rgba(21, 128, 61, 0.1);
  color: #15803D;
}

.stat-icon--emerald {
  background: rgba(16, 185, 129, 0.1);
  color: #10B981;
}

.stat-icon--gold {
  background: rgba(202, 138, 4, 0.1);
  color: #CA8A04;
}

.stat-card :deep(.el-statistic__head) {
  font-size: 13px;
  color: #6B7280;
  font-weight: 500;
  margin-bottom: 4px;
}

.stat-card :deep(.el-statistic__content) {
  font-size: 28px;
  font-weight: 700;
  color: #14532D;
}

/* Bots Card */
.bots-card {
  border-radius: 16px;
  border: none;
  box-shadow: 0 1px 3px rgba(21, 128, 61, 0.06), 0 4px 16px rgba(21, 128, 61, 0.04);
}

.bots-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid #E5E7EB;
}

.bots-card :deep(.el-card__body) {
  padding: 24px;
}

.card-header {
  display: flex;
  align-items: center;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 17px;
  font-weight: 600;
  color: #14532D;
}

.header-icon {
  color: #15803D;
  font-size: 20px;
}

.bot-col {
  margin-bottom: 16px;
}

/* Bot Card */
.bot-card {
  border-radius: 12px;
  border: 1px solid rgba(21, 128, 61, 0.08);
  transition: box-shadow 0.25s ease;
}

.bot-card:hover {
  box-shadow: 0 4px 12px rgba(21, 128, 61, 0.12), 0 8px 24px rgba(21, 128, 61, 0.08);
}

.bot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #F3F4F6;
}

.bot-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.bot-name-text {
  font-weight: 600;
  color: #14532D;
  font-size: 15px;
}

.platform-tag {
  font-size: 9px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 100px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.platform-qq {
  background: rgba(21, 128, 61, 0.1);
  color: #15803D;
}

.platform-wx {
  background: rgba(59, 130, 246, 0.1);
  color: #2563EB;
}

.status-tag {
  border-radius: 6px;
}

/* Bot Stats */
.bot-stats {
  display: flex;
  justify-content: space-between;
  margin-bottom: 12px;
}

.stat-item {
  text-align: center;
}

.stat-item .label {
  display: block;
  font-size: 11px;
  color: #9CA3AF;
  margin-bottom: 4px;
  font-weight: 500;
}

.stat-item .value {
  font-size: 15px;
  font-weight: 600;
  color: #14532D;
}

.stat-item .value.gold {
  color: #CA8A04;
}

.stat-item .value.help {
  color: #22C55E;
}

.bot-stats--secondary .stat-item .value {
  font-size: 14px;
}

/* Level Up Info */
.level-up-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 10px 14px;
  background: linear-gradient(135deg, #FEF9C3 0%, #FEF08A 100%);
  border-radius: 10px;
}

.level-up-label {
  font-size: 12px;
  color: #A16207;
  font-weight: 600;
  background: rgba(202, 138, 4, 0.15);
  padding: 2px 8px;
  border-radius: 4px;
}

.level-up-value {
  font-size: 13px;
  font-weight: 600;
  color: #854D0E;
}

/* Land Overview */
.land-overview {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 10px 14px;
  background: linear-gradient(135deg, #DCFCE7 0%, #BBF7D0 100%);
  border-radius: 10px;
}

.land-label {
  font-size: 12px;
  color: #166534;
  font-weight: 600;
  background: rgba(21, 128, 61, 0.15);
  padding: 2px 8px;
  border-radius: 4px;
}

.land-value {
  font-size: 13px;
  font-weight: 600;
  color: #14532D;
}

/* Land Grid */
.land-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
  margin-bottom: 12px;
  max-height: 180px;
  overflow-y: auto;
  padding: 2px;
}

.land-grid::-webkit-scrollbar {
  width: 4px;
}

.land-grid::-webkit-scrollbar-track {
  background: #F3F4F6;
  border-radius: 4px;
}

.land-grid::-webkit-scrollbar-thumb {
  background: #D1D5DB;
  border-radius: 4px;
}

.land-cell {
  background: #FAFAFA;
  border: 1px solid #E5E7EB;
  border-radius: 10px;
  padding: 6px;
  font-size: 11px;
  text-align: center;
  transition: all 0.2s ease;
}

.land-cell:hover {
  border-color: rgba(21, 128, 61, 0.3);
  background: #F0FDF4;
}

.land-cell-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.land-id {
  color: #9CA3AF;
  font-weight: 500;
  font-size: 10px;
}

.land-level {
  color: #15803D;
  font-weight: 600;
  font-size: 10px;
}

.land-crop {
  color: #14532D;
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
  border-radius: 4px !important;
}

/* Control Button */
.control-btn {
  width: 100%;
  border-radius: 8px;
  font-weight: 600;
  height: 36px;
  transition: all 0.2s ease;
}

.control-btn.el-button--success {
  background: #15803D;
  border-color: #15803D;
}

.control-btn.el-button--success:hover {
  background: #166534;
  border-color: #166534;
}

.control-btn.el-button--danger {
  background: #DC2626;
  border-color: #DC2626;
}

.control-btn.el-button--danger:hover {
  background: #B91C1C;
  border-color: #B91C1C;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .stats-row {
    margin-bottom: 16px;
  }
  
  .stat-card :deep(.el-card__body) {
    padding: 16px 20px;
  }
  
  .stat-card :deep(.el-statistic__content) {
    font-size: 24px;
  }
  
  .bots-card :deep(.el-card__body) {
    padding: 16px;
  }
  
  .bot-col {
    margin-bottom: 12px;
  }
}
</style>
