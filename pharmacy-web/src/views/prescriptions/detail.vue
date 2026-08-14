<template>
  <el-card v-if="detail" v-loading="loading">
    <div class="header">
      <div>
        <h3>处方 {{ detail.prescription_no }}</h3>
        <StatusTag :status="detail.status" :map="PRESCRIPTION_STATUS" />
        <el-tag v-if="detail.prescription_type > 0" type="danger" size="small" style="margin-left:8px">{{ PRESCRIPTION_TYPE[detail.prescription_type] }}</el-tag>
      </div>
      <div class="actions">
        <el-button v-if="detail.status === 'pending_review'" v-permission="'prescription:create'" @click="verifyOrder">核对医嘱</el-button>
        <el-button v-if="detail.status === 'pending_review'" v-permission="'prescription:create'" type="primary" @click="submit">提交审核</el-button>
        <el-button v-if="detail.status === 'pending_review'" v-permission="'prescription:review'" type="success" @click="review('pass')">审核通过</el-button>
        <el-button v-if="detail.status === 'pending_review'" v-permission="'prescription:review'" type="warning" @click="review('return')">退回医生</el-button>
        <el-button v-if="detail.status === 'pending_review'" v-permission="'prescription:review'" type="danger" @click="review('reject')">驳回</el-button>
        <el-button v-if="detail.status === 'reviewed_passed'" v-permission="'inventory:write'" type="primary" @click="dispense">调配</el-button>
        <el-button v-if="detail.status === 'dispensing'" v-permission="'inventory:write'" type="success" @click="confirmDispense">发药确认</el-button>
        <el-button v-if="detail.status === 'dispensed'" v-permission="'inventory:write'" type="warning" @click="openReturn">退药</el-button>
        <el-button v-if="detail.status === 'dispensed'" v-permission="'charge:write'" @click="charge">一键计费</el-button>
        <el-button v-if="['pending_review', 'reviewed_passed', 'dispensing'].includes(detail.status)" v-permission="'inventory:write'" type="danger" plain @click="cancel">作废</el-button>
      </div>
    </div>

    <el-descriptions :column="3" border style="margin-top:12px">
      <el-descriptions-item label="患者">{{ detail.patient_name }}</el-descriptions-item>
      <el-descriptions-item label="性别/年龄">{{ detail.patient_gender }} / {{ detail.patient_age }}</el-descriptions-item>
      <el-descriptions-item label="诊断">{{ detail.diagnosis_code ? `[${detail.diagnosis_code}] ` : '' }}{{ detail.diagnosis }}</el-descriptions-item>
      <el-descriptions-item label="科室">{{ detail.department }}</el-descriptions-item>
      <el-descriptions-item label="医生">{{ detail.doctor_name }}</el-descriptions-item>
      <el-descriptions-item label="金额"><MoneyText :amount="detail.total_amount" /></el-descriptions-item>
      <el-descriptions-item v-if="detail.is_pregnant || detail.is_lactating" label="个体化">
        <el-tag v-if="detail.is_pregnant" type="danger" size="small">妊娠</el-tag>
        <el-tag v-if="detail.is_lactating" type="warning" size="small">哺乳期</el-tag>
      </el-descriptions-item>
    </el-descriptions>

    <h4>明细</h4>
    <el-table :data="detail.items" border>
      <el-table-column prop="line_no" label="#" width="50" />
      <el-table-column prop="drug_name" label="药品" min-width="140" />
      <el-table-column prop="specification" label="规格" width="100" />
      <el-table-column prop="quantity" label="数量(LDU)" width="90" />
      <el-table-column label="单价" width="90"><template #default="{ row }"><MoneyText :amount="row.unit_price" /></template></el-table-column>
      <el-table-column label="金额" width="90"><template #default="{ row }"><MoneyText :amount="row.amount" /></template></el-table-column>
      <el-table-column prop="usage_text" label="用法" width="120" />
      <el-table-column prop="frequency" label="频次" width="80" />
      <el-table-column prop="days" label="天数" width="60" />
    </el-table>

    <template v-if="detail.allergies?.length">
      <h4>过敏史</h4>
      <el-alert type="danger" :closable="false" :title="detail.allergies.map((a) => a.drug_name).join('、')" />
    </template>

    <h4>流转日志</h4>
    <el-timeline style="padding-left:4px">
      <el-timeline-item v-for="log in detail.audit_logs" :key="log.id" :timestamp="log.created_at">
        <b>{{ ACTION_LABEL[log.action] ?? log.action }}</b>
        <span v-if="log.operator_name" style="color:#909399"> · {{ log.operator_name }}</span>
        <div v-if="log.remarks" style="color:#606266">{{ log.remarks }}</div>
      </el-timeline-item>
    </el-timeline>
  </el-card>

  <el-dialog v-model="returnVisible" title="退药" width="520px">
    <div v-for="it in detail?.items?.filter((i) => (i.dispensed_quantity ?? 0) > (i.returned_quantity ?? 0))" :key="it.id" class="item-row">
      <span style="flex:1">{{ it.drug_name }}</span>
      <span style="color:#909399">可退 {{ (it.dispensed_quantity ?? 0) - (it.returned_quantity ?? 0) }}</span>
      <el-input-number v-model="returnQty[it.id]" :min="0" :max="(it.dispensed_quantity ?? 0) - (it.returned_quantity ?? 0)" />
    </div>
    <template #footer>
      <el-button @click="returnVisible = false">取消</el-button>
      <el-button type="primary" @click="doReturn">确认退药</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getPrescription, verifyOrder as apiVerifyOrder, submitPrescription, reviewPrescription, dispensePrescription, confirmDispense as apiConfirmDispense, returnPrescription, cancelPrescription } from '@/api/prescriptions'
