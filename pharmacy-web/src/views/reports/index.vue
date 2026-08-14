<template>
  <el-card>
    <div class="toolbar">
      <el-date-picker
        v-model="range"
        type="daterange"
        value-format="YYYY-MM-DD"
        start-placeholder="开始"
        end-placeholder="结束"
      />
      <el-button type="primary" @click="load">查询</el-button>
    </div>
    <el-tabs v-model="tab" @tab-change="load">
      <el-tab-pane label="进销存汇总" name="summary">
        <ChartPanel v-if="summaryOption" :option="summaryOption" height="320px" />
        <el-table :data="rows" border>
          <el-table-column prop="drug_name" label="药品" min-width="140" />
          <el-table-column prop="opening" label="期初(LDU)" width="110" />
          <el-table-column prop="inbound" label="入库" width="100" />
          <el-table-column prop="outbound" label="出库" width="100" />
          <el-table-column prop="closing" label="期末" width="100" />
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="效期分析" name="expiry">
        <ChartPanel v-if="expiryOption" :option="expiryOption" height="300px" />
        <el-table :data="rows" border>
          <el-table-column prop="drug_name" label="药品" min-width="140" />
          <el-table-column prop="batch_no" label="批号" width="120" />
          <el-table-column prop="expired" label="已过期" width="90" />
          <el-table-column prop="in_3m" label="3月内" width="90" />
          <el-table-column prop="in_6m" label="6月内" width="90" />
          <el-table-column prop="in_12m" label="12月内" width="90" />
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="特殊药品使用" name="special">
        <ChartPanel v-if="specialOption" :option="specialOption" height="300px" />
        <el-table :data="rows" border>
          <el-table-column prop="drug_name" label="药品" min-width="140" />
          <el-table-column prop="batch_no" label="批号" width="120" />
          <el-table-column prop="month" label="月份" width="100" />
          <el-table-column prop="dispense_qty" label="发药" width="90" />
          <el-table-column prop="return_qty" label="退回" width="90" />
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="调配工作量" name="workload">
        <ChartPanel v-if="workloadOption" :option="workloadOption" height="300px" />
        <el-table :data="rows" border>
          <el-table-column prop="dispensed_name" label="发药人" width="120" />
          <el-table-column prop="date" label="日期" width="120" />
          <el-table-column prop="prescription_count" label="处方数" width="100" />
          <el-table-column prop="item_count" label="件数" width="100" />
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="拆零统计" name="split">
        <ChartPanel v-if="splitOption" :option="splitOption" height="300px" />
        <el-table :data="rows" border>
          <el-table-column prop="drug_name" label="药品" min-width="140" />
          <el-table-column prop="split_boxes" label="拆零盒数" width="100" />
          <el-table-column prop="split_units" label="入片数" width="90" />
          <el-table-column prop="loss_units" label="损耗" width="90" />
          <el-table-column label="成本" width="100"
            ><template #default="{ row }"><MoneyText :amount="row.split_cost" /></template
          ></el-table-column>
          <el-table-column label="收入" width="100"
            ><template #default="{ row }"><MoneyText :amount="row.split_revenue" /></template
          ></el-table-column>
          <el-table-column label="毛利" width="100"
            ><template #default="{ row }"><MoneyText :amount="row.margin" /></template
          ></el-table-column>
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="患者费用" name="patient">
        <ChartPanel v-if="patientOption" :option="patientOption" height="300px" />
        <el-table :data="rows" border>
          <el-table-column prop="patient_name" label="患者" width="120" />
          <el-table-column label="类型" width="110">
            <template #default="{ row }">{{
              CHARGE_ITEM_TYPES[row.item_type] ?? row.item_type
            }}</template>
          </el-table-column>
          <el-table-column prop="charge_count" label="收费笔数" width="100" />
          <el-table-column prop="refund_count" label="冲正笔数" width="100" />
          <el-table-column label="净额" width="110"
            ><template #default="{ row }"><MoneyText :amount="row.amount" /></template
          ></el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  inventorySummary,
  expiryAnalysis,
  specialDrugUsage,
  dispensingWorkload,
  splitStatistics,
  patientCharges,
} from '@/api/reports'
import MoneyText from '@/components/MoneyText.vue'
import ChartPanel from '@/components/ChartPanel.vue'
import { CHARGE_ITEM_TYPES } from '@/types/business'
import type { EChartsOption } from 'echarts'

const tab = ref('summary')
const range = ref<[string, string] | null>(null)
const rows = ref<any[]>([])

// 进销存：Top10 药品入/出对比柱状图
const summaryOption = computed<EChartsOption | null>(() => {
  const top = rows.value.slice(0, 10)
  if (!top.length) return null
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: ['入库', '出库'] },
    grid: { left: 40, right: 16, top: 36, bottom: 24 },
    xAxis: { type: 'category', data: top.map((r) => r.drug_name) },
    yAxis: { type: 'value' },
    series: [
      { name: '入库', type: 'bar', data: top.map((r) => r.inbound) },
      { name: '出库', type: 'bar', data: top.map((r) => r.outbound) },
    ],
  }
})

