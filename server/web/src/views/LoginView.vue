<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { authApi } from '@/api'
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
    } catch (error: any) {
      const message = error.response?.data?.error || '登录失败，请检查用户名和密码'
      ElMessage.error(message)
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
    <div class="login-bg">
      <div class="login-content">
        <ElCard class="login-card">
          <template #header>
            <div class="card-header">
              <span class="logo-icon">🌾</span>
              <h2>QQ农场管理系统</h2>
            </div>
          </template>
          
          <ElForm
            ref="formRef"
            :model="loginForm"
            :rules="rules"
            label-position="top"
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
            
            <ElFormItem>
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
            没有账号？<a @click="goToRegister">立即注册</a>
          </div>
        </ElCard>
        
        <p class="copyright">© 2024 QQ农场管理系统</p>
      </div>
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
}

.login-bg {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  position: relative;
  overflow: hidden;
}

.login-bg::before {
  content: '';
  position: absolute;
  width: 200%;
  height: 200%;
  top: -50%;
  left: -50%;
  background: radial-gradient(circle, rgba(255,255,255,0.1) 0%, transparent 60%);
  animation: pulse 15s infinite;
}

@keyframes pulse {
  0%, 100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.1);
    opacity: 0.8;
  }
}

.login-content {
  z-index: 1;
}

.login-card {
  width: 400px;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.card-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.logo-icon {
  font-size: 48px;
}

.card-header h2 {
  font-size: 22px;
  color: #303133;
  font-weight: 600;
  margin: 0;
}
.login-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
}

.register-link {
  text-align: center;
  margin-top: 16px;
  color: #909399;
  font-size: 14px;
}

.register-link a {
  color: #409eff;
  cursor: pointer;
  text-decoration: none;
}

.register-link a:hover {
  text-decoration: underline;
}

.copyright {
  text-align: center;
  color: rgba(255, 255, 255, 0.7);
  margin-top: 24px;
  font-size: 13px;
}

:deep(.el-card__header) {
  padding: 30px 20px 20px;
  border-bottom: none;
}

:deep(.el-card__body) {
  padding: 10px 30px 30px;
}

:deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

@media (max-width: 480px) {
  .login-card {
    width: 90%;
    margin: 0 16px;
  }
}
</style>
