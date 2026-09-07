<template>
  <div>
    <el-card>
      <el-tabs v-model="tab">
        <el-tab-pane label="采购单" name="orders">
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
        </el-tab-pane>

        <el-tab-pane label="收货单" name="receipts">
          <el-table v-loading="loading" :data="receipts" border>
            <el-table-column prop="receipt_no" label="收货单号" width="160" />
            <el-table-column prop="purchase_order_id" label="采购单ID" width="100" />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-tag :type="RECEIPT_TAG[row.status] ?? 'info'" size="small">{{
                  RECEIPT_LABEL[row.status] ?? row.status
                }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="金额" width="110">
              <template #default="{ row }"><MoneyText :amount="row.total_amount" /></template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="170" />
            <el-table-column label="操作" width="170" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openReceipt(row)">详情</el-button>
                <el-button
                  v-if="row.status === 'pending_quality'"
                  v-permission="'purchase:write'"
                  link
                  type="success"
                  @click="doCompleteReceipt(row)"
                  >确认入库</el-button
                >
              </template>
            </el-table-column>
          </el-table>
          <el-pagination
            v-model:current-page="receiptPage"
            class="pager"
            layout="total, prev, pager, next"
            :total="receiptTotal"
            :page-size="20"
            @current-change="loadReceipts"
          />
        </el-tab-pane>
      </el-tabs>
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

    <el-dialog v-model="receiveVisible" title="收货（质检登记）" width="760px">
      <el-table :data="receiveItems" border size="small">
        <el-table-column prop="drug_name" label="药品" min-width="140" />
        <el-table-column label="收货数量" width="140">
          <template #default="{ row }"
            ><el-input-number v-model="row.received_quantity" :min="0" size="small"
          /></template>
        </el-table-column>
        <el-table-column label="批号" width="130">
          <template #default="{ row }"><el-input v-model="row.batch_no" size="small" /></template>
        </el-table-column>
        <el-table-column label="效期" width="150">
          <template #default="{ row }"
            ><el-input v-model="row.expiry_date" placeholder="YYYY-MM-DD" size="small"
          /></template>
        </el-table-column>
        <el-table-column label="质检" width="120">
          <template #default="{ row }">
            <el-select v-model="row.qc_result" size="small"
              ><el-option label="合格" :value="1" /><el-option label="不合格" :value="2"
            /></el-select>
          </template>
        </el-table-column>
        <el-table-column label="质检备注" min-width="120">
          <template #default="{ row }"><el-input v-model="row.qc_notes" size="small" /></template>
        </el-table-column>
      </el-table>
      <div class="hint" style="margin-top: 8px">
        收货后进入待质检状态；每项质检结果必须登记（合格/不合格）后才能确认入库。
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

    <el-drawer
      v-model="receiptVisible"
      :title="`收货单 ${receiptDetail?.receipt_no ?? ''}`"
      size="720px"
    >
      <template v-if="receiptDetail">
        <el-descriptions :column="3" border size="small" style="margin-bottom: 12px">
          <el-descriptions-item label="状态">{{
            RECEIPT_LABEL[receiptDetail.status] ?? receiptDetail.status
          }}</el-descriptions-item>
          <el-descriptions-item label="采购单ID">{{
            receiptDetail.purchase_order_id
          }}</el-descriptions-item>
          <el-descriptions-item label="金额">
            <MoneyText :amount="receiptDetail.total_amount" />
          </el-descriptions-item>
        </el-descriptions>
        <el-table :data="receiptDetail.items ?? []" border size="small">
          <el-table-column prop="drug_id" label="药品ID" width="80" />
          <el-table-column prop="ordered_quantity" label="订购" width="70" />
          <el-table-column prop="received_quantity" label="实收" width="70" />
          <el-table-column prop="batch_no" label="批号" width="110" />
          <el-table-column label="效期" width="110">
            <template #default="{ row }">{{ String(row.expiry_date ?? '').slice(0, 10) }}</template>
          </el-table-column>
          <el-table-column label="单价" width="90">
            <template #default="{ row }"><MoneyText :amount="row.unit_price" /></template>
          </el-table-column>
          <el-table-column label="质检" width="90">
            <template #default="{ row }">
              <el-tag
                :type="row.qc_result === 1 ? 'success' : row.qc_result === 2 ? 'danger' : 'info'"
                size="small"
              >
                {{ row.qc_result === 1 ? '合格' : row.qc_result === 2 ? '不合格' : '未质检' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="qc_notes" label="质检备注" min-width="100" />
        </el-table>
      </template>
    </el-drawer>
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
  listReceipts,
  getReceipt,
  completeReceipt,
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
const RECEIPT_LABEL: Record<string, string> = {
  pending_quality: '待质检',
  received: '已入库',
  qc_failed: '质检不合格',
}
const RECEIPT_TAG: Record<string, string> = {
  pending_quality: 'warning',
  received: 'success',
  qc_failed: 'danger',
}

const tab = ref('orders')
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const receipts = ref<any[]>([])
const receiptTotal = ref(0)
const receiptPage = ref(1)
const loading = ref(false)
const suppliers = ref<any[]>([])
const dialogVisible = ref(false)
const receiveVisible = ref(false)
const suggestVisible = ref(false)
const suggestions = ref<any[]>([])
const receiveItems = ref<any[]>([])
const receivingOrderId = ref(0)
const receiptVisible = ref(false)
const receiptDetail = ref<any>(null)
const form = reactive<{ supplier_id: number | null; items: any[] }>({
  supplier_id: null,
  items: [],
})

onMounted(async () => {
  load()
  loadReceipts()
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
async function loadReceipts() {
  const res = await listReceipts({ page: receiptPage.value, page_size: 20 })
  receipts.value = res?.list ?? []
  receiptTotal.value = res?.total ?? 0
}
async function openReceipt(row: any) {
  receiptDetail.value = await getReceipt(row.id)
  receiptVisible.value = true
}
async function doCompleteReceipt(row: any) {
  await completeReceipt(row.id)
  ElMessage.success('已确认入库')
  loadReceipts()
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
    qc_notes: '',
  }))
  receiveVisible.value = true
}
async function doReceive() {
  for (const it of receiveItems.value) {
    if (it.received_quantity > 0 && (!it.batch_no || !it.expiry_date)) {
      ElMessage.warning('收货数量大于 0 的明细必须填写批号与效期')
      return
    }
  }
  // 效期转换为后端 time.Time 可解析的 RFC3339 格式
  const items = receiveItems.value.map((it) => ({
    ...it,
    expiry_date: it.expiry_date ? `${it.expiry_date}T00:00:00+08:00` : '',
  }))
  await receivePurchaseOrder(receivingOrderId.value, { items })
  ElMessage.success('已收货，待质检后确认入库')
  receiveVisible.value = false
  load()
  loadReceipts()
  tab.value = 'receipts'
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
.hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
