<template>
  <el-select
    :model-value="modelValue"
    filterable
    remote
    clearable
    :remote-method="search"
    :loading="loading"
    :placeholder="placeholder"
    value-key="id"
    @update:model-value="onSelect"
  >
    <el-option
      v-for="d in options"
      :key="d.id"
      :label="`${d.generic_name} ${d.specification ?? ''} ${d.manufacturer ?? ''}`"
      :value="d.id"
    >
      <span>{{ d.generic_name }}</span>
      <span style="color:#909399;margin-left:8px">{{ d.specification }} {{ d.manufacturer }}</span>
      <el-tag v-if="d.insurance_class" size="small" style="margin-left:8px">{{ d.insurance_class }}</el-tag>
      <el-tag v-if="d.vbp_batch" type="warning" size="small" style="margin-left:4px">集采{{ d.vbp_batch }}批</el-tag>
    </el-option>
  </el-select>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { listDrugs } from '@/api/drugs'
import type { Drug } from '@/types/entities'

const emit = defineEmits<{ (e: 'update:modelValue', v: number | null): void; (e: 'select', d: Drug): void }>()
withDefaults(
  defineProps<{ modelValue?: number | null; placeholder?: string }>(),
  { modelValue: null, placeholder: '搜索药品' },
)

const options = ref<Drug[]>([])
const loading = ref(false)

async function search(keyword: string) {
  loading.value = true
  try {
    const res = await listDrugs({ keyword, page: 1, page_size: 20 })
    options.value = res?.list ?? []
  } finally {
    loading.value = false
  }
}
search('')

function onSelect(id: number | null) {
  emit('update:modelValue', id)
  const drug = options.value.find((d) => d.id === id)
  if (drug) emit('select', drug)
}
</script>
