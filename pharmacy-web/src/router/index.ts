import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { hasPermission, type Permission, type Role } from '@/types/business'
import { cancelAllRequests } from '@/api/http'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录', public: true },
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '工作台' },
      },
      {
        path: 'patients',
        name: 'patients',
        component: () => import('@/views/patients/index.vue'),
        meta: { title: '患者管理', permission: 'patient:read' as Permission },
      },
      {
        path: 'prescriptions',
        name: 'prescriptions',
        component: () => import('@/views/prescriptions/index.vue'),
        meta: { title: '处方管理' },
      },
      {
        path: 'prescriptions/new',
        name: 'prescription-create',
        component: () => import('@/views/prescriptions/create.vue'),
        meta: { title: '开方', permission: 'prescription:create' as Permission },
      },
      {
        path: 'prescriptions/:id',
        name: 'prescription-detail',
        component: () => import('@/views/prescriptions/detail.vue'),
        meta: { title: '处方详情' },
      },
      {
        path: 'drugs',
        name: 'drugs',
        component: () => import('@/views/drugs/index.vue'),
        meta: { title: '药品主数据' },
      },
      {
        path: 'categories',
        name: 'categories',
        component: () => import('@/views/categories/index.vue'),
        meta: { title: '分类管理', permission: 'drug:write' as Permission },
      },
      {
        path: 'suppliers',
        name: 'suppliers',
        component: () => import('@/views/suppliers/index.vue'),
        meta: { title: '供应商', permission: 'purchase:write' as Permission },
      },
      {
        path: 'reference',
        name: 'reference',
        component: () => import('@/views/reference/index.vue'),
        meta: { title: '参考数据' },
      },
      {
        path: 'inventory',
        name: 'inventory',
        component: () => import('@/views/inventory/index.vue'),
        meta: { title: '库存管理' },
      },
      {
        path: 'purchase',
        name: 'purchase',
        component: () => import('@/views/purchase/index.vue'),
        meta: { title: '采购管理', permission: 'purchase:write' as Permission },
      },
      {
        path: 'billing',
        name: 'billing',
        component: () => import('@/views/billing/index.vue'),
        meta: { title: '计费管理', permission: 'billing:view' as Permission },
      },
      {
        path: 'pharma',
        name: 'pharma',
        component: () => import('@/views/pharma/index.vue'),
        meta: { title: '药学服务', permission: 'inventory:write' as Permission },
      },
      {
        path: 'special',
        name: 'special',
        component: () => import('@/views/specialdrugs/index.vue'),
        meta: { title: '特殊药品', permission: 'drug:write' as Permission },
      },
      {
        path: 'reports',
        name: 'reports',
        component: () => import('@/views/reports/index.vue'),
        meta: { title: '报表中心', permission: 'report:view' as Permission },
      },
      {
        path: 'admin',
        name: 'admin',
        component: () => import('@/views/admin/index.vue'),
        meta: { title: '系统管理', permission: 'user:admin' as Permission },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const store = useUserStore()
  const isPublic = to.meta.public === true

  // 路由切换取消未完成请求（docs/19）
  cancelAllRequests()

  if (isPublic) {
    if (store.token && to.path === '/login') return '/'
    return true
  }
  if (!store.token) return { path: '/login', query: { redirect: to.fullPath } }

  const required = to.meta.permission as Permission | undefined
  if (required) {
    const role = store.role as Role
    if (!role || !hasPermission(role, required)) return '/dashboard'
  }
  return true
})

export default router
