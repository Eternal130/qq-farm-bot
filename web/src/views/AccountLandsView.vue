<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { 
  dashboardApi, 
  type LandStatus,
  type DashboardStats
} from '@/api'
import { 
  ElTag, 
  ElButton, 
  ElIcon,
  ElEmpty
} from 'element-plus'
import { 
  Refresh, 
  Lock
} from '@element-plus/icons-vue'

const route = useRoute()

// Account ID from route
const accountId = computed(() => {
  const id = route.params.id
  return typeof id === 'string' ? parseInt(id, 10) : 0
})

// Account name
const accountName = ref('')

// Lands data
const lands = ref<LandStatus[]>([])
const lastUpdate = ref<Date | null>(null)
const isLoading = ref(false)
let refreshInterval: number | null = null

// Compute land statistics
const landStats = computed(() => {
  const harvestable = lands.value.filter(l => l.unlocked && l.phase === '成熟').length
  const growing = lands.value.filter(l => l.unlocked && l.phase && l.phase !== '成熟' && l.phase !== '枯萎').length
  const empty = lands.value.filter(l => l.unlocked && !l.crop_name).length
  const locked = lands.value.filter(l => !l.unlocked).length
  const needsAttention = lands.value.filter(l => 
    l.unlocked && l.crop_name && 
    (l.phase === '枯萎')
  ).length
  
  return { harvestable, growing, empty, locked, needsAttention }
})

// Get land level name
const getLandLevelName = (level: number): string => {
  const names: Record<number, string> = { 
    1: '黄土', 
    2: '红土', 
    3: '黑土', 
    4: '金土' 
  }
  return names[level] || `Lv.${level}`
}

// Get phase tag type
const getPhaseType = (phase: string | undefined): 'success' | 'info' | 'warning' | 'danger' | 'primary' => {
  if (!phase) return 'info'
  if (phase === '成熟') return 'success'
  if (phase === '枯萎') return 'danger'
  if (phase === '开花') return 'warning'
  if (['发芽', '小叶', '大叶'].includes(phase)) return 'primary'
  return 'info'
}

// Format time
const formatTime = (date: Date | null): string => {
  if (!date) return '--:--:--'
  return date.toLocaleTimeString('zh-CN', { 
    hour: '2-digit', 
    minute: '2-digit', 
    second: '2-digit' 
  })
}

// Fetch lands data
const fetchLands = async () => {
  if (accountId.value === 0) return
  
  isLoading.value = true
  try {
    const response = await dashboardApi.getStats()
    const data: DashboardStats = response.data
    
    const found = data.accounts.find(a => a.id === accountId.value)
    if (found) {
      accountName.value = found.name
      lands.value = found.lands || []
      lastUpdate.value = new Date()
    }
  } catch {
    // silently fail
  } finally {
    isLoading.value = false
  }
}

