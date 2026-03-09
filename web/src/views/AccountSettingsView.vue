<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import {
  accountApi,
  cropApi,
  getErrorMessage,
  type Account,
  type CropInfo
} from '@/api'
import { cropYieldData } from '@/data/cropYield'
import {
  ElCard,
  ElButton,
  ElInputNumber,
  ElSwitch,
  ElSelect,
  ElOption,
  ElTable,
  ElTableColumn,
  ElMessage,
  ElEmpty,
  ElIcon
} from 'element-plus'
import { Setting } from '@element-plus/icons-vue'

const route = useRoute()

// Account ID from route
const accountId = computed(() => {
  const id = route.params.id
  return typeof id === 'string' ? parseInt(id, 10) : 0
})

// Data
const account = ref<Account | null>(null)
const crops = ref<CropInfo[]>([])
const isLoading = ref(false)
const isSaving = ref(false)

// Form data
const formData = ref({
  farm_interval: 2,
  friend_interval: 1,
  auto_start: false,
  enable_anti_detection: false,
  plant_crop_id: 0,
  sell_crop_ids: [] as number[],
  steal_crop_ids: [] as number[],
  force_lowest: false,
  auto_use_fertilizer: false,
  auto_buy_fertilizer: false,
  fertilizer_target_count: 0,
  fertilizer_buy_daily_limit: 0
})

// Parse comma-separated string to number array
const parseIds = (ids: string): number[] => {
  if (!ids || ids.trim() === '') return []
  return ids.split(',').map(id => parseInt(id.trim(), 10)).filter(id => !isNaN(id))
}

// Convert number array to comma-separated string
const joinIds = (ids: number[]): string => {
  return ids.join(',')
}

// Fetch account and crops data
const fetchData = async () => {
  if (accountId.value === 0) return

  isLoading.value = true
  try {
    // Fetch account
    const accountRes = await accountApi.getAll()
    const found = accountRes.data.find(a => a.id === accountId.value)
    if (found) {
      account.value = found
      // Populate form data
      formData.value = {
        farm_interval: found.farm_interval,
        friend_interval: found.friend_interval,
        auto_start: found.auto_start,
        enable_anti_detection: found.enable_anti_detection,
        plant_crop_id: found.plant_crop_id,
        sell_crop_ids: parseIds(found.sell_crop_ids),
        steal_crop_ids: parseIds(found.steal_crop_ids),
        force_lowest: found.force_lowest,
        auto_use_fertilizer: found.auto_use_fertilizer,
        auto_buy_fertilizer: found.auto_buy_fertilizer,
        fertilizer_target_count: found.fertilizer_target_count,
        fertilizer_buy_daily_limit: found.fertilizer_buy_daily_limit
      }
    }

    // Fetch crops
    const cropRes = await cropApi.getAll()
    crops.value = cropRes.data
  } catch (error: unknown) {
    const message = getErrorMessage(error, '加载数据失败')
    ElMessage.error(message)
  } finally {
    isLoading.value = false
  }
}

// Save configuration
const saveConfig = async () => {
  if (!account.value) return

  isSaving.value = true
  try {
    await accountApi.update(account.value.id, {
      farm_interval: formData.value.farm_interval,
      friend_interval: formData.value.friend_interval,
      auto_start: formData.value.auto_start,
      enable_anti_detection: formData.value.enable_anti_detection,
      plant_crop_id: formData.value.plant_crop_id,
      sell_crop_ids: joinIds(formData.value.sell_crop_ids),
      steal_crop_ids: joinIds(formData.value.steal_crop_ids),
      force_lowest: formData.value.force_lowest,
      auto_use_fertilizer: formData.value.auto_use_fertilizer,
      auto_buy_fertilizer: formData.value.auto_buy_fertilizer,
      fertilizer_target_count: formData.value.fertilizer_target_count,
      fertilizer_buy_daily_limit: formData.value.fertilizer_buy_daily_limit
    } as Parameters<typeof accountApi.update>[1])
    ElMessage.success('配置已保存')
  } catch (error: unknown) {
    const message = getErrorMessage(error, '保存失败')
    ElMessage.error(message)
  } finally {
    isSaving.value = false
  }
}

// Get row class for table highlighting
const getRowClass = ({ row }: { row: { rank: number } }): string => {
  if (row.rank <= 3) return 'top-rank-row'
  return ''
}

onMounted(() => {
  fetchData()
})
</script>

