<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
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
    } catch (error: unknown) {
      ElMessage.error(getErrorMessage(error, '注册失败，请稍后重试'))
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
    <div class="register-content">
      <ElCard class="register-card">
        <div class="card-header">
          <div class="logo-wrapper">
            <svg class="logo-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2z" fill="var(--primary)"/>
              <path d="M8 14l2-4 2 3 2-2 2 3" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <h2 class="title">创建账号</h2>
          <p class="subtitle">加入QQ农场管理系统</p>
        </div>
        
        <ElForm
          ref="formRef"
          :model="registerForm"
          :rules="rules"
          label-position="top"
          class="register-form"
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
          
          <ElFormItem class="form-actions">
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
          <span>已有账号？</span>
          <a @click="goToLogin">立即登录</a>
        </div>
      </ElCard>
      
      <p class="copyright">© {{ new Date().getFullYear() }} QQ农场管理系统 · 让农场管理更简单</p>
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
  position: relative;
  overflow: hidden;
  background: var(--bg-page);
}

/* === Content === */
.register-content {
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
.register-card {
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
  padding: 44px 40px 28px;
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
.register-form {
  padding: 4px 40px 20px;
}

:deep(.el-form-item__label) {
  font-weight: 500;
  color: var(--text-primary);
  font-size: 14px;
  padding-bottom: 6px;
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

.register-btn {
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

.register-btn:hover {
  background: var(--primary-hover);
  box-shadow: var(--shadow-md);
}

.register-btn:active {
  transform: none;
}

/* === Login Link === */
.login-link {
  text-align: center;
  padding: 20px 40px 36px;
  border-top: 1px solid var(--border);
  background: var(--bg-card);
}

.login-link span {
  color: var(--text-secondary);
  font-size: 14px;
}

.login-link a {
  color: var(--primary);
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
  margin-left: 4px;
  transition: color var(--transition);
}

.login-link a:hover {
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
  .register-card {
    width: calc(100% - 32px);
    margin: 0 16px;
    border-radius: var(--radius-lg);
  }
  
  .card-header {
    padding: 32px 28px 20px;
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
  
  .register-form {
    padding: 4px 28px 16px;
  }
  
  .login-link {
    padding: 16px 28px 28px;
  }
}
</style>
