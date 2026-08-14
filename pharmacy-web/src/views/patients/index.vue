<template>
  <div>
    <el-card>
      <div class="toolbar">
        <el-input v-model="query.keyword" placeholder="姓名/卡号/电话" clearable style="width:220px" @keyup.enter="load" />
        <el-button type="primary" @click="load">查询</el-button>
        <el-button type="success" v-permission="'patient:write'" @click="openCreate">建档</el-button>
      </div>
      <el-table :data="list" v-loading="loading" border @row-click="openDetail">
        <el-table-column prop="card_no" label="卡号" width="160" />
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="gender" label="性别" width="70" />
        <el-table-column prop="age" label="年龄" width="80" />
        <el-table-column prop="phone" label="电话" width="140" />
        <el-table-column label="哺乳期" width="80">
          <template #default="{ row }"><el-tag v-if="row.is_lactating" type="warning" size="small">哺乳期</el-tag></template>
        </el-table-column>
      </el-table>
      <el-pagination class="pager" layout="total, prev, pager, next" :total="total" :page-size="query.page_size" v-model:current-page="query.page" @current-change="load" />
    </el-card>

    <el-dialog v-model="createVisible" title="患者建档" width="480px">
      <el-form label-width="80px">
        <el-form-item label="卡号" required><el-input v-model="form.card_no" /></el-form-item>
        <el-form-item label="姓名" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="性别"><el-select v-model="form.gender"><el-option label="男" value="男" /><el-option label="女" value="女" /></el-select></el-form-item>
        <el-form-item label="年龄"><el-input v-model="form.age" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="form.phone" /></el-form-item>
        <el-form-item label="哺乳期"><el-switch v-model="form.is_lactating" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="患者详情" size="520px">
      <template v-if="detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="卡号">{{ detail.card_no }}</el-descriptions-item>
          <el-descriptions-item label="姓名">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item label="性别/年龄">{{ detail.gender }} / {{ detail.age }}</el-descriptions-item>
          <el-descriptions-item label="电话">{{ detail.phone }}</el-descriptions-item>
        </el-descriptions>

        <h4>过敏史</h4>
        <div class="toolbar">
          <el-input v-model="allergyForm.drug_name" placeholder="过敏药名" style="width:160px" />
          <el-select v-model="allergyForm.severity" style="width:110px"><el-option label="轻" :value="1" /><el-option label="中" :value="2" /><el-option label="重" :value="3" /></el-select>
          <el-button type="primary" v-permission="'patient:write'" @click="addAllergy">新增</el-button>
        </div>
        <el-table :data="allergies" border size="small">
          <el-table-column prop="drug_name" label="药名" />
          <el-table-column prop="reaction" label="反应" />
          <el-table-column prop="severity" label="程度" width="60" />
          <el-table-column label="" width="60">
            <template #default="{ row }"><el-button link type="danger" @click="removeAllergy(row)">删</el-button></template>
          </el-table-column>
        </el-table>

        <h4>用药史</h4>
        <el-table :data="history" border size="small">
          <el-table-column prop="drug_name" label="药品" />
          <el-table-column prop="dosage" label="用法" />
          <el-table-column prop="begin_date" label="开药日期" width="110" />
        </el-table>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listPatients, getPatient, createPatient, listAllergies, addAllergy as apiAddAllergy, deleteAllergy, medicationHistory } from '@/api/patients'

const list = ref<any[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ keyword: '', page: 1, page_size: 20 })
const createVisible = ref(false)
const detailVisible = ref(false)
const form = reactive<Record<string, any>>({ gender: '男' })
const detail = ref<any>(null)
const allergies = ref<any[]>([])
const history = ref<any[]>([])
const allergyForm = reactive({ drug_name: '', severity: 1 })

onMounted(load)
async function load() {
  loading.value = true
  try {
    const res = await listPatients(query)
    list.value = res?.list ?? []
    total.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
function openCreate() {
  Object.keys(form).forEach((k) => delete form[k])
  form.gender = '男'
  createVisible.value = true
}
async function save() {
  await createPatient(form)
  ElMessage.success('已建档')
  createVisible.value = false
  load()
}
async function openDetail(row: any) {
  detail.value = await getPatient(row.id)
  allergies.value = (await listAllergies(row.id)) ?? []
  history.value = (await medicationHistory(row.id)) ?? []
  detailVisible.value = true
}
async function addAllergy() {
  await apiAddAllergy(detail.value.id, allergyForm)
  allergies.value = (await listAllergies(detail.value.id)) ?? []
  allergyForm.drug_name = ''
}
async function removeAllergy(row: any) {
  await deleteAllergy(row.id)
  allergies.value = (await listAllergies(detail.value.id)) ?? []
}
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; align-items: center; }
.pager { margin-top: 12px; justify-content: flex-end; }
h4 { margin: 16px 0 8px; }
</style>
