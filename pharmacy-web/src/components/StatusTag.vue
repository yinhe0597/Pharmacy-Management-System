<template>
  <el-tag :type="tag" size="small">{{ label }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: string
  map: Record<string, { label: string; tag: string }>
}>()

const label = computed(() => props.map[props.status]?.label ?? props.status)
const tag = computed(() => (props.map[props.status]?.tag ?? 'info') as 'success' | 'info' | 'warning' | 'danger' | 'primary')
</script>
