<template>
  <el-select
    :model-value="modelValue"
    filterable
    remote
    clearable
    :remote-method="search"
    :loading="loading"
    :placeholder="placeholder"
    value-key="code"
    @update:model-value="onSelect"
  >
    <el-option
      v-for="d in options"
      :key="d.code"
      :label="`${d.code} ${d.disease_name}`"
      :value="d.code"
    >
      <span>{{ d.code }}</span>
      <span style="margin-left:8px">{{ d.disease_name }}</span>
    </el-option>
  </el-select>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { listDiagnosisCodes } from '@/api/reference'

const emit = defineEmits<{ (e: 'update:modelValue', v: string): void; (e: 'select', v: { code: string; disease_name: string }): void }>()
withDefaults(
  defineProps<{ modelValue?: string; placeholder?: string }>(),
  { modelValue: '', placeholder: '搜索诊断（ICD-10）' },
)

const options = ref<{ code: string; disease_name: string }[]>([])
const loading = ref(false)

async function search(keyword: string) {
  loading.value = true
  try {
    const res = await listDiagnosisCodes({ keyword, page: 1, page_size: 20 })
    options.value = res?.list ?? []
  } finally {
    loading.value = false
  }
}
search('')

function onSelect(code: string) {
  emit('update:modelValue', code)
  const d = options.value.find((x) => x.code === code)
  if (d) emit('select', d)
}
</script>
