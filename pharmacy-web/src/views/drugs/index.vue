<template>
  <div>
    <el-card>
      <div class="toolbar">
        <el-input v-model="query.keyword" placeholder="药品名称/编码/拼音码" clearable style="width:220px" @keyup.enter="load" />
        <el-button type="primary" @click="load">查询</el-button>
        <el-button type="success" v-permission="'drug:write'" @click="openCreate">新增药品</el-button>
      </div>

      <el-table :data="list" v-loading="loading" border>
        <el-table-column prop="generic_name" label="通用名" min-width="140" />
        <el-table-column prop="specification" label="规格" width="100" />
        <el-table-column prop="manufacturer" label="厂家" min-width="120" />
        <el-table-column prop="dosage_form" label="剂型" width="90" />
        <el-table-column label="医保/集采" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.insurance_class" size="small">{{ row.insurance_class }}</el-tag>
            <el-tag v-if="row.vbp_batch" type="warning" size="small">集采{{ row.vbp_batch }}批</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="零售价" width="90">
          <template #default="{ row }"><MoneyText :amount="row.retail_price" /></template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" v-permission="'drug:write'" @click="openEdit(row)">编辑</el-button>
            <el-button link type="warning" v-permission="'drug:write'" @click="toggleStatus(row)">{{ row.status === 1 ? '停用' : '启用' }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        class="pager"
        layout="total, prev, pager, next"
        :total="total"
        :page-size="query.page_size"
        v-model:current-page="query.page"
        @current-change="load"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑药品' : '新增药品'" width="640px">
      <el-form :model="form" label-width="110px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="通用名" required><el-input v-model="form.generic_name" @blur="doMatch" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="编码"><el-input v-model="form.code" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="规格"><el-input v-model="form.specification" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="厂家"><el-input v-model="form.manufacturer" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="剂型"><el-input v-model="form.dosage_form" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="类型">
            <el-select v-model="form.item_type"><el-option label="药品" value="drug" /><el-option label="耗材" value="consumable" /></el-select>
          </el-form-item></el-col>
          <el-col :span="12"><el-form-item label="基本单位"><el-input v-model="form.base_unit" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="拆零单位"><el-input v-model="form.split_unit" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="包装含量"><el-input-number v-model="form.pack_size" :min="1" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="可拆零"><el-switch v-model="form.is_split_allowed" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="零售价(分)"><el-input-number v-model="form.retail_price" :min="0" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="进价(分)"><el-input-number v-model="form.purchase_price" :min="0" /></el-form-item></el-col>
        </el-row>
        <el-alert v-if="matchResult" type="info" :closable="false" style="margin-bottom:8px">
          目录匹配：<el-tag size="small">{{ matchResult.insurance_class || '无医保类别' }}</el-tag>
          <el-tag v-if="matchResult.vbp_batch" type="warning" size="small">集采{{ matchResult.vbp_batch }}批</el-tag>
        </el-alert>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listDrugs, createDrug, updateDrug, setDrugStatus } from '@/api/drugs'
import { matchDrug } from '@/api/reference'
import MoneyText from '@/components/MoneyText.vue'
import type { Drug } from '@/types/entities'
import type { ReferenceMatch } from '@/types/entities'

const list = ref<Drug[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ keyword: '', page: 1, page_size: 20 })
const dialogVisible = ref(false)
const form = reactive<Record<string, any>>({ item_type: 'drug', pack_size: 1 })
const matchResult = ref<ReferenceMatch | null>(null)

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await listDrugs(query)
    list.value = res?.list ?? []
    total.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.keys(form).forEach((k) => delete form[k])
  form.item_type = 'drug'
  form.pack_size = 1
  matchResult.value = null
  dialogVisible.value = true
}
function openEdit(row: Drug) {
  Object.assign(form, row)
  matchResult.value = null
  dialogVisible.value = true
}
async function doMatch() {
  if (!form.generic_name) return
  matchResult.value = await matchDrug(form.generic_name)
}
async function save() {
  const data = { ...form }
  if (form.id) await updateDrug(form.id, data)
  else await createDrug(data)
  ElMessage.success('已保存')
  dialogVisible.value = false
  load()
}
async function toggleStatus(row: Drug) {
  await setDrugStatus(row.id, row.status === 1 ? 0 : 1)
  ElMessage.success('已更新')
  load()
}
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.pager { margin-top: 12px; justify-content: flex-end; }
</style>
