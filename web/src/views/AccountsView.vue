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
  ElRadioGroup,
  ElRadioButton,
  ElCard
} from 'element-plus'
import { Plus, Edit, Delete, VideoPlay, VideoPause, Grid, CopyDocument, Key } from '@element-plus/icons-vue'
import QRCode from '@/components/QRCode.vue'
import type { QRStylePreset } from '@/components/QRCode.vue'

const loading = ref(false)
const accounts = ref<Account[]>([])
const crops = ref<CropInfo[]>([])
const dialogVisible = ref(false)
const qrDialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref<number | null>(null)

const formData = ref<CreateAccountRequest>({
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
  steal_crop_ids: '',
  enable_anti_detection: false,
  prefer_bag_seeds: false,
  planting_strategy: '',
  api_key: ''
})

const qrCodeData = ref<QRCodeResponse | null>(null)
const qrPolling = ref(false)
const currentQRAccountId = ref<number | null>(null)
let qrPollInterval: number | null = null
const autoStartAfterQR = ref(false)
const qrStylePreset = ref<QRStylePreset>('rounded')

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

const generateAPIKey = () => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  const segments = []
  for (let s = 0; s < 4; s++) {
    let segment = ''
    for (let i = 0; i < 8; i++) {
      segment += chars.charAt(Math.floor(Math.random() * chars.length))
    }
    segments.push(segment)
  }
  formData.value.api_key = 'fk_' + segments.join('-')
}

