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
    <!-- Decorative Background Elements -->
    <div class="bg-decoration">
      <div class="deco-circle deco-1"></div>
      <div class="deco-circle deco-2"></div>
      <div class="deco-circle deco-3"></div>
      <div class="deco-leaf deco-leaf-1"></div>
      <div class="deco-leaf deco-leaf-2"></div>
    </div>
    
    <div class="register-content">
      <ElCard class="register-card">
        <div class="card-header">
          <div class="logo-wrapper">
            <span class="logo-icon">🌾</span>
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
      
      <p class="copyright">© 2024 QQ农场管理系统 · 让农场管理更简单</p>
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
.register-form {
  padding: 24px 32px 16px;
}

:deep(.el-form-item__label) {
  font-weight: 500;
  color: #475569;
  font-size: 14px;
  padding-bottom: 6px;
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

.register-btn {
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

.register-btn:hover {
  background: linear-gradient(135deg, #16A34A 0%, #22C55E 100%);
  box-shadow: 0 6px 20px rgba(21, 128, 61, 0.35);
  transform: translateY(-1px);
}

.register-btn:active {
  transform: translateY(0);
}

/* === Login Link === */
.login-link {
  text-align: center;
  padding: 16px 32px 28px;
  border-top: 1px solid #F3F4F6;
  background: #FAFCFB;
}

.login-link span {
  color: #64748B;
  font-size: 14px;
}

.login-link a {
  color: #15803D;
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
  margin-left: 4px;
  transition: color var(--farm-transition);
}

.login-link a:hover {
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
  .register-card {
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
  
  .register-form {
    padding: 20px 24px 12px;
  }
  
  .login-link {
    padding: 12px 24px 24px;
  }
  
  .deco-1,
  .deco-2,
  .deco-3 {
    display: none;
  }
}
</style>
