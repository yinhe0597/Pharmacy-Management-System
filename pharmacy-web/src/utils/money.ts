// 金额单位为「分」（int64），仅展示层格式化（docs/16 §5）

/** 分 → 元字符串（如 1200 → "12.00"） */
export function formatCents(cents: number | null | undefined): string {
  const n = cents ?? 0
  const sign = n < 0 ? '-' : ''
  const abs = Math.abs(n)
  const yuan = Math.floor(abs / 100)
  const fen = abs % 100
  return `${sign}${yuan}.${fen.toString().padStart(2, '0')}`
}

/** 分 → 带 ¥ 符号 */
export function formatYuan(cents: number | null | undefined): string {
  return `¥${formatCents(cents)}`
}
