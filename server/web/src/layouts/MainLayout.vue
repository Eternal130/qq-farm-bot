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
  ElIcon,
  ElButton
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
      :width="isCollapse ? '64px' : '220px'" 
      class="sidebar"
    >
      <div class="logo">
        <span v-if="!isCollapse">🌾 农场管理</span>
        <span v-else>🌾</span>
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
    </ElAside>

    <ElContainer>
      <!-- Header -->
      <ElHeader class="header">
        <div class="header-left">
          <ElButton 
            text 
            @click="toggleSidebar"
            class="collapse-btn"
          >
            <ElIcon :size="20">
              <Fold v-if="!isCollapse" />
              <Expand v-else />
            </ElIcon>
          </ElButton>
        </div>
        
        <div class="header-right">
          <ElDropdown @command="handleCommand">
            <span class="user-info">
              <ElIcon><User /></ElIcon>
              <span class="username">{{ authStore.user?.username || '用户' }}</span>
            </span>
            <template #dropdown>
              <ElDropdownMenu>
                <ElDropdownItem command="logout">
                  <ElIcon><SwitchButton /></ElIcon>
                  退出登录
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

.sidebar {
  background-color: #304156;
  transition: width 0.3s ease;
  overflow: hidden;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  background-color: #263445;
  white-space: nowrap;
  overflow: hidden;
}

.sidebar-menu {
  border-right: none;
  background-color: #304156;
  --el-menu-bg-color: #304156;
  --el-menu-text-color: #bfcbd9;
  --el-menu-active-color: #409eff;
  --el-menu-hover-bg-color: #263445;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 220px;
}

:deep(.el-menu-item) {
  height: 50px;
  line-height: 50px;
}

:deep(.el-menu-item:hover) {
  background-color: #263445 !important;
}

:deep(.el-menu-item.is-active) {
  background-color: #409eff !important;
  color: #fff !important;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.header-left {
  display: flex;
  align-items: center;
}

.collapse-btn {
  padding: 8px;
}

.header-right {
  display: flex;
  align-items: center;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  color: #606266;
  padding: 8px 12px;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.user-info:hover {
  background-color: #f5f7fa;
}

.username {
  font-size: 14px;
}

.main-content {
  background-color: #f0f2f5;
  padding: 20px;
  overflow-y: auto;
}

/* Responsive */
@media (max-width: 768px) {
  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    height: 100vh;
    z-index: 1000;
  }
}
</style>
