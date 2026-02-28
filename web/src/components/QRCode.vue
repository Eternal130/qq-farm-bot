<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted } from 'vue'
import QRCodeStyling from 'qr-code-styling'
import type { DrawType, ErrorCorrectionLevel, DotType, CornerSquareType, CornerDotType, Options } from 'qr-code-styling'

// 预设样式
export type QRStylePreset = 'default' | 'rounded' | 'dots' | 'elegant' | 'minimal' | 'colorful'

interface Props {
  data: string
  width?: number
  height?: number
  preset?: QRStylePreset
}

const props = withDefaults(defineProps<Props>(), {
  width: 220,
  height: 220,
  preset: 'rounded'
})

const qrRef = ref<HTMLDivElement | null>(null)
let qrCode: QRCodeStyling | null = null

// 样式预设配置
const stylePresets: Record<QRStylePreset, Partial<Options>> = {
  default: {
    type: 'svg' as DrawType,
    dotsOptions: { type: 'square' as DotType, color: '#000000' },
    cornersSquareOptions: { type: 'square' as CornerSquareType, color: '#000000' },
    cornersDotOptions: { type: 'square' as CornerDotType, color: '#000000' },
    backgroundOptions: { color: '#ffffff' }
  },
  rounded: {
    type: 'svg' as DrawType,
    dotsOptions: { type: 'rounded' as DotType, color: '#15803D' },
    cornersSquareOptions: { type: 'extra-rounded' as CornerSquareType, color: '#14532D' },
    cornersDotOptions: { type: 'dot' as CornerDotType, color: '#166534' },
    backgroundOptions: { color: '#ffffff' }
  },
  dots: {
    type: 'svg' as DrawType,
    dotsOptions: { type: 'dots' as DotType, color: '#1e40af' },
    cornersSquareOptions: { type: 'dot' as CornerSquareType, color: '#1e3a8a' },
    cornersDotOptions: { type: 'dot' as CornerDotType, color: '#1d4ed8' },
    backgroundOptions: { color: '#f0f9ff' }
  },
  elegant: {
    type: 'svg' as DrawType,
    dotsOptions: { type: 'classy' as DotType, color: '#7c3aed' },
    cornersSquareOptions: { type: 'classy' as CornerSquareType, color: '#6d28d9' },
    cornersDotOptions: { type: 'classy' as CornerDotType, color: '#8b5cf6' },
    backgroundOptions: { color: '#faf5ff' }
  },
  minimal: {
    type: 'svg' as DrawType,
    dotsOptions: { type: 'classy-rounded' as DotType, color: '#374151' },
    cornersSquareOptions: { type: 'classy' as CornerSquareType, color: '#1f2937' },
    cornersDotOptions: { type: 'classy-rounded' as CornerDotType, color: '#4b5563' },
    backgroundOptions: { color: '#ffffff' }
  },
  colorful: {
    type: 'svg' as DrawType,
    dotsOptions: { 
      type: 'rounded' as DotType, 
      gradient: {
        type: 'linear',
        rotation: 45,
        colorStops: [
          { offset: 0, color: '#f97316' },
          { offset: 0.5, color: '#ec4899' },
          { offset: 1, color: '#8b5cf6' }
        ]
      }
    },
    cornersSquareOptions: { 
      type: 'extra-rounded' as CornerSquareType, 
      gradient: {
        type: 'linear',
        rotation: 135,
        colorStops: [
          { offset: 0, color: '#06b6d4' },
          { offset: 1, color: '#3b82f6' }
        ]
      }
    },
    cornersDotOptions: { 
      type: 'dot' as CornerDotType, 
      color: '#10b981' 
    },
    backgroundOptions: { color: '#ffffff' }
  }
}

function createQRCode() {
  if (!qrRef.value || !props.data) return

  const presetOptions = stylePresets[props.preset] || stylePresets.rounded

  const options: Options = {
    width: props.width,
    height: props.height,
    data: props.data,
    margin: 10,
    qrOptions: {
      errorCorrectionLevel: 'M' as ErrorCorrectionLevel
    },
    ...presetOptions
  }

  if (qrCode) {
    qrCode.update(options)
  } else {
    qrCode = new QRCodeStyling(options)
    qrCode.append(qrRef.value)
  }
}

// 监听数据变化更新二维码
watch(() => [props.data, props.preset, props.width, props.height], () => {
  createQRCode()
}, { deep: true })

onMounted(() => {
  createQRCode()
})

onUnmounted(() => {
  qrCode = null
})

// 暴露更新方法
defineExpose({
  update: (data: string) => {
    if (qrCode && data) {
      qrCode.update({ data })
    }
  },
  download: (filename = 'qrcode.png') => {
    if (qrCode) {
      qrCode.download({ name: filename, extension: 'png' })
    }
  }
})
</script>

<template>
  <div ref="qrRef" class="qr-code-container"></div>
</template>

<style scoped>
.qr-code-container {
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-code-container :deep(svg),
.qr-code-container :deep(canvas) {
  display: block;
}
</style>
