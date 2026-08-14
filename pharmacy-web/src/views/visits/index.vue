<template>
  <div>
    <el-card>
      <div class="toolbar">
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 130px">
          <el-option v-for="(v, k) in VISIT_STATUS" :key="k" :label="v.label" :value="k" />
        </el-select>
        <el-input v-model="query.patient_id" placeholder="患者ID" clearable style="width: 110px" />
        <el-date-picker
          v-model="range"
          type="daterange"
          value-format="YYYY-MM-DD"
          start-placeholder="开始"
          end-placeholder="结束"
        />
        <el-button type="primary" @click="load">查询</el-button>
        <el-button v-permission="'prescription:create'" type="success" @click="openRegister"
          >挂号</el-button
        >
      </div>
      <el-table v-loading="loading" :data="list" border>
        <el-table-column prop="visit_no" label="就诊号" width="150" />
        <el-table-column prop="patient_name" label="患者" width="110" />
        <el-table-column prop="department" label="科室" width="100" />
        <el-table-column prop="doctor_name" label="医生" width="100" />
        <el-table-column label="类型" width="90">
          <template #default="{ row }">{{
            VISIT_TYPES[row.visit_type] ?? row.visit_type
          }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="VISIT_STATUS[row.status]?.tag ?? 'info'" size="small">{{
              VISIT_STATUS[row.status]?.label ?? row.status
            }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="registered_by" label="挂号人" width="90" />
        <el-table-column prop="registered_at" label="挂号时间" width="170" />
        <el-table-column label="操作" min-width="230" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'waiting'"
              v-permission="'prescription:create'"
              link
              type="primary"
              @click="start(row)"
              >接诊</el-button
            >
            <el-button
              v-if="row.status === 'visiting'"
              v-permission="'prescription:create'"
              link
              type="success"
              @click="finish(row)"
              >结束</el-button
            >
            <el-button
              v-if="row.status === 'waiting'"
              v-permission="'prescription:create'"
              link
              type="danger"
              @click="cancel(row)"
              >退号</el-button
            >
            <el-button link type="primary" @click="openRecord(row)">病历</el-button>
            <el-button
              v-if="row.status === 'visiting' || row.status === 'finished'"
              v-permission="'charge:write'"
              link
              type="warning"
              @click="genCharge(row)"
              >结算</el-button
            >
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
    </el-card>

    <!-- 挂号 -->
    <el-dialog v-model="regVisible" title="挂号/分诊" width="460px">
      <el-form label-width="80px">
        <el-form-item label="患者ID" required
          ><el-input v-model="regForm.patient_id"
        /></el-form-item>
        <el-form-item label="科室"><el-input v-model="regForm.department" /></el-form-item>
        <el-form-item label="医生"><el-input v-model="regForm.doctor_name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="regForm.visit_type" style="width: 100%">
            <el-option label="门诊" value="outpatient" />
            <el-option label="住院" value="inpatient" />
            <el-option label="出院带药" value="refill" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="regForm.remarks" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="regVisible = false">取消</el-button>
        <el-button type="primary" @click="doRegister">保存</el-button>
      </template>
    </el-dialog>

    <!-- 病历 -->
    <el-drawer v-model="recordVisible" title="就诊病历" size="560px">
      <template v-if="current">
        <el-form label-width="90px">
          <el-form-item label="主诉"><el-input v-model="recForm.chief_complaint" /></el-form-item>
          <el-form-item label="现病史"
            ><el-input v-model="recForm.present_illness" type="textarea" :rows="2"
          /></el-form-item>
          <el-form-item label="既往史"
            ><el-input v-model="recForm.past_history" type="textarea" :rows="2"
          /></el-form-item>
          <el-form-item label="体格检查"
            ><el-input v-model="recForm.physical_exam" type="textarea" :rows="2"
          /></el-form-item>
          <el-form-item label="体征">
            <div class="vitals">
              <el-input-number
                v-model="recForm.temperature"
                :precision="1"
                :step="0.1"
                placeholder="体温℃"
              />
              <el-input-number v-model="recForm.systolic_pressure" placeholder="收缩压" />
              <el-input-number v-model="recForm.diastolic_pressure" placeholder="舒张压" />
              <el-input-number v-model="recForm.pulse" placeholder="脉搏" />
            </div>
          </el-form-item>
          <el-form-item label="诊断"
            ><el-input v-model="recForm.diagnosis" type="textarea" :rows="2"
          /></el-form-item>
          <el-form-item label="主诊断码"
            ><el-input v-model="recForm.diagnosis_code" placeholder="ICD-10"
          /></el-form-item>
        </el-form>
        <h4>结构化诊断</h4>
        <div v-for="(d, i) in recForm.diagnoses" :key="i" class="diag-row">
          <el-input v-model="d.diagnosis_code" placeholder="ICD-10" style="width: 110px" />
          <el-input v-model="d.diagnosis_name" placeholder="诊断名称" style="width: 180px" />
          <el-checkbox v-model="d.is_primary">主诊断</el-checkbox>
          <el-button link type="danger" @click="recForm.diagnoses.splice(i, 1)">删</el-button>
        </div>
        <el-button
          size="small"
          @click="
            recForm.diagnoses.push({ diagnosis_code: '', diagnosis_name: '', is_primary: false })
          "
          >+ 诊断</el-button
        >
        <div class="drawer-footer">
          <el-button @click="recordVisible = false">关闭</el-button>
          <el-button v-permission="'patient:write'" type="primary" @click="saveRecord"
            >保存病历</el-button
          >
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listVisits,
  registerVisit,
  startVisit,
  finishVisit,
  cancelVisit,
  getMedicalRecord,
  saveMedicalRecord,
  createCharge,
} from '@/api/clinical2'

