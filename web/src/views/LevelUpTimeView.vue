<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  calculateLevelUps,
  formatTimeShort,
  MAX_LEVEL,
  DEFAULT_LANDS,
  DEFAULT_LAND_GRADE,
  getLandGradeBuff,
  type LevelUpInfo
} from '@/data/levelUpCalc'
import {
  ElCard,
  ElTable,
  ElTableColumn,
  ElSwitch,
  ElTag,
  ElSlider
} from 'element-plus'

const showFertComparison = ref(false)
const numLands = ref(DEFAULT_LANDS)
const levelRange = ref<[number, number]>([1, MAX_LEVEL - 1])
const landGrade = ref(DEFAULT_LAND_GRADE)

const landGradeBuff = computed(() => getLandGradeBuff(landGrade.value))

const levelUpData = computed(() =>
  calculateLevelUps(numLands.value, levelRange.value[0], levelRange.value[1], landGrade.value),
)

/** Cumulative time from the start of the displayed range to each row. */
const cumulativeNoFert = computed(() => {
  let acc = 0
  return levelUpData.value.map(row => {
    acc += row.noFert.totalTimeSec
    return acc
  })
})
const cumulativeWithFert = computed(() => {
  let acc = 0
  return levelUpData.value.map(row => {
    acc += row.withFert.totalTimeSec
    return acc
  })
})
const getTimeType = (sec: number): 'success' | 'warning' | 'danger' | 'info' => {
  if (sec <= 3600) return 'success'       // ≤ 1h
  if (sec <= 43200) return 'warning'      // ≤ 12h
  return 'danger'                         // > 12h
}

const formatExp = (val: number): string => val.toLocaleString()

const timeSaved = (row: LevelUpInfo): number =>
  row.noFert.totalTimeSec - row.withFert.totalTimeSec

const timeSavedPct = (row: LevelUpInfo): string => {
  const saved = timeSaved(row)
  if (row.noFert.totalTimeSec <= 0) return '0%'
  return ((saved / row.noFert.totalTimeSec) * 100).toFixed(1) + '%'
}

const tableRowClassName = ({ row }: { row: LevelUpInfo; rowIndex: number }) => {
  // Highlight every 10 levels
  if (row.level % 10 === 0) return 'milestone-row'
  return ''
}
</script>

