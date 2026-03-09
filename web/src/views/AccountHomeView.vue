<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { 
  dashboardApi, 
  accountApi, 
  getErrorMessage, 
    type Account,
    type DashboardStats
} from '@/api'
import { 
  ElCard, 
  ElButton, 
  ElTag, 
  ElSwitch, 
  ElIcon,
  ElEmpty,
  ElMessage,
  ElTooltip
} from 'element-plus'
import { 
  User, 
  Coin, 
    TrendCharts, 
    VideoPlay, 
    VideoPause,
    Sunny,
    Grid,
    Handbag,
    Money,
    StarFilled,
    CircleCheck,
    InfoFilled
} from '@element-plus/icons-vue'

const route = useRoute()

// Account ID from route
const accountId = computed(() => {
  const id = route.params.id
  return typeof id === 'string' ? parseInt(id, 10) : 0
})

// Account data
const account = ref<Account | null>(null)
const botStatus = ref({
  running: false,
  level: 0,
    gold: 0,
    exp: 0
})

const isLoading = ref(false)
let refreshInterval: number | null = null

// Today's statistics (mock data - would come from API in production)
const todayStats = ref({
  harvest: 0,
  plant: 0,
  steal: 0,
  gold: 0,
  help: 0,
  task: 0
})

// Function toggles configuration
const toggles = [
  { key: 'enable_harvest', label: '自动收获', tooltip: '自动收获成熟的作物' },
  { key: 'enable_plant', label: '自动种植', tooltip: '收获后自动种植新作物' },
    { key: 'enable_sell', label: '自动出售', tooltip: '自动出售仓库中的果实' },
    { key: 'enable_weed', label: '自动除草', tooltip: '自动清除杂草' },
    { key: 'enable_bug', label: '自动除虫', tooltip: '自动消灭害虫' },
    { key: 'enable_water', label: '自动浇水', tooltip: '自动为缺水作物浇水' },
    { key: 'enable_remove_dead', label: '自动铲除', tooltip: '自动铲除枯死作物' },
    { key: 'enable_upgrade_land', label: '升级土地', tooltip: '自动升级和解锁土地' },
    { key: 'enable_help_friend', label: '帮好友', tooltip: '帮好友浇水/除草/除虫' },
    { key: 'enable_steal', label: '允许偷菜', tooltip: '自动偷取好友成熟作物' },
    { key: 'enable_claim_task', label: '领取任务', tooltip: '自动领取完成的任务奖励' },
    { key: 'enable_anti_detection', label: '防检测', tooltip: '随机化操作间隔，降低检测风险' }
]

// Toggle values (reactive map)
const toggleValues = ref<Record<string, boolean>>({})

// Fetch account data
const fetchAccountData = async () => {
  if (accountId.value === 0) return
    
  try {
        // Get dashboard stats to find account
        const response = await dashboardApi.getStats()
        const data: DashboardStats = response.data
        
        const found = data.accounts.find(a => a.id === accountId.value)
        if (found) {
            // Fetch full account details
            const accountRes = await accountApi.getAll()
            const fullAccount = accountRes.data.find(a => a.id === accountId.value)
            
            if (fullAccount) {
                account.value = fullAccount
                botStatus.value = {
                    running: fullAccount.status === 'running',
                    level: fullAccount.level,
                    gold: fullAccount.gold,
                    exp: fullAccount.exp,
                }
            }
            
            // Initialize toggle values from account
            toggles.forEach(t => {
                toggleValues.value[t.key] = (fullAccount as unknown as Record<string, boolean>)[t.key] as boolean
            })
        }
    } catch {
        // silently fail
    }
}

// Toggle bot
const toggleBot = async () => {
    if (!account.value) return
    
    isLoading.value = true
    try {
        if (botStatus.value.running) {
            await accountApi.stop(account.value.id)
            ElMessage.success(`已停止 ${account.value.name}`)
        } else {
            await accountApi.start(account.value.id)
            ElMessage.success(`已启动 ${account.value.name}`)
        }
        await fetchAccountData()
    } catch (error: unknown) {
        const message = getErrorMessage(error, '操作失败')
        ElMessage.error(message)
    } finally {
        isLoading.value = false
    }
}

