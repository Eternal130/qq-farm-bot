const fs = require('fs');
const path = require('path');

const PLANT_PATH = path.join(__dirname, '..', 'gameConfig', 'Plant.json');
const ITEM_PATH = path.join(__dirname, '..', 'gameConfig', 'ItemInfo.json');
const OUTPUT_PATH = path.join(__dirname, '..', 'crop-yield-list.md');

const NO_FERT_PLANT_SPEED = 9;
const NORMAL_FERT_PLANT_SPEED = 6;

function toNum(v, fallback = 0) {
    const n = Number(v);
    return Number.isFinite(n) ? n : fallback;
}

function parseArgs(argv) {
    const opts = { lands: 18, sortBy: 'expFert' };
    for (let i = 0; i < argv.length; i++) {
        if (argv[i] === '--lands' && argv[i + 1]) {
            opts.lands = Math.max(1, toNum(argv[++i], 18));
        } else if (argv[i] === '--sort' && argv[i + 1]) {
            opts.sortBy = argv[++i];
        }
    }
    return opts;
}

function parseGrowPhases(phases) {
    if (!phases || typeof phases !== 'string') return [];
    return phases
        .split(';')
        .map(x => x.trim())
        .filter(Boolean)
        .map(seg => {
            const parts = seg.split(':');
            return parts.length >= 2 ? toNum(parts[1], 0) : 0;
        })
        .filter(sec => sec > 0);
}

function loadItemPrices() {
    const items = JSON.parse(fs.readFileSync(ITEM_PATH, 'utf8'));
    const priceMap = new Map();
    for (const item of items) {
        const id = toNum(item.id);
        if (id > 0) {
            priceMap.set(id, toNum(item.price, 0));
        }
    }
    return priceMap;
}

function loadPlants() {
    return JSON.parse(fs.readFileSync(PLANT_PATH, 'utf8'));
}

function calcYield(plant, itemPrices, lands, plantSecondsNoFert, plantSecondsFert) {
    const seedId = toNum(plant.seed_id, 0);
    const plantId = toNum(plant.id, 0);
    const name = plant.name || `未知_${plantId}`;
    const exp = toNum(plant.exp, 0);
    const phases = parseGrowPhases(plant.grow_phases);
    
    const growTime = phases.reduce((sum, s) => sum + s, 0);
    if (growTime <= 0) return null;
    
    const firstPhase = phases.length > 0 ? phases[0] : 0;
    const growTimeFert = Math.max(1, growTime - firstPhase);
    
    const cycleTimeNoFert = growTime + plantSecondsNoFert;
    const cycleTimeFert = growTimeFert + plantSecondsFert;
    
    const fruitId = plant.fruit ? toNum(plant.fruit.id, 0) : 0;
    const fruitCount = plant.fruit ? toNum(plant.fruit.count, 0) : 0;
    const fruitPrice = itemPrices.get(fruitId) || 0;
    
    const goldPerCycle = fruitCount * fruitPrice;
    
    const expPerMinNoFert = (exp / cycleTimeNoFert) * 60;
    const expPerMinFert = (exp / cycleTimeFert) * 60;
    const goldPerMinNoFert = (goldPerCycle / cycleTimeNoFert) * 60;
    const goldPerMinFert = (goldPerCycle / cycleTimeFert) * 60;
    
    return {
        seedId,
        plantId,
        name,
        growTime,
        growTimeFert,
        cycleTimeNoFert,
        cycleTimeFert,
        exp,
        fruitId,
        fruitCount,
        fruitPrice,
        goldPerCycle,
        expPerMinNoFert,
        expPerMinFert,
        goldPerMinNoFert,
        goldPerMinFert,
    };
}

function formatTime(seconds) {
    const s = Math.max(0, Math.round(seconds));
    if (s < 60) return `${s}秒`;
    const m = Math.floor(s / 60);
    const r = s % 60;
    if (m < 60) return r > 0 ? `${m}分${r}秒` : `${m}分`;
    const h = Math.floor(m / 60);
    const mm = m % 60;
    return r > 0 ? `${h}时${mm}分${r}秒` : `${h}时${mm}分`;
}

