<template>
  <el-card>
    <div class="toolbar">
      <el-input v-model="code" placeholder="编码" style="width:160px" />
      <el-input v-model="name" placeholder="名称" style="width:160px" />
      <el-button type="success" v-permission="'drug:write'" @click="add">新增分类</el-button>
    </div>
    <el-table :data="list" v-loading="loading" border>
      <el-table-column prop="code" label="编码" width="120" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="sort_order" label="排序" width="80" />
    </el-table>
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listCategories, createCategory } from '@/api/drugs'
import type { Category } from '@/types/entities'

const list = ref<Category[]>([])
const loading = ref(false)
const code = ref('')
const name = ref('')

onMounted(load)
async function load() {
  loading.value = true
  try {
    list.value = (await listCategories()) ?? []
  } finally {
    loading.value = false
  }
}
async function add() {
  if (!code.value || !name.value) return ElMessage.warning('请填写编码与名称')
  await createCategory({ code: code.value, name: name.value })
  ElMessage.success('已新增')
  code.value = ''
  name.value = ''
  load()
}
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
</style>
