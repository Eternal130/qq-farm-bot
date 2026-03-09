<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAccountStore } from '@/stores/account'
import { useThemeStore } from '@/stores/theme'
import { 
  ElContainer, 
  ElAside, 
  ElHeader, 
  ElMain, 
  ElMenu, 
  ElMenuItem,
  ElDropdown,
  ElDropdownMenu,
  ElDropdownItem,
  ElIcon,
  ElSelect,
  ElOption,
  ElTag
} from 'element-plus'
import { 
  Odometer, 
  User, 
  Grid,
  Setting, 
  Document, 
  SwitchButton,
  Fold,
  Expand,
  TrendCharts,
  Timer,
  DataAnalysis,
  Sunny,
  Moon,
  PieChart
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const accountStore = useAccountStore()
const themeStore = useThemeStore()

const isCollapse = ref(false)
const activeMenu = computed(() => route.path)

const toggleSidebar = () => {
  isCollapse.value = !isCollapse.value
}

const handleLogout = () => {
  authStore.logout()
  router.push('/login')
}

const handleCommand = (command: string) => {
  if (command === 'logout') {
    handleLogout()
  }
}

// Account selection handling
const handleAccountChange = (accountId: number | null) => {
  accountStore.selectAccount(accountId)
  if (accountId !== null) {
    router.push(`/account/${accountId}/home`)
  }
}

// Check if account-specific routes should be enabled
const hasSelectedAccount = computed(() => accountStore.selectedAccountId !== null)

// Get account-specific route path
const getAccountRoute = (path: string) => {
  if (accountStore.selectedAccountId) {
    return `/account/${accountStore.selectedAccountId}${path}`
  }
  return ''
}

// Handle menu item click for account-specific routes
const handleMenuClick = (route: string) => {
  if (hasSelectedAccount.value) {
    router.push(getAccountRoute(route))
  }
}

// Load accounts on mount
onMounted(async () => {
  await accountStore.fetchAccounts()
  accountStore.loadPersistedAccount()
})

// Watch for account changes and update route if needed
watch(() => accountStore.selectedAccountId, (newId, oldId) => {
  // If we're on an account-specific route and the account changed, redirect to the new account's route
  if (oldId !== null && newId !== null && route.path.includes(`/account/${oldId}`)) {
    const newPath = route.path.replace(`/account/${oldId}`, `/account/${newId}`)
    router.push(newPath)
  }
})
</script>

<template>
  <ElContainer class="layout-container">
    <!-- Sidebar -->
    <ElAside 
      :width="isCollapse ? '72px' : '240px'" 
      class="sidebar"
    >
      <div class="logo">
        <div class="logo-content">
          <ElIcon class="logo-icon" :size="24"><Sunny /></ElIcon>
          <span class="logo-text" v-if="!isCollapse">QQ农场机器人</span>
        </div>
      </div>
      
      <!-- Account Selector -->
      <div class="account-selector" v-if="!isCollapse">
        <ElSelect
          v-model="accountStore.selectedAccountId"
          placeholder="选择账号"
          :teleported="false"
          @change="handleAccountChange"
          class="account-select"
        >
          <ElOption
            v-for="account in accountStore.accounts"
            :key="account.id"
            :label="account.name"
            :value="account.id"
          >
            <div class="account-option">
              <span class="account-name">{{ account.name }}</span>
              <ElTag 
                :type="account.status === 'running' ? 'success' : account.status === 'error' ? 'danger' : 'info'" 
                size="small"
              >
                {{ account.status === 'running' ? '运行' : account.status === 'error' ? '错误' : '停止' }}
              </ElTag>
            </div>
          </ElOption>
        </ElSelect>
      </div>
      
      <ElMenu
        :default-active="activeMenu"
        :collapse="isCollapse"
        :collapse-transition="false"
        router
        class="sidebar-menu"
      >
        <!-- Global Routes -->
        <ElMenuItem index="/dashboard">
          <ElIcon><Odometer /></ElIcon>
          <template #title>总览</template>
        </ElMenuItem>
        
        <!-- Account-specific Routes -->
        <ElMenuItem 
          :index="hasSelectedAccount ? getAccountRoute('/home') : ''"
          :disabled="!hasSelectedAccount"
          @click="hasSelectedAccount && handleMenuClick('/home')"
          :class="{ 'menu-item-disabled': !hasSelectedAccount }"
        >
          <ElIcon><User /></ElIcon>
          <template #title>首页</template>
        </ElMenuItem>
        
        <ElMenuItem 
          :index="hasSelectedAccount ? getAccountRoute('/lands') : ''"
          :disabled="!hasSelectedAccount"
          @click="hasSelectedAccount && handleMenuClick('/lands')"
          :class="{ 'menu-item-disabled': !hasSelectedAccount }"
        >
          <ElIcon><Grid /></ElIcon>
          <template #title>土地</template>
        </ElMenuItem>
        
        <ElMenuItem 
          :index="hasSelectedAccount ? getAccountRoute('/settings') : ''"
          :disabled="!hasSelectedAccount"
          @click="hasSelectedAccount && handleMenuClick('/settings')"
          :class="{ 'menu-item-disabled': !hasSelectedAccount }"
        >
          <ElIcon><Setting /></ElIcon>
          <template #title>配置</template>
        </ElMenuItem>
        
        <ElMenuItem 
          :index="hasSelectedAccount ? getAccountRoute('/logs') : ''"
          :disabled="!hasSelectedAccount"
          @click="hasSelectedAccount && handleMenuClick('/logs')"
          :class="{ 'menu-item-disabled': !hasSelectedAccount }"
        >
          <ElIcon><Document /></ElIcon>
          <template #title>日志</template>
        </ElMenuItem>
        
        <!-- Global Routes -->
        <ElMenuItem index="/crop-yield">
          <ElIcon><TrendCharts /></ElIcon>
          <template #title>种植排行</template>
        </ElMenuItem>
        
        <ElMenuItem index="/level-up-time">
          <ElIcon><Timer /></ElIcon>
          <template #title>升级计算</template>
        </ElMenuItem>
        
        <ElMenuItem index="/stats">
          <ElIcon><DataAnalysis /></ElIcon>
          <template #title>操作统计</template>
        </ElMenuItem>
        
        <ElMenuItem index="/data-summary">
          <ElIcon><PieChart /></ElIcon>
          <template #title>数据汇总</template>
        </ElMenuItem>
      </ElMenu>
      
      <!-- Collapse Toggle Button -->
      <div class="sidebar-footer">
        <button class="collapse-toggle" @click="toggleSidebar">
          <ElIcon :size="18">
            <Fold v-if="!isCollapse" />
            <Expand v-else />
          </ElIcon>
          <span v-if="!isCollapse" class="toggle-text">折叠</span>
        </button>
      </div>
    </ElAside>

    <ElContainer>
      <!-- Header -->
      <ElHeader class="header">
        <div class="header-left">
          <h1 class="page-title">{{ $route.meta.title || '农场管理' }}</h1>
        </div>
        
        <div class="header-right">
          <!-- Connection Status -->
          <div class="connection-status">
            <span class="status-dot"></span>
            <span class="status-text">已连接</span>
          </div>
          
          <!-- Theme Toggle -->
          <button 
            class="theme-toggle" 
            @click="themeStore.toggleTheme()" 
            :title="themeStore.theme === 'light' ? '切换暗色模式' : '切换亮色模式'"
          >
            <ElIcon :size="20">
              <Moon v-if="themeStore.theme === 'light'" />
              <Sunny v-else />
            </ElIcon>
          </button>
          
          <ElDropdown @command="handleCommand" trigger="click">
            <div class="user-info">
              <div class="user-avatar">
                <ElIcon :size="18"><User /></ElIcon>
              </div>
              <div class="user-details">
                <span class="username">{{ authStore.user?.username || '用户' }}</span>
                <ElTag v-if="authStore.user?.is_admin" type="primary" size="small" class="role-tag">管理员</ElTag>
              </div>
            </div>
            <template #dropdown>
              <ElDropdownMenu>
                <ElDropdownItem command="logout">
                  <ElIcon><SwitchButton /></ElIcon>
                  <span>退出登录</span>
                </ElDropdownItem>
              </ElDropdownMenu>
            </template>
          </ElDropdown>
        </div>
      </ElHeader>

      <!-- Main Content -->
      <ElMain class="main-content">
        <RouterView />
      </ElMain>
    </ElContainer>
  </ElContainer>
</template>

<style scoped>
.layout-container {
  height: 100vh;
  width: 100%;
}

/* === Sidebar Styles === */
.sidebar {
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border);
  transition: width var(--transition-slow);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-sidebar);
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 16px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.logo-content {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-icon {
  color: var(--primary);
}

.logo-text {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-heading);
  white-space: nowrap;
  letter-spacing: -0.02em;
}

