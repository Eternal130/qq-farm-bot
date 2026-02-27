<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
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
const authStore = useAuthStore()

const formRef = ref<FormInstance>()
const loading = ref(false)
const registerForm = ref({
  username: '',
  password: '',
  confirmPassword: ''
})

const validateConfirmPassword = (_rule: any, value: string, callback: (error?: Error) => void) => {
  if (value !== registerForm.value.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 32, message: '用户名长度为3-32个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

const handleRegister = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    
    loading.value = true
    try {
      const response = await authApi.register(registerForm.value.username, registerForm.value.password)
      authStore.setAuth(response.data.token, response.data.user)
      
      if (response.data.user.is_admin) {
        ElMessage.success('注册成功！您是第一个用户，已自动成为管理员')
      } else {
        ElMessage.success('注册成功')
      }
      
      router.push('/dashboard')
    } catch (error: any) {
      const message = error.response?.data?.error || '注册失败，请稍后重试'
      ElMessage.error(message)
    } finally {
      loading.value = false
    }
  })
}

const goToLogin = () => {
  router.push('/login')
}
</script>

<template>
  <div class="register-container">
    <div class="register-bg">
      <div class="register-content">
        <ElCard class="register-card">
          <template #header>
            <div class="card-header">
              <span class="logo-icon">🌾</span>
              <h2>注册账号</h2>
            </div>
          </template>
          
          <ElForm
            ref="formRef"
            :model="registerForm"
            :rules="rules"
            label-position="top"
            @submit.prevent="handleRegister"
          >
            <ElFormItem label="用户名" prop="username">
              <ElInput
                v-model="registerForm.username"
                placeholder="请输入用户名 (3-32字符)"
                :prefix-icon="User"
                size="large"
              />
            </ElFormItem>
            
            <ElFormItem label="密码" prop="password">
              <ElInput
                v-model="registerForm.password"
                type="password"
                placeholder="请输入密码 (至少6位)"
                :prefix-icon="Lock"
                size="large"
                show-password
              />
            </ElFormItem>
            
            <ElFormItem label="确认密码" prop="confirmPassword">
              <ElInput
                v-model="registerForm.confirmPassword"
                type="password"
                placeholder="请再次输入密码"
                :prefix-icon="Lock"
                size="large"
                show-password
                @keyup.enter="handleRegister"
              />
            </ElFormItem>
            
            <ElFormItem>
              <ElButton
                type="primary"
                size="large"
                :loading="loading"
                class="register-btn"
                @click="handleRegister"
              >
                {{ loading ? '注册中...' : '注 册' }}
              </ElButton>
            </ElFormItem>
          </ElForm>
          
          <div class="login-link">
            已有账号？<a @click="goToLogin">立即登录</a>
          </div>
        </ElCard>
        
        <p class="copyright">© 2024 QQ农场管理系统</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.register-container {
  width: 100%;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
}

.register-bg {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  position: relative;
  overflow: hidden;
}

.register-bg::before {
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

.register-content {
  z-index: 1;
}

.register-card {
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

.register-btn {
  width: 100%;
  height: 44px;
  font-size: 16px;
}

.login-link {
  text-align: center;
  margin-top: 16px;
  color: #909399;
  font-size: 14px;
}

.login-link a {
  color: #409eff;
  cursor: pointer;
  text-decoration: none;
}

.login-link a:hover {
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
  .register-card {
    width: 90%;
    margin: 0 16px;
  }
}
</style>