// Handle toggle change
const handleToggleChange = async (key: string, value: boolean) => {
    if (!account.value) return
    
    try {
        await accountApi.update(account.value.id, { [key]: value })
        toggleValues.value[key] = value
        ElMessage.success('设置已保存')
    } catch (error: unknown) {
        const message = getErrorMessage(error, '保存失败')
        ElMessage.error(message)
        // Revert on error
        toggleValues.value[key] = !value
    }
}

// Get status info
const getStatusInfo = (status: string): { text: string; type: 'success' | 'info' | 'danger' } => {
    if (status === 'running') return { text: '运行中', type: 'success' }
    if (status === 'error') return { text: '异常', type: 'danger' }
    return { text: '已停止', type: 'info' }
}

// Get avatar initial
const getAvatarInitial = (name: string): string => {
    return name ? name.charAt(0).toUpperCase() : 'U'
}

// Format date
const formatDate = (): string => {
    const now = new Date()
    return `${now.getFullYear()}/${now.getMonth() + 1}/${now.getDate()}`
}

// Lifecycle
onMounted(() => {
    fetchAccountData()
    refreshInterval = window.setInterval(fetchAccountData, 5000)
})

onUnmounted(() => {
    if (refreshInterval) {
        clearInterval(refreshInterval)
    }
})
</script>

