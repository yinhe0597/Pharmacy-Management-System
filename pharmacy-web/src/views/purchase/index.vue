<template>
  <div>
    <el-card>
      <div class="toolbar">
        <el-button v-permission="'purchase:write'" type="success" @click="openCreate"
          >创建采购单</el-button
        >
        <el-button @click="loadSuggestions">采购建议</el-button>
      </div>
      <el-table v-loading="loading" :data="list" border>
        <el-table-column prop="order_no" label="单号" width="150" />
        <el-table-column prop="supplier_name" label="供应商" min-width="140" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }"
            ><StatusTag :status="row.status" :map="ORDER_STATUS"
          /></template>
        </el-table-column>
        <el-table-column label="金额" width="100">
          <template #default="{ row }"><MoneyText :amount="row.total_amount" /></template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status === 'draft'" link type="primary" @click="submit(row)"
              >提交</el-button
            >
            <el-button
              v-if="row.status === 'submitted' || row.status === 'partial'"
              link
              type="success"
              @click="openReceive(row)"
              >收货</el-button
            >
            <el-button v-if="row.status === 'draft'" link type="danger" @click="cancel(row)"
              >作废</el-button
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
        @current-change="load"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" title="创建采购单" width="720px">
      <el-form label-width="90px">
        <el-form-item label="供应商">
          <el-select v-model="form.supplier_id" filterable>
            <el-option v-for="s in suppliers" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="明细">
          <div v-for="(it, idx) in form.items" :key="idx" class="item-row">
            <DrugPicker
              v-model="it.drug_id"
              style="flex: 1"
              @select="(d) => onDrugSelect(idx, d)"
            />
            <el-input-number v-model="it.quantity" :min="1" placeholder="数量" />
            <el-input-number v-model="it.unit_price" :min="0" placeholder="进价(分)" />
            <el-button link type="danger" @click="form.items.splice(idx, 1)">删</el-button>
          </div>
          <el-button
            link
            type="primary"
            @click="form.items.push({ drug_id: null, quantity: 1, unit_price: 0 })"
            >+ 添加明细</el-button
          >
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="receiveVisible" title="收货" width="720px">
      <div v-for="(it, idx) in receiveItems" :key="idx" class="item-row">
        <span style="flex: 1">{{ it.drug_name ?? '明细#' + it.id }}</span>
        <el-input-number v-model="it.received_quantity" :min="0" placeholder="收货数量" />
        <el-input v-model="it.batch_no" placeholder="批号" />
        <el-input v-model="it.expiry_date" placeholder="效期 YYYY-MM-DD" />
        <el-select v-model="it.qc_result" style="width: 110px"
          ><el-option label="合格" :value="1" /><el-option label="不合格" :value="2"
        /></el-select>
      </div>
      <template #footer>
        <el-button @click="receiveVisible = false">取消</el-button>
        <el-button type="primary" @click="doReceive">确认收货</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="suggestVisible" title="采购建议" width="720px">
      <el-table :data="suggestions" border>
        <el-table-column prop="drug_name" label="药品" />
        <el-table-column prop="available" label="可用 LDU" width="100" />
        <el-table-column prop="suggest_qty" label="建议补货(盒)" width="120" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  listPurchaseOrders,
  getPurchaseOrder,
  createPurchaseOrder,
  submitPurchaseOrder,
  cancelPurchaseOrder,
  receivePurchaseOrder,
  purchaseSuggestions,
} from '@/api/purchase'
import { listSuppliers } from '@/api/suppliers'
import DrugPicker from '@/components/DrugPicker.vue'
import StatusTag from '@/components/StatusTag.vue'
import MoneyText from '@/components/MoneyText.vue'

const ORDER_STATUS: Record<string, { label: string; tag: string }> = {
  draft: { label: '草稿', tag: 'info' },
  submitted: { label: '已提交', tag: 'primary' },
  partial: { label: '部分收货', tag: 'warning' },
  received: { label: '已入库', tag: 'success' },
  cancelled: { label: '已作废', tag: 'danger' },
}

const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const suppliers = ref<any[]>([])
const dialogVisible = ref(false)
const receiveVisible = ref(false)
const suggestVisible = ref(false)
const suggestions = ref<any[]>([])
const receiveItems = ref<any[]>([])
const receivingOrderId = ref(0)
const form = reactive<{ supplier_id: number | null; items: any[] }>({
  supplier_id: null,
  items: [],
})

onMounted(async () => {
  load()
  suppliers.value = (await listSuppliers({ page: 1, page_size: 200 }))?.list ?? []
})

async function load() {
  loading.value = true
  try {
    const res = await listPurchaseOrders({ page: page.value, page_size: 20 })
    list.value = res?.list ?? []
    total.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
}
function openCreate() {
  form.supplier_id = null
  form.items = [{ drug_id: null, quantity: 1, unit_price: 0 }]
  dialogVisible.value = true
}
function onDrugSelect(idx: number, d: any) {
  form.items[idx].unit_price = d.purchase_price ?? 0
}
async function save() {
  await createPurchaseOrder({ supplier_id: form.supplier_id, items: form.items })
  ElMessage.success('已创建')
  dialogVisible.value = false
  load()
}
async function submit(row: any) {
  await submitPurchaseOrder(row.id)
  ElMessage.success('已提交')
  load()
}
async function cancel(row: any) {
  await cancelPurchaseOrder(row.id)
  load()
}
async function openReceive(row: any) {
  receivingOrderId.value = row.id
  const detail = await getPurchaseOrder(row.id)
  receiveItems.value = (detail?.items ?? []).map((it: any) => ({
    order_item_id: it.id,
    drug_name: it.drug_name,
    received_quantity: (it.quantity ?? 0) - (it.received_quantity ?? 0),
    batch_no: '',
    expiry_date: '',
    qc_result: 1,
  }))
  receiveVisible.value = true
}
async function doReceive() {
  await receivePurchaseOrder(receivingOrderId.value, { items: receiveItems.value })
  ElMessage.success('已收货')
  receiveVisible.value = false
  load()
}
async function loadSuggestions() {
  suggestions.value = (await purchaseSuggestions()) ?? []
  suggestVisible.value = true
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
.item-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
</style>
