<template>
  <el-card>
    <h3>开方</h3>
    <el-form :model="form" label-width="100px" style="max-width: 1080px">
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
      <el-form-item label="处方类型">
        <el-select v-model="form.prescription_type" style="width: 220px">
          <el-option
            v-for="t in PRESCRIPTION_TYPE_OPTIONS"
            :key="t.value"
            :label="t.label"
            :value="t.value"
          />
        </el-select>
        <el-alert
          v-if="isSpecial"
          type="warning"
          :closable="false"
          show-icon
          style="margin-top: 6px"
          :title="specialAlert"
        />
      </el-form-item>
      <el-row :gutter="12">
        <el-col :span="8"
          ><el-form-item label="妊娠"><el-switch v-model="form.is_pregnant" /></el-form-item
        ></el-col>
        <el-col :span="8"
          ><el-form-item label="哺乳期"><el-switch v-model="form.is_lactating" /></el-form-item
        ></el-col>
      </el-row>

      <el-divider content-position="left">医嘱分组（按给药途径分批，配伍检查覆盖整方）</el-divider>

      <div
        v-for="(group, gi) in groups"
        :key="gi"
        class="group-card"
        :style="{ borderColor: groupColors[gi % groupColors.length] }"
      >
        <div class="group-header" :style="{ background: groupColors[gi % groupColors.length] }">
          <el-input v-model="group.name" class="group-name" size="small" />
          <el-button size="small" text @click="removeGroup(gi)">删除组</el-button>
        </div>
        <div v-for="(it, ii) in group.items" :key="ii" class="item-block">
          <div class="item-row">
            <el-select v-model="it.route" style="width: 130px" placeholder="给药途径">
              <el-option v-for="r in ROUTES" :key="r.value" :label="r.label" :value="r.value" />
            </el-select>
            <DrugPicker
              v-model="it.drug_id"
              style="flex: 1"
              @select="(d) => onDrugSelect(group, ii, d)"
            />
            <span>数量</span><el-input-number v-model="it.quantity" :min="1" />
            <el-checkbox v-model="it.is_split" :disabled="!it.split_allowed">拆零</el-checkbox>
            <el-button link type="danger" @click="group.items.splice(ii, 1)">删</el-button>
          </div>
          <div class="item-row">
            <el-input
              v-model="it.usage_text"
              placeholder="用法（如 口服/溶于250ml盐水）"
              style="width: 200px"
            />
            <el-input v-model="it.frequency" placeholder="频次（如 tid/qd）" style="width: 140px" />
            <span>单次</span><el-input-number v-model="it.single_dose" :min="0" /> <span>日总</span
            ><el-input-number v-model="it.total_daily_dose" :min="0" /> <span>天数</span
            ><el-input-number v-model="it.days" :min="0" />
          </div>
        </div>
        <el-button link type="primary" @click="group.items.push(newItem())"
          >+ 本组添加药品</el-button
        >
      </div>

      <div class="group-actions">
        <el-button type="primary" plain @click="addGroup()">+ 新建分组</el-button>
        <el-alert
          type="info"
          :closable="false"
          style="flex: 1"
          title="提示：静滴/注射剂建议单药一组或同瓶配伍一组，避免分批混淆；配伍禁忌检查覆盖整方全部明细。"
        />
      </div>

      <el-form-item style="margin-top: 16px">
        <el-button type="primary" @click="save(false)">保存草稿</el-button>
        <el-button type="success" @click="save(true)">保存并提交审核</el-button>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  createPrescription,
  submitPrescription,
  type PrescriptionInput,
  type PrescriptionItemInput,
} from '@/api/prescriptions'
import { listPatients } from '@/api/patients'
import { PRESCRIPTION_TYPES, PRESCRIPTION_TYPE_DAY_LIMITS } from '@/types/business'
import DrugPicker from '@/components/DrugPicker.vue'
import DiagnosisPicker from '@/components/DiagnosisPicker.vue'

const ROUTES = [
  { value: 'oral', label: '口服' },
  { value: 'external', label: '外用' },
  { value: 'iv', label: '静脉注射' },
  { value: 'im', label: '肌注' },
  { value: 'iv_drip', label: '静滴' },
  { value: 'inhale', label: '雾化吸入' },
  { value: 'other', label: '其他' },
]
const ROUTE_USAGE: Record<string, string> = {
  oral: '口服',
  external: '外用',
  iv: '静脉注射',
  im: '肌注',
  iv_drip: '静脉滴注',
  inhale: '雾化吸入',
  other: '',
}
const groupColors = ['#409eff', '#67c23a', '#e6a23c', '#f56c6c', '#9254de']
const GROUP_NAMES = ['口服组', '外用组', '输液组1', '输液组2', '输液组3']

// 处方类型选项：复用共享字典（types/business.ts），避免第四份硬编码副本。
const PRESCRIPTION_TYPE_OPTIONS = Object.entries(PRESCRIPTION_TYPES).map(([value, label]) => ({
  value: Number(value),
  label: label + '处方',
}))

type LocalItem = PrescriptionItemInput & { split_allowed?: boolean }
interface Group {
  name: string
  items: LocalItem[]
}

