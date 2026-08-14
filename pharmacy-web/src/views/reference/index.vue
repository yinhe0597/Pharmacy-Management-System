<template>
  <el-card>
    <el-tabs v-model="tab">
      <el-tab-pane label="ICD-10 诊断" name="diagnosis">
        <el-input
          v-model="keyword"
          placeholder="编码/名称/拼音码"
          clearable
          style="width: 240px; margin-bottom: 8px"
          @keyup.enter="load"
        />
        <el-table :data="rows" border>
          <el-table-column prop="code" label="编码" width="100" />
          <el-table-column prop="disease_name" label="诊断名称" min-width="160" />
          <el-table-column prop="chapter_name" label="章节" width="140" />
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="医保目录" name="nhsa">
        <el-input
          v-model="keyword"
          placeholder="药名/拼音码"
          clearable
          style="width: 240px; margin-bottom: 8px"
          @keyup.enter="load"
        />
        <el-table :data="rows" border>
          <el-table-column prop="drug_name" label="药品名称" min-width="160" />
          <el-table-column prop="dosage_form" label="剂型" width="120" />
          <el-table-column prop="insurance_class" label="类别" width="80" />
          <el-table-column prop="drug_category" label="分类" min-width="160" />
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="集采目录" name="vbp">
        <el-input
          v-model="keyword"
          placeholder="通用名/拼音码"
          clearable
          style="width: 240px; margin-bottom: 8px"
          @keyup.enter="load"
        />
        <el-table :data="rows" border>
          <el-table-column prop="generic_name" label="通用名" min-width="160" />
          <el-table-column prop="dosage_form" label="剂型" width="120" />
          <el-table-column prop="vbp_batch" label="集采批次" width="100" />
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="耗材目录" name="consumable">
        <el-input
          v-model="keyword"
          placeholder="名称/拼音码/分类"
          clearable
          style="width: 240px; margin-bottom: 8px"
          @keyup.enter="load"
        />
        <el-table :data="rows" border>
          <el-table-column prop="item_name" label="耗材名称" min-width="160" />
          <el-table-column prop="category" label="分类" width="120" />
          <el-table-column prop="nmpa_class" label="NMPA 类别" width="110" />
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="非医保药品" name="noninsurance">
        <el-input
          v-model="keyword"
          placeholder="药名/拼音码/分类"
          clearable
          style="width: 240px; margin-bottom: 8px"
          @keyup.enter="load"
        />
        <el-table :data="rows" border>
          <el-table-column prop="drug_name" label="药品名称" min-width="160" />
          <el-table-column prop="rx_otc_class" label="处方/OTC" width="100" />
          <el-table-column prop="category" label="分类" min-width="140" />
        </el-table>
      </el-tab-pane>
    </el-tabs>
    <el-pagination
      v-model:current-page="page"
      class="pager"
      layout="total, prev, pager, next"
      :total="total"
      :page-size="20"
      @current-change="load"
    />
  </el-card>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  listDiagnosisCodes,
  listNHSADrugs,
  listVBPDrugs,
  listConsumables,
  listNonInsuranceDrugs,
} from '@/api/reference'

const tab = ref('diagnosis')
const keyword = ref('')
const rows = ref<any[]>([])
const total = ref(0)
const page = ref(1)

watch(tab, () => {
  page.value = 1
  load()
})

async function load() {
  const apiMap: Record<string, (p: Record<string, unknown>) => Promise<any>> = {
    diagnosis: listDiagnosisCodes,
    nhsa: listNHSADrugs,
    vbp: listVBPDrugs,
    consumable: listConsumables,
    noninsurance: listNonInsuranceDrugs,
  }
  const api = apiMap[tab.value]
  const res = await api({ keyword: keyword.value, page: page.value, page_size: 20 })
  rows.value = res?.list ?? []
  total.value = res?.total ?? 0
}
load()
</script>

<style scoped>
.pager {
  margin-top: 12px;
  justify-content: flex-end;
}
</style>
