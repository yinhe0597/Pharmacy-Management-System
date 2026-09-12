import { describe, expect, it } from 'vitest'
import { hasPermission, type Role } from './business'

describe('hasPermission', () => {
  it('admin 通配符拥有任意权限', () => {
    expect(hasPermission('admin', 'drug:write')).toBe(true)
    expect(hasPermission('admin', 'report:view')).toBe(true)
  })

  it('药师拥有 drug:write 但不拥有 user:admin', () => {
    expect(hasPermission('pharmacist', 'drug:write')).toBe(true)
    expect(hasPermission('pharmacist', 'user:admin')).toBe(false)
  })

  it('采购员仅拥有 purchase:write', () => {
    expect(hasPermission('buyer', 'purchase:write')).toBe(true)
    expect(hasPermission('buyer', 'patient:read')).toBe(false)
  })

  it('医生不拥有 report:view', () => {
    expect(hasPermission('doctor', 'report:view')).toBe(false)
  })

  it('未知角色返回 false', () => {
    const unknown = 'ghost' as unknown as Role
    expect(hasPermission(unknown, 'drug:write')).toBe(false)
  })
})
