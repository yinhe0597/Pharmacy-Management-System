import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join, relative, resolve } from 'node:path'

/**
 * 前后端字段契约守卫。
 *
 * 背景：第六轮审计发现 6 处前后端字段契约断裂——后端从未返回该字段，
 * 前端却按它渲染，结果是整列空白或字段显示 undefined：
 *   - PurchaseOrder.order_no   → 后端是 purchase_no
 *   - Inventory.drug_name / location_name → 仓储层未 JOIN，需行 DTO
 *   - PurchaseOrderItem.drug_name / supplier_name → 后端未落列
 *   - SplitOrder.units_in → 后端是 units（曾致入库数量存 0）
 *   - AdverseReaction.drug_name / reaction → 后端是 reaction_desc
 * 根因是 types/entities.ts 手写、后端不校验，TS 类型给出「已对齐」的虚假保证。
 *
 * 契约来源取 internal/ 下所有 Go struct 的 json tag（而非 docs/swagger.json）：
 * 多数 handler 走泛型 Body 返回，swag 不会为其生成 definition，用 swagger
 * 会产生 61 个误报；Go tag 才是接口真实返回字段的定义处。
 *
 * 已知局限：这里只校验「字段名在后端任意 DTO 中存在」，不校验「存在于该
 * 端点返回的那个 DTO 中」。预警列表最初正是靠这层局限漏掉 drug_name /
 * location_name（它们在 InventoryRow 里存在），已由
 * internal/service/alert_row_integration_test.go 在后端侧补守卫。
 */

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const goRoot = resolve(webRoot, '..')

describe('前后端字段契约', () => {
  it('后端源码目录可访问（守卫自身）', () => {
    // 没有这层，CI 若只 checkout 了 pharmacy-web/ 子目录会抛 ENOENT，报错难以定位
    const goInternal = join(goRoot, 'internal')
    expect(
      statSync(goInternal).isDirectory(),
      `未找到后端源码目录 ${goInternal}：本用例需在仓库内运行（依赖 ../internal 的 struct json tag）`,
    ).toBe(true)
  })
})

/** 收集 internal/ 下所有 Go struct 的 json tag 字段名。 */
function collectContractFields(): Set<string> {
  const fields = new Set<string>()
  const walk = (dir: string) => {
    for (const entry of readdirSync(dir)) {
      const full = join(dir, entry)
      if (statSync(full).isDirectory()) {
        walk(full)
      } else if (full.endsWith('.go') && !full.endsWith('_test.go')) {
        const src = readFileSync(full, 'utf8')
        for (const m of src.matchAll(/json:"([a-zA-Z_0-9]+)(?:,[^"]*)?"/g)) {
          fields.add(m[1])
        }
      }
    }
  }
  walk(join(goRoot, 'internal'))
  return fields
}

/** 收集所有 .vue 文件路径。 */
function collectVueFiles(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    if (statSync(full).isDirectory()) collectVueFiles(full, out)
    else if (full.endsWith('.vue')) out.push(full)
  }
  return out
}

/**
 * 模板中静态绑定的表格列 prop。
 * 排除 :prop（动态绑定，如 v-for 里的 :prop="col"），只校验字面量列。
 */
function collectTemplateProps(): Map<string, Set<string>> {
  const usage = new Map<string, Set<string>>()
  for (const file of collectVueFiles(join(webRoot, 'src'))) {
    const src = readFileSync(file, 'utf8')
    const rel = relative(webRoot, file)
    for (const m of src.matchAll(/<el-table-column(?!:)[^>]*?\sprop="([a-zA-Z_0-9]+)"/g)) {
      const prop = m[1]
      if (!usage.has(prop)) usage.set(prop, new Set())
      usage.get(prop)!.add(rel)
    }
  }
  return usage
}

const contract = collectContractFields()
const templateProps = collectTemplateProps()

