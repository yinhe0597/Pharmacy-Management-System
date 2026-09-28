<template>
  <el-card>
    <el-tabs v-model="tab">
      <el-tab-pane label="计费记录" name="charges">
        <div class="toolbar">
          <el-input
            v-model="query.keyword"
            placeholder="患者/项目"
            clearable
            style="width: 160px"
            @keyup.enter="loadCharges"
          />
          <el-select v-model="query.item_type" placeholder="类型" clearable style="width: 130px">
            <el-option label="药品" value="drug" /><el-option
              label="耗材"
              value="consumable"
            /><el-option label="诊疗项目" value="clinical_service" />
          </el-select>
          <PatientPicker v-model="query.patient_id" placeholder="患者" style="width: 190px" />
          <el-button type="primary" @click="loadCharges">查询</el-button>
          <el-button v-permission="'charge:write'" type="success" @click="openCreate"
            >手工计费</el-button
          >
        </div>
        <el-table v-loading="loading" :data="charges" border>
          <el-table-column prop="patient_name" label="患者" width="100" />
          <el-table-column label="类型" width="100">
            <template #default="{ row }">{{
              CHARGE_ITEM_TYPES[row.item_type] ?? row.item_type
            }}</template>
          </el-table-column>
          <el-table-column prop="item_name" label="项目" min-width="140" />
          <el-table-column prop="quantity" label="数量" width="70" />
          <el-table-column label="金额" width="100">
            <template #default="{ row }"
              ><MoneyText :amount="row.amount" :voided="row.voided"
            /></template>
          </el-table-column>
          <el-table-column prop="ref_type" label="来源" width="120" />
          <el-table-column prop="created_at" label="时间" width="170" />
          <el-table-column label="操作" width="90" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="!row.voided && row.amount > 0"
                v-permission="'charge:write'"
                link
                type="danger"
                @click="doVoid(row)"
                >红冲</el-button
              >
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          v-model:current-page="page"
          class="pager"
          layout="total, prev, pager, next"
          :total="total"
          :page-size="20"
          @current-change="loadCharges"
        />
      </el-tab-pane>

      <el-tab-pane label="诊疗项目目录" name="services">
        <div class="toolbar">
          <el-button v-permission="'drug:write'" type="success" @click="openService"
            >新增项目</el-button
          >
        </div>
        <el-table v-loading="loading" :data="services" border>
          <el-table-column prop="code" label="编码" width="120" />
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="category" label="分类" width="120" />
          <el-table-column label="单价" width="100"
            ><template #default="{ row }"><MoneyText :amount="row.unit_price" /></template
          ></el-table-column>
          <el-table-column prop="unit" label="单位" width="70" />
          <el-table-column label="状态" width="80">
            <template #default="{ row }"
              ><el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{
                row.status === 1 ? '启用' : '停用'
              }}</el-tag></template
            >
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button v-permission="'drug:write'" link type="primary" @click="openService(row)"
                >编辑</el-button
              >
              <el-button
                v-permission="'drug:write'"
                link
                type="warning"
                @click="toggleService(row)"
                >{{ row.status === 1 ? '停用' : '启用' }}</el-button
              >
              <el-button v-permission="'drug:write'" link type="danger" @click="removeService(row)"
                >删除</el-button
              >
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="chargeVisible" title="手工计费" width="480px">
      <el-form label-width="90px">
        <el-form-item label="患者" required
          ><el-input v-model="chargeForm.patient_name"
        /></el-form-item>
        <el-form-item label="关联就诊">
          <el-select
            v-model="chargeForm.visit_id"
            filterable
            clearable
            remote
            :remote-method="searchVisits"
            :loading="visitLoading"
            placeholder="按患者姓名搜索就诊（不选则按时间窗口归集）"
            style="width: 100%"
          >
            <el-option
              v-for="v in visits"
              :key="v.id"
              :label="`${v.visit_no} · ${v.status === 'visiting' ? '就诊中' : v.status === 'finished' ? '已结束' : v.status}`"
              :value="v.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="chargeForm.item_type">
            <el-option label="药品" value="drug" /><el-option
              label="耗材"
              value="consumable"
            /><el-option label="诊疗项目" value="clinical_service" />
          </el-select>
        </el-form-item>
        <el-form-item label="项目名" required
          ><el-input v-model="chargeForm.item_name"
        /></el-form-item>
        <el-form-item label="数量"
          ><el-input-number v-model="chargeForm.quantity" :min="1"
        /></el-form-item>
        <el-form-item label="单价(分)"
          ><el-input-number v-model="chargeForm.unit_price" :min="0"
        /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="chargeVisible = false">取消</el-button>
        <el-button type="primary" @click="saveCharge">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="serviceVisible"
      :title="serviceForm.id ? '编辑诊疗项目' : '新增诊疗项目'"
      width="440px"
    >
      <el-form label-width="80px">
        <el-form-item label="编码" required
          ><el-input v-model="serviceForm.code" :disabled="!!serviceForm.id"
        /></el-form-item>
        <el-form-item label="名称" required><el-input v-model="serviceForm.name" /></el-form-item>
        <el-form-item label="分类"><el-input v-model="serviceForm.category" /></el-form-item>
        <el-form-item label="单价(分)"
          ><el-input-number v-model="serviceForm.unit_price" :min="0"
        /></el-form-item>
        <el-form-item label="单位"
          ><el-input v-model="serviceForm.unit" placeholder="次"
        /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="serviceVisible = false">取消</el-button>
        <el-button type="primary" @click="saveService">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listChargeRecords,
  createCharge,
  voidCharge,
  listClinicalServices,
  createClinicalService,
  updateClinicalService,
  deleteClinicalService,
  setClinicalServiceStatus,
} from '@/api/billing'
import { listVisits } from '@/api/clinical2'
import MoneyText from '@/components/MoneyText.vue'
import PatientPicker from '@/components/PatientPicker.vue'
import { CHARGE_ITEM_TYPES } from '@/types/business'