// 效期分析：分档总量饼图
const expiryOption = computed<EChartsOption | null>(() => {
  const sum = (k: string) => rows.value.reduce((s, r) => s + Number(r[k] ?? 0), 0)
  const data = [
    { name: '已过期', value: sum('expired') },
    { name: '3月内', value: sum('in_3m') },
    { name: '6月内', value: sum('in_6m') },
    { name: '12月内', value: sum('in_12m') },
    { name: '12月以上', value: sum('after_12m') },
  ].filter((d) => d.value > 0)
  if (!data.length) return null
  return {
    tooltip: { trigger: 'item' },
    legend: { orient: 'vertical', left: 'left' },
    series: [{ type: 'pie', radius: '60%', data }],
  }
})

// 特殊药品使用：按月发药/退回趋势（聚合全部药品/批号）
const specialOption = computed<EChartsOption | null>(() => {
  const byMonth = new Map<string, { dispense: number; ret: number }>()
  for (const r of rows.value) {
    const m = String(r.month ?? '').slice(0, 7) || 'unknown'
    const cur = byMonth.get(m) ?? { dispense: 0, ret: 0 }
    cur.dispense += Number(r.dispense_qty ?? 0)
    cur.ret += Number(r.return_qty ?? 0)
    byMonth.set(m, cur)
  }
  const months = [...byMonth.keys()].sort()
  if (!months.length) return null
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: ['发药', '退回'] },
    grid: { left: 40, right: 16, top: 36, bottom: 24 },
    xAxis: { type: 'category', data: months },
    yAxis: { type: 'value' },
    series: [
      {
        name: '发药',
        type: 'bar',
        stack: 'qty',
        data: months.map((m) => byMonth.get(m)!.dispense),
      },
      { name: '退回', type: 'bar', stack: 'qty', data: months.map((m) => byMonth.get(m)!.ret) },
    ],
  }
})

// 调配工作量：发药人处方数/件数对比
const workloadOption = computed<EChartsOption | null>(() => {
  const byName = new Map<string, { rx: number; items: number }>()
  for (const r of rows.value) {
    const n = String(r.dispensed_name ?? '未知')
    const cur = byName.get(n) ?? { rx: 0, items: 0 }
    cur.rx += Number(r.prescription_count ?? 0)
    cur.items += Number(r.item_count ?? 0)
    byName.set(n, cur)
  }
  const names = [...byName.keys()]
  if (!names.length) return null
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: ['处方数', '件数'] },
    grid: { left: 40, right: 16, top: 36, bottom: 24 },
    xAxis: { type: 'category', data: names },
    yAxis: { type: 'value' },
    series: [
      { name: '处方数', type: 'bar', data: names.map((n) => byName.get(n)!.rx) },
      { name: '件数', type: 'bar', data: names.map((n) => byName.get(n)!.items) },
    ],
  }
})

// 拆零统计：Top10 药品毛利柱状图
const splitOption = computed<EChartsOption | null>(() => {
  const top = [...rows.value]
    .sort((a, b) => Number(b.margin ?? 0) - Number(a.margin ?? 0))
    .slice(0, 10)
  if (!top.length) return null
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 16, top: 36, bottom: 24 },
    xAxis: { type: 'category', data: top.map((r) => r.drug_name) },
    yAxis: { type: 'value' },
    series: [{ name: '毛利(分)', type: 'bar', data: top.map((r) => Number(r.margin ?? 0)) }],
  }
})

// 患者费用：费用类型分布饼图（净额）
const patientOption = computed<EChartsOption | null>(() => {
  const byType = new Map<string, number>()
  for (const r of rows.value) {
    const t = CHARGE_ITEM_TYPES[r.item_type] ?? String(r.item_type ?? '其他')
    byType.set(t, (byType.get(t) ?? 0) + Number(r.amount ?? 0))
  }
  const data = [...byType.entries()]
    .map(([name, value]) => ({ name, value }))
    .filter((d) => d.value !== 0)
  if (!data.length) return null
  return {
    tooltip: { trigger: 'item', valueFormatter: (v) => `¥${(Number(v) / 100).toFixed(2)}` },
    legend: { orient: 'vertical', left: 'left' },
    series: [{ type: 'pie', radius: '60%', data }],
  }
})

async function load() {
  const params = range.value
    ? { start: `${range.value[0]}T00:00:00+08:00`, end: `${range.value[1]}T23:59:59+08:00` }
    : {}
  switch (tab.value) {
    case 'summary':
      rows.value = (await inventorySummary(params)) ?? []
      break
    case 'expiry':
      rows.value = (await expiryAnalysis()) ?? []
      break
    case 'special':
      rows.value = (await specialDrugUsage(params)) ?? []
      break
    case 'workload':
      rows.value = (await dispensingWorkload(params)) ?? []
      break
    case 'split':
      rows.value = (await splitStatistics(params)) ?? []
      break
    case 'patient':
      rows.value = (await patientCharges(params)) ?? []
      break
  }
}
load()
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
</style>
