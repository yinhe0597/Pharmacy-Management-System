import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { hasPermission, type Permission, type Role } from '@/types/business'

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
      { path: 'dashboard', name: 'dashboard', component: () => import('@/views/dashboard/index.vue'), meta: { title: '工作台' } },
      { path: 'patients', name: 'patients', component: () => import('@/views/Placeholder.vue'), meta: { title: '患者管理', permission: 'patient:read' as Permission } },
      { path: 'prescriptions', name: 'prescriptions', component: () => import('@/views/Placeholder.vue'), meta: { title: '处方管理' } },
      { path: 'drugs', name: 'drugs', component: () => import('@/views/Placeholder.vue'), meta: { title: '药品主数据' } },
      { path: 'inventory', name: 'inventory', component: () => import('@/views/Placeholder.vue'), meta: { title: '库存管理' } },
      { path: 'purchase', name: 'purchase', component: () => import('@/views/Placeholder.vue'), meta: { title: '采购管理', permission: 'purchase:write' as Permission } },
      { path: 'billing', name: 'billing', component: () => import('@/views/Placeholder.vue'), meta: { title: '计费管理', permission: 'billing:view' as Permission } },
      { path: 'reports', name: 'reports', component: () => import('@/views/Placeholder.vue'), meta: { title: '报表中心', permission: 'report:view' as Permission } },
      { path: 'admin', name: 'admin', component: () => import('@/views/Placeholder.vue'), meta: { title: '系统管理', permission: 'user:admin' as Permission } },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫：登录校验 + 角色权限
router.beforeEach((to) => {
  const store = useUserStore()
  const isPublic = to.meta.public === true

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
