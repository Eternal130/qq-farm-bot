<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { authApi, getErrorMessage } from '@/api'
import { 
  ElCard, 
  ElForm, 
  ElFormItem, 
  ElInput, 
  ElButton, 
  ElMessage,
  FormInstance,
  FormRules
} from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const formRef = ref<FormInstance>()
const loading = ref(false)
const loginForm = ref({
  username: '',
  password: ''
})

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    
    loading.value = true
    try {
      const response = await authApi.login(loginForm.value.username, loginForm.value.password)
      authStore.setAuth(response.data.token, response.data.user)
      ElMessage.success('登录成功')
      
      const redirect = route.query.redirect as string
      router.push(redirect || '/dashboard')
    } catch (error: unknown) {
      ElMessage.error(getErrorMessage(error, '登录失败，请检查用户名和密码'))
    } finally {
      loading.value = false
    }
  })
}

const goToRegister = () => {
  router.push('/register')
}
</script>

<template>
  <div class="login-container">
    <div class="login-content">
      <ElCard class="login-card">
        <div class="card-header">
          <div class="logo-wrapper">
            <svg class="logo-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2z" fill="var(--primary)"/>
              <path d="M8 14l2-4 2 3 2-2 2 3" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <h2 class="title">QQ农场管理系统</h2>
          <p class="subtitle">智能农场自动化管理平台</p>
        </div>
        
        <ElForm
          ref="formRef"
          :model="loginForm"
          :rules="rules"
          label-position="top"
          class="login-form"
          @submit.prevent="handleLogin"
        >
          <ElFormItem label="用户名" prop="username">
            <ElInput
              v-model="loginForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
              size="large"
            />
          </ElFormItem>
          
          <ElFormItem label="密码" prop="password">
            <ElInput
              v-model="loginForm.password"
              type="password"
              placeholder="请输入密码"
              :prefix-icon="Lock"
              size="large"
              show-password
              @keyup.enter="handleLogin"
            />
          </ElFormItem>
          
          <ElFormItem class="form-actions">
            <ElButton
              type="primary"
              size="large"
              :loading="loading"
              class="login-btn"
              @click="handleLogin"
            >
              {{ loading ? '登录中...' : '登 录' }}
            </ElButton>
          </ElFormItem>
        </ElForm>
        
        <div class="register-link">
          <span>还没有账号？</span>
          <a @click="goToRegister">立即注册</a>
        </div>
      </ElCard>
      
      <p class="copyright">© {{ new Date().getFullYear() }} QQ农场管理系统 · 让农场管理更简单</p>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  width: 100%;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background: var(--bg-page);
}

/* === Content === */
.login-content {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  animation: fadeInUp 0.6s ease-out;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* === Card Styles === */
.login-card {
  width: 420px;
  border-radius: var(--radius-xl);
  border: 1px solid var(--border);
  box-shadow: var(--shadow-card);
  background: var(--bg-card);
  overflow: hidden;
}

:deep(.el-card__body) {
  padding: 0;
}

/* === Header === */
.card-header {
  text-align: center;
  padding: 48px 40px 32px;
  background: var(--bg-card);
}

.logo-wrapper {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  background: var(--primary-bg);
  border-radius: var(--radius-xl);
  margin-bottom: 20px;
  border: none;
}

.logo-icon {
  width: 44px;
  height: 44px;
}

.title {
  font-size: 26px;
  font-weight: 600;
  color: var(--text-heading);
  margin: 0 0 8px;
  letter-spacing: -0.02em;
}

.subtitle {
  font-size: 15px;
  color: var(--text-secondary);
  margin: 0;
}

/* === Form === */
.login-form {
  padding: 8px 40px 24px;
}

:deep(.el-form-item__label) {
  font-weight: 500;
  color: var(--text-primary);
  font-size: 14px;
  padding-bottom: 8px;
}

:deep(.el-input__wrapper) {
  border-radius: var(--radius-md);
  background-color: var(--bg-input) !important;
  box-shadow: 0 0 0 1px var(--border) inset !important;
  transition: all var(--transition);
  padding: 4px 12px;
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px var(--border-light) inset !important;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--primary) inset !important;
}

:deep(.el-input__inner) {
  color: var(--text-primary) !important;
}

:deep(.el-input__inner::placeholder) {
  color: var(--text-muted) !important;
}

:deep(.el-input__prefix) {
  color: var(--text-muted);
}

.form-actions {
  margin-top: 12px;
  margin-bottom: 0;
}

.login-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: var(--radius-md);
  background: var(--primary);
  border: none;
  box-shadow: var(--shadow-sm);
  transition: all var(--transition);
}

.login-btn:hover {
  background: var(--primary-hover);
  box-shadow: var(--shadow-md);
}

.login-btn:active {
  transform: none;
}

/* === Register Link === */
.register-link {
  text-align: center;
  padding: 24px 40px 36px;
  border-top: 1px solid var(--border);
  background: var(--bg-card);
}

.register-link span {
  color: var(--text-secondary);
  font-size: 14px;
}

.register-link a {
  color: var(--primary);
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
  margin-left: 4px;
  transition: color var(--transition);
}

.register-link a:hover {
  color: var(--primary-hover);
  text-decoration: none;
}

/* === Copyright === */
.copyright {
  text-align: center;
  color: var(--text-muted);
  margin-top: 28px;
  font-size: 13px;
  opacity: 0.8;
}

/* === Responsive === */
@media (max-width: 480px) {
  .login-card {
    width: calc(100% - 32px);
    margin: 0 16px;
    border-radius: var(--radius-lg);
  }
  
  .card-header {
    padding: 36px 28px 24px;
  }
  
  .logo-wrapper {
    width: 72px;
    height: 72px;
  }
  
  .logo-icon {
    width: 40px;
    height: 40px;
  }
  
  .title {
    font-size: 22px;
  }
  
  .login-form {
    padding: 8px 28px 20px;
  }
  
  .register-link {
    padding: 20px 28px 28px;
  }
}
</style>