function main() {
    const opts = parseArgs(process.argv.slice(2));
    const lands = opts.lands;
    
    const plantSecondsNoFert = lands / NO_FERT_PLANT_SPEED;
    const plantSecondsFert = lands / NORMAL_FERT_PLANT_SPEED;
    
    console.log(`[配置] 地块数: ${lands}`);
    console.log(`[配置] 种植速度(不施肥): ${NO_FERT_PLANT_SPEED} 块/秒，整场耗时: ${plantSecondsNoFert.toFixed(2)} 秒`);
    console.log(`[配置] 种植速度(施肥): ${NORMAL_FERT_PLANT_SPEED} 块/秒，整场耗时: ${plantSecondsFert.toFixed(2)} 秒`);
    
    const itemPrices = loadItemPrices();
    console.log(`[加载] 物品价格表: ${itemPrices.size} 条`);
    
    const plants = loadPlants();
    console.log(`[加载] 作物配置: ${plants.length} 条`);
    
    const seenSeeds = new Set();
    const results = [];
    
    for (const plant of plants) {
        const seedId = toNum(plant.seed_id, 0);
        if (seedId <= 0 || seedId >= 29000 || seenSeeds.has(seedId)) continue;
        seenSeeds.add(seedId);
        
        const row = calcYield(plant, itemPrices, lands, plantSecondsNoFert, plantSecondsFert);
        if (row) results.push(row);
    }
    
    console.log(`[计算] 有效作物: ${results.length} 种`);
    
    const sortKey = opts.sortBy === 'goldFert' ? 'goldPerMinFert' : 
                    opts.sortBy === 'expNoFert' ? 'expPerMinNoFert' : 
                    opts.sortBy === 'goldNoFert' ? 'goldPerMinNoFert' : 'expPerMinFert';
    results.sort((a, b) => b[sortKey] - a[sortKey]);
    
    const lines = [];
    lines.push('# 作物收益列表');
    lines.push('');
    lines.push(`导出时间: ${new Date().toISOString()}`);
    lines.push('');
    lines.push('## 配置参数');
    lines.push('');
    lines.push(`- 地块数: ${lands}`);
    lines.push(`- 种植速度(不施肥): ${NO_FERT_PLANT_SPEED} 块/秒`);
    lines.push(`- 种植速度(施肥): ${NORMAL_FERT_PLANT_SPEED} 块/秒`);
    lines.push(`- 施肥规则: 减少首阶段生长时间`);
    lines.push('');
    lines.push('## 计算公式');
    lines.push('');
    lines.push('```');
    lines.push('经验收益 = 收获经验 / 轮次时间 * 60');
    lines.push('金币收益 = (果实数量 * 单价) / 轮次时间 * 60');
    lines.push('轮次时间 = 生长时间 + 种植时间');
    lines.push('```');
    lines.push('');
    lines.push('## 收益列表');
    lines.push('');
    lines.push('| 排名 | 作物ID | 种子ID | 名称 | 生长时间 | 施肥后生长 | 收获经验 | 果实数量 | 果实单价 | 经验/分钟(不施肥) | 经验/分钟(施肥) | 金币/分钟(不施肥) | 金币/分钟(施肥) |');
    lines.push('|------|--------|--------|------|----------|------------|----------|----------|----------|-------------------|-----------------|-------------------|-----------------|');
    
    for (let i = 0; i < results.length; i++) {
        const r = results[i];
        lines.push(`| ${i + 1} | ${r.plantId} | ${r.seedId} | ${r.name} | ${formatTime(r.growTime)} | ${formatTime(r.growTimeFert)} | ${r.exp} | ${r.fruitCount} | ${r.fruitPrice} | ${r.expPerMinNoFert.toFixed(2)} | ${r.expPerMinFert.toFixed(2)} | ${r.goldPerMinNoFert.toFixed(2)} | ${r.goldPerMinFert.toFixed(2)} |`);
    }
    
    lines.push('');
    lines.push(`共 ${results.length} 种作物`);
    
    fs.writeFileSync(OUTPUT_PATH, lines.join('\n'), 'utf8');
    console.log(`[输出] ${OUTPUT_PATH}`);
    
    console.log('');
    console.log('Top 10 (按施肥后每分钟经验收益):');
    console.log('排名 | 名称           | 经验/分钟(施肥) | 金币/分钟(施肥)');
    for (let i = 0; i < Math.min(10, results.length); i++) {
        const r = results[i];
        console.log(
            `${String(i + 1).padStart(2)} | ${r.name.padEnd(14)} | ${r.expPerMinFert.toFixed(2).padStart(12)} | ${r.goldPerMinFert.toFixed(2).padStart(12)}`
        );
    }
}

main();
