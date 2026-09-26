// 打印工具：新窗口输出精简单据（处方签/结算单），调用浏览器打印。
// 不污染主页面样式，各单据调用方自行组装 body HTML（金额已换算为元）。

export function yuanText(cents: number | null | undefined): string {
  return `¥${((Number(cents) || 0) / 100).toFixed(2)}`
}

function escapeHtml(s: unknown): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export { escapeHtml }

export function printDocument(title: string, bodyHtml: string): void {
  const w = window.open('', '_blank', 'width=800,height=600')
  if (!w) return
  w.document.write(`<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8" />
<title>${escapeHtml(title)}</title>
<style>
body { font-family: "SimSun", serif; color: #000; padding: 24px; }
h2 { text-align: center; margin: 0 0 8px; }
.meta { display: flex; flex-wrap: wrap; gap: 4px 24px; margin: 12px 0; font-size: 14px; }
table { width: 100%; border-collapse: collapse; font-size: 13px; margin-top: 8px; }
th, td { border: 1px solid #000; padding: 4px 8px; text-align: left; }
tfoot td { font-weight: bold; }
.footer { margin-top: 16px; font-size: 13px; display: flex; justify-content: space-between; }
@media print { body { padding: 0; } }
</style></head><body>${bodyHtml}
<script>window.onload = function () { window.print(); }</script>
</body></html>`)
  w.document.close()
}