const VISIT_STATUS: Record<string, { label: string; tag: string }> = {
  waiting: { label: '待诊', tag: 'warning' },
  visiting: { label: '就诊中', tag: 'primary' },
  finished: { label: '已结束', tag: 'success' },
  cancelled: { label: '退号', tag: 'info' },
}
const VISIT_TYPES: Record<string, string> = {
  outpatient: '门诊',
  inpatient: '住院',
  refill: '出院带药',
}

const list = ref<any[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ status: '', patient_id: '', page: 1, page_size: 20 })
const range = ref<[string, string] | null>(null)

const regVisible = ref(false)
const regForm = reactive({
  patient_id: '',
  department: '',
  doctor_name: '',
  visit_type: 'outpatient',
  remarks: '',
})

const recordVisible = ref(false)
const current = ref<any>(null)
const recForm = reactive({
  chief_complaint: '',
  present_illness: '',
  past_history: '',
  physical_exam: '',
  temperature: undefined as number | undefined,
  systolic_pressure: undefined as number | undefined,
  diastolic_pressure: undefined as number | undefined,
  pulse: undefined as number | undefined,
  diagnosis: '',
  diagnosis_code: '',
  diagnoses: [] as any[],
})

async function load() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: query.page, page_size: query.page_size }
    if (query.status) params.status = query.status
    if (query.patient_id) params.patient_id = query.patient_id
    if (range.value) {
      params.start = `${range.value[0]}T00:00:00+08:00`
      params.end = `${range.value[1]}T23:59:59+08:00`
    }
    const r = await listVisits(params)
    list.value = r?.list ?? []
    total.value = r?.total ?? 0
  } finally {
    loading.value = false
  }
}

function openRegister() {
  Object.assign(regForm, {
    patient_id: '',
    department: '',
    doctor_name: '',
    visit_type: 'outpatient',
    remarks: '',
  })
  regVisible.value = true
}

async function doRegister() {
  if (!regForm.patient_id) return ElMessage.warning('请填写患者ID')
  await registerVisit({
    patient_id: Number(regForm.patient_id),
    department: regForm.department,
    doctor_name: regForm.doctor_name,
    visit_type: regForm.visit_type,
    remarks: regForm.remarks,
  })
  ElMessage.success('挂号成功')
  regVisible.value = false
  load()
}

async function start(row: any) {
  await ElMessageBox.confirm(`确认接诊 ${row.patient_name}？`, '接诊')
  await startVisit(row.id)
  ElMessage.success('已接诊')
  load()
}

async function finish(row: any) {
  await ElMessageBox.confirm(`确认结束 ${row.patient_name} 的就诊？`, '结束就诊')
  await finishVisit(row.id)
  ElMessage.success('已结束')
  load()
}

async function cancel(row: any) {
  await ElMessageBox.confirm(`确认退号 ${row.patient_name}？`, '退号')
  await cancelVisit(row.id)
  ElMessage.success('已退号')
  load()
}

async function openRecord(row: any) {
  current.value = row
  Object.assign(recForm, {
    chief_complaint: '',
    present_illness: '',
    past_history: '',
    physical_exam: '',
    temperature: undefined,
    systolic_pressure: undefined,
    diastolic_pressure: undefined,
    pulse: undefined,
    diagnosis: '',
    diagnosis_code: '',
    diagnoses: [],
  })
  try {
    const rec = await getMedicalRecord(row.id)
    if (rec) {
      Object.assign(recForm, rec, { diagnoses: rec.diagnoses ?? [] })
    }
  } catch {
    /* 无病历则为空表单 */
  }
  recordVisible.value = true
}

async function saveRecord() {
  if (!current.value) return
  await saveMedicalRecord(current.value.id, { ...recForm })
  ElMessage.success('病历已保存')
  recordVisible.value = false
}

async function genCharge(row: any) {
  await ElMessageBox.confirm(`为 ${row.patient_name} 生成合并结算单？`, '合并结算')
  await createCharge(row.id, { discount: 0 })
  ElMessage.success('结算单已生成')
}

onMounted(load)
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
.vitals {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.diag-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.drawer-footer {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