const copyAPIKey = async () => {
  if (!formData.value.api_key) return
  try {
    await navigator.clipboard.writeText(formData.value.api_key)
    ElMessage.success('API Key 已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

const clearAPIKey = () => {
  formData.value.api_key = ''
}

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
    steal_crop_ids: '',
    enable_anti_detection: false,
    prefer_bag_seeds: false,
    planting_strategy: '',
    api_key: ''
  }
  dialogVisible.value = true
}

const openEditDialog = (row: Account) => {
  isEdit.value = true
  currentId.value = row.id
  formData.value = {
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
    steal_crop_ids: row.steal_crop_ids,
    enable_anti_detection: row.enable_anti_detection,
    prefer_bag_seeds: row.prefer_bag_seeds,
    planting_strategy: row.planting_strategy || '',
    api_key: row.api_key || ''
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
            <span class="account-name">{{ row.name || '账号 #' + row.id }}</span>
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
          <ElFormItem label="背包优先">
            <div class="switch-row">
              <ElSwitch v-model="formData.prefer_bag_seeds" />
              <span class="form-hint">优先使用背包内已有的种子进行种植</span>
            </div>
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
                :max="3600"
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
                :max="3600"
                style="width: 120px"
              />
              <span class="form-hint">秒</span>
            </div>
          </ElFormItem>
          <ElFormItem label="防检测">
            <div class="switch-row">
              <ElSwitch v-model="formData.enable_anti_detection" />
              <span class="form-hint">随机化访问顺序与间隔，降低被检测风险</span>
            </div>
          </ElFormItem>
        </div>

        <div class="form-section">
          <div class="form-section-title">外部 API</div>
          <ElFormItem label="API Key">
            <div class="api-key-input">
              <ElInput
                v-model="formData.api_key"
                placeholder="点击生成按钮创建 API Key"
                readonly
                class="api-key-field"
              />
              <ElButton
                v-if="formData.api_key"
                :icon="CopyDocument"
                @click="copyAPIKey"
                class="api-key-btn"
                title="复制"
              />
              <ElButton
                v-if="formData.api_key"
                type="danger"
                text
                :icon="Delete"
                @click="clearAPIKey"
                class="api-key-btn"
                title="清除"
              />
              <ElButton
                type="primary"
                :icon="Key"
                @click="generateAPIKey"
                class="api-key-btn"
              >
                {{ formData.api_key ? '重新生成' : '生成' }}
              </ElButton>
            </div>
            <div class="form-hint" style="margin-top: 6px;">
              设置后可通过此 Key 调用外部 API，仅能操作当前账号
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
      width="440px"
      @close="closeQRDialog"
      class="qr-dialog"
    >
      <div class="qr-container">
        <p class="qr-tip">请使用手机QQ扫描下方二维码登录</p>
        <div class="qr-image-wrapper">
          <QRCode 
            v-if="qrCodeData"
            :data="qrCodeData.qr_code_url"
            :width="200"
            :height="200"
            :preset="qrStylePreset"
          />
        </div>
        
        <!-- QR Style Selector -->
        <div class="qr-style-selector">
          <ElRadioGroup v-model="qrStylePreset" size="small">
            <ElRadioButton value="rounded">圆角</ElRadioButton>
            <ElRadioButton value="dots">点状</ElRadioButton>
            <ElRadioButton value="elegant">优雅</ElRadioButton>
            <ElRadioButton value="colorful">炫彩</ElRadioButton>
          </ElRadioGroup>
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
  border-radius: var(--radius-lg);
  border: none;
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
}

.table-card :deep(.el-card__header) {
  padding: var(--space-5) var(--space-6);
  border-bottom: 1px solid var(--border-light);
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
  color: var(--text-heading);
}

.add-btn {
  background: var(--primary) !important;
  border-color: var(--primary) !important;
  border-radius: var(--radius-sm);
  font-weight: 600;
  transition: all var(--transition);
}

.add-btn:hover {
  background: var(--primary-hover) !important;
  border-color: var(--primary-hover) !important;
}

/* Table Styles */
.accounts-table {
  --el-table-border-color: var(--border-light);
  --el-table-bg-color: var(--bg-card);
  --el-table-header-bg-color: var(--bg-elevated);
  --el-table-header-text-color: var(--text-secondary);
  --el-table-text-color: var(--text-primary);
  --el-table-row-hover-bg-color: var(--bg-hover);
  --el-table-tr-bg-color: var(--bg-card);
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
  background: var(--bg-elevated);
}

.account-name {
  font-weight: 500;
  color: var(--text-primary);
}

/* Platform & Status Tags */
.platform-tag,
.status-tag {
  border-radius: var(--radius-xs);
  font-weight: 500;
}

.platform-tag:deep(.el-tag--success) {
  background: var(--success-bg);
  border-color: transparent;
  color: var(--success);
}

.platform-tag:deep(.el-tag--primary) {
  background: var(--primary-bg);
  border-color: transparent;
  color: var(--primary);
}

.status-tag:deep(.el-tag--success) {
  background: var(--success-bg);
  border-color: transparent;
  color: var(--success);
}

.status-tag:deep(.el-tag--danger) {
  background: var(--danger-bg);
  border-color: transparent;
  color: var(--danger);
}

.status-tag:deep(.el-tag--info) {
  background: rgba(142, 142, 147, 0.1);
  border-color: transparent;
  color: var(--text-secondary);
}

.level-value {
  font-weight: 600;
  color: var(--success);
}

.gold-value {
  font-weight: 600;
  color: var(--gold);
}

/* Action Buttons */
.action-btn {
  border-radius: var(--radius-xs);
  font-weight: 500;
  transition: all var(--transition);
}

.action-btn.el-button--success {
  background: var(--success);
  border-color: var(--success);
  color: #FFFFFF;
}

.action-btn.el-button--success:hover {
  background: color-mix(in srgb, var(--success), #000 15%);
  border-color: color-mix(in srgb, var(--success), #000 15%);
  color: #FFFFFF;
}

.action-btn.el-button--danger:not(.is-text) {
  background: var(--danger);
  border-color: var(--danger);
  color: #FFFFFF;
}

.action-btn.el-button--danger:not(.is-text):hover {
  background: color-mix(in srgb, var(--danger), #000 15%);
  border-color: color-mix(in srgb, var(--danger), #000 15%);
  color: #FFFFFF;
}

.action-btn--qr {
  border-color: var(--gold);
  color: var(--gold);
  background: transparent;
}

.action-btn--qr:hover {
  background: var(--warning-bg);
  border-color: var(--gold);
  color: var(--gold-light);
}

.action-btn.el-button--primary.is-text {
  color: var(--primary);
  background: transparent;
  border-color: transparent;
}

.action-btn.el-button--primary.is-text:hover {
  background: var(--primary-bg);
  color: var(--primary-light);
}

.action-btn.el-button--danger.is-text {
  color: var(--danger);
  background: transparent;
  border-color: transparent;
}

.action-btn.el-button--danger.is-text:hover {
  background: var(--danger-bg);
  color: var(--danger-light);
}

/* Dialog Styles */
.account-dialog :deep(.el-dialog) {
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  border: none;
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}

.account-dialog :deep(.el-dialog__header) {
  padding: var(--space-5) var(--space-6);
  border-bottom: 1px solid var(--border-light);
  margin-right: 0;
  background: var(--bg-card);
}

.account-dialog :deep(.el-dialog__title) {
  font-weight: 600;
  color: var(--text-heading);
}

.account-dialog :deep(.el-dialog__headerbtn) {
  top: var(--space-5);
}

.account-dialog :deep(.el-dialog__headerbtn .el-dialog__close) {
  color: var(--text-muted);
}

.account-dialog :deep(.el-dialog__headerbtn:hover .el-dialog__close) {
  color: var(--text-primary);
}

.account-dialog :deep(.el-dialog__body) {
  padding: var(--space-6);
  background: var(--bg-card);
  max-height: 65vh;
  overflow-y: auto;
}

.account-form :deep(.el-form-item) {
  margin-bottom: var(--space-5);
}

.account-form :deep(.el-form-item__label) {
  font-weight: 500;
  color: var(--text-secondary);
}

.account-form :deep(.el-input__wrapper),
.account-form :deep(.el-textarea__inner),
.account-form :deep(.el-select .el-input__wrapper) {
  border-radius: var(--radius-sm);
  background-color: var(--bg-elevated) !important;
  box-shadow: 0 0 0 1px var(--border-light) inset !important;
  transition: all var(--transition);
}

.account-form :deep(.el-input__wrapper:hover),
.account-form :deep(.el-textarea__inner:hover),
.account-form :deep(.el-select .el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px var(--text-muted) inset !important;
}

.account-form :deep(.el-input__wrapper.is-focus),
.account-form :deep(.el-textarea__inner:focus),
.account-form :deep(.el-select .el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--primary) inset !important;
}

.account-form :deep(.el-input__inner),
.account-form :deep(.el-textarea__inner) {
  color: var(--text-primary) !important;
}

.account-form :deep(.el-input__inner::placeholder),
.account-form :deep(.el-textarea__inner::placeholder) {
  color: var(--text-muted) !important;
}

.interval-input {
  display: flex;
  align-items: center;
  gap: 10px;
}

.switch-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.form-hint {
  font-size: 13px;
  color: var(--text-muted);
}

.api-key-input {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
}

.api-key-field {
  flex: 1;
}

.api-key-field :deep(.el-input__wrapper) {
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  letter-spacing: 0.5px;
  background: var(--bg-elevated) !important;
}

.api-key-btn {
  flex-shrink: 0;
}

.form-section {
  margin-bottom: var(--space-2);
  padding-bottom: var(--space-2);
}

.form-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-heading);
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--border-light);
}

.toggle-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3) var(--space-6);
}