// Lifecycle
onMounted(() => {
  fetchLands()
  refreshInterval = window.setInterval(fetchLands, 5000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <div class="account-lands">
    <!-- Top section: Summary stats row -->
    <div class="stats-header">
      <div class="stat-badges">
        <div class="stat-badge harvestable">
          <span class="badge-count">{{ landStats.harvestable }}</span>
          <span class="badge-label">可收获</span>
        </div>
        
        <div class="stat-badge growing">
          <span class="badge-count">{{ landStats.growing }}</span>
          <span class="badge-label">生长中</span>
        </div>
        
        <div class="stat-badge empty">
          <span class="badge-count">{{ landStats.empty }}</span>
          <span class="badge-label">空地</span>
        </div>
        
        <div class="stat-badge attention">
          <span class="badge-count">{{ landStats.needsAttention }}</span>
          <span class="badge-label">需处理</span>
        </div>
        
        <div class="stat-badge locked">
          <span class="badge-count">{{ landStats.locked }}</span>
          <span class="badge-label">未解锁</span>
        </div>
      </div>
      
      <ElButton 
        type="primary" 
        :icon="Refresh" 
        :loading="isLoading"
        @click="fetchLands"
        class="refresh-btn"
      >
        刷新
      </ElButton>
    </div>
    
    <!-- Title section -->
    <div class="title-section">
      <h2 class="page-title">土地状态</h2>
      <p class="page-subtitle">
        共 {{ lands.length }} 块土地
        <span v-if="lastUpdate" class="update-time">
          更新于 {{ formatTime(lastUpdate) }}
        </span>
      </p>
    </div>
    
    <!-- Empty state -->
    <ElEmpty 
      v-if="lands.length === 0" 
      description="暂无土地数据" 
      class="empty-state"
    />
    
    <!-- Land grid -->
    <div v-else class="land-grid">
      <div 
        v-for="land in lands" 
        :key="land.id" 
        class="land-card"
        :class="{ 'land-locked': !land.unlocked }"
      >
        <!-- Card header -->
        <div class="land-header">
          <span class="land-id">土地 #{{ land.id }}</span>
          <span class="land-level" :class="`level-${land.level}`">
            {{ getLandLevelName(land.level) }}
          </span>
        </div>
        
        <!-- Locked land -->
        <template v-if="!land.unlocked">
          <div class="locked-content">
            <ElIcon class="lock-icon"><Lock /></ElIcon>
            <ElTag type="danger" size="small" class="locked-tag">未解锁</ElTag>
          </div>
        </template>
        
        <!-- Unlocked land -->
        <template v-else>
          <div class="land-crop">
            {{ land.crop_name || '空地' }}
          </div>
          
          <ElTag 
            v-if="land.phase"
            :type="getPhaseType(land.phase)" 
            size="small"
            class="phase-tag"
          >
            {{ land.phase }}
          </ElTag>
          <ElTag 
            v-else 
            type="info" 
            size="small"
            class="phase-tag"
          >
            空地
          </ElTag>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.account-lands {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Stats Header */
.stats-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
}

.stat-badges {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.stat-badge {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 10px 16px;
  border-radius: var(--radius-md);
  background-color: var(--bg-card);
  border: 1px solid var(--border);
  min-width: 70px;
}

.badge-count {
  font-size: 20px;
  font-weight: 700;
  line-height: 1;
  margin-bottom: 4px;
}

.badge-label {
  font-size: 11px;
  color: var(--text-muted);
}

.stat-badge.harvestable .badge-count { color: var(--success); }
.stat-badge.growing .badge-count { color: var(--primary); }
.stat-badge.empty .badge-count { color: var(--text-muted); }
.stat-badge.attention .badge-count { color: var(--warning); }
.stat-badge.locked .badge-count { color: var(--danger); }

.refresh-btn {
  flex-shrink: 0;
}

/* Title Section */
.title-section {
  margin-bottom: 4px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-heading);
  margin: 0 0 4px 0;
}

.page-subtitle {
  font-size: 13px;
  color: var(--text-muted);
  margin: 0;
}

.update-time {
  margin-left: 8px;
  color: var(--text-secondary);
}

/* Empty State */
.empty-state {
  background-color: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 40px;
}

/* Land Grid */
.land-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
}

@media (min-width: 1400px) {
  .land-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

/* Land Card */
.land-card {
  background-color: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 16px;
  transition: all var(--transition-fast) ease;
}

.land-card:hover {
  border-color: var(--border-focus);
  background-color: var(--bg-elevated);
}

.land-card.land-locked {
  opacity: 0.6;
}

.land-card.land-locked:hover {
  opacity: 0.8;
}

/* Land Header */
.land-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.land-id {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
}

.land-level {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-xs);
}

.land-level.level-1 { 
  background-color: rgba(180, 83, 9, 0.15); 
  color: #D97706; 
}

.land-level.level-2 { 
  background-color: rgba(220, 38, 38, 0.15); 
  color: var(--danger); 
}

.land-level.level-3 { 
  background-color: rgba(30, 58, 138, 0.15); 
  color: var(--primary); 
}

.land-level.level-4 { 
  background-color: rgba(202, 138, 4, 0.15); 
  color: var(--gold); 
}

/* Locked Content */
.locked-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
}

.lock-icon {
  font-size: 24px;
  color: var(--text-muted);
}

.locked-tag {
  font-size: 11px;
}

/* Land Crop */
.land-crop {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Phase Tag */
.phase-tag {
  font-size: 11px !important;
}

/* Responsive */
@media (max-width: 768px) {
  .stats-header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .stat-badges {
    justify-content: center;
  }
  
  .refresh-btn {
    width: 100%;
  }
  
  .land-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 480px) {
  .land-grid {
    grid-template-columns: 1fr;
  }
  
  .stat-badge {
    min-width: 60px;
    padding: 8px 12px;
  }
  
  .badge-count {
    font-size: 18px;
  }
}
</style>
