<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { accountApi, logsApi, createLogWebSocket, type Account, type LogEntry } from '@/api'
import { 
  ElCard, 
  ElSelect, 
  ElOption, 
  ElSwitch,
  ElButton,
  ElEmpty,
  ElIcon,
  ElTag
} from 'element-plus'
import { Refresh, Connection } from '@element-plus/icons-vue'

const accounts = ref<Account[]>([])
const selectedAccountId = ref<number | null>(null)
const logs = ref<LogEntry[]>([])
const realtimeMode = ref(true)
const loading = ref(false)
const connected = ref(false)
const autoScroll = ref(true)

let websocket: WebSocket | null = null
let logContainer: HTMLElement | null = null

const fetchAccounts = async () => {
  try {
    const response = await accountApi.getAll()
    accounts.value = response.data
    if (accounts.value.length > 0 && !selectedAccountId.value) {
      selectedAccountId.value = accounts.value[0].id
    }
  } catch (error) {
    console.error('Failed to fetch accounts:', error)
  }
}

const fetchHistoricalLogs = async () => {
  if (!selectedAccountId.value) return
  
  loading.value = true
  try {
    const response = await logsApi.getHistorical(selectedAccountId.value, 200)
    logs.value = response.data.reverse()
    if (autoScroll.value) {
      await nextTick()
      scrollToBottom()
    }
  } catch (error) {
    console.error('Failed to fetch logs:', error)
  } finally {
    loading.value = false
  }
}

const connectWebSocket = () => {
  if (!selectedAccountId.value) return
  
  disconnectWebSocket()
  
  logs.value = []
  websocket = createLogWebSocket(selectedAccountId.value)
  
  websocket.onopen = () => {
    connected.value = true
    console.log('WebSocket connected')
  }
  
  websocket.onmessage = (event) => {
    try {
      const log: LogEntry = JSON.parse(event.data)
      logs.value.push(log)
      
      // Limit log count
      if (logs.value.length > 500) {
        logs.value.shift()
      }
      
      if (autoScroll.value) {
        nextTick(() => scrollToBottom())
      }
    } catch (error) {
      console.error('Failed to parse log:', error)
    }
  }
  
  websocket.onerror = (error) => {
    console.error('WebSocket error:', error)
    connected.value = false
  }
  
  websocket.onclose = () => {
    connected.value = false
    console.log('WebSocket disconnected')
  }
}

const disconnectWebSocket = () => {
  if (websocket) {
    websocket.close()
    websocket = null
  }
  connected.value = false
}

const scrollToBottom = () => {
  if (logContainer) {
    logContainer.scrollTop = logContainer.scrollHeight
  }
}

const clearLogs = () => {
  logs.value = []
}

const getLevelClass = (level: string): string => {
  const classes: Record<string, string> = {
    info: 'log-info',
    warn: 'log-warn',
    error: 'log-error',
    debug: 'log-debug'
  }
  return classes[level] || 'log-info'
}

const formatTime = (timestamp: string): string => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('zh-CN', { 
    hour: '2-digit', 
    minute: '2-digit', 
    second: '2-digit' 
  })
}

const handleAccountChange = () => {
  logs.value = []
  if (realtimeMode.value) {
    connectWebSocket()
  } else {
    fetchHistoricalLogs()
  }
}

const handleModeChange = () => {
  if (realtimeMode.value) {
    connectWebSocket()
  } else {
    disconnectWebSocket()
    fetchHistoricalLogs()
  }
}

const handleRefresh = () => {
  if (realtimeMode.value) {
    logs.value = []
    connectWebSocket()
  } else {
    fetchHistoricalLogs()
  }
}

watch(selectedAccountId, () => {
  handleAccountChange()
})

watch(realtimeMode, () => {
  handleModeChange()
})