<template>
    <div class="account-home">
        <!-- Account not found -->
        <ElEmpty v-if="!account" description="账号不存在或未加载" />
        
        <template v-else>
            <!-- Section 1: Account Profile Card -->
            <ElCard class="profile-card" shadow="never">
                <div class="profile-header">
                    <div class="profile-left">
                        <div class="avatar">
                            {{ getAvatarInitial(account.name) }}
                        </div>
                    </div>
                    
                    <div class="profile-middle">
                        <div class="account-name">{{ account.name }}</div>
                        <div class="account-meta">
                            <ElTag 
                                :type="account.platform === 'qq' ? 'primary' : 'success'" 
                                size="small"
                                class="platform-tag"
                            >
                                {{ account.platform.toUpperCase() }}
                            </ElTag>
                            <span class="user-id">ID: {{ account.id }}</span>
                        </div>
                    </div>
                    
                    <div class="profile-right">
                        <ElTag 
                            :type="getStatusInfo(account.status).type" 
                            size="default"
                            class="status-tag"
                        >
                            {{ getStatusInfo(account.status).text }}
                        </ElTag>
                        <ElButton 
                            :type="botStatus.running ? 'warning' : 'success'"
                            :loading="isLoading"
                            @click="toggleBot"
                            class="control-btn"
                        >
                            <ElIcon>
                                <VideoPause v-if="botStatus.running" />
                                <VideoPlay v-else />
                            </ElIcon>
                            {{ botStatus.running ? '停止Bot' : '启动Bot' }}
                        </ElButton>
                    </div>
                </div>
                
                <!-- Stats row -->
                <div class="stats-row">
                    <div class="stat-box level-box">
                        <div class="stat-icon level-icon">
                            <ElIcon :size="20"><TrendCharts /></ElIcon>
                        </div>
                        <div class="stat-content">
                            <span class="stat-label">等级</span>
                            <span class="stat-value">Lv.{{ botStatus.level }}</span>
                        </div>
                    </div>
                    
                    <div class="stat-box gold-box">
                        <div class="stat-icon gold-icon">
                            <ElIcon :size="20"><Coin /></ElIcon>
                        </div>
                        <div class="stat-content">
                            <span class="stat-label">金币</span>
                            <span class="stat-value">{{ botStatus.gold.toLocaleString() }}</span>
                        </div>
                    </div>
                    
                    <div class="stat-box exp-box">
                        <div class="stat-icon exp-icon">
                            <ElIcon :size="20"><User /></ElIcon>
                        </div>
                        <div class="stat-content">
                            <span class="stat-label">经验</span>
                            <span class="stat-value">{{ botStatus.exp.toLocaleString() }}</span>
                        </div>
                    </div>
                </div>
            </ElCard>
            
            <!-- Section 2: Function Toggles -->
            <ElCard class="toggles-card" shadow="never">
                <template #header>
                    <div class="section-header">
                        <span class="section-title">功能开关</span>
                    </div>
                </template>
                
                <div class="toggles-grid">
                    <div 
                        v-for="toggle in toggles" 
                        :key="toggle.key" 
                        class="toggle-item"
                    >
                        <div class="toggle-label-wrapper">
                            <span class="toggle-label">{{ toggle.label }}</span>
                            <ElTooltip :content="toggle.tooltip" placement="top">
                                <ElIcon class="info-icon"><InfoFilled /></ElIcon>
                            </ElTooltip>
                        </div>
                        <ElSwitch 
                            v-model="toggleValues[toggle.key]"
                            @change="(val: string | number | boolean) => handleToggleChange(toggle.key, Boolean(val))"
                        />
                    </div>
                </div>
            </ElCard>
            
            <!-- Section 3: Today's Statistics -->
            <ElCard class="today-stats-card" shadow="never">
                <template #header>
                    <div class="section-header">
                        <span class="section-title">今日统计</span>
                        <span class="section-date">{{ formatDate() }}</span>
                    </div>
                </template>
                
                <div class="stats-grid">
                    <div class="today-stat-item harvest">
                        <div class="stat-icon-circle">
                            <ElIcon :size="18"><Sunny /></ElIcon>
                        </div>
                        <span class="stat-count">{{ todayStats.harvest }}</span>
                        <span class="stat-name">收获</span>
                    </div>
                    
                    <div class="today-stat-item plant">
                        <div class="stat-icon-circle">
                            <ElIcon :size="18"><Grid /></ElIcon>
                        </div>
                        <span class="stat-count">{{ todayStats.plant }}</span>
                        <span class="stat-name">种植</span>
                    </div>
                    
                    <div class="today-stat-item steal">
                        <div class="stat-icon-circle">
                            <ElIcon :size="18"><Handbag /></ElIcon>
                        </div>
                        <span class="stat-count">{{ todayStats.steal }}</span>
                        <span class="stat-name">偷菜</span>
                    </div>
                    
                    <div class="today-stat-item gold">
                        <div class="stat-icon-circle">
                            <ElIcon :size="18"><Money /></ElIcon>
                        </div>
                        <span class="stat-count">{{ todayStats.gold.toLocaleString() }}</span>
                        <span class="stat-name">金币</span>
                    </div>
                    
                    <div class="today-stat-item help">
                        <div class="stat-icon-circle">
                            <ElIcon :size="18"><StarFilled /></ElIcon>
                        </div>
                        <span class="stat-count">{{ todayStats.help }}</span>
                        <span class="stat-name">帮忙</span>
                    </div>
                    
                    <div class="today-stat-item task">
                        <div class="stat-icon-circle">
                            <ElIcon :size="18"><CircleCheck /></ElIcon>
                        </div>
                        <span class="stat-count">{{ todayStats.task }}</span>
                        <span class="stat-name">任务</span>
                    </div>
                </div>
            </ElCard>
        </template>
    </div>
</template>

<style scoped>
.account-home {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
}

/* Profile Card */
.profile-card {
    background-color: var(--bg-card);
    border: none;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
    }

.profile-card :deep(.el-card__body) {
    padding: var(--space-6);
    }

.profile-header {
    display: flex;
    align-items: center;
    gap: var(--space-5);
    margin-bottom: var(--space-6);
    }

.profile-left {
    flex-shrink: 0;
    }

.avatar {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    background: linear-gradient(135deg, var(--primary) 0%, var(--primary-light) 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;
    font-weight: 700;
    color: #FFFFFF;
    }

.profile-middle {
    flex: 1;
    min-width: 0;
    }

.account-name {
    font-size: 20px;
    font-weight: 600;
    color: var(--text-heading);
    margin-bottom: 6px;
    }

.account-meta {
    display: flex;
    align-items: center;
    gap: 10px;
    }

.platform-tag {
    font-size: 11px;
    font-weight: 600;
    }

.user-id {
    font-size: 13px;
    color: var(--text-muted);
    }

.profile-right {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 10px;
    }

.status-tag {
    font-weight: 500;
    }

.control-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    }

