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
        基于 18 块地、普通肥料计算。点击表头可排序。
      </div>

      <ElTable
        :data="filteredData"
        stripe
        :row-class-name="tableRowClassName"
        :default-sort="{ prop: 'expPerMinFert', order: 'descending' }"
        style="width: 100%"
        max-height="calc(100vh - 260px)"
      >
        <ElTableColumn prop="rank" label="排名" width="70" sortable align="center" fixed />
        <ElTableColumn prop="name" label="名称" min-width="110" fixed>
          <template #default="{ row }">
            <span class="crop-name">{{ row.name }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="growTime" label="生长时间" min-width="100" sortable>
          <template #default="{ row }">
            <ElTag :type="getGrowTimeType(row.growTime)" size="small">
              {{ row.growTime }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="growTimeFert" label="施肥后" min-width="110" sortable>
          <template #default="{ row }">
            <ElTag :type="getGrowTimeType(row.growTimeFert)" size="small" effect="plain">
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
            {{ formatRate(row.expPerMinNoFert) }}
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
            {{ formatRate(row.goldPerMinNoFert) }}
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

.table-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.title {
  font-size: 16px;
  font-weight: 500;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.crop-count {
  font-size: 13px;
  color: #909399;
  white-space: nowrap;
}

.search-input {
  width: 200px;
}

.table-tip {
  font-size: 12px;
  color: #909399;
  margin-bottom: 12px;
  padding: 8px 12px;
  background-color: #f4f4f5;
  border-radius: 4px;
}

.crop-name {
  font-weight: 500;
  color: #303133;
}

.value-exp {
  color: #409eff;
  font-weight: 500;
}

.value-gold {
  color: #e6a23c;
  font-weight: 500;
}

.value-highlight {
  color: #409eff;
  font-weight: 600;
}

.value-gold-rate {
  color: #e6a23c;
  font-weight: 600;
}

.header-multi {
  line-height: 1.3;
  text-align: center;
}

.header-multi small {
  color: #909399;
  font-weight: normal;
}

:deep(.top-row) {
  background-color: #fdf6ec !important;
}

:deep(.good-row) {
  background-color: #f0f9eb !important;
}

:deep(.el-table .cell) {
  padding: 0 8px;
}

:deep(.el-table th .cell) {
  font-size: 13px;
}

@media (max-width: 768px) {
  .search-input {
    width: 150px;
  }

  .header-right {
    gap: 8px;
  }
}
</style>