.toggle-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0;
}

.toggle-label {
  font-size: 14px;
  color: var(--text-secondary);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-6);
  border-top: 1px solid var(--border-light);
  background: var(--bg-card);
}

.btn-cancel {
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  border-color: var(--border-light);
  color: var(--text-secondary);
}

.btn-cancel:hover {
  background: var(--bg-hover);
  border-color: var(--text-muted);
  color: var(--text-primary);
}

.btn-submit {
  background: var(--primary);
  border-color: var(--primary);
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.btn-submit:hover {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}

/* QR Dialog */
.qr-dialog :deep(.el-dialog) {
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  border: none;
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}

.qr-dialog :deep(.el-dialog__header) {
  padding: var(--space-5) var(--space-6);
  border-bottom: 1px solid var(--border-light);
  margin-right: 0;
  background: var(--bg-card);
}

.qr-dialog :deep(.el-dialog__title) {
  color: var(--text-heading);
}

.qr-dialog :deep(.el-dialog__body) {
  padding: 0;
  background: var(--bg-card);
}

.qr-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--space-8) var(--space-6);
}

.qr-tip {
  color: var(--text-secondary);
  font-size: 14px;
  margin-bottom: var(--space-6);
  text-align: center;
}

.qr-image-wrapper {
  background: #FFFFFF;
  border: 2px solid var(--border-light);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  box-shadow: var(--shadow-md);
}

.qr-style-selector {
  margin-top: var(--space-5);
  display: flex;
  justify-content: center;
}

.qr-style-selector :deep(.el-radio-group) {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--space-2);
}

.qr-style-selector :deep(.el-radio-button__inner) {
  border-radius: var(--radius-xs) !important;
  padding: 6px 12px;
  font-size: 12px;
  background: var(--bg-elevated);
  border-color: var(--border-light);
  color: var(--text-secondary);
}

.qr-style-selector :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  background: var(--primary);
  border-color: var(--primary);
  color: #FFFFFF;
  box-shadow: none;
}

.qr-style-selector :deep(.el-radio-button:first-child .el-radio-button__inner) {
  border-radius: var(--radius-xs) !important;
}

.qr-style-selector :deep(.el-radio-button:last-child .el-radio-button__inner) {
  border-radius: var(--radius-xs) !important;
}

.qr-status {
  margin-top: var(--space-6);
  color: var(--success);
  font-size: 14px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.waiting-dot {
  width: 8px;
  height: 8px;
  background-color: var(--success);
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
    padding: var(--space-4);
  }
  
  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-3);
  }
  
  .account-dialog :deep(.el-dialog) {
    width: 95% !important;
    margin: 20px auto;
  }
}
</style>
