<template>
  <div ref="el" :style="{ width: '100%', height }" />
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { init, type EChartsOption, type EChartsType } from '@/utils/echarts'

const props = withDefaults(defineProps<{ option: EChartsOption; height?: string }>(), {
  height: '300px',
})

const el = ref<HTMLElement>()
let chart: EChartsType | null = null

function render() {
  if (!el.value) return
  if (!chart) chart = init(el.value)
  chart.setOption(props.option, true)
}

onMounted(render)
watch(() => props.option, render, { deep: true })
onBeforeUnmount(() => {
  chart?.dispose()
  chart = null
})
</script>