const router = useRouter()
const form = reactive<Omit<PrescriptionInput, 'items'> & { diagnosis_code: string }>({
  patient_id: undefined,
  patient_name: '',
  patient_gender: '男',
  patient_age: '',
  patient_card_no: '',
  diagnosis_code: '',
  diagnosis: '',
  department: '',
  doctor_name: '',
  prescription_type: 0,
  is_pregnant: false,
  is_lactating: false,
})
const patients = ref<any[]>([])
const patientLoading = ref(false)

function newItem(): LocalItem {
  return { drug_id: 0, quantity: 1, is_split: false, split_allowed: false, route: 'oral' }
}
const groups = reactive<Group[]>([{ name: GROUP_NAMES[0], items: [newItem()] }])

// 已选药品的全部明细（跨分组），用于开方前的限量自检
const allItems = computed<LocalItem[]>(() =>
  groups.flatMap((g) => g.items).filter((it) => it.drug_id > 0),
)
// prescription_type 在契约中为可选字段，这里收敛为确定的数字以便查字典。
const rxType = computed(() => form.prescription_type ?? 0)
const isSpecial = computed(() => rxType.value !== 0)
const currentType = computed(() => PRESCRIPTION_TYPES[rxType.value] ?? '普通')
const dayLimit = computed(() => PRESCRIPTION_TYPE_DAY_LIMITS[rxType.value] ?? 0)
// 超出限量的明细（仅对有限量天数的类型有意义）
const overLimitItems = computed(() => {
  const limit = dayLimit.value
  if (limit <= 0) return []
  return allItems.value.filter((it) => !it.days || it.days <= 0 || it.days > limit)
})
const specialAlert = computed(() => {
  const limit = dayLimit.value
  const parts = ['麻精毒放专管处方：须填写患者卡号与诊断，并全程专人负责、双人核对。']
  if (limit > 0) {
    parts.push(`单张处方限量 ${limit} 日，每条明细「天数」须为 1~${limit}。`)
  }
  if (overLimitItems.value.length > 0) {
    parts.push(`当前有 ${overLimitItems.value.length} 条明细「天数」不合规，提交将被拒绝。`)
  }
  if (!form.patient_card_no || !form.diagnosis) {
    parts.push('患者卡号与诊断为必填项。')
  }
  return parts.join(' ')
})

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

function onDrugSelect(group: Group, idx: number, d: any) {
  group.items[idx].split_allowed = !!d.is_split_allowed
  const route = group.items[idx].route ?? ''
  if (route && !group.items[idx].usage_text) {
    group.items[idx].usage_text = ROUTE_USAGE[route] ?? ''
  }
}

function addGroup() {
  const n = groups.length + 1
  const name = n <= GROUP_NAMES.length ? GROUP_NAMES[n - 1] : `分组${n}`
  groups.push({ name, items: [newItem()] })
}
function removeGroup(gi: number) {
  if (groups.length <= 1) return ElMessage.warning('至少保留一个分组')
  groups.splice(gi, 1)
}

async function save(submit: boolean) {
  const items: PrescriptionItemInput[] = groups.flatMap((g) =>
    g.items
      .filter((it) => it.drug_id > 0)
      .map((it) => ({
        drug_id: it.drug_id,
        quantity: it.quantity,
        is_split: it.is_split,
        usage_text: it.usage_text,
        frequency: it.frequency,
        route: it.route,
        batch_group: g.name || '',
        single_dose: it.single_dose,
        total_daily_dose: it.total_daily_dose,
        days: it.days,
      })),
  )
  if (!form.patient_name) return ElMessage.warning('请填写患者姓名')
  if (!items.length) return ElMessage.warning('请至少添加一条药品明细')
  // 麻精毒放专管处方：卡号/诊断为必填（后端 4001 校验），限量类型的天数须落在 1~limit
  // （后端 4002 校验）。此处前置拦截，避免提交后才报错。
  if (isSpecial.value) {
    if (!form.patient_card_no || !form.diagnosis) {
      return ElMessage.warning('麻精毒放专管处方必须填写患者卡号与诊断')
    }
    if (overLimitItems.value.length > 0) {
      return ElMessage.warning(
        `有 ${overLimitItems.value.length} 条明细的「天数」不合规：${currentType.value}处方限量 ${dayLimit.value} 日，请填写 1~${dayLimit.value}`,
      )
    }
  }
  const created = await createPrescription({ ...form, items })
  ElMessage.success('已保存')
  if (submit) {
    await submitPrescription(created.id)
    ElMessage.success('已提交审核')
  }
  router.push(`/prescriptions/${created.id}`)
}
</script>

<style scoped>
.group-card {
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  margin-bottom: 12px;
  padding: 0 12px 12px;
  border-left-width: 4px;
}
.group-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 0 -12px 8px;
  padding: 6px 12px;
  border-radius: 3px 3px 0 0;
}
.group-name {
  width: 180px;
}
.item-block {
  border-bottom: 1px dashed #ebeef5;
  padding: 8px 0;
}
.item-block:last-of-type {
  border-bottom: none;
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
.group-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