import { chargePrescription } from '@/api/billing'
import StatusTag from '@/components/StatusTag.vue'
import MoneyText from '@/components/MoneyText.vue'
import { PRESCRIPTION_STATUS } from '@/types/business'
import type { PrescriptionDetail } from '@/types/entities'

const PRESCRIPTION_TYPE: Record<number, string> = { 1: '麻醉', 2: '精神一类', 3: '精神二类', 4: '毒性', 5: '放射性' }
const ACTION_LABEL: Record<string, string> = {
  create: '创建', submit: '提交审核', re_submit: '重新提交', verify_order: '核对医嘱',
  review_pass: '审核通过', review_reject: '审核驳回', review_return: '退回医生',
  dispense: '调配', confirm_dispense: '发药确认', return: '退药', cancel: '作废',
}

const route = useRoute()
const detail = ref<PrescriptionDetail | null>(null)
const loading = ref(false)
const returnVisible = ref(false)
const returnQty = reactive<Record<number, number>>({})

onMounted(load)
async function load() {
  loading.value = true
  try {
    detail.value = await getPrescription(Number(route.params.id))
  } finally {
    loading.value = false
  }
}

async function verifyOrder() {
  await apiVerifyOrder(detail.value!.id)
  ElMessage.success('医嘱已核对')
  load()
}
async function submit() {
  await submitPrescription(detail.value!.id)
  ElMessage.success('已提交审核')
  load()
}
async function review(action: string) {
  await reviewPrescription(detail.value!.id, action)
  ElMessage.success('已处理')
  load()
}
async function dispense() {
  await dispensePrescription(detail.value!.id)
  load()
}
async function confirmDispense() {
  await apiConfirmDispense(detail.value!.id)
  ElMessage.success('发药确认完成')
  load()
}
function openReturn() {
  Object.keys(returnQty).forEach((k) => delete returnQty[Number(k)])
  returnVisible.value = true
}
async function doReturn() {
  const items = Object.entries(returnQty)
    .filter(([, q]) => q > 0)
    .map(([id, q]) => ({ item_id: Number(id), return_quantity: q }))
  if (!items.length) return ElMessage.warning('请填写退药数量')
  await returnPrescription(detail.value!.id, items)
  ElMessage.success('退药完成（已自动冲正）')
  returnVisible.value = false
  load()
}
async function charge() {
  await chargePrescription(detail.value!.id)
  ElMessage.success('已生成计费')
  load()
}
async function cancel() {
  await cancelPrescription(detail.value!.id)
  load()
}
</script>

<style scoped>
.header { display: flex; justify-content: space-between; align-items: flex-start; }
.actions { display: flex; flex-wrap: wrap; gap: 4px; justify-content: flex-end; }
h4 { margin: 16px 0 8px; }
.item-row { display: flex; gap: 8px; align-items: center; margin-bottom: 8px; }
</style>
