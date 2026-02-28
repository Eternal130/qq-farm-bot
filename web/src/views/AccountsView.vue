<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { accountApi, cropApi, getErrorMessage, type Account, type CreateAccountRequest, type QRCodeResponse, type CropInfo } from '@/api'
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
  ElImage,
  ElCard
} from 'element-plus'
import { Plus, Edit, Delete, VideoPlay, VideoPause, Grid } from '@element-plus/icons-vue'

const loading = ref(false)
const accounts = ref<Account[]>([])
const crops = ref<CropInfo[]>([])
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
  force_lowest: false,
  enable_harvest: true,
  enable_plant: true,
  enable_sell: true,
  enable_weed: true,
  enable_bug: true,
  enable_water: true,
  enable_remove_dead: true,
  enable_upgrade_land: true,
  enable_help_friend: true,
  enable_claim_task: true,
  plant_crop_id: 0,
  sell_crop_ids: '',
  steal_crop_ids: ''
})

const qrCodeData = ref<QRCodeResponse | null>(null)
const qrPolling = ref(false)
const currentQRAccountId = ref<number | null>(null)
let qrPollInterval: number | null = null
const autoStartAfterQR = ref(false)

const dialogTitle = computed(() => isEdit.value ? '编辑账号' : '添加账号')

const sellCropIdsArray = computed({
  get: () => {
    if (!formData.value.sell_crop_ids) return []
    return formData.value.sell_crop_ids.split(',').map(Number).filter(n => n > 0)
  },
  set: (val: number[]) => {
    formData.value.sell_crop_ids = val.join(',')
  }
})

const stealCropIdsArray = computed({
  get: () => {
    if (!formData.value.steal_crop_ids) return []
    return formData.value.steal_crop_ids.split(',').map(Number).filter(n => n > 0)
  },
  set: (val: number[]) => {
    formData.value.steal_crop_ids = val.join(',')
  }
})

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

const fetchCrops = async () => {
  try {
    const response = await cropApi.getAll()
    crops.value = response.data
  } catch { }
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
    force_lowest: false,
    enable_harvest: true,
    enable_plant: true,
    enable_sell: true,
    enable_weed: true,
    enable_bug: true,
    enable_water: true,
    enable_remove_dead: true,
    enable_upgrade_land: true,
    enable_help_friend: true,
    enable_claim_task: true,
    plant_crop_id: 0,
    sell_crop_ids: '',
    steal_crop_ids: ''
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
    force_lowest: row.force_lowest,
    enable_harvest: row.enable_harvest,
    enable_plant: row.enable_plant,
    enable_sell: row.enable_sell,
    enable_weed: row.enable_weed,
    enable_bug: row.enable_bug,
    enable_water: row.enable_water,
    enable_remove_dead: row.enable_remove_dead,
    enable_upgrade_land: row.enable_upgrade_land,
    enable_help_friend: row.enable_help_friend,
    enable_claim_task: row.enable_claim_task,
    plant_crop_id: row.plant_crop_id,
    sell_crop_ids: row.sell_crop_ids,
    steal_crop_ids: row.steal_crop_ids
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
  } catch (error: unknown) {
    ElMessage.error(getErrorMessage(error, '操作失败'))
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
  } catch (error: unknown) {
    if (error !== 'cancel') {
      ElMessage.error(getErrorMessage(error, '删除失败'))
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
  } catch (error: unknown) {
    ElMessage.error(getErrorMessage(error, '操作失败'))
  }
}

const startQRLogin = async (row: Account) => {
  try {
    const response = await accountApi.getQRCode(row.id)
    qrCodeData.value = response.data
    currentQRAccountId.value = row.id
    qrDialogVisible.value = true
    startQRPolling(row.id, response.data.login_code)
  } catch (error: unknown) {
    ElMessage.error(getErrorMessage(error, '获取二维码失败'))
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
    } catch {
      // polling error - will retry on next interval
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
  fetchCrops()
})

