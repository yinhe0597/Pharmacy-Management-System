<template>
  <el-card>
    <el-tabs v-model="tab">
      <el-tab-pane label="库存列表" name="stock">
        <div class="toolbar">
          <el-input v-model="query.keyword" placeholder="药品名称" clearable style="width:220px" @keyup.enter="loadStock" />
          <el-button type="primary" @click="loadStock">查询</el-button>
        </div>
        <el-table :data="stock" v-loading="loading" border>
          <el-table-column prop="drug_name" label="药品" min-width="140" />
          <el-table-column prop="batch_no" label="批号" width="120" />
          <el-table-column prop="location_name" label="库房" width="100" />
          <el-table-column prop="expiry_date" label="效期" width="110" />
          <el-table-column label="口径" width="70">
            <template #default="{ row }">{{ row.is_split ? '拆零' : '整盒' }}</template>
          </el-table-column>
          <el-table-column prop="quantity" label="数量" width="80" />
          <el-table-column prop="reserved_quantity" label="已预占" width="80" />
        </el-table>
        <el-pagination class="pager" layout="total, prev, pager, next" :total="stockTotal" :page-size="20" v-model:current-page="query.page" @current-change="loadStock" />
      </el-tab-pane>

      <el-tab-pane label="领用/补发登记单" name="requisition">
        <div class="toolbar">
          <el-button type="success" v-permission="'inventory:write'" @click="openRequisition">新建补发登记单</el-button>
        </div>
        <el-table :data="orders" v-loading="loading" border>
          <el-table-column prop="requisition_no" label="单号" width="150" />
          <el-table-column prop="purpose" label="目的" width="100">
            <template #default="{ row }">{{ PURPOSE[row.purpose] ?? row.purpose }}</template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" min-width="140" />
          <el-table-column prop="operator_name" label="操作人" width="100" />
          <el-table-column prop="created_at" label="时间" width="170" />
        </el-table>
        <el-pagination class="pager" layout="total, prev, pager, next" :total="orderTotal" :page-size="20" v-model:current-page="orderPage" @current-change="loadOrders" />
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="reqVisible" title="新建补发登记单" width="640px">
      <el-form label-width="90px">
        <el-form-item label="库房"><el-input-number v-model="reqForm.location_id" :min="1" /></el-form-item>
        <el-form-item label="目的">
          <el-select v-model="reqForm.purpose">
            <el-option label="补发" value="supplement" />
            <el-option label="临床领用" value="clinical" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="原因"><el-input v-model="reqForm.reason" /></el-form-item>
        <el-form-item label="明细">
          <div v-for="(it, idx) in reqForm.items" :key="idx" class="item-row">
            <DrugPicker v-model="it.drug_id" style="flex:1" />
            <el-input-number v-model="it.quantity" :min="1" placeholder="数量(LDU)" />
            <el-button link type="danger" @click="reqForm.items.splice(idx, 1)">删</el-button>
          </div>
          <el-button link type="primary" @click="reqForm.items.push({ drug_id: null, quantity: 1 })">+ 添加明细</el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reqVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRequisition">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listInventory, listRequisitionOrders, createRequisitionOrder } from '@/api/inventory'
import DrugPicker from '@/components/DrugPicker.vue'

const PURPOSE: Record<string, string> = { supplement: '补发', clinical: '临床领用', other: '其他' }

const tab = ref('stock')
const stock = ref<any[]>([])
const stockTotal = ref(0)
const orders = ref<any[]>([])
const orderTotal = ref(0)
const orderPage = ref(1)
const loading = ref(false)
const query = reactive({ keyword: '', page: 1, page_size: 20 })
const reqVisible = ref(false)
const reqForm = reactive<{ location_id: number; purpose: string; reason: string; items: any[] }>({
  location_id: 2, purpose: 'supplement', reason: '', items: [{ drug_id: null, quantity: 1 }],
})

onMounted(() => { loadStock(); loadOrders() })

async function loadStock() {
  loading.value = true
  try {
    const res = await listInventory(query)
    stock.value = res?.list ?? []
    stockTotal.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
async function loadOrders() {
  loading.value = true
  try {
    const res = await listRequisitionOrders({ page: orderPage.value, page_size: 20 })
    orders.value = res?.list ?? []
    orderTotal.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
function openRequisition() {
  reqForm.location_id = 2
  reqForm.purpose = 'supplement'
  reqForm.reason = ''
  reqForm.items = [{ drug_id: null, quantity: 1 }]
  reqVisible.value = true
}
async function saveRequisition() {
  await createRequisitionOrder(reqForm)
  ElMessage.success('已登记')
  reqVisible.value = false
  loadOrders()
}
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.pager { margin-top: 12px; justify-content: flex-end; }
.item-row { display: flex; gap: 8px; align-items: center; margin-bottom: 8px; }
</style>
