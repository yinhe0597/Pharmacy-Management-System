<template>
  <div>
    <el-card>
      <div class="toolbar">
        <el-select v-model="query.status" placeholder="状态" clearable style="width: 130px">
          <el-option v-for="(v, k) in CHARGE_STATUS" :key="k" :label="v.label" :value="k" />
        </el-select>
        <PatientPicker v-model="query.patient_id" placeholder="患者" style="width: 200px" />
        <el-date-picker
          v-model="range"
          type="daterange"
          value-format="YYYY-MM-DD"
          start-placeholder="开始"
          end-placeholder="结束"
        />
        <el-button type="primary" @click="load">查询</el-button>
      </div>
      <el-table v-loading="loading" :data="list" border @row-click="openDetail">
        <el-table-column prop="charge_no" label="结算单号" width="160" />
        <el-table-column prop="patient_name" label="患者" width="110" />
        <el-table-column label="合计" width="110">
          <template #default="{ row }"><MoneyText :amount="row.total_amount" /></template>
        </el-table-column>
        <el-table-column label="优惠" width="100">
          <template #default="{ row }"><MoneyText :amount="row.discount_amount" /></template>
        </el-table-column>
        <el-table-column label="应收" width="110">
          <template #default="{ row }"><MoneyText :amount="row.payable_amount" /></template>
        </el-table-column>
        <el-table-column label="实收" width="110">
          <template #default="{ row }"><MoneyText :amount="row.paid_amount" /></template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="CHARGE_STATUS[row.status]?.tag ?? 'info'" size="small">{{
              CHARGE_STATUS[row.status]?.label ?? row.status
            }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="operator_name" label="操作人" width="100" />
        <el-table-column prop="created_at" label="创建时间" width="170" />
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'pending'"
              v-permission="'charge:write'"
              link
              type="primary"
              @click.stop="pay(row)"
              >收费</el-button
            >
            <el-button
              v-if="row.status === 'paid'"
              v-permission="'charge:write'"
              link
              type="danger"
              @click.stop="refund(row)"
              >退费</el-button
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

    <el-drawer v-model="detailVisible" title="结算单详情" size="560px">
      <template v-if="detail">
        <div class="toolbar">
          <el-button size="small" @click="printCharge">打印结算单</el-button>
        </div>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="结算单号">{{ detail.charge_no }}</el-descriptions-item>
          <el-descriptions-item label="患者"
            >{{ detail.patient_name }}（ID {{ detail.patient_id }}）</el-descriptions-item
          >
          <el-descriptions-item label="状态">
            <el-tag :type="CHARGE_STATUS[detail.status]?.tag ?? 'info'" size="small">{{
              CHARGE_STATUS[detail.status]?.label ?? detail.status
            }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>
        <h4>费用明细</h4>
        <el-table :data="detail.items ?? []" border size="small">
          <el-table-column label="类型" width="100">
            <template #default="{ row }">{{
              CHARGE_ITEM_TYPES[row.item_type] ?? row.item_type
            }}</template>
          </el-table-column>
          <el-table-column prop="item_name" label="项目" min-width="140" />
          <el-table-column prop="quantity" label="数量" width="60" />
          <el-table-column label="单价" width="90">
            <template #default="{ row }"><MoneyText :amount="row.unit_price" /></template>
          </el-table-column>
          <el-table-column label="金额" width="90">
            <template #default="{ row }"><MoneyText :amount="row.amount" /></template>
          </el-table-column>
        </el-table>
        <el-descriptions :column="1" border class="summary">
          <el-descriptions-item label="合计">
            <MoneyText :amount="detail.total_amount" />
          </el-descriptions-item>
          <el-descriptions-item label="应收">
            <MoneyText :amount="detail.payable_amount" />
          </el-descriptions-item>
        </el-descriptions>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listCharges, getCharge, payCharge, refundCharge } from '@/api/clinical2'
import MoneyText from '@/components/MoneyText.vue'
import PatientPicker from '@/components/PatientPicker.vue'
import { printDocument, yuanText, escapeHtml } from '@/utils/print'
import { CHARGE_ITEM_TYPES } from '@/types/business'

const CHARGE_STATUS: Record<string, { label: string; tag: string }> = {
  pending: { label: '待收', tag: 'warning' },
  paid: { label: '已收', tag: 'success' },
  refunded: { label: '已退', tag: 'info' },
}

const list = ref<any[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({
  status: '',
  patient_id: undefined as number | undefined,
  page: 1,
  page_size: 20,
})
const range = ref<[string, string] | null>(null)
const detailVisible = ref(false)
const detail = ref<any>(null)

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
    const r = await listCharges(params)
    list.value = r?.list ?? []
    total.value = r?.total ?? 0
  } finally {
    loading.value = false
  }
}

async function openDetail(row: any) {
  detail.value = await getCharge(row.id)
  detailVisible.value = true
}

async function pay(row: any) {
  const { value } = await ElMessageBox.prompt('实收金额（分）', '收费', {
    inputValue: String(row.payable_amount ?? 0),
    inputPattern: /^\d+$/,
    inputErrorMessage: '请输入整数金额（分）',
  })
  await payCharge(row.id, { paid_amount: Number(value) })
  ElMessage.success('收费成功')
  load()
}

async function refund(row: any) {
  await ElMessageBox.confirm(`确认退费 ${row.patient_name} 的结算单？`, '退费')
  await refundCharge(row.id)
  ElMessage.success('已退费')
  load()
}

function printCharge() {
  const d = detail.value
  if (!d) return
  const rows = (d.items ?? [])
    .map(
      (it: any, i: number) =>
        `<tr><td>${i + 1}</td><td>${escapeHtml(CHARGE_ITEM_TYPES[it.item_type] ?? it.item_type)}</td>` +
        `<td>${escapeHtml(it.item_name)}</td><td>${escapeHtml(it.quantity)}</td>` +
        `<td>${yuanText(it.unit_price)}</td><td>${yuanText(it.amount)}</td></tr>`,
    )
    .join('')
  printDocument(
    `结算单 ${d.charge_no ?? ''}`,
    `<h2>收费结算单</h2>
<div class="meta"><span>结算单号：${escapeHtml(d.charge_no)}</span>` +
      `<span>患者：${escapeHtml(d.patient_name)}</span>` +
      `<span>状态：${escapeHtml(CHARGE_STATUS[d.status]?.label ?? d.status)}</span></div>
<table><thead><tr><th>#</th><th>类型</th><th>项目</th><th>数量</th><th>单价</th><th>金额</th></tr></thead>
<tbody>${rows}</tbody>
<tfoot><tr><td colspan="5">合计</td><td>${yuanText(d.total_amount)}</td></tr>` +
      `<tr><td colspan="5">应收</td><td>${yuanText(d.payable_amount)}</td></tr></tfoot></table>
<div class="footer"><span>打印时间：${new Date().toLocaleString('zh-CN')}</span><span>收费员签字：________</span></div>`,
  )
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
.summary {
  margin-top: 12px;
}
</style>