onUnmounted(() => {
  closeQRDialog()
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
            <ElTag size="small" :type="row.platform === 'qq' ? 'success' : 'primary'" class="platform-tag">
              {{ row.platform.toUpperCase() }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <ElTag :type="getStatusType(row.status)" size="small" class="status-tag">
              {{ getStatusText(row.status) }}
            </ElTag>
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
        <ElTableColumn label="操作" width="280" fixed="right" align="center">
          <template #default="{ row }">
            <ElSpace wrap>
              <ElButton
                :type="row.status === 'running' ? 'danger' : 'success'"
                size="small"
                :icon="row.status === 'running' ? VideoPause : VideoPlay"
                @click="toggleBot(row)"
                class="action-btn"
              >
                {{ row.status === 'running' ? '停止' : '启动' }}
              </ElButton>
              <ElButton
                v-if="row.platform === 'qq'"
                type="warning"
                size="small"
                :icon="Grid"
                @click="startQRLogin(row)"
                class="action-btn action-btn--qr"
              >
                扫码
              </ElButton>
              <ElButton
                type="primary"
                size="small"
                text
                :icon="Edit"
                @click="openEditDialog(row)"
                class="action-btn"
              >
                编辑
              </ElButton>
              <ElButton
                type="danger"
                size="small"
                text
                :icon="Delete"
                @click="handleDelete(row)"
                class="action-btn"
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
      width="640px"
      destroy-on-close
      class="account-dialog"
    >
      <ElForm :model="formData" label-width="100px" class="account-form">
        <div class="form-section">
          <div class="form-section-title">基本信息</div>
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
        </div>

        <div class="form-section">
          <div class="form-section-title">自动化开关</div>
          <div class="toggle-grid">
            <div class="toggle-item">
              <span class="toggle-label">自动收获</span>
              <ElSwitch v-model="formData.enable_harvest" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">自动种植</span>
              <ElSwitch v-model="formData.enable_plant" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">自动出售</span>
              <ElSwitch v-model="formData.enable_sell" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">自动除草</span>
              <ElSwitch v-model="formData.enable_weed" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">自动除虫</span>
              <ElSwitch v-model="formData.enable_bug" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">自动浇水</span>
              <ElSwitch v-model="formData.enable_water" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">自动铲除</span>
              <ElSwitch v-model="formData.enable_remove_dead" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">升级土地</span>
              <ElSwitch v-model="formData.enable_upgrade_land" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">帮好友</span>
              <ElSwitch v-model="formData.enable_help_friend" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">允许偷菜</span>
              <ElSwitch v-model="formData.enable_steal" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">领取任务</span>
              <ElSwitch v-model="formData.enable_claim_task" />
            </div>
            <div class="toggle-item">
              <span class="toggle-label">强制最低级</span>
              <ElSwitch v-model="formData.force_lowest" />
            </div>
          </div>
        </div>

        <div class="form-section">
          <div class="form-section-title">作物选择</div>
          <ElFormItem label="种植作物">
            <ElSelect v-model="formData.plant_crop_id" filterable style="width: 100%">
              <ElOption :value="0" label="自动选择（最优经验）" />
              <ElOption 
                v-for="crop in crops" 
                :key="crop.id" 
                :value="crop.id" 
                :label="crop.required_level ? `${crop.name} (Lv.${crop.required_level})` : crop.name"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="出售作物">
            <ElSelect 
              v-model="sellCropIdsArray" 
              multiple 
              filterable 
              collapse-tags 
              collapse-tags-tooltip
              placeholder="全部出售"
              style="width: 100%"
            >
              <ElOption 
                v-for="crop in crops" 
                :key="crop.id" 
                :value="crop.id" 
                :label="crop.name"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="偷取作物">
            <ElSelect 
              v-model="stealCropIdsArray" 
              multiple 
              filterable 
              collapse-tags 
              collapse-tags-tooltip
              placeholder="全部偷取"
              style="width: 100%"
            >
              <ElOption 
                v-for="crop in crops" 
                :key="crop.id" 
                :value="crop.id" 
                :label="crop.name"
              />
            </ElSelect>
          </ElFormItem>
        </div>

        <div class="form-section">
          <div class="form-section-title">运行设置</div>
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
        </div>
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

/* Platform & Status Tags */
.platform-tag,
.status-tag {
  border-radius: 6px;
  font-weight: 500;
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
.action-btn {
  border-radius: 6px;
  font-weight: 500;
}

.action-btn.el-button--success {
  background: #15803D;
  border-color: #15803D;
  color: #fff;
}

.action-btn.el-button--success:hover {
  background: #166534;
  border-color: #166534;
  color: #fff;
}

.action-btn.el-button--danger:not(.is-text) {
  background: #DC2626;
  border-color: #DC2626;
  color: #fff;
}

.action-btn.el-button--danger:not(.is-text):hover {
  background: #B91C1C;
  border-color: #B91C1C;
  color: #fff;
}

.action-btn--qr {
  border-color: #CA8A04;
  color: #CA8A04;
  background: transparent;
}

.action-btn--qr:hover {
  background: rgba(202, 138, 4, 0.1);
  border-color: #CA8A04;
  color: #A16207;
}

.action-btn.el-button--primary.is-text {
  color: #15803D;
  background: transparent;
  border-color: transparent;
}

.action-btn.el-button--primary.is-text:hover {
  background: rgba(21, 128, 61, 0.08);
  color: #166534;
}

.action-btn.el-button--danger.is-text {
  color: #DC2626;
  background: transparent;
  border-color: transparent;
}

.action-btn.el-button--danger.is-text:hover {
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

.form-section {
  margin-bottom: 8px;
  padding-bottom: 8px;
}
.form-section-title {
  font-size: 14px;
  font-weight: 600;
  color: #14532D;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid #E5E7EB;
}
.toggle-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 24px;
}
.toggle-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0;
}
.toggle-label {
  font-size: 14px;
  color: #374151;
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
  
  .account-dialog :deep(.el-dialog) {
    width: 95% !important;
    margin: 20px auto;
  }
}
</style>
