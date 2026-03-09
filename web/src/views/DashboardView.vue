<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { dashboardApi, accountApi, getErrorMessage, type LandStatus } from '@/api'
import { 
  ElButton,
  ElTag,
  ElIcon,
  ElEmpty,
  ElMessage
} from 'element-plus'
import { 
  User, 
  VideoPlay,
  VideoPause,
  Warning,
  Plus
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

const router = useRouter()

// Stats and bot cards state
const stats = ref({
  totalAccounts: 0,
  runningBots: 0,
  errorBots: 0,
  stoppedBots: 0
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
      errorBots: data.accounts.filter((a: { status: string }) => a.status === 'error').length,
      stoppedBots: data.accounts.filter((a: { status: string }) => a.status === 'stopped').length
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

const goToAddAccount = () => {
  router.push('/accounts')
}

const getStatusText = (status: string): string => {
  if (status === 'running') return '运行中'
  if (status === 'error') return '异常'
  return '已停止'
}

const getStatusClass = (status: string): string => {
  if (status === 'running') return 'status-running'
  if (status === 'error') return 'status-error'
  return 'status-stopped'
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

const getAvatarLetter = (name: string): string => {
  return (name || '?').charAt(0).toUpperCase()
}

const displayedLands = computed(() => {
  return (lands: LandStatus[]) => {
    if (!lands) return []
    return lands.filter(l => l.unlocked).slice(0, 9)
  }
})

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
    <!-- Header with Add Account Button -->
    <div class="dashboard-header">
      <h1 class="page-title">总览</h1>
      <ElButton type="primary" class="add-account-btn" @click="goToAddAccount">
        <ElIcon class="btn-icon"><Plus /></ElIcon>
        添加账号
      </ElButton>
    </div>

    <!-- Stats Row -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-icon stat-icon--blue">
          <ElIcon :size="22"><User /></ElIcon>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.totalAccounts }}</span>
          <span class="stat-label">账号总数</span>
        </div>
      </div>
      
      <div class="stat-card">
        <div class="stat-icon stat-icon--green">
          <ElIcon :size="22"><VideoPlay /></ElIcon>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.runningBots }}</span>
          <span class="stat-label">运行中</span>
        </div>
      </div>
      
      <div class="stat-card">
        <div class="stat-icon stat-icon--red">
          <ElIcon :size="22"><Warning /></ElIcon>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.errorBots }}</span>
          <span class="stat-label">异常</span>
        </div>
      </div>
      
      <div class="stat-card">
        <div class="stat-icon stat-icon--gray">
          <ElIcon :size="22"><VideoPause /></ElIcon>
        </div>
        <div class="stat-info">
          <span class="stat-value">{{ stats.stoppedBots }}</span>
          <span class="stat-label">已停止</span>
        </div>
      </div>
    </div>

    <!-- Bot Cards -->
    <div class="accounts-section">
      <div class="section-header">
        <span class="section-title">账号列表</span>
        <span class="section-count">{{ botCards.length }} 个账号</span>
      </div>
      
      <ElEmpty v-if="botCards.length === 0" description="暂无账号，请先添加账号" class="empty-state" />
      
      <div v-else class="accounts-grid">
        <div 
          v-for="bot in botCards" 
          :key="bot.id"
          class="account-card"
          :class="{ 'account-card--stopped': bot.status !== 'running' }"
        >
          <!-- Card Header -->
          <div class="card-header">
            <div class="avatar">{{ getAvatarLetter(bot.name) }}</div>
            <div class="account-info">
              <div class="account-name">{{ bot.name || '账号 #' + bot.id }}</div>
              <div class="account-meta">
                <span class="platform-tag" :class="bot.platform === 'qq' ? 'platform-qq' : 'platform-wx'">
                  {{ bot.platform.toUpperCase() }}
                </span>
                <span class="status-tag" :class="getStatusClass(bot.status)">
                  {{ getStatusText(bot.status) }}
                </span>
              </div>
            </div>
          </div>

          <!-- Stats Row -->
          <div class="stats-row-inner">
            <div class="stat-item">
              <span class="stat-label">等级</span>
              <span class="stat-value">Lv.{{ bot.level }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">金币</span>
              <span class="stat-value stat-value--gold">{{ bot.gold.toLocaleString() }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">经验</span>
              <span class="stat-value">{{ bot.exp.toLocaleString() }}</span>
            </div>
          </div>

          <!-- Secondary Stats -->
          <div class="secondary-stats">
            <div class="stat-mini">
              <span class="stat-mini-value">{{ bot.total_steal }}</span>
              <span class="stat-mini-label">偷菜</span>
            </div>
            <div class="stat-mini">
              <span class="stat-mini-value stat-mini-value--green">{{ bot.total_help }}</span>
              <span class="stat-mini-label">帮助</span>
            </div>
            <div class="stat-mini">
              <span class="stat-mini-value">{{ bot.friends_count }}</span>
              <span class="stat-mini-label">好友</span>
            </div>
          </div>

          <!-- Level Up Info -->
          <div class="level-up-box" v-if="bot.status === 'running' && bot.exp_to_next_level > 0">
            <span class="level-up-icon">UP</span>
            <span class="level-up-text">
              预计 {{ formatLevelUpTime(bot.hours_to_next_level) }} 升级
            </span>
          </div>

          <!-- Land Overview -->
          <div class="land-overview">
            <span class="land-icon">LAND</span>
            <span class="land-text">{{ bot.unlocked_lands }}/{{ bot.total_lands }} 土地已解锁</span>
          </div>

          <!-- Land Grid Preview -->
          <div class="land-grid" v-if="displayedLands(bot.lands).length > 0">
            <div 
              class="land-cell"
              v-for="land in displayedLands(bot.lands)"
              :key="land.id"
              :class="{ 'land-cell--mature': land.phase === '成熟' }"
            >
              <span class="land-level">{{ getLandLevelName(land.level) }}</span>
              <ElTag 
                v-if="land.phase" 
                :type="getPhaseType(land.phase)" 
                size="small"
                class="phase-tag"
              >
                {{ land.phase }}
              </ElTag>
              <span v-else class="land-empty">空</span>
            </div>
          </div>

          <!-- Control Button -->
          <ElButton
            :type="bot.status === 'running' ? 'danger' : 'success'"
            class="control-btn"
            @click="toggleBot(bot)"
          >
            <ElIcon>
              <VideoPause v-if="bot.status === 'running'" />
              <VideoPlay v-else />
            </ElIcon>
            {{ bot.status === 'running' ? '停止运行' : '启动运行' }}
          </ElButton>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 0;
  min-height: 100%;
}

/* Header */
.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-6);
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-heading);
  margin: 0;
}

