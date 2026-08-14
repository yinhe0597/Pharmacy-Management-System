<template>
  <el-card>
    <div class="toolbar">
      <el-input v-model="query.keyword" placeholder="供应商名称/编码" clearable style="width:220px" @keyup.enter="load" />
      <el-button type="primary" @click="load">查询</el-button>
      <el-button type="success" v-permission="'purchase:write'" @click="openCreate">新增供应商</el-button>
    </div>
    <el-table :data="list" v-loading="loading" border>
      <el-table-column prop="code" label="编码" width="120" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="contact" label="联系人" width="120" />
      <el-table-column prop="phone" label="电话" width="140" />
      <el-table-column prop="address" label="地址" min-width="160" />
    </el-table>
    <el-pagination class="pager" layout="total, prev, pager, next" :total="total" :page-size="query.page_size" v-model:current-page="query.page" @current-change="load" />

    <el-dialog v-model="dialogVisible" title="新增供应商" width="480px">
      <el-form label-width="80px">
        <el-form-item label="编码" required><el-input v-model="form.code" /></el-form-item>
        <el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="联系人"><el-input v-model="form.contact" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="form.phone" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="form.address" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listSuppliers, createSupplier } from '@/api/suppliers'
import type { Supplier } from '@/types/entities'

const list = ref<Supplier[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ keyword: '', page: 1, page_size: 20 })
const dialogVisible = ref(false)
const form = reactive<Record<string, any>>({})

onMounted(load)
async function load() {
  loading.value = true
  try {
    const res = await listSuppliers(query)
    list.value = res?.list ?? []
    total.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
function openCreate() {
  Object.keys(form).forEach((k) => delete form[k])
  dialogVisible.value = true
}
async function save() {
  await createSupplier(form)
  ElMessage.success('已新增')
  dialogVisible.value = false
  load()
}
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.pager { margin-top: 12px; justify-content: flex-end; }
</style>