const tab = ref('charges')
const charges = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const query = reactive({ keyword: '', item_type: '', patient_id: undefined as number | undefined })
const chargeVisible = ref(false)
const chargeForm = reactive({
  // visit_id 决定该费用项被哪张结算单精确归集（后端 charge_records.visit_id，000040）。
  // 不传时后端回退到时间窗口猜测归属，可能错归到同患者的其它就诊。
  visit_id: undefined as number | undefined,
  patient_name: '',
  item_type: 'clinical_service',
  item_name: '',
  quantity: 1,
  unit_price: 0,
})
const visits = ref<any[]>([])
const visitLoading = ref(false)
const services = ref<any[]>([])
const serviceVisible = ref(false)
const serviceForm = reactive<Record<string, any>>({
  id: 0,
  code: '',
  name: '',
  category: '',
  unit_price: 0,
  unit: '次',
})

onMounted(() => {
  loadCharges()
  loadServices()
})

async function loadCharges() {
  loading.value = true
  try {
    const res = await listChargeRecords({ ...query, page: page.value, page_size: 20 })
    charges.value = res?.list ?? []
    total.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
async function loadServices() {
  services.value = (await listClinicalServices({ page: 1, page_size: 200 }))?.list ?? []
}
function openCreate() {
  Object.assign(chargeForm, {
    visit_id: undefined,
    patient_name: '',
    item_type: 'clinical_service',
    item_name: '',
    quantity: 1,
    unit_price: 0,
  })
  visits.value = []
  chargeVisible.value = true
}
async function searchVisits(keyword: string) {
  if (!keyword) {
    visits.value = []
    return
  }
  visitLoading.value = true
  try {
    visits.value = (await listVisits({ keyword, page: 1, page_size: 20 }))?.list ?? []
  } finally {
    visitLoading.value = false
  }
}
async function saveCharge() {
  await createCharge(chargeForm)
  ElMessage.success('已计费')
  chargeVisible.value = false
  loadCharges()
}
async function doVoid(row: any) {
  await voidCharge(row.id)
  ElMessage.success('已红冲')
  loadCharges()
}
function openService(row?: any) {
  Object.assign(serviceForm, {
    id: row?.id ?? 0,
    code: row?.code ?? '',
    name: row?.name ?? '',
    category: row?.category ?? '',
    unit_price: row?.unit_price ?? 0,
    unit: row?.unit ?? '次',
  })
  serviceVisible.value = true
}
async function saveService() {
  const { id, ...payload } = serviceForm
  if (id) {
    await updateClinicalService(id, payload)
  } else {
    await createClinicalService(payload)
  }
  ElMessage.success('已保存')
  serviceVisible.value = false
  loadServices()
}
async function removeService(row: any) {
  await ElMessageBox.confirm(`确认删除诊疗项目「${row.name}」？`, '提示', { type: 'warning' })
  await deleteClinicalService(row.id)
  ElMessage.success('已删除')
  loadServices()
}
async function toggleService(row: any) {
  await setClinicalServiceStatus(row.id, row.status === 1 ? 0 : 1)
  loadServices()
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
