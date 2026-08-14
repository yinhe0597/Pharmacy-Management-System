// 业务枚举/常量（与后端 internal/domain/enum 对齐，docs/03 §2、docs/18）

export type Role =
  | 'admin'
  | 'pharmacy_director'
  | 'pharmacist'
  | 'doctor'
  | 'clinic_nurse'
  | 'pharmacy_nurse'
  | 'buyer'
  | 'finance'

export const ROLE_LABELS: Record<Role, string> = {
  admin: '管理员',
  pharmacy_director: '药房主任',
  pharmacist: '药师',
  doctor: '医生',
  clinic_nurse: '跟诊护士',
  pharmacy_nurse: '药房护士',
  buyer: '采购员',
  finance: '财务',
}

// 权限点（对应后端路由分组）
export type Permission =
  | 'user:admin' // UserAdmin：用户管理
  | 'drug:write' // DrugAdmin：药品/分类/配伍/交互规则/特殊药品目录写
  | 'purchase:write' // Purchase：采购/供应商写
  | 'inventory:write' // Pharmacy：库存写/领用补发/处方执行
  | 'prescription:create' // Clinical：处方开立/医嘱核对
  | 'prescription:review' // 药师审核（pharmacist/director/admin）
  | 'patient:write' // PatientAdmin：患者档案写
  | 'patient:read' // PatientRead：患者档案读
  | 'charge:write' // ChargeStaff：计费录入/红冲
  | 'billing:view' // Billing：计费查看
  | 'report:view' // Report：报表

// 角色 → 权限点集合（与后端 enum.RoleGroups 对齐）
export const ROLE_PERMISSIONS: Record<Role, (Permission | '*')[]> = {
  admin: ['*'],
  pharmacy_director: [
    'user:admin',
    'drug:write',
    'purchase:write',
    'inventory:write',
    'prescription:create',
    'prescription:review',
    'patient:write',
    'patient:read',
    'charge:write',
    'billing:view',
    'report:view',
  ],
  pharmacist: [
    'drug:write',
    'inventory:write',
    'prescription:create',
    'prescription:review',
    'patient:read',
    'charge:write',
    'billing:view',
    'report:view',
  ],
  doctor: [
    'inventory:write',
    'prescription:create',
    'patient:write',
    'patient:read',
    'charge:write',
    'billing:view',
  ],
  clinic_nurse: [
    'prescription:create',
    'patient:write',
    'patient:read',
    'charge:write',
    'billing:view',
  ],
  pharmacy_nurse: ['inventory:write', 'patient:read', 'charge:write', 'billing:view'],
  buyer: ['purchase:write'],
  finance: ['report:view', 'billing:view'],
}

export function hasPermission(role: Role, permission: Permission): boolean {
  const perms = ROLE_PERMISSIONS[role]
  if (!perms) return false
  return perms.includes('*') || perms.includes(permission)
}

// 处方状态（domain/prescription）
export const PRESCRIPTION_STATUS: Record<string, { label: string; tag: string }> = {
  pending_review: { label: '待审核', tag: 'warning' },
  reviewed_passed: { label: '审核通过', tag: 'primary' },
  reviewed_rejected: { label: '审核驳回', tag: 'danger' },
  dispensing: { label: '调配中', tag: 'primary' },
  dispensed: { label: '已发药', tag: 'success' },
  returned: { label: '已退药', tag: 'info' },
  cancelled: { label: '已作废', tag: 'info' },
}

// 计费项目类型
export const CHARGE_ITEM_TYPES: Record<string, string> = {
  drug: '药品',
  consumable: '耗材',
  clinical_service: '诊疗项目',
}
