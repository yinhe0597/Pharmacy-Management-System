<template>
  <div>
    <el-card class="welcome">
      <div class="welcome-row">
        <div>
          <h3>👋 {{ greeting }}，{{ store.user?.name }}</h3>
          <p class="tip">
            {{ todayText }} · 角色：<el-tag size="small">{{ store.roleLabel }}</el-tag>
          </p>
        </div>
      </div>
    </el-card>

    <el-row :gutter="16" class="cards">
      <el-col v-for="card in statCards" :key="card.title" :span="6">
        <el-card class="stat-card" shadow="hover" @click="card.to && $router.push(card.to)">
          <div class="stat-value" :style="{ color: card.color }">{{ card.value ?? '—' }}</div>
          <div class="stat-title">{{ card.title }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card header="快捷入口" class="shortcuts">
      <el-button v-for="s in quickActions" :key="s.to" size="large" @click="$router.push(s.to)">
        {{ s.title }}
      </el-button>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useUserStore } from '@/stores/user'
import { hasPermission, type Permission, type Role } from '@/types/business'
import { listPrescriptions } from '@/api/prescriptions'
import { listAlerts, listExpiryWarnings } from '@/api/inventory'
import { listVisits } from '@/api/clinical2'

const store = useUserStore()
const pendingReview = ref<number | null>(null)
const dispensing = ref<number | null>(null)
const visiting = ref<number | null>(null)
const stockAlerts = ref<number | null>(null)
const expiryAlerts = ref<number | null>(null)

const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return '凌晨好'
  if (h < 12) return '上午好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})
const todayText = computed(() =>
  new Date().toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'long',
  }),
)

const canSeePrescription = computed(
  () => store.role !== '' && hasPermission(store.role as Role, 'prescription:create' as Permission),
)
const canSeePatient = computed(
  () => store.role !== '' && hasPermission(store.role as Role, 'patient:read' as Permission),
)

const statCards = computed(() => {
  const cards: { title: string; value: number | null; color: string; to?: string }[] = []
  if (canSeePrescription.value) {
    cards.push({
      title: '待审核处方',
      value: pendingReview.value,
      color: '#e6a23c',
      to: '/prescriptions',
    })
    cards.push({
      title: '调配中处方',
      value: dispensing.value,
      color: '#409eff',
      to: '/prescriptions',
    })
  }
  if (canSeePatient.value) {
    cards.push({ title: '就诊中患者', value: visiting.value, color: '#67c23a', to: '/visits' })
  }
  cards.push({
    title: '库存下限预警',
    value: stockAlerts.value,
    color: '#f56c6c',
    to: '/inventory',
  })
  cards.push({ title: '近效期批次', value: expiryAlerts.value, color: '#e6a23c', to: '/inventory' })
  return cards.slice(0, 4)
})

const quickActions = computed(() => {
  const acts: { title: string; to: string }[] = []
  if (canSeePrescription.value) acts.push({ title: '📝 开处方', to: '/prescriptions/new' })
  if (canSeePatient.value) acts.push({ title: '🩺 就诊登记', to: '/visits' })
  if (canSeePatient.value) acts.push({ title: '👥 患者管理', to: '/patients' })
  acts.push({ title: '📦 库存管理', to: '/inventory' })
  if (canSeePatient.value) acts.push({ title: '💰 收费台', to: '/charges' })
  if (store.role !== '' && hasPermission(store.role as Role, 'purchase:write' as Permission))
    acts.push({ title: '🛒 采购', to: '/purchase' })
  if (store.role !== '' && hasPermission(store.role as Role, 'report:view' as Permission))
    acts.push({ title: '📊 报表', to: '/reports' })
  return acts
})

onMounted(async () => {
  const loads: Promise<void>[] = []
  if (canSeePrescription.value) {
    loads.push(
      listPrescriptions({ status: 'pending_review', page: 1, page_size: 1 }).then(
        (r) => (pendingReview.value = r?.total ?? 0),
      ),
      listPrescriptions({ status: 'dispensing', page: 1, page_size: 1 }).then(
        (r) => (dispensing.value = r?.total ?? 0),
      ),
    )
  }
  if (canSeePatient.value) {
    loads.push(
      listVisits({ status: 'visiting', page: 1, page_size: 1 }).then(
        (r) => (visiting.value = r?.total ?? 0),
      ),
    )
  }
  loads.push(
    listAlerts({ page: 1, page_size: 1 }).then((r) => (stockAlerts.value = r?.total ?? 0)),
    listExpiryWarnings({ page: 1, page_size: 1 }).then((r) => (expiryAlerts.value = r?.total ?? 0)),
  )
  await Promise.allSettled(loads)
})
</script>

<style scoped>
.welcome {
  margin-bottom: 16px;
}
.welcome-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.welcome h3 {
  margin: 0 0 6px;
}
.tip {
  color: #909399;
  font-size: 13px;
  margin: 0;
}
.cards {
  margin-bottom: 16px;
}
.stat-card {
  cursor: pointer;
  text-align: center;
}
.stat-value {
  font-size: 30px;
  font-weight: 700;
  line-height: 1.4;
}
.stat-title {
  color: #909399;
  font-size: 13px;
}
.shortcuts :deep(.el-button) {
  margin: 0 12px 12px 0;
}
</style>