<template>
  <div class="account-settings">
    <ElEmpty v-if="!account && !isLoading" description="账号不存在" />

    <div v-else class="settings-container">
      <!-- Section 1: Parameter Config -->
      <ElCard class="config-card" shadow="never">
        <template #header>
          <div class="card-header">
            <ElIcon><Setting /></ElIcon>
            <span>参数配置</span>
          </div>
        </template>

        <div class="config-form">
          <!-- Basic Settings -->
          <div class="form-section">
            <div class="form-row">
              <div class="form-item">
                <label class="form-label">农场巡查间隔</label>
                <div class="input-with-unit">
                  <ElInputNumber
                    v-model="formData.farm_interval"
                    :min="1"
                    :max="3600"
                    :step="1"
                    controls-position="right"
                  />
                  <span class="unit">秒</span>
                </div>
              </div>
              <div class="form-item">
                <label class="form-label">好友巡查间隔</label>
                <div class="input-with-unit">
                  <ElInputNumber
                    v-model="formData.friend_interval"
                    :min="1"
                    :max="3600"
                    :step="1"
                    controls-position="right"
                  />
                  <span class="unit">秒</span>
                </div>
              </div>
            </div>

            <div class="form-row">
              <div class="form-item switch-item">
                <label class="form-label">自动启动</label>
                <ElSwitch v-model="formData.auto_start" />
              </div>
              <div class="form-item switch-item">
                <div class="label-with-desc">
                  <label class="form-label">防检测模式</label>
                  <span class="form-desc">随机化操作间隔，降低被检测风险</span>
                </div>
                <ElSwitch v-model="formData.enable_anti_detection" />
              </div>
            </div>
          </div>

          <!-- Crop Selection -->
          <div class="form-section">
            <div class="section-title">作物选择</div>

            <div class="form-row">
              <div class="form-item">
                <label class="form-label">种植作物</label>
                <ElSelect
                  v-model="formData.plant_crop_id"
                  placeholder="自动选最优"
                  clearable
                  class="full-width"
                >
                  <ElOption :value="0" label="自动选最优" />
                  <ElOption
                    v-for="crop in crops"
                    :key="crop.id"
                    :value="crop.id"
                    :label="`${crop.name} (Lv.${crop.required_level})`"
                  />
                </ElSelect>
              </div>
              <div class="form-item switch-item">
                <label class="form-label">强制最低级</label>
                <ElSwitch v-model="formData.force_lowest" />
              </div>
            </div>

            <div class="form-row">
              <div class="form-item">
                <label class="form-label">出售作物</label>
                <ElSelect
                  v-model="formData.sell_crop_ids"
                  multiple
                  collapse-tags
                  collapse-tags-tooltip
                  placeholder="全部出售"
                  clearable
                  class="full-width"
                >
                  <ElOption
                    v-for="crop in crops"
                    :key="crop.id"
                    :value="crop.id"
                    :label="crop.name"
                  />
                </ElSelect>
              </div>
              <div class="form-item">
                <label class="form-label">偷取作物</label>
                <ElSelect
                  v-model="formData.steal_crop_ids"
                  multiple
                  collapse-tags
                  collapse-tags-tooltip
                  placeholder="全部偷取"
                  clearable
                  class="full-width"
                >
                  <ElOption
                    v-for="crop in crops"
                    :key="crop.id"
                    :value="crop.id"
                    :label="crop.name"
                  />
                </ElSelect>
              </div>
            </div>
          </div>

          <!-- Fertilizer Settings -->
          <div class="form-section">
            <div class="section-title">肥料管理</div>

            <div class="form-row">
              <div class="form-item switch-item">
                <label class="form-label">自动使用肥料</label>
                <ElSwitch v-model="formData.auto_use_fertilizer" />
              </div>
              <div class="form-item switch-item">
                <label class="form-label">自动购买肥料</label>
                <ElSwitch v-model="formData.auto_buy_fertilizer" />
              </div>
            </div>

            <div class="form-row">
              <div class="form-item">
                <label class="form-label">肥料目标数量</label>
                <ElInputNumber
                  v-model="formData.fertilizer_target_count"
                  :min="0"
                  :max="9999"
                  :step="10"
                  controls-position="right"
                  class="full-width"
                />
              </div>
              <div class="form-item">
                <label class="form-label">每日购买上限</label>
                <ElInputNumber
                  v-model="formData.fertilizer_buy_daily_limit"
                  :min="0"
                  :max="999"
                  :step="1"
                  controls-position="right"
                  class="full-width"
                />
              </div>
            </div>
          </div>

          <!-- Save Button -->
          <div class="form-actions">
            <ElButton
              type="primary"
              :loading="isSaving"
              @click="saveConfig"
              class="save-btn"
            >
              保存配置
            </ElButton>
          </div>
        </div>
      </ElCard>

      <!-- Section 2: Crop Efficiency Ranking -->
      <ElCard class="ranking-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>种植效率排行</span>
          </div>
        </template>

        <ElTable
          :data="cropYieldData"
          :row-class-name="getRowClass"
          class="ranking-table"
          max-height="600"
        >
          <ElTableColumn prop="rank" label="排名" width="70" align="center">
            <template #default="{ row }">
              <span :class="['rank-badge', row.rank <= 3 ? 'top-rank' : '']">
                {{ row.rank }}
              </span>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="name" label="名称" min-width="100" />
          <ElTableColumn prop="requiredLevel" label="等级" width="70" align="center">
            <template #default="{ row }">
              <span class="level-text">Lv.{{ row.requiredLevel }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="expPerMinFert" label="经验/分钟" width="100" align="right">
            <template #default="{ row }">
              <span class="exp-value">{{ row.expPerMinFert.toFixed(2) }}</span>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>
    </div>
  </div>
</template>

<style scoped>
.account-settings {
  padding: 0;
}

.settings-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
    gap: var(--space-5);
}