.add-account-btn {
  background: var(--primary);
  border-color: var(--primary);
  border-radius: var(--radius);
  font-weight: 500;
  padding: 10px 20px;
  transition: all var(--transition);
}

.add-account-btn:hover {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}

.btn-icon {
  margin-right: 6px;
}

/* Stats Row - Apple Bento Box */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
  margin-bottom: var(--space-6);
}

.stat-card {
  background: var(--bg-card);
  border: none;
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  display: flex;
  align-items: center;
  gap: var(--space-4);
  box-shadow: var(--shadow-card);
  transition: all var(--transition);
}

.stat-card:hover {
  box-shadow: var(--shadow-card-hover);
  transform: translateY(-1px);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon--blue {
  background: var(--primary-bg);
  color: var(--primary);
}

.stat-icon--green {
  background: var(--success-bg);
  color: var(--success);
}

.stat-icon--red {
  background: var(--danger-bg);
  color: var(--danger);
}

.stat-icon--gray {
  background: rgba(142, 142, 147, 0.1);
  color: var(--text-secondary);
}

.stat-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-info .stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-heading);
  line-height: 1.2;
}

.stat-info .stat-label {
  font-size: 13px;
  color: var(--text-secondary);
}

/* Accounts Section */
.accounts-section {
  background: var(--bg-card);
  border: none;
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  box-shadow: var(--shadow-card);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-5);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--border-light);
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-heading);
}

.section-count {
  font-size: 13px;
  color: var(--text-muted);
}

.empty-state {
  padding: 40px 0;
}

/* Accounts Grid */
.accounts-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
}

/* Account Card - Apple Bento Style */
.account-card {
  background: var(--bg-card);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  transition: all var(--transition);
  box-shadow: var(--shadow-xs);
}

.account-card:hover {
  border-color: var(--border);
  box-shadow: var(--shadow-md);
}

.account-card--stopped {
  opacity: 0.7;
}

.account-card--stopped:hover {
  opacity: 0.85;
}

/* Card Header */
.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border-light);
}

.avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary) 0%, var(--primary-light) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 600;
  color: #FFFFFF;
  flex-shrink: 0;
}

.account-info {
  flex: 1;
  min-width: 0;
}

.account-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-heading);
  margin-bottom: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.account-meta {
  display: flex;
  align-items: center;
  gap: 6px;
}

.platform-tag {
  font-size: 9px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: var(--radius-full);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.platform-qq {
  background: var(--primary-bg);
  color: var(--primary);
}

.platform-wx {
  background: var(--success-bg);
  color: var(--success);
}

.status-tag {
  font-size: 10px;
  font-weight: 500;
  padding: 3px 8px;
  border-radius: var(--radius-xs);
}

.status-running {
  background: var(--success-bg);
  color: var(--success);
}

.status-stopped {
  background: rgba(142, 142, 147, 0.1);
  color: var(--text-secondary);
}

.status-error {
  background: var(--danger-bg);
  color: var(--danger);
}

/* Stats Row Inner */
.stats-row-inner {
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--space-3);
}

.stats-row-inner .stat-item {
  text-align: center;
  flex: 1;
}

.stats-row-inner .stat-label {
  display: block;
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 4px;
}

.stats-row-inner .stat-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.stats-row-inner .stat-value--gold {
  color: var(--gold);
}

/* Secondary Stats */
.secondary-stats {
  display: flex;
  justify-content: space-around;
  padding: 10px 0;
  margin-bottom: var(--space-3);
  border-top: 1px solid var(--border-light);
  border-bottom: 1px solid var(--border-light);
}

.stat-mini {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.stat-mini-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.stat-mini-value--green {
  color: var(--success);
}

.stat-mini-label {
  font-size: 10px;
  color: var(--text-muted);
}

/* Level Up Box */
.level-up-box {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  margin-bottom: var(--space-3);
}

.level-up-icon {
  font-size: 11px;
  font-weight: 700;
  color: var(--gold);
}

.level-up-text {
  font-size: 12px;
  font-weight: 500;
  color: var(--gold);
}

/* Land Overview */
.land-overview {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  margin-bottom: var(--space-3);
}

.land-icon {
  font-size: 11px;
  font-weight: 700;
  color: var(--success);
}

.land-text {
  font-size: 12px;
  font-weight: 500;
  color: var(--success);
}

/* Land Grid */
.land-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
  margin-bottom: var(--space-3);
}

.land-cell {
  background: var(--bg-elevated);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-sm);
  padding: 6px 4px;
  text-align: center;
  font-size: 10px;
  transition: all var(--transition-fast);
}

.land-cell:hover {
  border-color: var(--primary);
}

.land-cell--mature {
  border-color: var(--success);
  background: var(--success-bg);
}

.land-level {
  display: block;
  color: var(--text-muted);
  font-size: 9px;
  margin-bottom: 2px;
}

.phase-tag {
  font-size: 9px !important;
  padding: 1px 4px !important;
  height: auto !important;
  line-height: 1.4 !important;
}

.land-empty {
  font-size: 9px;
  color: var(--text-muted);
}

/* Control Button */
.control-btn {
  width: 100%;
  border-radius: var(--radius-sm);
  font-weight: 500;
  height: 36px;
  margin-top: auto;
  transition: all var(--transition);
}

.control-btn.el-button--success {
  background: var(--success);
  border-color: var(--success);
}

.control-btn.el-button--success:hover {
  background: color-mix(in srgb, var(--success), #000 15%);
  border-color: color-mix(in srgb, var(--success), #000 15%);
}

.control-btn.el-button--danger {
  background: var(--danger);
  border-color: var(--danger);
}

.control-btn.el-button--danger:hover {
  background: color-mix(in srgb, var(--danger), #000 15%);
  border-color: color-mix(in srgb, var(--danger), #000 15%);
}

/* Mobile Responsive */
@media (max-width: 1200px) {
  .accounts-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 900px) {
  .accounts-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
    gap: var(--space-3);
  }

  .stat-card {
    padding: var(--space-4);
  }

  .stat-info .stat-value {
    font-size: 24px;
  }

  .accounts-section {
    padding: var(--space-4);
  }

  .accounts-grid {
    grid-template-columns: 1fr;
    gap: var(--space-3);
  }

  .dashboard-header {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-4);
  }

  .add-account-btn {
    width: 100%;
  }
}
</style>
