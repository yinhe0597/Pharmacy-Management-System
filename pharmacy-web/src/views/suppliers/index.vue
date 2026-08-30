<template>
  <el-card>
    <div class="toolbar">
      <el-input
        v-model="query.keyword"
        placeholder="供应商名称/编码"
        clearable
        style="width: 220px"
        @keyup.enter="load"
      />
      <el-button type="primary" @click="load">查询</el-button>
      <el-button v-permission="'purchase:write'" type="success" @click="openCreate"
        >新增供应商</el-button
      >
    </div>
    <el-table v-loading="loading" :data="list" border>
      <el-table-column prop="code" label="编码" width="120" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="contact_person" label="联系人" width="120" />
      <el-table-column prop="phone" label="电话" width="140" />
      <el-table-column prop="address" label="地址" min-width="160" />
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination
      v-model:current-page="query.page"
      class="pager"
      layout="total, prev, pager, next"
      :total="total"
      :page-size="query.page_size"
      @current-change="load"
    />

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑供应商' : '新增供应商'" width="480px">
      <el-form label-width="80px">
        <el-form-item label="编码" required><el-input v-model="form.code" :disabled="!!form.id" /></el-form-item>
        <el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="联系人"><el-input v-model="form.contact_person" /></el-form-item>
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
import { ElMessage, ElMessageBox } from 'element-plus'
import { listSuppliers, createSupplier, updateSupplier, deleteSupplier } from '@/api/suppliers'
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
function openEdit(row: any) {
  Object.keys(form).forEach((k) => delete form[k])
  Object.assign(form, {
    id: row.id,
    code: row.code,
    name: row.name,
    contact_person: row.contact_person,
    phone: row.phone,
    address: row.address,
  })
  dialogVisible.value = true
}
async function save() {
  if (form.id) await updateSupplier(form.id, form)
  else await createSupplier(form)
  ElMessage.success('已保存')
  dialogVisible.value = false
  load()
}
async function remove(row: any) {
  await ElMessageBox.confirm(`确认删除供应商「${row.name}」？`, '提示', { type: 'warning' })
  await deleteSupplier(row.id)
  ElMessage.success('已删除')
  load()
}
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
.pager {
  margin-top: 12px;
  justify-content: flex-end;
}
</style>
