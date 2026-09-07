<template>
  <el-card>
    <el-tabs v-model="tab">
      <el-tab-pane label="麻精处方" name="prescriptions">
        <el-table :data="prescriptions" border>
          <el-table-column prop="prescription_no" label="处方号" width="150" />
          <el-table-column prop="patient_name" label="患者" width="100" />
          <el-table-column prop="patient_card_no" label="卡号" width="160" />
          <el-table-column label="类型" width="90">
            <template #default="{ row }">{{ TYPE_LABEL[row.prescription_type] }}</template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }"
              ><StatusTag :status="row.status" :map="PRESCRIPTION_STATUS"
            /></template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="空安瓿回收" name="ampoule">
        <div class="toolbar">
          <el-button type="success" @click="openAmpoule">登记回收</el-button>
        </div>
        <el-table :data="ampoules" border>
          <el-table-column prop="drug_id" label="药品ID" width="80" />
          <el-table-column prop="quantity" label="数量" width="80" />
          <el-table-column prop="return_date" label="回收日期" width="120" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'verified' ? 'success' : 'warning'" size="small">{{
                row.status === 'verified' ? '已核对' : '待核对'
              }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button v-if="row.status !== 'verified'" link type="primary" @click="verify(row)"
                >核对</el-button
              >
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="专账查询" name="ledgers">
        <div class="toolbar">
          <el-button v-permission="'drug:write'" type="success" @click="openRegister"
            >发药登记（补录）</el-button
          >
        </div>
        <el-table :data="ledgers" border>
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="drug_id" label="药品ID" width="80" />
          <el-table-column prop="batch_no" label="批号" width="120" />
          <el-table-column prop="log_type" label="类型" width="90" />
          <el-table-column prop="quantity" label="数量" width="80" />
          <el-table-column prop="patient_name" label="患者" width="100" />
          <el-table-column prop="operator_name" label="经手人" width="100" />
          <el-table-column prop="notes" label="备注" min-width="120" />
          <el-table-column prop="created_at" label="时间" width="170" />
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="ampouleVisible" title="空安瓿回收登记" width="480px">
      <el-form label-width="90px">
        <el-form-item label="药品" required>
          <DrugPicker v-model="ampouleForm.drug_id" />
        </el-form-item>
        <el-form-item label="批号"><el-input v-model="ampouleForm.batch_no" /></el-form-item>
        <el-form-item label="回收数量" required
          ><el-input-number v-model="ampouleForm.quantity" :min="1"
        /></el-form-item>
        <el-form-item label="经手人"><el-input v-model="ampouleForm.returned_by" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="ampouleForm.notes" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ampouleVisible = false">取消</el-button>
        <el-button type="primary" @click="saveAmpoule">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="registerVisible" title="麻精发药专册登记（手工补录）" width="520px">
      <el-form label-width="90px">
        <el-form-item label="药品" required>
          <DrugPicker v-model="registerForm.drug_id" />
        </el-form-item>
        <el-form-item label="批号" required
          ><el-input v-model="registerForm.batch_no"
        /></el-form-item>
        <el-form-item label="数量" required>
          <el-input-number v-model="registerForm.quantity" />
          <div class="hint">正数 = 发药出账，负数 = 退药冲正</div>
        </el-form-item>
        <el-form-item label="患者姓名"
          ><el-input v-model="registerForm.patient_name"
        /></el-form-item>
        <el-form-item label="患者卡号"
          ><el-input v-model="registerForm.patient_card_no"
        /></el-form-item>
        <el-form-item label="经手人"
          ><el-input v-model="registerForm.operator_name"
        /></el-form-item>
        <el-form-item label="备注"><el-input v-model="registerForm.notes" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="registerVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRegister">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  listSpecialPrescriptions,
  listAmpouleReturns,
  createAmpouleReturn,
  verifyAmpouleReturn,
  listLedgers,
  registerDispense,
} from '@/api/specialdrugs'
import StatusTag from '@/components/StatusTag.vue'
import DrugPicker from '@/components/DrugPicker.vue'
import { PRESCRIPTION_STATUS } from '@/types/business'

const TYPE_LABEL: Record<number, string> = {
  1: '麻醉',
  2: '精神一类',
  3: '精神二类',
  4: '毒性',
  5: '放射性',
}

const tab = ref('prescriptions')
const prescriptions = ref<any[]>([])
const ampoules = ref<any[]>([])
const ledgers = ref<any[]>([])
const ampouleVisible = ref(false)
const ampouleForm = reactive<Record<string, any>>({
  drug_id: null,
  batch_no: '',
  quantity: 1,
  returned_by: '',
  notes: '',
})
const registerVisible = ref(false)
const registerForm = reactive<Record<string, any>>({
  drug_id: null,
  batch_no: '',
  quantity: 1,
  patient_name: '',
  patient_card_no: '',
  operator_name: '',
  notes: '',
})

onMounted(async () => {
  prescriptions.value = (await listSpecialPrescriptions({ page: 1, page_size: 50 }))?.list ?? []
  ampoules.value = (await listAmpouleReturns({ page: 1, page_size: 50 }))?.list ?? []
  ledgers.value = (await listLedgers({ page: 1, page_size: 50 }))?.list ?? []
})
function openAmpoule() {
  Object.assign(ampouleForm, {
    drug_id: null,
    batch_no: '',
    quantity: 1,
    returned_by: '',
    notes: '',
  })
  ampouleVisible.value = true
}
async function saveAmpoule() {
  if (!ampouleForm.drug_id || !ampouleForm.quantity) {
    ElMessage.warning('请选择药品并填写数量')
    return
  }
  await createAmpouleReturn(ampouleForm)
  ElMessage.success('已登记')
  ampouleVisible.value = false
  ampoules.value = (await listAmpouleReturns({ page: 1, page_size: 50 }))?.list ?? []
}
async function verify(row: any) {
  await verifyAmpouleReturn(row.id)
  ElMessage.success('已核对')
  ampoules.value = (await listAmpouleReturns({ page: 1, page_size: 50 }))?.list ?? []
}
function openRegister() {
  Object.assign(registerForm, {
    drug_id: null,
    batch_no: '',
    quantity: 1,
    patient_name: '',
    patient_card_no: '',
    operator_name: '',
    notes: '',
  })
  registerVisible.value = true
}
async function saveRegister() {
  if (!registerForm.drug_id || !registerForm.batch_no || !registerForm.quantity) {
    ElMessage.warning('请填写药品/批号/数量')
    return
  }
  await registerDispense(registerForm)
  ElMessage.success('已登记专账')
  registerVisible.value = false
  ledgers.value = (await listLedgers({ page: 1, page_size: 50 }))?.list ?? []
}
</script>

<style scoped>
.toolbar {
  margin-bottom: 12px;
}
</style>