/* === Account Selector === */
.account-selector {
  padding: 12px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.account-select {
  width: 100%;
}

.account-select :deep(.el-input__wrapper) {
  background-color: var(--bg-elevated) !important;
  box-shadow: 0 0 0 1px var(--border) inset !important;
  border-radius: var(--radius-md);
}

.account-select :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px var(--border-light) inset !important;
}

.account-select :deep(.el-input__inner) {
  color: var(--text-primary) !important;
}

.account-select :deep(.el-input__inner::placeholder) {
  color: var(--text-muted) !important;
}

.account-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.account-name {
  color: var(--text-primary);
}

/* === Menu Styles === */
.sidebar-menu {
  border-right: none !important;
  background-color: transparent !important;
  flex: 1;
  padding: 8px;
  overflow-y: auto;
  --el-menu-bg-color: transparent;
  --el-menu-text-color: var(--text-secondary);
  --el-menu-active-color: var(--primary);
  --el-menu-hover-bg-color: var(--bg-hover);
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 224px;
}

:deep(.el-menu-item) {
  height: 48px;
  line-height: 48px;
  margin: 4px 0;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  transition: all var(--transition);
}

:deep(.el-menu-item .el-icon) {
  color: var(--text-muted);
  transition: color var(--transition);
}

:deep(.el-menu-item:hover) {
  background-color: var(--bg-hover) !important;
  color: var(--text-primary);
}

