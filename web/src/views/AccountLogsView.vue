<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { logsApi, createLogWebSocket, type LogEntry } from '@/api'
import { 
  ElCard, 
  ElSelect, 
  ElOption, 
  ElButton,
  ElEmpty
} from 'element-plus'
import { Delete } from '@element-plus/icons-vue'

const route = useRoute()
const logs = ref<LogEntry[]>([])
const loading = ref(false)
const autoScroll = ref(true)
const categoryFilter = ref<string>('')
const levelFilter = ref<string>('')

let websocket: WebSocket | null = null
const logContainerRef = ref<HTMLElement | null>(null)

// Account ID from route
const accountId = computed(() => {
  const id = route.params.id
  return Array.isArray(id) ? parseInt(id[0]) : parseInt(id)
})

// Category mapping for filtering
const categoryMap: Record<string, string[]> = {
  '农场': ['农场'],
  '好友': ['好友'],
  '仓库': ['仓库'],
  '施肥': ['施肥'],
  '任务': ['任务'],
  '系统': ['系统', '登录', '连接']
}

// Filtered logs
const filteredLogs = computed(() => {
  let result = logs.value
  
  // Filter by category
  if (categoryFilter.value) {
    const tags = categoryMap[categoryFilter.value] || []
    result = result.filter(log => tags.includes(log.tag))
  }
  
  // Filter by level
  if (levelFilter.value) {
    const levelMap: Record<string, string[]> = {
      'INF': ['info'],
      'WRN': ['warn'],
      'ERR': ['error']
    }
    const levels = levelMap[levelFilter.value] || []
    result = result.filter(log => levels.includes(log.level))
  }
  
  return result
})

// Log count display
const logCount = computed(() => filteredLogs.value.length)

// Reverse mapping for tag to category
const tagToCategory = computed(() => {
  const map: Record<string, string> = {}
  Object.entries(categoryMap).forEach(([category, tags]) => {
    tags.forEach(tag => {
      map[tag] = category
    })
  })
  return map
})

// Get category tag color
const getCategoryColor = (tag: string): string => {
  const category = tagToCategory.value[tag]
  const colorMap: Record<string, string> = {
    '农场': '#22C55E',
    '好友': '#A855F7',
    '仓库': '#F59E0B',
    '施肥': '#06B6D4',
    '任务': '#3B82F6',
    '系统': '#6B7280'
  }
  return colorMap[category || '系统'] || '#6B7280'
}

// Get level badge style
const getLevelBadgeClass = (level: string): string => {
  const classes: Record<string, string> = {
    info: 'level-info',
    warn: 'level-warn',
    error: 'level-error',
    debug: 'level-debug'
  }
  return classes[level] || 'level-info'
}

// Format timestamp
const formatTime = (timestamp: string): string => {
  const date = new Date(timestamp)
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  const seconds = date.getSeconds().toString().padStart(2, '0')
  return `${hours}:${minutes}:${seconds}`
}

const fetchHistoricalLogs = async () => {
  if (!accountId.value) return
  
  loading.value = true
  try {
    const response = await logsApi.getHistorical(accountId.value, 500)
    logs.value = response.data.reverse()
    if (autoScroll.value) {
      await nextTick()
      scrollToBottom()
    }
  } catch {
    // silently fail
  } finally {
    loading.value = false
  }
}

const connectWebSocket = () => {
  if (!accountId.value) return
  
  disconnectWebSocket()
  
  websocket = createLogWebSocket(accountId.value)
  
  websocket.onmessage = (event) => {
    try {
      const log: LogEntry = JSON.parse(event.data)
      // Filter: only show logs for this account
      if (log.account_id !== accountId.value) return
      
      logs.value.push(log)
      
      // Limit log count
      if (logs.value.length > 1000) {
        logs.value.shift()
      }
      
      if (autoScroll.value) {
        nextTick(() => scrollToBottom())
      }
    } catch {
      // malformed message, skip
    }
  }
}

const disconnectWebSocket = () => {
  if (websocket) {
    websocket.close()
    websocket = null
  }
}

const scrollToBottom = () => {
  if (logContainerRef.value) {
    logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
  }
}

const clearLogs = () => {
  logs.value = []
}

onMounted(() => {
  fetchHistoricalLogs()
  connectWebSocket()
})

onUnmounted(() => {
  disconnectWebSocket()
})
</script>

