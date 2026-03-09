<script setup lang="ts">
import { ref, computed } from 'vue'
import { cropYieldData, type CropYield } from '@/data/cropYield'
import {
  ElCard,
  ElTable,
  ElTableColumn,
  ElInput,
  ElTag
} from 'element-plus'
import { Search } from '@element-plus/icons-vue'

const searchQuery = ref('')

const filteredData = computed(() => {
  if (!searchQuery.value) return cropYieldData
  const query = searchQuery.value.toLowerCase()
  return cropYieldData.filter(crop => crop.name.toLowerCase().includes(query))
})

const getGrowTimeType = (growTime: string): 'success' | 'warning' | 'danger' | 'info' => {
  if (growTime.includes('分') && !growTime.includes('时')) return 'success'
  if (growTime.includes('时') && !growTime.includes('24')) return 'warning'
  return 'danger'
}

const formatRate = (val: number): string => {
  return val.toFixed(2)
}

const tableRowClassName = ({ row }: { row: CropYield }) => {
  if (row.rank <= 10) return 'top-row'
  if (row.rank <= 20) return 'good-row'
  return ''
}
</script>

<template>
  <div class="crop-yield-view">
    <ElCard shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span class="title">作物收益表</span>
          <div class="header-right">
            <span class="crop-count">共 {{ filteredData.length }} 种作物</span>
            <ElInput
              v-model="searchQuery"
              placeholder="搜索作物名称"
              :prefix-icon="Search"
              clearable
              class="search-input"
            />
          </div>
        </div>
      </template>

      <div class="table-tip">
        基于 18 块地、普通肥料、最优阶段施肥计算。多季作物显示全季合计。点击表头可排序。
      </div>

      <ElTable
        :data="filteredData"
        stripe
        :row-class-name="tableRowClassName"
        :default-sort="{ prop: 'expPerMinFert', order: 'descending' }"
        style="width: 100%"
        max-height="calc(100vh - 260px)"
        class="crop-table"
      >
        <ElTableColumn prop="rank" label="排名" width="70" sortable align="center" fixed>
          <template #default="{ row }">
            <span class="rank-value" :class="{ 'rank-top': row.rank <= 10, 'rank-good': row.rank > 10 && row.rank <= 20 }">
              {{ row.rank }}
            </span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="name" label="名称" min-width="140" fixed>
          <template #default="{ row }">
            <span class="crop-name">{{ row.name }}</span>
            <ElTag type="info" size="small" class="level-tag">Lv.{{ row.requiredLevel }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="seasons" label="季" width="60" sortable align="center">
          <template #default="{ row }">
            <ElTag v-if="row.seasons >= 2" type="warning" size="small" effect="plain" class="season-tag">{{ row.seasons }}季</ElTag>
            <span v-else class="value-normal">1</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="growTime" label="生长时间" min-width="100" sortable>
          <template #default="{ row }">
            <ElTag :type="getGrowTimeType(row.growTime)" size="small" class="time-tag">
              {{ row.growTime }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="growTimeFert" label="施肥后" min-width="110" sortable>
          <template #default="{ row }">
            <ElTag :type="getGrowTimeType(row.growTimeFert)" size="small" effect="plain" class="time-tag">
              {{ row.growTimeFert }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="harvestExp" label="收获经验" min-width="100" sortable align="right">
          <template #default="{ row }">
            <span class="value-exp">{{ row.harvestExp.toLocaleString() }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="fruitCount" label="果实数量" min-width="90" sortable align="right" />
        <ElTableColumn prop="fruitPrice" label="果实单价" min-width="90" sortable align="right">
          <template #default="{ row }">
            <span class="value-gold">{{ row.fruitPrice }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="expPerMinNoFert" label="经验/分钟" min-width="105" sortable align="right">
          <template #header>
            <span class="header-multi">经验/分钟<br /><small>不施肥</small></span>
          </template>
          <template #default="{ row }">
            <span class="value-normal">{{ formatRate(row.expPerMinNoFert) }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="expPerMinFert" label="经验/分钟(施肥)" min-width="105" sortable align="right">
          <template #header>
            <span class="header-multi">经验/分钟<br /><small>施肥</small></span>
          </template>
          <template #default="{ row }">
            <span class="value-highlight">{{ formatRate(row.expPerMinFert) }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="goldPerMinNoFert" label="金币/分钟" min-width="105" sortable align="right">
          <template #header>
            <span class="header-multi">金币/分钟<br /><small>不施肥</small></span>
          </template>
          <template #default="{ row }">
            <span class="value-normal">{{ formatRate(row.goldPerMinNoFert) }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="goldPerMinFert" label="金币/分钟(施肥)" min-width="105" sortable align="right">
          <template #header>
            <span class="header-multi">金币/分钟<br /><small>施肥</small></span>
          </template>
          <template #default="{ row }">
            <span class="value-gold-rate">{{ formatRate(row.goldPerMinFert) }}</span>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<style scoped>
.crop-yield-view {
  padding: 0;
}

/* Table Card */
.table-card {
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
}

.table-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.table-card :deep(.el-card__body) {
  padding: 0 24px 24px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-heading);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.crop-count {
  font-size: 13px;
  color: var(--text-muted);
  white-space: nowrap;
  font-weight: 500;
}

/* Search Input */
.search-input {
  width: 220px;
}

.search-input :deep(.el-input__wrapper) {
  border-radius: var(--radius);
  box-shadow: 0 0 0 1px var(--border) inset !important;
  background-color: var(--bg-input) !important;
  transition: all 0.2s ease;
}

.search-input :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px var(--border-focus) inset !important;
}

.search-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px var(--primary) inset !important;
}

.search-input :deep(.el-input__inner) {
  color: var(--text-primary) !important;
}

.search-input :deep(.el-input__inner::placeholder) {
  color: var(--text-muted) !important;
}

.search-input :deep(.el-input__prefix) {
  color: var(--text-muted);
}

/* Table Tip */
.table-tip {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 -24px;
  margin-bottom: 16px;
  padding: 12px 24px;
  background: var(--primary-bg);
  border-bottom: 1px solid rgba(59, 130, 246, 0.2);
}

/* Table Styles */
.crop-table {
  --el-table-bg-color: var(--bg-card);
  --el-table-tr-bg-color: var(--bg-card);
  --el-table-header-bg-color: var(--bg-elevated);
  --el-table-row-hover-bg-color: var(--bg-elevated);
  --el-table-border-color: var(--border);
  --el-table-text-color: var(--text-primary);
  --el-table-header-text-color: var(--text-secondary);
}

.crop-table :deep(.el-table__header th) {
  font-weight: 600;
  font-size: 13px;
  padding: 14px 0;
  background: var(--bg-elevated);
}

.crop-table :deep(.el-table__body td) {
  padding: 14px 8px;
}

.crop-table :deep(.el-table__row--striped) {
  background: var(--bg-hover);
}

/* Top 10 Rows - Gold Tint */
.crop-table :deep(.top-row) {
  background-color: rgba(234, 179, 8, 0.08) !important;
}

.crop-table :deep(.top-row:hover > td) {
  background-color: rgba(234, 179, 8, 0.15) !important;
}

/* Top 20 Rows - Green Tint */
.crop-table :deep(.good-row) {
  background-color: rgba(34, 197, 94, 0.05) !important;
}

.crop-table :deep(.good-row:hover > td) {
  background-color: rgba(34, 197, 94, 0.1) !important;
}

/* Sort Icons */
.crop-table :deep(.el-table__column-filter-trigger),
.crop-table :deep(.caret-wrapper) {
  color: var(--text-muted);
}

.crop-table :deep(.el-table__column-filter-trigger:hover),
.crop-table :deep(.caret-wrapper:hover) {
  color: var(--primary);
}

.crop-table :deep(.sort-caret.ascending) {
  border-bottom-color: var(--primary);
}

.crop-table :deep(.sort-caret.descending) {
  border-top-color: var(--primary);
}

/* Rank Values */
.rank-value {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  height: 28px;
  border-radius: var(--radius);
  font-weight: 700;
  font-size: 13px;
  background: var(--bg-elevated);
  color: var(--text-muted);
}

.rank-top {
  background: linear-gradient(135deg, var(--gold) 0%, var(--gold-light) 100%);
  color: var(--gold);
  border: 1px solid rgba(234, 179, 8, 0.3);
}

.rank-good {
  background: linear-gradient(135deg, var(--success) 0%, var(--success-bg) 100%);
  color: var(--success);
  border: 1px solid rgba(34, 197, 94, 0.2);
}

/* Crop Name */
.crop-name {
  font-weight: 600;
  color: var(--text-primary);
}

.level-tag {
  margin-left: 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  vertical-align: middle;
}

/* Time Tags */
.time-tag {
  border-radius: var(--radius-sm);
  font-weight: 600;
}

.season-tag {
  border-radius: var(--radius-sm);
  font-weight: 600;
}

/* Value Colors */
.value-normal {
  color: var(--text-secondary);
  font-weight: 500;
}

.value-exp {
  color: var(--success);
  font-weight: 600;
}

.value-gold {
  color: var(--gold);
  font-weight: 600;
}

.value-highlight {
  color: var(--success);
  font-weight: 700;
}

.value-gold-rate {
  color: var(--gold);
  font-weight: 700;
}

/* Header Multi-line */
.header-multi {
  line-height: 1.35;
  text-align: center;
  display: inline-block;
}

.header-multi small {
  color: var(--text-muted);
  font-weight: normal;
  font-size: 11px;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .table-card :deep(.el-card__header) {
    padding: 16px;
  }

  .table-card :deep(.el-card__body) {
    padding: 0 16px 16px;
  }

  .table-tip {
    margin: 0 -16px;
    margin-bottom: 12px;
    padding: 10px 16px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .header-right {
    width: 100%;
    justify-content: space-between;
  }

  .search-input {
    width: 160px;
  }
}
</style>
