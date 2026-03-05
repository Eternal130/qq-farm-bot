import { cropYieldData } from './cropYield'

/**
 * Parse formatted time string to seconds.
 * Handles: "4时0分", "1小时20分", "30秒", "1分", "12时0分", "1时52分30秒"
 */
export function parseTimeToSec(timeStr: string): number {
  let total = 0
  const hourMatch = timeStr.match(/(\d+)(?:小)?时/)
  if (hourMatch) total += parseInt(hourMatch[1]) * 3600
  const minMatch = timeStr.match(/(\d+)分/)
  if (minMatch) total += parseInt(minMatch[1]) * 60
  const secMatch = timeStr.match(/(\d+)秒/)
  if (secMatch) total += parseInt(secMatch[1])
  return total
}

/**
 * Format seconds to compact human-readable string.
 */
export function formatTime(seconds: number): string {
  if (seconds <= 0) return '0秒'
  if (!isFinite(seconds)) return '-'

  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60

  const parts: string[] = []
  if (days > 0) parts.push(`${days}天`)
  if (hours > 0) parts.push(`${hours}时`)
  if (mins > 0) parts.push(`${mins}分`)
  if (parts.length === 0 && secs > 0) parts.push(`${secs}秒`)
  return parts.join('')
}

/**
 * Format seconds to short display for table cells.
 */
export function formatTimeShort(seconds: number): string {
  if (seconds <= 0) return '0秒'
  if (!isFinite(seconds)) return '-'

  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const mins = Math.floor((seconds % 3600) / 60)

  if (days > 0) {
    if (hours > 0) return `${days}天${hours}时`
    return `${days}天`
  }
  if (hours > 0) {
    if (mins > 0) return `${hours}时${mins}分`
    return `${hours}时`
  }
  if (mins > 0) return `${mins}分`
  return `${seconds}秒`
}