<template>
  <div class="level-up-view">
    <ElCard shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span class="title">升级时间计算器</span>
          <div class="header-controls">
            <div class="control-group">
              <span class="control-label">土地数量</span>
              <ElSlider
                v-model="numLands"
                :min="1"
                :max="24"
                :step="1"
                :show-tooltip="true"
                class="land-slider"
              />
              <span class="land-value">{{ numLands }} 块</span>
            </div>
            <div class="control-group">
              <span class="control-label">土地等阶</span>
              <ElSlider
                v-model="landGrade"
                :min="1"
                :max="4"
                :step="1"
                :show-tooltip="true"
                :format-tooltip="(val: number) => `Lv${val}`"
                class="land-slider"
              />
              <span class="land-value">
                Lv{{ landGrade }}
                <template v-if="landGradeBuff.expBonusPct > 0 || landGradeBuff.timeReductionPct > 0">
                  (经验+{{ landGradeBuff.expBonusPct }}%、时间-{{ landGradeBuff.timeReductionPct }}%)
                </template>
              </span>
            </div>
            <div class="control-group fert-toggle">
              <span class="control-label">对比施肥效果</span>
              <ElSwitch
                v-model="showFertComparison"
                active-text="开"
                inactive-text="关"
                active-color="var(--primary)"
              />
            </div>
          </div>
        </div>
      </template>

      <div class="table-tip">
        基于 {{ numLands }} 块地计算，作物择优选择（最短时间升级而非最高经验效率），忽略操作时间。
        施肥采用最优阶段施肥策略（跳过最长生长阶段)。
      </div>

      <div class="level-range-filter">
        <span class="range-label">等级范围</span>
        <ElSlider
          v-model="levelRange"
          :min="1"
          :max="MAX_LEVEL - 1"
          range
          :show-tooltip="true"
          class="range-slider"
        />
        <span class="range-value">Lv.{{ levelRange[0] }} — Lv.{{ levelRange[1] }}</span>
      </div>

      <ElTable
        :data="levelUpData"
        stripe
        :row-class-name="tableRowClassName"
        style="width: 100%"
        max-height="calc(100vh - 340px)"
        class="level-table"
      >
        <!-- Level -->
        <ElTableColumn prop="level" label="等级" width="75" fixed align="center">
          <template #default="{ row }">
            <span class="level-value" :class="{ 'level-milestone': row.level % 10 === 0 }">
              {{ row.level }}
            </span>
          </template>
        </ElTableColumn>

        <!-- EXP needed -->
        <ElTableColumn prop="expToNext" label="升级经验" min-width="110" align="right">
          <template #default="{ row }">
            <span class="value-exp">{{ formatExp(row.expToNext) }}</span>
          </template>
        </ElTableColumn>

        <!-- ═══ No Fertilizer Section ═══ -->
        <ElTableColumn label="不施肥" align="center" class-name="section-no-fert">
          <ElTableColumn prop="noFert.cropName" label="最优作物" min-width="120">
            <template #default="{ row }">
              <span class="crop-name">{{ row.noFert.cropName }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="noFert.cycles" label="轮次" width="70" align="center">
            <template #default="{ row }">
              <span class="value-normal">{{ row.noFert.cycles }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn label="用时" min-width="110" align="center" sortable
            :sort-method="(a: LevelUpInfo, b: LevelUpInfo) => a.noFert.totalTimeSec - b.noFert.totalTimeSec">
            <template #default="{ row }">
              <ElTag :type="getTimeType(row.noFert.totalTimeSec)" size="small" class="time-tag">
                {{ formatTimeShort(row.noFert.totalTimeSec) }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn label="累计" min-width="110" align="center">
            <template #default="{ $index }">
              <span class="value-cumulative">{{ formatTimeShort(cumulativeNoFert[$index]) }}</span>
            </template>
          </ElTableColumn>
        </ElTableColumn>

        <!-- ═══ With Fertilizer Section (conditional) ═══ -->
        <ElTableColumn v-if="showFertComparison" label="施肥" align="center" class-name="section-fert">
          <ElTableColumn prop="withFert.cropName" label="最优作物" min-width="120">
            <template #default="{ row }">
              <span class="crop-name crop-name-fert">{{ row.withFert.cropName }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="withFert.cycles" label="轮次" width="70" align="center">
            <template #default="{ row }">
              <span class="value-normal">{{ row.withFert.cycles }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn label="用时" min-width="110" align="center" sortable
            :sort-method="(a: LevelUpInfo, b: LevelUpInfo) => a.withFert.totalTimeSec - b.withFert.totalTimeSec">
            <template #default="{ row }">
              <ElTag :type="getTimeType(row.withFert.totalTimeSec)" size="small" class="time-tag">
                {{ formatTimeShort(row.withFert.totalTimeSec) }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn label="累计" min-width="110" align="center">
            <template #default="{ $index }">
              <span class="value-cumulative">{{ formatTimeShort(cumulativeWithFert[$index]) }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn label="节省" min-width="110" align="center">
            <template #default="{ row }">
              <div class="saved-cell">
                <span class="value-saved">{{ formatTimeShort(timeSaved(row)) }}</span>
                <span class="value-saved-pct">{{ timeSavedPct(row) }}</span>
              </div>
            </template>
          </ElTableColumn>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<style scoped>
.level-up-view {
  padding: 0;
}

 /* ── Card ── */
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

/* ── Header ── */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
}

.title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-heading);
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 24px;
  flex-wrap: wrap;
}

.control-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

.control-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  white-space: nowrap;
}

.land-slider {
  width: 120px;
}

.land-slider :deep(.el-slider__runway) {
  height: 4px;
  background-color: var(--border);
}

.land-slider :deep(.el-slider__bar) {
  background-color: var(--primary);
  height: 4px;
}

.land-slider :deep(.el-slider__button) {
  border-color: var(--primary);
  width: 14px;
  height: 14px;
}

.land-value {
  font-size: 13px;
  font-weight: 600;
  color: var(--primary);
  min-width: 36px;
}

.fert-toggle {
  padding-left: 8px;
  border-left: 1px solid var(--border);
}

/* ── Tip ── */
.table-tip {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 -24px;
  margin-bottom: 0;
  padding: 12px 24px;
  background: var(--primary-bg);
  border-bottom: 1px solid rgba(59, 130, 246, 0.2);
}

/* ── Level range filter ── */
.level-range-filter {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
}

.range-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  white-space: nowrap;
}

.range-slider {
  flex: 1;
  max-width: 400px;
}

.range-slider :deep(.el-slider__runway) {
  height: 4px;
  background-color: var(--border);
}

.range-slider :deep(.el-slider__bar) {
  background-color: var(--primary);
  height: 4px;
}

.range-slider :deep(.el-slider__button) {
  border-color: var(--primary);
  width: 14px;
  height: 14px;
}

.range-value {
  font-size: 13px;
  font-weight: 600;
  color: var(--primary);
  white-space: nowrap;
}

/* ── Table ── */
.level-table {
  --el-table-border-color: var(--border);
  --el-table-bg-color: var(--bg-card);
  --el-table-tr-bg-color: var(--bg-card);
  --el-table-header-bg-color: var(--bg-elevated);
  --el-table-header-text-color: var(--text-secondary);
  --el-table-text-color: var(--text-primary);
  --el-table-row-hover-bg-color: var(--bg-elevated);
}

.level-table :deep(.el-table__header th) {
  font-weight: 600;
  font-size: 13px;
  padding: 12px 0;
  background: var(--bg-elevated);
}

.level-table :deep(.el-table__body td) {
  padding: 10px 8px;
}

.level-table :deep(.el-table__row--striped) {
  background: var(--bg-hover);
}

/* Milestone rows (every 10 levels) */
.level-table :deep(.milestone-row) {
  background-color: var(--gold-bg) !important;
}

.level-table :deep(.milestone-row:hover > td) {
  background-color: rgba(234, 179, 8, 0.15) !important;
}

/* Section header styling */
.level-table :deep(.section-no-fert) {
  border-left: 2px solid var(--primary);
  opacity: 0.7;
}

.level-table :deep(.section-fert) {
  border-left: 2px solid var(--gold);
  opacity: 0.7;
}

/* ── Values ── */
.level-value {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 36px;
  height: 26px;
  border-radius: var(--radius-sm);
  font-weight: 600;
  font-size: 13px;
  background: var(--bg-elevated);
  color: var(--text-secondary);
}

.level-milestone {
  background: linear-gradient(135deg, var(--gold) 0%, var(--gold-light) 100%);
  color: var(--gold);
  border: 1px solid rgba(234, 179, 8, 0.3);
}

.value-exp {
  color: var(--success);
  font-weight: 600;
  font-size: 13px;
}

.crop-name {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 13px;
}

.crop-name-fert {
  color: var(--gold);
}

.value-normal {
  color: var(--text-secondary);
  font-weight: 500;
}

.time-tag {
  border-radius: var(--radius-sm);
  font-weight: 600;
}

.time-tag:deep(.el-tag--success) {
  background: var(--success-bg);
  border-color: rgba(34, 197, 94, 0.3);
  color: var(--success);
}

.time-tag:deep(.el-tag--warning) {
  background: var(--gold-bg);
  border-color: rgba(234, 179, 8, 0.3);
  color: var(--gold);
}

.time-tag:deep(.el-tag--danger) {
  background: var(--danger-bg);
  border-color: rgba(239, 68, 68, 0.3);
  color: var(--danger);
}

.value-cumulative {
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 500;
}

.saved-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.value-saved {
  color: var(--success);
  font-weight: 600;
  font-size: 12px;
}

.value-saved-pct {
  color: var(--success);
  font-size: 11px;
  font-weight: 500;
}

/* ── Sort icons ── */
.level-table :deep(.caret-wrapper) {
  color: var(--text-muted);
}

.level-table :deep(.caret-wrapper:hover) {
  color: var(--primary);
}

.level-table :deep(.sort-caret.ascending) {
  border-bottom-color: var(--primary);
}

.level-table :deep(.sort-caret.descending) {
  border-top-color: var(--primary);
}

/* ── Responsive ── */
@media (max-width: 768px) {
  .table-card :deep(.el-card__header) {
    padding: 16px;
  }

  .table-card :deep(.el-card__body) {
    padding: 0 16px 16px;
  }

  .table-tip {
    margin: 0 -16px;
    padding: 10px 16px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .header-controls {
    width: 100%;
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .fert-toggle {
    padding-left: 0;
    border-left: none;
  }

  .level-range-filter {
    flex-wrap: wrap;
  }

  .range-slider {
    width: 100%;
    max-width: none;
  }
}
</style>