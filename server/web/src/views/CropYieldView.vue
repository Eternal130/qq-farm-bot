<script setup lang="ts">
import { ref, computed } from 'vue'
import { cropYieldData, type CropYield } from '@/data/cropYield'
import {
  ElCard,
  ElTable,
  ElTableColumn,
  ElInput,
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
        基于 18 块地、普通肥料计算。点击表头可排序。
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
        <ElTableColumn prop="name" label="名称" min-width="110" fixed>
          <template #default="{ row }">
            <span class="crop-name">{{ row.name }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="growTime" label="生长时间" min-width="100" sortable>
          <template #default="{ row }">
            <span class="time-tag" :class="'time-' + getGrowTimeType(row.growTime)">
              {{ row.growTime }}
            </span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="growTimeFert" label="施肥后" min-width="110" sortable>
          <template #default="{ row }">
            <span class="time-tag time-outline" :class="'time-' + getGrowTimeType(row.growTimeFert)">
              {{ row.growTimeFert }}
            </span>
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
  border-radius: 16px;
  border: none;
  box-shadow: 0 1px 3px rgba(21, 128, 61, 0.06), 0 4px 16px rgba(21, 128, 61, 0.04);
}

.table-card :deep(.el-card__header) {
  padding: 20px 24px;
  border-bottom: 1px solid #E5E7EB;
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
  color: #14532D;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.crop-count {
  font-size: 13px;
  color: #6B7280;
  white-space: nowrap;
  font-weight: 500;
}

/* Search Input */
.search-input {
  width: 220px;
}

.search-input :deep(.el-input__wrapper) {
  border-radius: 8px;
  box-shadow: 0 0 0 1px #D1D5DB;
  transition: all 0.2s ease;
}

.search-input :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #9CA3AF;
}

.search-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px rgba(21, 128, 61, 0.2), 0 0 0 1px #15803D !important;
}

.search-input :deep(.el-input__prefix) {
  color: #9CA3AF;
}

/* Table Tip */
.table-tip {
  font-size: 13px;
  color: #166534;
  margin: 0 -24px;
  margin-bottom: 16px;
  padding: 12px 24px;
  background: linear-gradient(135deg, #DCFCE7 0%, #F0FDF4 100%);
  border-bottom: 1px solid #BBF7D0;
}

/* Table Styles */
.crop-table {
  --el-table-border-color: #E5E7EB;
  --el-table-header-bg-color: #F9FAFB;
  --el-table-header-text-color: #374151;
  --el-table-text-color: #14532D;
  --el-table-row-hover-bg-color: #F0FDF4;
}

.crop-table :deep(.el-table__header th) {
  font-weight: 600;
  font-size: 13px;
  padding: 14px 0;
  background: #F9FAFB;
}

.crop-table :deep(.el-table__body td) {
  padding: 14px 8px;
}

.crop-table :deep(.el-table__row--striped) {
  background: #FAFFF7;
}

/* Top 10 Rows - Gold Tint */
.crop-table :deep(.top-row) {
  background-color: #FFFBEB !important;
}

.crop-table :deep(.top-row:hover > td) {
  background-color: #FEF3C7 !important;
}

/* Top 20 Rows - Green Tint */
.crop-table :deep(.good-row) {
  background-color: #F0FDF4 !important;
}

.crop-table :deep(.good-row:hover > td) {
  background-color: #DCFCE7 !important;
}

/* Sort Icons */
.crop-table :deep(.el-table__column-filter-trigger),
.crop-table :deep(.caret-wrapper) {
  color: #9CA3AF;
}

.crop-table :deep(.el-table__column-filter-trigger:hover),
.crop-table :deep(.caret-wrapper:hover) {
  color: #15803D;
}

.crop-table :deep(.sort-caret.ascending) {
  border-bottom-color: #15803D;
}

.crop-table :deep(.sort-caret.descending) {
  border-top-color: #15803D;
}

/* Rank Values */
.rank-value {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  height: 28px;
  border-radius: 8px;
  font-weight: 700;
  font-size: 13px;
  background: #F3F4F6;
  color: #6B7280;
}

.rank-top {
  background: linear-gradient(135deg, #FEF08A 0%, #FDE047 100%);
  color: #854D0E;
}

.rank-good {
  background: linear-gradient(135deg, #BBF7D0 0%, #86EFAC 100%);
  color: #166534;
}

/* Crop Name */
.crop-name {
  font-weight: 600;
  color: #14532D;
}

/* Time Tags */
.time-tag {
  display: inline-block;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 6px;
}

.time-success {
  background: rgba(34, 197, 94, 0.12);
  color: #16A34A;
}

.time-warning {
  background: rgba(245, 158, 11, 0.12);
  color: #D97706;
}

.time-danger {
  background: rgba(220, 38, 38, 0.1);
  color: #DC2626;
}

.time-outline.time-success {
  background: transparent;
  border: 1px solid rgba(34, 197, 94, 0.3);
}

.time-outline.time-warning {
  background: transparent;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.time-outline.time-danger {
  background: transparent;
  border: 1px solid rgba(220, 38, 38, 0.25);
}

/* Value Colors */
.value-normal {
  color: #14532D;
  font-weight: 500;
}

.value-exp {
  color: #15803D;
  font-weight: 600;
}

.value-gold {
  color: #CA8A04;
  font-weight: 600;
}

.value-highlight {
  color: #15803D;
  font-weight: 700;
}

.value-gold-rate {
  color: #CA8A04;
  font-weight: 700;
}

/* Header Multi-line */
.header-multi {
  line-height: 1.35;
  text-align: center;
  display: inline-block;
}

.header-multi small {
  color: #9CA3AF;
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
