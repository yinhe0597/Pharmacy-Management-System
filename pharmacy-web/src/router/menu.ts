import type { Permission } from '@/types/business'

export interface MenuItem {
  path: string
  title: string
  icon: string
  permission?: Permission
}

// 主菜单（按角色权限收敛，docs/16 §三、docs/18）
export const MENU: MenuItem[] = [
  { path: '/dashboard', title: '工作台', icon: 'Odometer' },
  { path: '/patients', title: '患者管理', icon: 'User', permission: 'patient:read' },
  { path: '/prescriptions', title: '处方管理', icon: 'Document' },
  { path: '/drugs', title: '药品主数据', icon: 'FirstAidKit' },
  { path: '/inventory', title: '库存管理', icon: 'Box' },
  { path: '/purchase', title: '采购管理', icon: 'ShoppingCart', permission: 'purchase:write' },
  { path: '/billing', title: '计费管理', icon: 'Money', permission: 'billing:view' },
  { path: '/reports', title: '报表中心', icon: 'DataAnalysis', permission: 'report:view' },
  { path: '/admin', title: '系统管理', icon: 'Setting', permission: 'user:admin' },
]
