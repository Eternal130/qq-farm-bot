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
    <!-- Decorative Background Elements -->
    <div class="bg-decoration">
      <div class="deco-circle deco-1"></div>
      <div class="deco-circle deco-2"></div>
      <div class="deco-circle deco-3"></div>
      <div class="deco-leaf deco-leaf-1"></div>
      <div class="deco-leaf deco-leaf-2"></div>
    </div>
    
    <div class="login-content">
      <ElCard class="login-card">
        <div class="card-header">
          <div class="logo-wrapper">
            <span class="logo-icon">🌾</span>
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
  background: linear-gradient(135deg, #DCFCE7 0%, #F0FDF4 40%, #FEF9C3 100%);
}

/* === Decorative Background Elements === */
.bg-decoration {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
}

.deco-circle {
  position: absolute;
  border-radius: 50%;
  opacity: 0.6;
}

.deco-1 {
  width: 400px;
  height: 400px;
  background: radial-gradient(circle, rgba(34, 197, 94, 0.15) 0%, transparent 70%);
  top: -100px;
  right: -100px;
}

.deco-2 {
  width: 300px;
  height: 300px;
  background: radial-gradient(circle, rgba(202, 138, 4, 0.12) 0%, transparent 70%);
  bottom: -50px;
  left: -50px;
}

.deco-3 {
  width: 200px;
  height: 200px;
  background: radial-gradient(circle, rgba(134, 239, 172, 0.2) 0%, transparent 70%);
  top: 50%;
  left: 10%;
  transform: translateY(-50%);
}

.deco-leaf {
  position: absolute;
  width: 60px;
  height: 60px;
  border-radius: 0 50% 50% 50%;
  transform: rotate(45deg);
}

.deco-leaf-1 {
  background: linear-gradient(135deg, rgba(34, 197, 94, 0.2) 0%, rgba(134, 239, 172, 0.1) 100%);
  top: 15%;
  right: 15%;
}

.deco-leaf-2 {
  background: linear-gradient(135deg, rgba(202, 138, 4, 0.15) 0%, rgba(254, 249, 195, 0.1) 100%);
  bottom: 20%;
  right: 25%;
  width: 40px;
  height: 40px;
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
  border-radius: 20px;
  border: none;
  box-shadow: 
    0 4px 6px rgba(21, 128, 61, 0.04),
    0 10px 40px rgba(21, 128, 61, 0.1);
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  overflow: hidden;
}

:deep(.el-card__body) {
  padding: 0;
}

/* === Header === */
.card-header {
  text-align: center;
  padding: 36px 32px 24px;
  background: linear-gradient(180deg, #F0FDF4 0%, #FFFFFF 100%);
  border-bottom: 1px solid #F0FDF4;
}

.logo-wrapper {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 72px;
  height: 72px;
  background: linear-gradient(135deg, #DCFCE7 0%, #BBF7D0 100%);
  border-radius: 20px;
  margin-bottom: 16px;
  box-shadow: 0 4px 12px rgba(21, 128, 61, 0.15);
}

.logo-icon {
  font-size: 36px;
  line-height: 1;
}

.title {
  font-size: 24px;
  font-weight: 700;
  color: #14532D;
  margin: 0 0 8px;
  letter-spacing: -0.02em;
}

.subtitle {
  font-size: 14px;
  color: #64748B;
  margin: 0;
}

/* === Form === */
.login-form {
  padding: 28px 32px 20px;
}

:deep(.el-form-item__label) {
  font-weight: 500;
  color: #475569;
  font-size: 14px;
  padding-bottom: 8px;
}

:deep(.el-input__wrapper) {
  border-radius: 10px;
  box-shadow: 0 1px 2px rgba(21, 128, 61, 0.04);
  border: 1px solid #E5E7EB;
  transition: all var(--farm-transition);
  padding: 4px 12px;
}

:deep(.el-input__wrapper:hover) {
  border-color: #BBF7D0;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #15803D;
  box-shadow: 0 0 0 3px rgba(21, 128, 61, 0.1);
}

:deep(.el-input__inner) {
  color: #14532D;
}

:deep(.el-input__inner::placeholder) {
  color: #94A3B8;
}

.form-actions {
  margin-top: 8px;
  margin-bottom: 0;
}

.login-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 12px;
  background: linear-gradient(135deg, #15803D 0%, #16A34A 100%);
  border: none;
  box-shadow: 0 4px 12px rgba(21, 128, 61, 0.25);
  transition: all var(--farm-transition);
}

.login-btn:hover {
  background: linear-gradient(135deg, #16A34A 0%, #22C55E 100%);
  box-shadow: 0 6px 20px rgba(21, 128, 61, 0.35);
  transform: translateY(-1px);
}

.login-btn:active {
  transform: translateY(0);
}

/* === Register Link === */
.register-link {
  text-align: center;
  padding: 20px 32px 28px;
  border-top: 1px solid #F3F4F6;
  background: #FAFCFB;
}

.register-link span {
  color: #64748B;
  font-size: 14px;
}

.register-link a {
  color: #15803D;
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
  margin-left: 4px;
  transition: color var(--farm-transition);
}

.register-link a:hover {
  color: #16A34A;
  text-decoration: underline;
}

/* === Copyright === */
.copyright {
  text-align: center;
  color: #64748B;
  margin-top: 24px;
  font-size: 13px;
  opacity: 0.8;
}

/* === Responsive === */
@media (max-width: 480px) {
  .login-card {
    width: calc(100% - 32px);
    margin: 0 16px;
    border-radius: 16px;
  }
  
  .card-header {
    padding: 28px 24px 20px;
  }
  
  .logo-wrapper {
    width: 64px;
    height: 64px;
  }
  
  .logo-icon {
    font-size: 32px;
  }
  
  .title {
    font-size: 20px;
  }
  
  .login-form {
    padding: 24px 24px 16px;
  }
  
  .register-link {
    padding: 16px 24px 24px;
  }
  
  .deco-1,
  .deco-2,
  .deco-3 {
    display: none;
  }
}
</style>