@media (max-width: 1200px) {
  .settings-container {
    grid-template-columns: 1fr;
  }
}

/* Card Styles */
.config-card,
.ranking-card {
    background-color: var(--bg-card);
    border: none;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
}

.config-card :deep(.el-card__header),
.ranking-card :deep(.el-card__header) {
    padding: var(--space-4) var(--space-6);
    border-bottom: 1px solid var(--border-light);
}

.config-card :deep(.el-card__body),
.ranking-card :deep(.el-card__body) {
    padding: var(--space-6);
}

.card-header {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--text-heading);
    font-size: 16px;
    font-weight: 600;
 }

/* Form Styles */
.config-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
 }

.form-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
 }

.section-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-secondary);
    padding-bottom: var(--space-2);
    border-bottom: 1px solid var(--border-light);
}

.form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-4);
 }

@media (max-width: 640px) {
    .form-row {
        grid-template-columns: 1fr;
    }
}

.form-item {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
 }

.form-item.switch-item {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-3) var(--space-4);
    background-color: var(--bg-elevated);
    border-radius: var(--radius-sm);
    border: none;
}

.form-label {
    font-size: 14px;
    color: var(--text-primary);
    font-weight: 500;
 }

.label-with-desc {
    display: flex;
    flex-direction: column;
    gap: 2px;
 }

.form-desc {
    font-size: 11px;
    color: var(--text-muted);
}

.input-with-unit {
    display: flex;
    align-items: center;
    gap: var(--space-2);
 }

.input-with-unit .unit {
    font-size: 14px;
    color: var(--text-secondary);
    flex-shrink: 0;
}

.full-width {
    width: 100%;
}

/* Input Number Dark Theme */
.form-item :deep(.el-input-number) {
    width: 100%;
}

.form-item :deep(.el-input-number .el-input__wrapper) {
    background-color: var(--bg-elevated);
    border-color: var(--border-light);
    box-shadow: none;
}

.form-item :deep(.el-input-number .el-input__inner) {
    color: var(--text-primary);
 }

.form-item :deep(.el-input-number .el-input-number__decrease),
.form-item :deep(.el-input-number .el-input-number__increase) {
    background-color: var(--bg-card);
    border-color: var(--border-light);
    color: var(--text-secondary);
 }

/* Select Dark Theme */
.form-item :deep(.el-select) {
    width: 100%;
}

.form-item :deep(.el-select .el-input__wrapper) {
    background-color: var(--bg-elevated);
    border-color: var(--border-light);
    box-shadow: none;
 }

.form-item :deep(.el-select .el-input__inner) {
    color: var(--text-primary);
 }

.form-item :deep(.el-select .el-input__suffix) {
    color: var(--text-secondary);
 }

/* Switch Dark Theme */
.switch-item :deep(.el-switch) {
    --el-switch-off-color: var(--border-light);
 }

/* Save Button */
.form-actions {
    display: flex;
    justify-content: flex-end;
    padding-top: var(--space-2);
 }

.save-btn {
    background-color: var(--primary);
    border-color: var(--primary);
    padding: 10px 32px;
    font-weight: 500;
 transition: all var(--transition);
 }

.save-btn:hover {
    background-color: var(--primary-hover);
    border-color: var(--primary-hover);
 }

/* Table Dark Theme */
.ranking-table {
    --el-table-bg-color: var(--bg-card);
    --el-table-header-bg-color: var(--bg-elevated);
    --el-table-tr-bg-color: var(--bg-card);
    --el-table-row-hover-bg-color: var(--bg-hover);
    --el-table-border-color: var(--border-light);
    --el-table-text-color: var(--text-primary);
    --el-table-header-text-color: var(--text-secondary);
    background-color: transparent;
 }

.ranking-table :deep(.el-table__cell) {
    border-bottom: 1px solid var(--border-light);
 }

.ranking-table :deep(.el-table__row) {
    transition: background-color var(--transition);
 }

/* Top Rank Row Highlight */
.ranking-table :deep(.top-rank-row) {
    background-color: var(--warning-bg);
 }

.ranking-table :deep(.top-rank-row:hover > td) {
    background-color: rgba(255, 159, 10, 0.12) !important;
 }

.rank-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 24px;
    height: 24px;
    border-radius: var(--radius-xs);
    font-size: 11px;
    font-weight: 600;
    background-color: var(--bg-elevated);
    color: var(--text-secondary);
 }

.rank-badge.top-rank {
    background-color: var(--warning-bg);
    color: var(--gold);
 }

.level-text {
    font-size: 13px;
    color: var(--text-secondary);
 }

.exp-value {
    font-size: 14px;
    font-weight: 600;
    color: var(--success);
 }

/* Scrollbar */
.ranking-table :deep(.el-scrollbar__wrap) {
    overflow-x: hidden;
 }
</style>
