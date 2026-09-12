import { describe, expect, it } from 'vitest'
import { formatCents, formatYuan } from './money'

describe('formatCents', () => {
  it('整元金额', () => {
    expect(formatCents(1200)).toBe('12.00')
  })

  it('分位补零', () => {
    expect(formatCents(1005)).toBe('10.05')
  })

  it('零与负数', () => {
    expect(formatCents(0)).toBe('0.00')
    expect(formatCents(-1200)).toBe('-12.00')
    expect(formatCents(-5)).toBe('-0.05')
  })

  it('null/undefined 视为 0', () => {
    expect(formatCents(null)).toBe('0.00')
    expect(formatCents(undefined)).toBe('0.00')
  })
})

describe('formatYuan', () => {
  it('带 ¥ 前缀', () => {
    expect(formatYuan(1234)).toBe('¥12.34')
  })
})
