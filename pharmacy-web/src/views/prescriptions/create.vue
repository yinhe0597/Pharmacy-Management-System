<template>
  <el-card>
    <h3>开方</h3>
    <el-form :model="form" label-width="100px" style="max-width: 960px">
      <el-form-item label="患者">
        <el-select
          v-model="form.patient_id"
          filterable
          remote
          clearable
          :remote-method="searchPatients"
          :loading="patientLoading"
          placeholder="搜索患者（可留空手工填写）"
          style="width: 320px"
          @change="onPatientSelect"
        >
          <el-option
            v-for="p in patients"
            :key="p.id"
            :label="`${p.name} ${p.card_no ?? ''}`"
            :value="p.id"
          />
        </el-select>
      </el-form-item>
      <el-row :gutter="12">
        <el-col :span="8"
          ><el-form-item label="姓名" required
            ><el-input v-model="form.patient_name" /></el-form-item
        ></el-col>
        <el-col :span="8"
          ><el-form-item label="性别"
            ><el-select v-model="form.patient_gender"
              ><el-option label="男" value="男" /><el-option
                label="女"
                value="女" /></el-select></el-form-item
        ></el-col>
        <el-col :span="8"
          ><el-form-item label="年龄"><el-input v-model="form.patient_age" /></el-form-item
        ></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :span="8"
          ><el-form-item label="卡号"><el-input v-model="form.patient_card_no" /></el-form-item
        ></el-col>
        <el-col :span="8"
          ><el-form-item label="科室"><el-input v-model="form.department" /></el-form-item
        ></el-col>
        <el-col :span="8"
          ><el-form-item label="医生"><el-input v-model="form.doctor_name" /></el-form-item
        ></el-col>
      </el-row>
      <el-form-item label="诊断编码"
        ><DiagnosisPicker
          v-model="form.diagnosis_code"
          @select="(d) => (form.diagnosis = d.disease_name)"
      /></el-form-item>
      <el-form-item label="诊断"
        ><el-input v-model="form.diagnosis" type="textarea"
      /></el-form-item>
      <el-row :gutter="12">
        <el-col :span="8"
          ><el-form-item label="妊娠"><el-switch v-model="form.is_pregnant" /></el-form-item
        ></el-col>
        <el-col :span="8"
          ><el-form-item label="哺乳期"><el-switch v-model="form.is_lactating" /></el-form-item
        ></el-col>
      </el-row>

      <el-divider>明细</el-divider>
      <div v-for="(it, idx) in form.items" :key="idx" class="item-card">
        <div class="item-row">
          <DrugPicker v-model="it.drug_id" style="flex: 1" @select="(d) => onDrugSelect(idx, d)" />
          <span>数量(LDU)</span><el-input-number v-model="it.quantity" :min="1" />
          <el-checkbox v-model="it.is_split" :disabled="!it.split_allowed">拆零</el-checkbox>
          <el-button link type="danger" @click="form.items.splice(idx, 1)">删</el-button>
        </div>
        <div class="item-row">
          <el-input v-model="it.usage_text" placeholder="用法（如 口服）" style="width: 160px" />
          <el-input v-model="it.frequency" placeholder="频次（如 tid）" style="width: 140px" />
          <span>单次</span><el-input-number v-model="it.single_dose" :min="0" /> <span>日总</span
          ><el-input-number v-model="it.total_daily_dose" :min="0" /> <span>天数</span
          ><el-input-number v-model="it.days" :min="0" />
        </div>
      </div>
      <el-button
        link
        type="primary"
        @click="form.items.push({ drug_id: 0, quantity: 1, is_split: false, split_allowed: false })"
        >+ 添加明细</el-button
      >

      <el-form-item style="margin-top: 16px">
        <el-button type="primary" @click="save">保存草稿</el-button>
        <el-button type="success" @click="saveAndSubmit">保存并提交审核</el-button>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  createPrescription,
  submitPrescription,
  type PrescriptionInput,
  type PrescriptionItemInput,
} from '@/api/prescriptions'
import { listPatients } from '@/api/patients'
import DrugPicker from '@/components/DrugPicker.vue'
import DiagnosisPicker from '@/components/DiagnosisPicker.vue'

type LocalItem = PrescriptionItemInput & { split_allowed?: boolean }

const router = useRouter()
const form = reactive<
  Omit<PrescriptionInput, 'items'> & { diagnosis_code: string; items: LocalItem[] }
>({
  patient_id: undefined,
  patient_name: '',
  patient_gender: '男',
  patient_age: '',
  patient_card_no: '',
  diagnosis_code: '',
  diagnosis: '',
  department: '',
  doctor_name: '',
  is_pregnant: false,
  is_lactating: false,
  items: [{ drug_id: 0, quantity: 1, is_split: false, split_allowed: false }],
})
const patients = ref<any[]>([])
const patientLoading = ref(false)

async function searchPatients(keyword: string) {
  patientLoading.value = true
  try {
    patients.value = (await listPatients({ keyword, page: 1, page_size: 20 }))?.list ?? []
  } finally {
    patientLoading.value = false
  }
}
searchPatients('')

function onPatientSelect(id: number) {
  const p = patients.value.find((x) => x.id === id)
  if (!p) return
  form.patient_name = p.name
  form.patient_gender = p.gender ?? ''
  form.patient_age = p.age ?? ''
  form.patient_card_no = p.card_no ?? ''
  if (p.is_lactating) form.is_lactating = true
}
function onDrugSelect(idx: number, d: any) {
  form.items[idx].split_allowed = !!d.is_split_allowed
}

async function doSave(submit: boolean) {
  const payload: PrescriptionInput = {
    ...form,
    items: form.items.map((it) => ({
      drug_id: it.drug_id,
      quantity: it.quantity,
      is_split: it.is_split,
      usage_text: it.usage_text,
      frequency: it.frequency,
      single_dose: it.single_dose,
      total_daily_dose: it.total_daily_dose,
      days: it.days,
    })),
  }
  const created = await createPrescription(payload)
  ElMessage.success('已保存')
  if (submit) {
    await submitPrescription(created.id)
    ElMessage.success('已提交审核')
  }
  router.push(`/prescriptions/${created.id}`)
}
function save() {
  doSave(false)
}
function saveAndSubmit() {
  doSave(true)
}
</script>

<style scoped>
.item-card {
  border: 1px dashed #dcdfe6;
  padding: 8px;
  border-radius: 4px;
  margin-bottom: 8px;
}
.item-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.item-row:last-child {
  margin-bottom: 0;
}
</style>