:deep(.el-menu-item:hover .el-icon) {
  color: var(--text-primary);
}

:deep(.el-menu-item.is-active) {
  background-color: var(--primary-bg) !important;
  color: var(--primary) !important;
  font-weight: 500;
  border-radius: var(--radius-md);
}

:deep(.el-menu-item.is-active .el-icon) {
  color: var(--primary);
}

:deep(.el-menu-item.menu-item-disabled) {
  opacity: 0.4;
  cursor: not-allowed;
}

:deep(.el-menu-item.menu-item-disabled:hover) {
  background-color: transparent !important;
  color: var(--text-secondary);
}

/* === Sidebar Footer === */
.sidebar-footer {
  padding: 12px;
  border-top: 1px solid var(--border);
  flex-shrink: 0;
}

.collapse-toggle {
  width: 100%;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--transition);
}

.collapse-toggle:hover {
  background: var(--bg-hover);
  border-color: var(--border-light);
  color: var(--text-primary);
}

.toggle-text {
  font-size: 14px;
}

/* === Header Styles === */
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  backdrop-filter: blur(20px);
  background: var(--bg-header);
  border-bottom: 1px solid var(--border);
  padding: 0 24px;
  height: 64px;
  box-shadow: var(--shadow-xs);
}

.header-left {
  display: flex;
  align-items: center;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-heading);
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 20px;
}

/* Connection Status */
.connection-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: var(--success-bg);
  border-radius: var(--radius-full);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--success);
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.status-text {
  font-size: 13px;
  color: var(--success);
  font-weight: 500;
}

/* Theme Toggle */
.theme-toggle {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--transition);
}

.theme-toggle:hover {
  background: var(--bg-hover);
  border-color: var(--border-light);
  color: var(--warning);
}

/* User Info */
.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  transition: all var(--transition);
}

.user-info:hover {
  background: var(--bg-hover);
}

.user-avatar {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--primary) 0%, var(--primary-hover) 100%);
  border-radius: var(--radius-md);
  color: #fff;
}

.user-details {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.username {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.role-tag {
  font-size: 11px;
  padding: 0 6px;
  height: 18px;
  line-height: 16px;
}

/* === Main Content === */
.main-content {
  background-color: var(--bg-page);
  padding: 24px;
  overflow-y: auto;
  min-height: 0;
}

/* === Dropdown Menu === */
:deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  color: var(--text-secondary);
}

:deep(.el-dropdown-menu__item:hover) {
  background-color: var(--bg-hover);
  color: var(--text-primary);
}

/* === Responsive === */
@media (max-width: 768px) {
  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    height: 100vh;
    z-index: 1000;
    box-shadow: var(--shadow-lg);
  }
  
  .main-content {
    padding: 16px;
  }
  
  .page-title {
    font-size: 16px;
  }
  
  .username {
    display: none;
  }
  
  .role-tag {
    display: none;
  }
  
  .connection-status {
    display: none;
  }
  
  .theme-toggle {
    display: none;
  }
}
</style>