<template>
  <div class="account-logs">
    <ElCard shadow="never" class="logs-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <span class="header-title">运行日志</span>
            
            <!-- Category Filter -->
            <ElSelect 
              v-model="categoryFilter"
              placeholder="全部分类"
              class="filter-select"
              clearable
            >
              <ElOption label="全部分类" value="" />
              <ElOption label="农场" value="农场" />
              <ElOption label="好友" value="好友" />
              <ElOption label="仓库" value="仓库" />
              <ElOption label="施肥" value="施肥" />
              <ElOption label="任务" value="任务" />
              <ElOption label="系统" value="系统" />
            </ElSelect>

            <!-- Level Filter -->
            <ElSelect 
              v-model="levelFilter"
              placeholder="全部级别"
              class="filter-select"
              clearable
            >
              <ElOption label="全部级别" value="" />
              <ElOption label="INF" value="INF" />
              <ElOption label="WRN" value="WRN" />
              <ElOption label="ERR" value="ERR" />
            </ElSelect>
          </div>
          
          <div class="header-right">
            <!-- Auto Scroll Toggle -->
            <ElButton 
              size="small"
              class="control-btn"
              :class="{ 'is-active': autoScroll }"
              @click="autoScroll = !autoScroll"
            >
              自动滚动
            </ElButton>
            
            <!-- Clear Button -->
            <ElButton 
              :icon="Delete"
              size="small"
              class="control-btn clear-btn"
              @click="clearLogs"
            >
              清空
            </ElButton>

            <!-- Log Count -->
            <div class="log-count">
              <span class="count-number">{{ logCount }}</span>
              <span class="count-label">条</span>
            </div>
          </div>
        </div>
      </template>

      <!-- Log Container -->
      <div class="log-container" ref="logContainerRef">
        <ElEmpty v-if="filteredLogs.length === 0 && !loading" description="暂无日志" class="empty-state" />
        
        <div v-else class="log-list">
          <div 
            v-for="log in filteredLogs" 
            :key="log.id" 
            class="log-entry"
          >
            <!-- Timestamp -->
            <span class="log-time">{{ formatTime(log.created_at) }}</span>
            
            <!-- Level Badge -->
            <span 
              class="log-level" 
              :class="getLevelBadgeClass(log.level)"
            >
              {{ log.level.toUpperCase().slice(0, 3) }}
            </span>
            
            <!-- Category Tag -->
            <span 
              class="log-category"
              :style="{ color: getCategoryColor(log.tag) }"
            >
              [{{ log.tag }}]
            </span>
            
            <!-- Message -->
            <span class="log-message" :class="'level-' + log.level">
              {{ log.message }}
            </span>
          </div>
        </div>
      </div>
    </ElCard>
  </div>
</template>

<style scoped>
.account-logs {
  padding: 0;
  height: 100%;
}

/* Card Styles */
.logs-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  height: 100%;
  display: flex;
  flex-direction: column;
}

.logs-card :deep(.el-card__header) {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
}

.logs-card :deep(.el-card__body) {
  padding: 0;
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-heading);
}

/* Select Styles */
.filter-select {
  width: 120px;
}

/* Control Buttons */
.control-btn {
  border-radius: var(--radius-sm);
  border-color: var(--border);
  background: var(--bg-elevated);
  color: var(--text-secondary);
  font-weight: 500;
}

.control-btn:hover {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-bg);
}

.control-btn.is-active {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}

.control-btn.is-active:hover {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}

.clear-btn:hover {
  border-color: var(--danger);
  color: var(--danger);
  background: var(--danger-bg);
}

/* Log Count */
.log-count {
  display: flex;
  align-items: baseline;
  gap: 4px;
  padding: 6px 12px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
}

.count-number {
  font-size: 16px;
  font-weight: 700;
  color: var(--primary);
  font-family: 'Courier New', monospace;
}

.count-label {
  font-size: 12px;
  color: var(--text-muted);
}

/* Empty State */
.empty-state {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
}

.empty-state :deep(.el-empty__description) {
  color: var(--text-muted);
}

/* Log Container - Terminal Style (always dark for terminal aesthetic) */
.log-container {
  flex: 1;
  overflow-y: auto;
  background: var(--bg-code);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.6;
  padding: 16px 20px;
}

/* Custom Scrollbar for Terminal */
.log-container::-webkit-scrollbar {
  width: 8px;
}

.log-container::-webkit-scrollbar-track {
  background: var(--bg-code);
}

.log-container::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 4px;
}

.log-container::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}

/* Log List */
.log-list {
  min-height: 100%;
}

.log-entry {
  padding: 4px 0;
  white-space: pre-wrap;
  word-break: break-all;
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

/* Timestamp - Theme aware */
.log-time {
  color: var(--success);
  font-weight: 500;
  flex-shrink: 0;
  min-width: 75px;
}

/* Level Badges */
.log-level {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  flex-shrink: 0;
  min-width: 32px;
}

.log-level.level-info {
  background: var(--primary-bg);
  color: var(--primary-light);
}

.log-level.level-warn {
  background: var(--warning-bg);
  color: var(--warning);
}

.log-level.level-error {
  background: var(--danger-bg);
  color: var(--danger);
}

.log-level.level-debug {
  background: rgba(142, 142, 147, 0.1);
  color: var(--text-muted);
}

/* Category Tag */
.log-category {
  font-weight: 600;
  flex-shrink: 0;
}

/* Message Text */
.log-message {
  color: var(--text-secondary);
  flex: 1;
}

.log-message.level-info {
  color: var(--text-primary);
}

.log-message.level-warn {
  color: var(--warning);
}

.log-message.level-error {
  color: var(--danger);
}

.log-message.level-debug {
  color: var(--text-muted);
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .header-left {
    width: 100%;
  }
  
  .header-right {
    width: 100%;
    justify-content: flex-end;
  }
  
  .filter-select {
    width: 50%;
  }
  
  .logs-card :deep(.el-card__header) {
    padding: 12px 16px;
  }
  
  .header-left {
    gap: 8px;
  }
  
  .header-right {
    gap: 6px;
    flex-wrap: wrap;
  }
  
  .log-count {
    padding: 4px 8px;
  }
  
  .control-btn {
    padding: 6px 10px;
    font-size: 12px;
  }
  
  .log-container {
    padding: 12px;
    font-size: 12px;
  }
  
  .log-time {
    min-width: 65px;
    font-size: 11px;
  }
  
  .log-level {
    font-size: 9px;
    min-width: 28px;
  }
}
</style>
