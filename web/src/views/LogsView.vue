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
const logContainerRef = ref<HTMLElement | null>(null)

const fetchAccounts = async () => {
  try {
    const response = await accountApi.getAll()
    accounts.value = response.data
    if (accounts.value.length > 0 && !selectedAccountId.value) {
      selectedAccountId.value = accounts.value[0].id
    }
  } catch {
    // silently fail - accounts list will be empty
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
  } catch {
    // silently fail - will show empty log area
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
    } catch {
      // malformed message, skip
    }
  }
  
  websocket.onerror = () => {
    connected.value = false
  }
  
  websocket.onclose = () => {
    connected.value = false
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
  if (logContainerRef.value) {
    logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
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
            <span class="header-title">实时日志</span>
            
            <ElSelect 
              v-model="selectedAccountId" 
              placeholder="选择账号"
              class="account-select"
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
                  class="account-status-tag"
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
              <ElIcon :color="connected ? '#22C55E' : '#9CA3AF'" class="connection-icon">
                <Connection />
              </ElIcon>
              <span class="connection-text" :class="connected ? 'connected' : 'disconnected'">
                {{ connected ? '已连接' : '未连接' }}
              </span>
            </div>
            
            <ElButton 
              :icon="Refresh" 
              circle 
              size="small"
              class="refresh-btn"
              @click="handleRefresh"
            />
            
            <ElButton 
              size="small"
              class="scroll-btn"
              :class="{ 'is-active': autoScroll }"
              @click="autoScroll = !autoScroll"
            >
              自动滚动
            </ElButton>
            
            <ElButton 
              size="small"
              class="clear-btn"
              @click="clearLogs"
            >
              清空
            </ElButton>
          </div>
        </div>
      </template>

      <ElEmpty v-if="!selectedAccountId" description="请先选择一个账号" class="empty-state" />
      
      <div v-else class="log-container" ref="logContainerRef">
        <ElEmpty v-if="logs.length === 0 && !loading" description="暂无日志" class="empty-state" />
        
        <div v-else class="log-list">
          <div 
            v-for="log in logs" 
            :key="log.id" 
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

/* Card Styles */
.logs-card {
  border-radius: 16px;
  border: none;
  box-shadow: 0 1px 3px rgba(21, 128, 61, 0.06), 0 4px 16px rgba(21, 128, 61, 0.04);
}

.logs-card :deep(.el-card__header) {
  padding: 16px 24px;
  border-bottom: 1px solid #E5E7EB;
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

.header-title {
  font-size: 17px;
  font-weight: 600;
  color: #14532D;
}

/* Account Select */
.account-select {
  width: 200px;
}

.account-select :deep(.el-input__wrapper) {
  border-radius: 8px;
  box-shadow: 0 0 0 1px #D1D5DB;
}

.account-select :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #9CA3AF;
}

.account-select :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px rgba(21, 128, 61, 0.2), 0 0 0 1px #15803D !important;
}

.account-status-tag {
  margin-left: 8px;
}

/* Header Right */
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* Mode Switch */
.mode-switch {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 12px;
  height: 32px;
  background: #F9FAFB;
  border-radius: 8px;
}

.switch-label {
  font-size: 13px;
  font-weight: 500;
  color: #6B7280;
}

/* Connection Status */
.connection-status {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  height: 32px;
  background: #F9FAFB;
  border-radius: 8px;
}

.connection-icon {
  font-size: 16px;
}

.connection-text {
  font-size: 13px;
  font-weight: 500;
}

.connection-text.connected {
  color: #22C55E;
}

.connection-text.disconnected {
  color: #9CA3AF;
}

/* Buttons */
.refresh-btn {
  border-radius: 8px;
  border-color: #E5E7EB;
  color: #6B7280;
}

.refresh-btn:hover {
  border-color: #15803D;
  color: #15803D;
  background: rgba(21, 128, 61, 0.05);
}

.scroll-btn {
  border-radius: 8px;
  border-color: #E5E7EB;
  color: #6B7280;
  font-weight: 500;
}

.scroll-btn:hover {
  border-color: #15803D;
  color: #15803D;
  background: rgba(21, 128, 61, 0.05);
}

.scroll-btn.is-active {
  background: #15803D;
  border-color: #15803D;
  color: #FFFFFF;
}

.scroll-btn.is-active:hover {
  background: #166534;
  border-color: #166534;
}

.clear-btn {
  border-radius: 8px;
  border-color: #E5E7EB;
  color: #6B7280;
  font-weight: 500;
}

.clear-btn:hover {
  border-color: #DC2626;
  color: #DC2626;
  background: rgba(220, 38, 38, 0.05);
}

/* Empty State */
.empty-state {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

/* Log Container - Terminal Style */
.log-container {
  height: 100%;
  overflow-y: auto;
  background: #0C1222;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.7;
  padding: 16px 20px;
}

/* Custom Scrollbar */
.log-container::-webkit-scrollbar {
  width: 8px;
}

.log-container::-webkit-scrollbar-track {
  background: #0C1222;
}

.log-container::-webkit-scrollbar-thumb {
  background: #1E3A5F;
  border-radius: 4px;
}

.log-container::-webkit-scrollbar-thumb:hover {
  background: #2D4A6F;
}

/* Log List */
.log-list {
  min-height: 100%;
}

.log-entry {
  padding: 3px 0;
  white-space: pre-wrap;
  word-break: break-all;
}

/* Time - Soft Green */
.log-time {
  color: #4ADE80;
  margin-right: 12px;
  font-weight: 500;
}

/* Tag - Soft Blue */
.log-tag {
  color: #60A5FA;
  margin-right: 12px;
  font-weight: 500;
}

/* Log Levels */
.log-info .log-message {
  color: #94A3B8;
}

.log-warn .log-message {
  color: #FBBF24;
}

.log-error .log-message {
  color: #F87171;
}

.log-debug .log-message {
  color: #64748B;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .logs-card :deep(.el-card__header) {
    padding: 12px 16px;
  }
  
  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  
  .header-right {
    flex-wrap: wrap;
    gap: 8px;
  }
  
  .account-select {
    width: 100%;
  }
  
  .mode-switch,
  .connection-status {
    padding: 0 10px;
    height: 28px;
  }
}
</style>