// Level EXP thresholds: LEVEL_EXP[level] = cumulative XP required to reach that level.
// Index 0 is unused. Level 1 = 0 exp, Level 2 = 100 exp, ..., Level 200 = 167034400 exp.
export const LEVEL_EXP: number[] = [
  0,          // index 0 (unused)
  0,          // level 1
  100,        // level 2
  300,        // level 3
  700,        // level 4
  1300,       // level 5
  2300,       // level 6
  4000,       // level 7
  6600,       // level 8
  10100,      // level 9
  14300,      // level 10
  19300,      // level 11
  25100,      // level 12
  31800,      // level 13
  39500,      // level 14
  48300,      // level 15
  58300,      // level 16
  69500,      // level 17
  82000,      // level 18
  95900,      // level 19
  111300,     // level 20
  128300,     // level 21
  146900,     // level 22
  167200,     // level 23
  189300,     // level 24
  213300,     // level 25
  239300,     // level 26
  267300,     // level 27
  297400,     // level 28
  329700,     // level 29
  364300,     // level 30
  401300,     // level 31
  440700,     // level 32
  482600,     // level 33
  527100,     // level 34
  574300,     // level 35
  624300,     // level 36
  677100,     // level 37
  732800,     // level 38
  791500,     // level 39
  853300,     // level 40
  918300,     // level 41
  986500,     // level 42
  1058000,    // level 43
  1132900,    // level 44
  1211300,    // level 45
  1293300,    // level 46
  1378900,    // level 47
  1468200,    // level 48
  1561300,    // level 49
  1658300,    // level 50
  1759300,    // level 51
  1864300,    // level 52
  1973400,    // level 53
  2086700,    // level 54
  2204300,    // level 55
  2326300,    // level 56
  2452700,    // level 57
  2583600,    // level 58
  2719100,    // level 59
  2859300,    // level 60
  3004300,    // level 61
  3154100,    // level 62
  3308800,    // level 63
  3468500,    // level 64
  3633300,    // level 65
  3803300,    // level 66
  3978500,    // level 67
  4159000,    // level 68
  4344900,    // level 69
  4536300,    // level 70
  4733300,    // level 71
  4935900,    // level 72
  5144200,    // level 73
  5358300,    // level 74
  5578300,    // level 75
  5804300,    // level 76
  6036300,    // level 77
  6274400,    // level 78
  6518700,    // level 79
  6769300,    // level 80
  7026300,    // level 81
  7289700,    // level 82
  7559600,    // level 83
  7836100,    // level 84
  8119300,    // level 85
  8409300,    // level 86
  8706100,    // level 87
  9009800,    // level 88
  9320500,    // level 89
  9638300,    // level 90
  9963300,    // level 91
  10295500,   // level 92
  10635000,   // level 93
  10981900,   // level 94
  11336300,   // level 95
  11698300,   // level 96
  12067900,   // level 97
  12445200,   // level 98
  12830300,   // level 99
  13223300,   // level 100
  13624300,   // level 101
  14185200,   // level 102
  14760100,   // level 103
  15349200,   // level 104
  15952700,   // level 105
  16570700,   // level 106
  17203500,   // level 107
  17851200,   // level 108
  18513900,   // level 109
  19191900,   // level 110
  19885400,   // level 111
  20594500,   // level 112
  21319400,   // level 113
  22060300,   // level 114
  22817400,   // level 115
  23590900,   // level 116
  24381000,   // level 117
  25187800,   // level 118
  26011600,   // level 119
  26852500,   // level 120
  27710700,   // level 121
  28586400,   // level 122
  29479800,   // level 123
  30391100,   // level 124
  31320500,   // level 125
  32268100,   // level 126
  33234200,   // level 127
  34218900,   // level 128
  35222400,   // level 129
  36245000,   // level 130
  37286800,   // level 131
  38348000,   // level 132
  39428800,   // level 133
  40529400,   // level 134
  41650000,   // level 135
  42790800,   // level 136
  43952000,   // level 137
  45133800,   // level 138
  46336400,   // level 139
  47559900,   // level 140
  48804600,   // level 141
  50070700,   // level 142
  51358300,   // level 143
  52667700,   // level 144
  53999100,   // level 145
  55352600,   // level 146
  56728500,   // level 147
  58127000,   // level 148
  59548200,   // level 149
  60992400,   // level 150
  62459800,   // level 151
  63950500,   // level 152
  65464800,   // level 153
  67002900,   // level 154
  68564900,   // level 155
  70151100,   // level 156
  71761700,   // level 157
  73396900,   // level 158
  75056900,   // level 159
  76741900,   // level 160
  78452100,   // level 161
  80187700,   // level 162
  81948900,   // level 163
  83735900,   // level 164
  85548900,   // level 165
  87388200,   // level 166
  89253900,   // level 167
  91146300,   // level 168
  93065500,   // level 169
  95011800,   // level 170
  96985300,   // level 171
  98986300,   // level 172
  101015000,  // level 173
  103071600,  // level 174
  105156300,  // level 175
  107269400,  // level 176
  109411000,  // level 177
  111581300,  // level 178
  113780600,  // level 179
  116009100,  // level 180
  118267000,  // level 181
  120554500,  // level 182
  122871800,  // level 183
  125219100,  // level 184
  127596600,  // level 185
  130004600,  // level 186
  132443200,  // level 187
  134912700,  // level 188
  137413300,  // level 189
  139945200,  // level 190
  142508700,  // level 191
  145103900,  // level 192
  147731100,  // level 193
  150390400,  // level 194
  153082100,  // level 195
  155806500,  // level 196
  158563700,  // level 197
  161353900,  // level 198
  164177400,  // level 199
  167034400,  // level 200
]

export const MAX_LEVEL = LEVEL_EXP.length - 1 // 200

// Default farmable land count
export const DEFAULT_LANDS = 24

// ---------------------------------------------------------------------------
// Land grade buff data (lv0 = locked, lv1-lv4)
// Server uses basis-point style (10000 = 100%), we store as plain percentages.
// ---------------------------------------------------------------------------

export interface LandGradeBuff {
  level: number
  label: string
  expBonusPct: number      // e.g. 20 means +20%
  timeReductionPct: number // e.g. 20 means -20% grow time
  yieldBonusPct: number    // e.g. 300 means +300% yield
}

export const LAND_GRADE_BUFFS: LandGradeBuff[] = [
  { level: 1, label: 'Lv1', expBonusPct: 0,  timeReductionPct: 0,  yieldBonusPct: 0 },
  { level: 2, label: 'Lv2', expBonusPct: 0,  timeReductionPct: 0,  yieldBonusPct: 0 },
  { level: 3, label: 'Lv3', expBonusPct: 0,  timeReductionPct: 10, yieldBonusPct: 200 },
  { level: 4, label: 'Lv4', expBonusPct: 20, timeReductionPct: 20, yieldBonusPct: 300 },
]

export const DEFAULT_LAND_GRADE = 1

export function getLandGradeBuff(level: number): LandGradeBuff {
  return LAND_GRADE_BUFFS.find(g => g.level === level) ?? LAND_GRADE_BUFFS[0]
}

