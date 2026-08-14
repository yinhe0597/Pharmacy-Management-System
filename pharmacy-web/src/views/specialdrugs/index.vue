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
        <el-table :data="ledgers" border>
          <el-table-column prop="drug_id" label="药品ID" width="80" />
          <el-table-column prop="batch_no" label="批号" width="120" />
          <el-table-column prop="log_type" label="类型" width="90" />
          <el-table-column prop="quantity" label="数量" width="80" />
          <el-table-column prop="patient_name" label="患者" width="100" />
          <el-table-column prop="created_at" label="时间" width="170" />
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="ampouleVisible" title="空安瓿回收登记" width="440px">
      <el-form label-width="80px">
        <el-form-item label="药品ID"
          ><el-input-number v-model="ampouleForm.drug_id" :min="1"
        /></el-form-item>
        <el-form-item label="数量"
          ><el-input-number v-model="ampouleForm.quantity" :min="1"
        /></el-form-item>
        <el-form-item label="备注"><el-input v-model="ampouleForm.notes" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ampouleVisible = false">取消</el-button>
        <el-button type="primary" @click="saveAmpoule">保存</el-button>
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
} from '@/api/specialdrugs'
import StatusTag from '@/components/StatusTag.vue'
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
const ampouleForm = reactive({ drug_id: 1, quantity: 1, notes: '' })

onMounted(async () => {
  prescriptions.value = (await listSpecialPrescriptions({ page: 1, page_size: 50 }))?.list ?? []
  ampoules.value = (await listAmpouleReturns({ page: 1, page_size: 50 }))?.list ?? []
  ledgers.value = (await listLedgers({ page: 1, page_size: 50 }))?.list ?? []
})
function openAmpoule() {
  ampouleForm.drug_id = 1
  ampouleForm.quantity = 1
  ampouleForm.notes = ''
  ampouleVisible.value = true
}
async function saveAmpoule() {
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
</script>

<style scoped>
.toolbar {
  margin-bottom: 12px;
}
</style>
