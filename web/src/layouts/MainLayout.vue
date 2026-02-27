<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
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
  ElIcon
} from 'element-plus'
import { 
  Odometer, 
  User, 
  Document, 
  SwitchButton,
  Fold,
  Expand,
  TrendCharts
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

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
          <span class="logo-text" v-if="!isCollapse">🌾 农场管理</span>
          <span class="logo-icon-only" v-else>🌾</span>
        </div>
      </div>
      
      <ElMenu
        :default-active="activeMenu"
        :collapse="isCollapse"
        :collapse-transition="false"
        router
        class="sidebar-menu"
      >
        <ElMenuItem index="/dashboard">
          <ElIcon><Odometer /></ElIcon>
          <template #title>仪表盘</template>
        </ElMenuItem>
        
        <ElMenuItem index="/accounts">
          <ElIcon><User /></ElIcon>
          <template #title>账号管理</template>
        </ElMenuItem>
        
        <ElMenuItem index="/logs">
          <ElIcon><Document /></ElIcon>
          <template #title>实时日志</template>
        </ElMenuItem>
        
        <ElMenuItem index="/crop-yield">
          <ElIcon><TrendCharts /></ElIcon>
          <template #title>作物收益</template>
        </ElMenuItem>
      </ElMenu>
      
      <!-- Collapse Toggle Button (Mobile-friendly) -->
      <div class="sidebar-footer">
        <button class="collapse-toggle" @click="toggleSidebar">
          <ElIcon :size="18">
            <Fold v-if="!isCollapse" />
            <Expand v-else />
          </ElIcon>
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
          <ElDropdown @command="handleCommand" trigger="click">
            <div class="user-info">
              <div class="user-avatar">
                <ElIcon :size="18"><User /></ElIcon>
              </div>
              <span class="username">{{ authStore.user?.username || '用户' }}</span>
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
  background: #FFFFFF;
  border-right: 1px solid #E5E7EB;
  transition: width var(--farm-transition-slow);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 2px 0 8px rgba(21, 128, 61, 0.04);
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 16px;
  border-bottom: 1px solid #F3F4F6;
  flex-shrink: 0;
}

.logo-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.logo-text {
  font-size: 17px;
  font-weight: 600;
  color: #15803D;
  white-space: nowrap;
  letter-spacing: -0.02em;
}

.logo-icon-only {
  font-size: 24px;
}

/* === Menu Styles === */
.sidebar-menu {
  border-right: none !important;
  background-color: transparent !important;
  flex: 1;
  padding: 8px;
  --el-menu-bg-color: transparent;
  --el-menu-text-color: #475569;
  --el-menu-active-color: #15803D;
  --el-menu-hover-bg-color: #F0FDF4;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 224px;
}

:deep(.el-menu-item) {
  height: 48px;
  line-height: 48px;
  margin: 4px 0;
  border-radius: 10px;
  color: #475569;
  transition: all var(--farm-transition);
}

:deep(.el-menu-item .el-icon) {
  color: #64748B;
  transition: color var(--farm-transition);
}

:deep(.el-menu-item:hover) {
  background-color: #F0FDF4 !important;
  color: #15803D;
}

:deep(.el-menu-item:hover .el-icon) {
  color: #15803D;
}

:deep(.el-menu-item.is-active) {
  background-color: #DCFCE7 !important;
  color: #15803D !important;
  font-weight: 500;
}

:deep(.el-menu-item.is-active .el-icon) {
  color: #15803D;
}

/* === Sidebar Footer === */
.sidebar-footer {
  padding: 12px;
  border-top: 1px solid #F3F4F6;
  flex-shrink: 0;
}

.collapse-toggle {
  width: 100%;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #F9FAFB;
  border: 1px solid #E5E7EB;
  border-radius: 10px;
  cursor: pointer;
  color: #64748B;
  transition: all var(--farm-transition);
}

.collapse-toggle:hover {
  background: #F0FDF4;
  border-color: #BBF7D0;
  color: #15803D;
}

/* === Header Styles === */
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #FFFFFF;
  border-bottom: 1px solid #E5E7EB;
  padding: 0 24px;
  height: 64px;
  box-shadow: 0 1px 3px rgba(21, 128, 61, 0.04);
}

.header-left {
  display: flex;
  align-items: center;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: #14532D;
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 12px;
  transition: all var(--farm-transition);
}

.user-info:hover {
  background: #F0FDF4;
}

.user-avatar {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #DCFCE7 0%, #BBF7D0 100%);
  border-radius: 10px;
  color: #15803D;
}

.username {
  font-size: 14px;
  font-weight: 500;
  color: #475569;
}

/* === Main Content === */
.main-content {
  background-color: #F0FDF4;
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
}

:deep(.el-dropdown-menu__item:hover) {
  background-color: #F0FDF4;
  color: #15803D;
}

/* === Responsive === */
@media (max-width: 768px) {
  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    height: 100vh;
    z-index: 1000;
    box-shadow: 4px 0 16px rgba(21, 128, 61, 0.12);
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
}
</style>