/* Stats Row */
.stats-row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--space-4);
    }

.stat-box {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: var(--space-4);
    background-color: var(--bg-elevated);
    border-radius: var(--radius);
    border: none;
    }

.stat-icon {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    }

.level-icon {
    background-color: rgba(142, 142, 147, 0.1);
    color: var(--text-secondary);
    }

.gold-icon {
    background-color: var(--warning-bg);
    color: var(--gold);
    }

    .exp-icon {
    background-color: var(--primary-bg);
    color: var(--primary);
    }

    .stat-content {
    display: flex;
    flex-direction: column;
    gap: 2px;
    }

.stat-label {
    font-size: 12px;
    color: var(--text-muted);
    }

.stat-value {
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
    }

    .gold-box .stat-value {
    color: var(--gold);
}

    /* Toggles Card */
    .toggles-card {
    background-color: var(--bg-card);
    border: none;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
    }

    .toggles-card :deep(.el-card__header) {
    padding: var(--space-4) var(--space-6);
    border-bottom: 1px solid var(--border-light);
    }

    .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    }

    .section-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-heading);
    }

    .section-date {
    font-size: 13px;
    color: var(--text-muted);
    }

    .toggles-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: var(--space-3);
    }

    .toggle-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-3) var(--space-4);
    background-color: var(--bg-elevated);
    border-radius: var(--radius-sm);
    border: none;
    transition: all var(--transition);
    }

    .toggle-item:hover {
    background-color: var(--bg-hover);
    }

    .toggle-label-wrapper {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    }

    .toggle-label {
    font-size: 14px;
    color: var(--text-primary);
    }

    .info-icon {
    color: var(--text-muted);
    cursor: help;
    font-size: 14px;
    }

    .info-icon:hover {
    color: var(--text-secondary);
    }

    /* Today Stats Card */
.today-stats-card {
    background-color: var(--bg-card);
    border: none;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
}

.today-stats-card :deep(.el-card__header) {
    padding: var(--space-4) var(--space-6);
    border-bottom: 1px solid var(--border-light);
}

.stats-grid {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: var(--space-4);
}

.today-stat-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-4) var(--space-2);
    background-color: var(--bg-elevated);
    border-radius: var(--radius);
    border: none;
}

.stat-icon-circle {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
}

.today-stat-item.harvest .stat-icon-circle {
    background-color: var(--warning-bg);
    color: var(--gold);
}

.today-stat-item.plant .stat-icon-circle {
    background-color: var(--success-bg);
    color: var(--success);
}

.today-stat-item.steal .stat-icon-circle {
    background-color: rgba(139, 92, 246, 0.15);
    color: #8B5CF6;
}

.today-stat-item.gold .stat-icon-circle {
    background-color: var(--warning-bg);
    color: var(--gold);
}

.today-stat-item.help .stat-icon-circle {
    background-color: rgba(236, 72, 153, 0.15);
    color: #EC4899;
}

.today-stat-item.task .stat-icon-circle {
    background-color: var(--primary-bg);
    color: var(--primary);
}

.stat-count {
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
}

.today-stat-item.harvest .stat-count { color: var(--gold); }
.today-stat-item.plant .stat-count { color: var(--success); }
.today-stat-item.steal .stat-count { color: #8B5CF6; }
.today-stat-item.gold .stat-count { color: var(--gold); }
.today-stat-item.help .stat-count { color: #EC4899; }
.today-stat-item.task .stat-count { color: var(--primary); }

.stat-name {
    font-size: 12px;
    color: var(--text-muted);
}

/* Responsive */
@media (max-width: 1024px) {
    .stats-grid {
        grid-template-columns: repeat(3, 1fr);
    }
}

@media (max-width: 768px) {
    .profile-header {
        flex-wrap: wrap;
    }

    .profile-right {
        width: 100%;
        flex-direction: row;
        justify-content: space-between;
        align-items: center;
    }

    .stats-row {
        grid-template-columns: 1fr;
    }

    .toggles-grid {
        grid-template-columns: 1fr;
    }

    .stats-grid {
        grid-template-columns: repeat(2, 1fr);
    }
}
</style>
