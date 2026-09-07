<template>
  <el-select
    :model-value="modelValue"
    filterable
    remote
    clearable
    :remote-method="search"
    :loading="loading"
    :placeholder="placeholder"
    @update:model-value="onSelect"
  >
    <el-option v-for="p in options" :key="p.id" :label="`${p.name}（${p.card_no}）`" :value="p.id">
      <span>{{ p.name }}</span>
      <span style="color: #909399; margin-left: 8px">{{ p.card_no }}</span>
      <span v-if="p.gender || p.age" style="color: #909399; margin-left: 8px">
        {{ p.gender }} {{ p.age }}
      </span>
    </el-option>
  </el-select>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { listPatients } from '@/api/patients'
import type { Patient } from '@/types/entities'

const emit = defineEmits<{
  (e: 'update:modelValue', v: number | null): void
  (e: 'select', p: Patient): void
}>()
withDefaults(defineProps<{ modelValue?: number | null; placeholder?: string }>(), {
  modelValue: null,
  placeholder: '搜索患者（姓名/卡号）',
})

const options = ref<Patient[]>([])
const loading = ref(false)

async function search(keyword: string) {
  if (!keyword) {
    options.value = []
    return
  }
  loading.value = true
  try {
    const res = await listPatients({ keyword, page: 1, page_size: 20 })
    options.value = res?.list ?? []
  } finally {
    loading.value = false
  }
}
function onSelect(v: number | null) {
  emit('update:modelValue', v)
  const found = options.value.find((p) => p.id === v)
  if (found) emit('select', found)
}
</script>