onMounted(() => {
  fetchAccounts().then(() => {
    if (realtimeMode.value && selectedAccountId.value) {
      connectWebSocket()
    }
  })
  
  // Get log container reference
  logContainer = document.querySelector('.log-container')
})

onUnmounted(() => {
  disconnectWebSocket()
})
</script>

<template>
  <div class="logs-view">
    <ElCard shadow="never" class="logs-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <span>实时日志</span>
            
            <ElSelect 
              v-model="selectedAccountId" 
              placeholder="选择账号"
              style="width: 200px"
              clearable
            >
              <ElOption
                v-for="account in accounts"
                :key="account.id"
                :label="account.name"
                :value="account.id"
              >
                <span>{{ account.name }}</span>
                <ElTag 
                  size="small" 
                  :type="account.status === 'running' ? 'success' : 'info'"
                  style="margin-left: 8px"
                >
                  {{ account.status === 'running' ? '运行中' : '已停止' }}
                </ElTag>
              </ElOption>
            </ElSelect>
          </div>
          
          <div class="header-right">
            <div class="mode-switch">
              <span class="switch-label">实时模式</span>
              <ElSwitch v-model="realtimeMode" />
            </div>
            
            <div class="connection-status" v-if="realtimeMode">
              <ElIcon :color="connected ? '#67c23a' : '#909399'">
                <Connection />
              </ElIcon>
              <span :class="connected ? 'connected' : 'disconnected'">
                {{ connected ? '已连接' : '未连接' }}
              </span>
            </div>
            
            <ElButton 
              :icon="Refresh" 
              circle 
              size="small"
              @click="handleRefresh"
            />
            
            <ElButton 
              size="small"
              @click="autoScroll = !autoScroll"
              :type="autoScroll ? 'primary' : 'default'"
            >
              自动滚动
            </ElButton>
            
            <ElButton 
              size="small"
              @click="clearLogs"
            >
              清空
            </ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!selectedAccountId" description="请先选择一个账号" />
      
      <div v-else class="log-container" ref="logContainer">
        <ElEmpty v-if="logs.length === 0 && !loading" description="暂无日志" />
        
        <div v-else class="log-list">
          <div 
            v-for="(log, index) in logs" 
            :key="index" 
            class="log-entry"
            :class="getLevelClass(log.level)"
          >
            <span class="log-time">{{ formatTime(log.created_at) }}</span>
            <span class="log-tag">[{{ log.tag }}]</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
        </div>
      </div>
    </ElCard>
  </div>
</template>

<style scoped>
.logs-view {
  padding: 0;
}

.logs-card {
  border-radius: 8px;
}

.logs-card :deep(.el-card__body) {
  padding: 0;
  height: calc(100vh - 240px);
  min-height: 500px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.mode-switch {
  display: flex;
  align-items: center;
  gap: 8px;
}

.switch-label {
  font-size: 14px;
  color: #606266;
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
}

.connected {
  color: #67c23a;
}

.disconnected {
  color: #909399;
}

.log-container {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
  background-color: #1e1e1e;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.log-list {
  height: 100%;
}

.log-entry {
  padding: 2px 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-time {
  color: #6a9955;
  margin-right: 8px;
}

.log-tag {
  color: #569cd6;
  margin-right: 8px;
}

.log-message {
  color: #d4d4d4;
}

/* Log level colors in dark theme */
.log-info .log-message {
  color: #d4d4d4;
}

.log-warn .log-message {
  color: #dcdcaa;
}

.log-error .log-message {
  color: #f14c4c;
}

.log-debug .log-message {
  color: #808080;
}

/* Custom scrollbar for dark log container */
.log-container::-webkit-scrollbar {
  width: 8px;
}

.log-container::-webkit-scrollbar-track {
  background: #2d2d2d;
}

.log-container::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 4px;
}

.log-container::-webkit-scrollbar-thumb:hover {
  background: #666;
}

@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .header-right {
    flex-wrap: wrap;
  }
}
</style>
