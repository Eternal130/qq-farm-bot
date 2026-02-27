<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { accountApi, type Account, type CreateAccountRequest, type QRCodeResponse } from '@/api'
import { 
  ElTable, 
  ElTableColumn, 
  ElButton, 
  ElTag, 
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElSelect,
  ElOption,
  ElSwitch,
  ElInputNumber,
  ElMessage,
  ElMessageBox,
  ElSpace,
  ElImage
} from 'element-plus'
import { Plus, Edit, Delete, VideoPlay, VideoPause, Grid } from '@element-plus/icons-vue'

const loading = ref(false)
const accounts = ref<Account[]>([])
const dialogVisible = ref(false)
const qrDialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref<number | null>(null)

const formData = ref<CreateAccountRequest>({
  name: '',
  platform: 'qq',
  code: '',
  auto_start: false,
  farm_interval: 10,
  friend_interval: 10,
  enable_steal: true,
  force_lowest: false
})

const qrCodeData = ref<QRCodeResponse | null>(null)
const qrPolling = ref(false)
const currentQRAccountId = ref<number | null>(null)
let qrPollInterval: number | null = null
const autoStartAfterQR = ref(false)

const dialogTitle = computed(() => isEdit.value ? '编辑账号' : '添加账号')

const fetchAccounts = async () => {
  loading.value = true
  try {
    const response = await accountApi.getAll()
    accounts.value = response.data
  } catch (error) {
    ElMessage.error('获取账号列表失败')
  } finally {
    loading.value = false
  }
}

const openAddDialog = () => {
  isEdit.value = false
  currentId.value = null
  formData.value = {
    name: '',
    platform: 'qq',
    code: '',
    auto_start: false,
    farm_interval: 10,
    friend_interval: 10,
    enable_steal: true,
    force_lowest: false
  }
  dialogVisible.value = true
}