// ---------------------------------------------------------------------------
// Crop calculation data — derived from pre-computed cropYield data
// ---------------------------------------------------------------------------

interface CropCalcInfo {
  name: string
  requiredLevel: number
  /** Total EXP per plant for the complete cycle (all seasons) */
  expPerCycle: number
  seasons: number
  /** Total growth seconds for all seasons, no fertilizer */
  growTimeNoFertSec: number
  /** Total growth seconds for all seasons, with optimal fertilizer */
  growTimeFertSec: number
}

// Build crop calc data from the existing cropYield dataset.
// cropYieldData.harvestExp = base exp per plant per complete cycle (all seasons).
// cropYieldData.growTime / growTimeFert = formatted total time across all seasons.
const cropCalcData: CropCalcInfo[] = cropYieldData
  .map(c => ({
    name: c.name,
    requiredLevel: c.requiredLevel,
    expPerCycle: c.harvestExp,
    seasons: c.seasons,
    growTimeNoFertSec: parseTimeToSec(c.growTime),
    growTimeFertSec: parseTimeToSec(c.growTimeFert),
  }))
  .filter(c => c.growTimeNoFertSec > 0 && c.growTimeFertSec > 0 && c.expPerCycle > 0)

// ---------------------------------------------------------------------------
// Core calculation: find the crop that minimizes total level-up time
// ---------------------------------------------------------------------------

export interface OptimalResult {
  cropName: string
  cycles: number
  totalTimeSec: number
}

export interface LevelUpInfo {
  level: number
  /** EXP needed to reach next level */
  expToNext: number
  /** Optimal result without fertilizer */
  noFert: OptimalResult
  /** Optimal result with fertilizer */
  withFert: OptimalResult
}

/**
 * Find the crop that reaches the required EXP in the shortest wall-clock time.
 *
 * Key insight: A crop with lower exp-per-minute but shorter cycle time can be
 * faster to level up than a high-exp crop with a long cycle, because you only
 * need enough *cycles × exp* to cover the gap, and short crops waste less
 * time on the final partial-need cycle.
 */
function findOptimalCrop(
  expNeeded: number,
  level: number,
  numLands: number,
  useFert: boolean,
  landGrade: number,
): OptimalResult {
  let bestTime = Infinity
  let bestCrop = ''
  let bestCycles = 0

  for (const crop of cropCalcData) {
    if (crop.requiredLevel > level) continue

    const gradeBuff = getLandGradeBuff(landGrade)
    // Apply land grade exp bonus
    const adjustedExp = crop.expPerCycle * (1 + gradeBuff.expBonusPct / 100)
    const expPerFullCycle = adjustedExp * numLands
    if (expPerFullCycle <= 0) continue

    // Apply land grade time reduction
    const baseTimeSec = useFert ? crop.growTimeFertSec : crop.growTimeNoFertSec
    const cycleTimeSec = Math.max(1, Math.round(baseTimeSec * (1 - gradeBuff.timeReductionPct / 100)))
    if (cycleTimeSec <= 0) continue

    const cycles = Math.ceil(expNeeded / expPerFullCycle)
    const totalTime = cycles * cycleTimeSec

    // Prefer shorter time; on tie prefer fewer cycles (less busy-work)
    if (totalTime < bestTime || (totalTime === bestTime && cycles < bestCycles)) {
      bestTime = totalTime
      bestCrop = crop.name
      bestCycles = cycles
    }
  }

  return { cropName: bestCrop, cycles: bestCycles, totalTimeSec: bestTime }
}

/**
 * Calculate the optimal level-up plan for every level from `startLevel` to `endLevel`.
 */
export function calculateLevelUps(
  numLands: number = DEFAULT_LANDS,
  startLevel: number = 1,
  endLevel: number = MAX_LEVEL - 1,
  landGrade: number = DEFAULT_LAND_GRADE,
): LevelUpInfo[] {
  const results: LevelUpInfo[] = []

  const lo = Math.max(1, startLevel)
  const hi = Math.min(MAX_LEVEL - 1, endLevel)

  for (let level = lo; level <= hi; level++) {
    const expToNext = LEVEL_EXP[level + 1] - LEVEL_EXP[level]
    if (expToNext <= 0) continue

    const noFert = findOptimalCrop(expToNext, level, numLands, false, landGrade)
    const withFert = findOptimalCrop(expToNext, level, numLands, true, landGrade)

    results.push({ level, expToNext, noFert, withFert })
  }

  return results
}