describe('前后端字段契约', () => {
  it('契约字段集非空（解析器未失效）', () => {
    // 守卫自身：若将来正则失效导致集合为空，下面的用例会全部假通过
    expect(contract.size).toBeGreaterThan(200)
  })

  it('模板采集到足够的列（守卫自身）', () => {
    expect(templateProps.size).toBeGreaterThan(100)
  })

  it('表格列 prop 均存在于后端字段契约中', () => {
    const missing: string[] = []
    for (const [prop, files] of [...templateProps].sort()) {
      if (!contract.has(prop)) {
        missing.push(`${prop}（${[...files].join('、')}）`)
      }
    }
    expect(
      missing,
      `以下列在后端没有任何 struct 返回，整列必然空白：\n  ${missing.join('\n  ')}`,
    ).toEqual([])
  })
})

/**
 * 源码编码守卫。
 *
 * 背景：src/api/prescriptions.ts 与 src/api/purchase.ts 的中文注释是 **GBK 字节**，
 * 混在一个 UTF-8 仓库里。后果是连锁的：
 *   1. 任何 UTF-8 工具读它们都会得到乱码；
 *   2. `prettier format:check` 恒报这两个文件「格式不合规」——但那根本不是格式问题；
 *   3. `prettier --write` 会把非法字节替换成 U+FFFD，**不可逆销毁注释内容**。
 * 其中 prescriptions.ts 的损坏吃掉了换行，导致 `batch_group?: string` 被并进上一行的
 * 注释里——该字段是后端真实字段（model.PrescriptionItem.BatchGroup）且 create.vue 正在
 * 发送，却因不在接口中而完全脱离类型检查。
 *
 * 损坏先于本轮存在且长期无人察觉，说明「prettier 报格式错」掩盖了「文件根本不是 UTF-8」
 * 这个更严重的事实。这里把编码校验变成硬门禁，避免再次靠 `npm run format` 踩雷。
 */
const TEXT_EXTS = new Set([
  '.ts',
  '.vue',
  '.js',
  '.json',
  '.md',
  '.yml',
  '.yaml',
  '.css',
  '.html',
  '.go',
  '.sql',
  '.sh',
])
const SKIP_DIRS = new Set(['node_modules', '.git', 'dist', 'vendor', '.idea', '.vscode', 'bin'])

/** 严格 UTF-8 解码：非法字节序列直接抛错（而非常规解码那样静默产出 U+FFFD）。 */
function collectNonUtf8Files(): string[] {
  const bad: string[] = []
  const decoder = new TextDecoder('utf-8', { fatal: true })
  const walk = (dir: string) => {
    for (const entry of readdirSync(dir)) {
      if (SKIP_DIRS.has(entry)) continue
      const full = join(dir, entry)
      if (statSync(full).isDirectory()) {
        walk(full)
        continue
      }
      const dot = entry.lastIndexOf('.')
      if (dot < 0) continue
      if (!TEXT_EXTS.has(entry.slice(dot).toLowerCase())) continue
      try {
        decoder.decode(readFileSync(full))
      } catch {
        bad.push(relative(webRoot, full))
      }
    }
  }
  walk(join(goRoot, 'internal'))
  walk(join(webRoot, 'src'))
  return bad.sort()
}

describe('源码编码', () => {
  it('所有源码文件均为合法 UTF-8', () => {
    const bad = collectNonUtf8Files()
    expect(
      bad,
      `以下文件不是合法 UTF-8（很可能是 GBK 字节混入）：\n  ${bad.join('\n  ')}\n` +
        '后果：prettier 会把它们报成「格式不合规」，而 --write 会把注释替换成 U+FFFD 永久销毁。',
    ).toEqual([])
  })

  it('.gitattributes 对无扩展名点文件声明 eol=lf', () => {
    // core.autocrlf=true 的 Windows 检出中，无扩展名点文件不受按扩展名的规则约束，
    // 会被转成 CRLF，导致 format:check 在 Windows 上恒失败
    const attrs = readFileSync(join(goRoot, '.gitattributes'), 'utf8')
    expect(attrs).toMatch(/^\.prettierrc\s+.*eol=lf/m)
  })
})