const openEditDialog = (row: Account) => {
  isEdit.value = true
  currentId.value = row.id
  formData.value = {
    name: row.name,
    platform: row.platform,
    code: row.code,
    auto_start: row.auto_start,
    farm_interval: row.farm_interval,
    friend_interval: row.friend_interval,
    enable_steal: row.enable_steal,
    force_lowest: row.force_lowest
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  try {
    if (isEdit.value && currentId.value) {
      await accountApi.update(currentId.value, formData.value)
      ElMessage.success('更新成功')
    } else {
      await accountApi.create(formData.value)
      ElMessage.success('添加成功')
    }
    dialogVisible.value = false
    fetchAccounts()
  } catch (error: any) {
    const message = error.response?.data?.error || '操作失败'
    ElMessage.error(message)
  }
}

const handleDelete = async (row: Account) => {
  try {
    await ElMessageBox.confirm(`确定要删除账号 "${row.name}" 吗？`, '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await accountApi.delete(row.id)
    ElMessage.success('删除成功')
    fetchAccounts()
  } catch (error: any) {
    if (error !== 'cancel') {
      const message = error.response?.data?.error || '删除失败'
      ElMessage.error(message)
    }
  }
}

const toggleBot = async (row: Account) => {
  try {
    if (row.status === 'running') {
      await accountApi.stop(row.id)
      ElMessage.success(`已停止 ${row.name}`)
    } else {
      // QQ account without code: trigger QR login flow first, then auto-start
      if (row.platform === 'qq' && !row.code) {
        autoStartAfterQR.value = true
        startQRLogin(row)
        return
      }
      await accountApi.start(row.id)
      ElMessage.success(`已启动 ${row.name}`)
    }
    fetchAccounts()
  } catch (error: any) {
    const message = error.response?.data?.error || '操作失败'
    ElMessage.error(message)
  }
}

const startQRLogin = async (row: Account) => {
  try {
    const response = await accountApi.getQRCode(row.id)
    qrCodeData.value = response.data
    currentQRAccountId.value = row.id
    qrDialogVisible.value = true
    startQRPolling(row.id, response.data.login_code)
  } catch (error: any) {
    const message = error.response?.data?.error || '获取二维码失败'
    ElMessage.error(message)
  }
}

const startQRPolling = (accountId: number, loginCode: string) => {
  qrPolling.value = true
  qrPollInterval = window.setInterval(async () => {
    try {
      const response = await accountApi.pollQRCode(accountId, loginCode)
      const data = response.data
      
      if (data.status === 'ok' && data.code) {
        // Update account with new code
        await accountApi.update(accountId, { code: data.code })
        ElMessage.success('扫码登录成功！')
        closeQRDialog()
        // Auto-start bot if triggered from start button
        if (autoStartAfterQR.value) {
          autoStartAfterQR.value = false
          try {
            await accountApi.start(accountId)
            ElMessage.success('已自动启动')
          } catch (e: any) {
            const msg = e.response?.data?.error || '自动启动失败'
            ElMessage.error(msg)
          }
        }
        fetchAccounts()
      } else if (data.status === 'expired') {
        ElMessage.warning('二维码已过期，请重新获取')
        closeQRDialog()
      } else if (data.status === 'error') {
        ElMessage.error(data.message || '扫码登录失败')
        closeQRDialog()
      }
    } catch (error) {
      console.error('QR polling error:', error)
    }
  }, 2000)
}

const closeQRDialog = () => {
  qrDialogVisible.value = false
  qrCodeData.value = null
  currentQRAccountId.value = null
  autoStartAfterQR.value = false
  if (qrPollInterval) {
    clearInterval(qrPollInterval)
    qrPollInterval = null
  }
  qrPolling.value = false
}

const getStatusType = (status: string): 'success' | 'info' | 'danger' => {
  if (status === 'running') return 'success'
  if (status === 'error') return 'danger'
  return 'info'
}

const getStatusText = (status: string): string => {
  if (status === 'running') return '运行中'
  if (status === 'error') return '错误'
  return '已停止'
}

onMounted(() => {
  fetchAccounts()
})
</script>

<template>
  <div class="accounts-view">
    <ElCard shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span>账号列表</span>
          <ElButton type="primary" :icon="Plus" @click="openAddDialog">
            添加账号
          </ElButton>
        </div>
      </template>

      <ElTable :data="accounts" v-loading="loading" stripe style="width: 100%">
        <ElTableColumn prop="id" label="ID" width="70" />
        <ElTableColumn prop="name" label="名称" min-width="120" />
        <ElTableColumn prop="platform" label="平台" width="80">
          <template #default="{ row }">
            <ElTag size="small" :type="row.platform === 'qq' ? 'primary' : 'success'">
              {{ row.platform.toUpperCase() }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="status" label="状态" width="100">
          <template #default="{ row }">
            <ElTag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="level" label="等级" width="80">
          <template #default="{ row }">
            Lv.{{ row.level || 0 }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="gold" label="金币" width="100">
          <template #default="{ row }">
            {{ (row.gold || 0).toLocaleString() }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <ElSpace>
              <ElButton
                :type="row.status === 'running' ? 'danger' : 'success'"
                size="small"
                :icon="row.status === 'running' ? VideoPause : VideoPlay"
                @click="toggleBot(row)"
              >
                {{ row.status === 'running' ? '停止' : '启动' }}
              </ElButton>
              <ElButton
                v-if="row.platform === 'qq'"
                type="warning"
                size="small"
                :icon="Grid"
                @click="startQRLogin(row)"
              >
                扫码
              </ElButton>
              <ElButton
                type="primary"
                size="small"
                :icon="Edit"
                @click="openEditDialog(row)"
              >
                编辑
              </ElButton>
              <ElButton
                type="danger"
                size="small"
                :icon="Delete"
                @click="handleDelete(row)"
              >
                删除
              </ElButton>
            </ElSpace>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>

    <!-- Add/Edit Dialog -->
    <ElDialog 
      v-model="dialogVisible" 
      :title="dialogTitle"
      width="500px"
      destroy-on-close
    >
      <ElForm :model="formData" label-width="100px">
        <ElFormItem label="名称" required>
          <ElInput v-model="formData.name" placeholder="请输入账号名称" />
        </ElFormItem>
        
        <ElFormItem label="平台" required>
          <ElSelect v-model="formData.platform" style="width: 100%">
            <ElOption label="QQ小程序" value="qq" />
            <ElOption label="微信小程序" value="wx" />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="登录Code">
          <ElInput 
            v-model="formData.code" 
            placeholder="请输入登录Code（QQ可扫码获取）" 
            type="textarea"
            :rows="2"
          />
        </ElFormItem>
        
        <ElFormItem label="自动启动">
          <ElSwitch v-model="formData.auto_start" />
        </ElFormItem>
        
        <ElFormItem label="农场间隔">
          <ElInputNumber 
            v-model="formData.farm_interval" 
            :min="1" 
            :max="60"
            style="width: 100%"
          />
          <span class="form-hint">秒</span>
        </ElFormItem>
        
        <ElFormItem label="好友间隔">
          <ElInputNumber 
            v-model="formData.friend_interval" 
            :min="1" 
            :max="60"
            style="width: 100%"
          />
          <span class="form-hint">秒</span>
        </ElFormItem>
        
        <ElFormItem label="允许偷菜">
          <ElSwitch v-model="formData.enable_steal" />
        </ElFormItem>
        
        <ElFormItem label="强制最低级">
          <ElSwitch v-model="formData.force_lowest" />
          <span class="form-hint">种植最低等级作物</span>
        </ElFormItem>
      </ElForm>
      
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" @click="handleSubmit">确定</ElButton>
      </template>
    </ElDialog>

    <!-- QR Code Dialog -->
    <ElDialog 
      v-model="qrDialogVisible" 
      title="扫码登录"
      width="400px"
      @close="closeQRDialog"
    >
      <div class="qr-container">
        <p class="qr-tip">请使用手机QQ扫描下方二维码登录</p>
        <ElImage 
          v-if="qrCodeData"
          :src="`https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=${encodeURIComponent(qrCodeData.qr_code_url)}`"
          fit="contain"
          class="qr-image"
        />
        <p class="qr-status" v-if="qrPolling">
          <span class="waiting-dot"></span>
          等待扫码中...
        </p>
      </div>
    </ElDialog>
  </div>
</template>

<style scoped>
.accounts-view {
  padding: 0;
}

.table-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-hint {
  margin-left: 10px;
  color: #909399;
  font-size: 12px;
}

.qr-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
}

.qr-tip {
  color: #606266;
  margin-bottom: 20px;
}

.qr-image {
  width: 250px;
  height: 250px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
}

.qr-status {
  margin-top: 20px;
  color: #409eff;
  display: flex;
  align-items: center;
  gap: 8px;
}

.waiting-dot {
  width: 8px;
  height: 8px;
  background-color: #409eff;
  border-radius: 50%;
  animation: blink 1s infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}
</style>
