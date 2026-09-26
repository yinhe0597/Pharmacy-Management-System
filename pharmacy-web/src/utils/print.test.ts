import { describe, expect, it } from 'vitest'
import { yuanText, escapeHtml } from './print'

describe('print utils', () => {
  it('yuanText 分转元保留两位', () => {
    expect(yuanText(0)).toBe('¥0.00')
    expect(yuanText(1)).toBe('¥0.01')
    expect(yuanText(1234)).toBe('¥12.34')
    expect(yuanText(null)).toBe('¥0.00')
    expect(yuanText(undefined)).toBe('¥0.00')
  })

  it('escapeHtml 转义标签字符', () => {
    expect(escapeHtml('<b>&"')).toBe('&lt;b&gt;&amp;&quot;')
    expect(escapeHtml(null)).toBe('')
    expect(escapeHtml(123)).toBe('123')
  })
})
