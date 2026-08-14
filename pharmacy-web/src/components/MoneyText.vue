<template>
  <span :class="cls">{{ formatYuan(amount) }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatYuan } from '@/utils/money'

const props = defineProps<{ amount?: number | null; voided?: boolean }>()
const cls = computed(() => ({
  negative: (props.amount ?? 0) < 0,
  voided: props.voided,
}))
</script>

<style scoped>
.negative {
  color: #f56c6c;
}
.voided {
  color: #c0c4cc;
  text-decoration: line-through;
}
</style>
