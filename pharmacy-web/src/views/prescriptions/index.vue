<template>
  <el-card>
    <div class="toolbar">
      <el-select v-model="query.status" placeholder="状态" clearable style="width: 130px">
        <el-option v-for="(v, k) in PRESCRIPTION_STATUS" :key="k" :label="v.label" :value="k" />
      </el-select>
      <el-select
        v-model="query.prescription_type"
        placeholder="类型"
        clearable
        style="width: 130px"
      >
        <el-option label="普通" :value="0" /><el-option label="麻醉" :value="1" /><el-option
          label="精神一类"
          :value="2"
        />
        <el-option label="精神二类" :value="3" /><el-option label="毒性" :value="4" /><el-option
          label="放射性"
          :value="5"
        />
      </el-select>
      <el-input
        v-model="query.patient_name"
        placeholder="患者姓名"
        clearable
        style="width: 160px"
        @keyup.enter="load"
      />
      <el-button type="primary" @click="load">查询</el-button>
      <el-button
        v-permission="'prescription:create'"
        type="success"
        @click="$router.push('/prescriptions/new')"
        >开方</el-button
      >
    </div>
    <el-table
      v-loading="loading"
      :data="list"
      border
      @row-click="(r: any) => $router.push(`/prescriptions/${r.id}`)"
    >
      <el-table-column prop="prescription_no" label="处方号" width="150" />
      <el-table-column prop="patient_name" label="患者" width="100" />
      <el-table-column prop="diagnosis" label="诊断" min-width="140" show-overflow-tooltip />
      <el-table-column label="类型" width="90">
        <template #default="{ row }">{{
          PRESCRIPTION_TYPE[row.prescription_type] ?? '普通'
        }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }"
          ><StatusTag :status="row.status" :map="PRESCRIPTION_STATUS"
        /></template>
      </el-table-column>
      <el-table-column label="金额" width="90">
        <template #default="{ row }"><MoneyText :amount="row.total_amount" /></template>
      </el-table-column>
      <el-table-column prop="created_at" label="时间" width="170" />
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
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { listPrescriptions } from '@/api/prescriptions'
import StatusTag from '@/components/StatusTag.vue'
import MoneyText from '@/components/MoneyText.vue'
import { PRESCRIPTION_STATUS } from '@/types/business'

const PRESCRIPTION_TYPE: Record<number, string> = {
  1: '麻醉',
  2: '精神一类',
  3: '精神二类',
  4: '毒性',
  5: '放射性',
}

const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const query = reactive({
  status: '',
  prescription_type: undefined as number | undefined,
  patient_name: '',
  page: 1,
  page_size: 20,
})

onMounted(load)
async function load() {
  loading.value = true
  try {
    const res = await listPrescriptions({ ...query, page: page.value })
    list.value = res?.list ?? []
    total.value = res?.total ?? 0
  } finally {
    loading.value = false
  }
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
