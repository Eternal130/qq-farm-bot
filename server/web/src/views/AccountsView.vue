<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { accountApi, type Account, type CreateAccountRequest, type QRCodeResponse } from '@/api'
import { 
  ElTable, 
  ElTableColumn, 
  ElButton, 
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
          <span class="header-title">账号列表</span>
          <ElButton type="primary" :icon="Plus" @click="openAddDialog" class="add-btn">
            添加账号
          </ElButton>
        </div>
      </template>

      <ElTable :data="accounts" v-loading="loading" class="accounts-table" style="width: 100%">
        <ElTableColumn prop="id" label="ID" width="70" align="center" />
        <ElTableColumn prop="name" label="名称" min-width="120">
          <template #default="{ row }">
            <span class="account-name">{{ row.name }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="platform" label="平台" width="90" align="center">
          <template #default="{ row }">
            <span class="platform-badge" :class="row.platform === 'qq' ? 'platform-qq' : 'platform-wx'">
              {{ row.platform.toUpperCase() }}
            </span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <div class="status-tag" :class="'status-' + row.status">
              <span class="status-dot"></span>
              <span class="status-text">{{ getStatusText(row.status) }}</span>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="level" label="等级" width="80" align="center">
          <template #default="{ row }">
            <span class="level-value">Lv.{{ row.level || 0 }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="gold" label="金币" width="110" align="right">
          <template #default="{ row }">
            <span class="gold-value">{{ (row.gold || 0).toLocaleString() }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="260" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <ElButton
                :type="row.status === 'running' ? 'danger' : 'success'"
                size="small"
                :icon="row.status === 'running' ? VideoPause : VideoPlay"
                @click="toggleBot(row)"
                class="action-btn action-btn--toggle"
              >
                {{ row.status === 'running' ? '停止' : '启动' }}
              </ElButton>
              <ElButton
                v-if="row.platform === 'qq'"
                size="small"
                :icon="Grid"
                @click="startQRLogin(row)"
                class="action-btn action-btn--qr"
              >
                扫码
              </ElButton>
              <ElButton
                size="small"
                text
                :icon="Edit"
                @click="openEditDialog(row)"
                class="action-btn action-btn--edit"
              >
                编辑
              </ElButton>
              <ElButton
                size="small"
                text
                :icon="Delete"
                @click="handleDelete(row)"
                class="action-btn action-btn--delete"
              >
                删除
              </ElButton>
            </div>
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
      class="account-dialog"
    >
      <ElForm :model="formData" label-width="100px" class="account-form">
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
          <div class="interval-input">
            <ElInputNumber 
              v-model="formData.farm_interval" 
              :min="1" 
              :max="60"
              style="width: 120px"
            />
            <span class="form-hint">秒</span>
          </div>
        </ElFormItem>
        
        <ElFormItem label="好友间隔">
          <div class="interval-input">
            <ElInputNumber 
              v-model="formData.friend_interval" 
              :min="1" 
              :max="60"
              style="width: 120px"
            />
            <span class="form-hint">秒</span>
          </div>
        </ElFormItem>
        
        <ElFormItem label="允许偷菜">
          <ElSwitch v-model="formData.enable_steal" />
        </ElFormItem>
        
        <ElFormItem label="强制最低级">
          <div class="switch-row">
            <ElSwitch v-model="formData.force_lowest" />
            <span class="form-hint">种植最低等级作物</span>
          </div>
        </ElFormItem>
      </ElForm>
      
      <template #footer>
        <div class="dialog-footer">
          <ElButton @click="dialogVisible = false" class="btn-cancel">取消</ElButton>
          <ElButton type="primary" @click="handleSubmit" class="btn-submit">确定</ElButton>
        </div>
      </template>
    </ElDialog>

    <!-- QR Code Dialog -->
    <ElDialog 
      v-model="qrDialogVisible" 
      title="扫码登录"
      width="400px"
      @close="closeQRDialog"
      class="qr-dialog"
    >
      <div class="qr-container">
        <p class="qr-tip">请使用手机QQ扫描下方二维码登录</p>
        <div class="qr-image-wrapper">
          <ElImage 
            v-if="qrCodeData"
            :src="`https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=${encodeURIComponent(qrCodeData.qr_code_url)}`"
            fit="contain"
            class="qr-image"
          />
        </div>
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

/* Table Card */
.table-card {
  border-radius: 16px;
  border: none;
  box-shadow: 0 1px 3px rgba(21, 128, 61, 0.06), 0 4px 16px rgba(21, 128, 61, 0.04);
}

.table-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid #E5E7EB;
}

.table-card :deep(.el-card__body) {
  padding: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-title {
  font-size: 17px;
  font-weight: 600;
  color: #14532D;
}

.add-btn {
  background: #15803D !important;
  border-color: #15803D !important;
  border-radius: 8px;
  font-weight: 600;
}

.add-btn:hover {
  background: #166534 !important;
  border-color: #166534 !important;
}

/* Table Styles */
.accounts-table {
  --el-table-border-color: #E5E7EB;
  --el-table-header-bg-color: #F9FAFB;
  --el-table-header-text-color: #374151;
  --el-table-text-color: #14532D;
  --el-table-row-hover-bg-color: #F0FDF4;
}

.accounts-table :deep(.el-table__header th) {
  font-weight: 600;
  font-size: 13px;
  padding: 14px 0;
}

.accounts-table :deep(.el-table__body td) {
  padding: 14px 0;
}

.accounts-table :deep(.el-table__row--striped) {
  background: #FAFFF7;
}

.account-name {
  font-weight: 500;
  color: #14532D;
}

/* Platform Badge */
.platform-badge {
  display: inline-block;
  font-size: 11px;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 100px;
  letter-spacing: 0.3px;
}

.platform-qq {
  background: rgba(21, 128, 61, 0.12);
  color: #15803D;
}

.platform-wx {
  background: rgba(59, 130, 246, 0.12);
  color: #2563EB;
}

/* Status Tag */
.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-running .status-dot {
  background: #22C55E;
  box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.2);
}

.status-running .status-text {
  color: #22C55E;
}

.status-error .status-dot {
  background: #DC2626;
  box-shadow: 0 0 0 3px rgba(220, 38, 38, 0.2);
}

.status-error .status-text {
  color: #DC2626;
}

.status-stopped .status-dot,
.status-info .status-dot {
  background: #9CA3AF;
}

.status-stopped .status-text,
.status-info .status-text {
  color: #6B7280;
}

.level-value {
  font-weight: 600;
  color: #15803D;
}

.gold-value {
  font-weight: 600;
  color: #CA8A04;
}

/* Action Buttons */
.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.action-btn {
  border-radius: 6px;
  font-size: 12px;
  padding: 5px 10px;
  height: 28px;
}

.action-btn--toggle.el-button--success {
  background: #15803D;
  border-color: #15803D;
}

.action-btn--toggle.el-button--success:hover {
  background: #166534;
  border-color: #166534;
}

.action-btn--toggle.el-button--danger {
  background: #DC2626;
  border-color: #DC2626;
}

.action-btn--toggle.el-button--danger:hover {
  background: #B91C1C;
  border-color: #B91C1C;
}

.action-btn--qr {
  background: transparent;
  border: 1px solid #CA8A04;
  color: #CA8A04;
}

.action-btn--qr:hover {
  background: rgba(202, 138, 4, 0.1);
  border-color: #CA8A04;
  color: #A16207;
}

.action-btn--edit {
  color: #15803D;
}

.action-btn--edit:hover {
  background: rgba(21, 128, 61, 0.08);
  color: #166534;
}

.action-btn--delete {
  color: #DC2626;
}

.action-btn--delete:hover {
  background: rgba(220, 38, 38, 0.08);
  color: #B91C1C;
}

/* Dialog Styles */
.account-dialog :deep(.el-dialog) {
  border-radius: 16px;
  overflow: hidden;
}

.account-dialog :deep(.el-dialog__header) {
  padding: 20px 24px;
  border-bottom: 1px solid #E5E7EB;
  margin-right: 0;
}

.account-dialog :deep(.el-dialog__title) {
  font-weight: 600;
  color: #14532D;
}

.account-dialog :deep(.el-dialog__body) {
  padding: 24px;
}

.account-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.account-form :deep(.el-form-item__label) {
  font-weight: 500;
  color: #374151;
}

.account-form :deep(.el-input__wrapper),
.account-form :deep(.el-textarea__inner),
.account-form :deep(.el-select .el-input__wrapper) {
  border-radius: 8px;
  box-shadow: 0 0 0 1px #D1D5DB;
  transition: all 0.2s ease;
}

.account-form :deep(.el-input__wrapper:hover),
.account-form :deep(.el-textarea__inner:hover),
.account-form :deep(.el-select .el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #9CA3AF;
}

.account-form :deep(.el-input__wrapper.is-focus),
.account-form :deep(.el-textarea__inner:focus),
.account-form :deep(.el-select .el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px rgba(21, 128, 61, 0.2), 0 0 0 1px #15803D !important;
}

.interval-input {
  display: flex;
  align-items: center;
  gap: 10px;
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.form-hint {
  font-size: 13px;
  color: #6B7280;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid #E5E7EB;
}

.btn-cancel {
  border-radius: 8px;
}

.btn-submit {
  background: #15803D;
  border-color: #15803D;
  border-radius: 8px;
  font-weight: 500;
}

.btn-submit:hover {
  background: #166534;
  border-color: #166534;
}

/* QR Dialog */
.qr-dialog :deep(.el-dialog) {
  border-radius: 16px;
  overflow: hidden;
}

.qr-dialog :deep(.el-dialog__header) {
  padding: 20px 24px;
  border-bottom: 1px solid #E5E7EB;
  margin-right: 0;
}

.qr-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.qr-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 24px;
}

.qr-tip {
  color: #374151;
  font-size: 14px;
  margin-bottom: 24px;
  text-align: center;
}

.qr-image-wrapper {
  background: #FFFFFF;
  border: 2px solid #E5E7EB;
  border-radius: 16px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.qr-image {
  width: 220px;
  height: 220px;
  display: block;
}

.qr-status {
  margin-top: 24px;
  color: #15803D;
  font-size: 14px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}

.waiting-dot {
  width: 8px;
  height: 8px;
  background-color: #22C55E;
  border-radius: 50%;
  animation: pulse-dot 1.5s ease-in-out infinite;
}

@keyframes pulse-dot {
  0%, 100% { 
    opacity: 1;
    transform: scale(1);
  }
  50% { 
    opacity: 0.5;
    transform: scale(0.8);
  }
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .table-card :deep(.el-card__header) {
    padding: 16px;
  }
  
  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  
  .action-buttons {
    flex-wrap: wrap;
  }
  
  .account-dialog :deep(.el-dialog) {
    width: 95% !important;
    margin: 20px auto;
  }
}
</style>
