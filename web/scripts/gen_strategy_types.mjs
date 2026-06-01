#!/usr/bin/env node
// Generates manager/strategy_types.go from web/src/lib/bbgo/strategies.ts
// Usage: node scripts/gen_strategy_types.mjs
// Or:    pnpm gen-strategy-types

import { readFileSync, writeFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const tsPath = resolve(__dirname, '../src/lib/bbgo/strategies.ts')
const outPath = resolve(__dirname, '../../manager/strategy_types.go')

const ts = readFileSync(tsPath, 'utf-8')

// ── Parse strategies.ts ──────────────────────────────────────────────

function extractTopLevelBlocks(text) {
  const blocks = []
  let searchFrom = text.indexOf('const STRATEGY_SCHEMAS')
  if (searchFrom === -1) return blocks

  const arrayEnd = text.indexOf('\nexport function', searchFrom)
  const arrayText = text.substring(searchFrom, arrayEnd > 0 ? arrayEnd : text.length)

  // Find each top-level strategy object by scanning for { id: '
  let pos = 0
  while (true) {
    const nextIdx = arrayText.indexOf("id: '", pos)
    if (nextIdx === -1) break

    // Verify this is a strategy id (preceded by { on same or prior line, not inside a field def)
    let objStart = -1
    let depth = 0
    for (let i = nextIdx; i >= Math.max(0, nextIdx - 200); i--) {
      if (arrayText[i] === '}') depth++
      if (arrayText[i] === '{') {
        if (depth === 0) { objStart = i; break }
        depth--
      }
    }
    if (objStart === -1) { pos = nextIdx + 1; continue }

    // Count braces to find matching close
    depth = 1
    let objEnd = objStart + 1
    for (; objEnd < arrayText.length; objEnd++) {
      if (arrayText[objEnd] === '{') depth++
      else if (arrayText[objEnd] === '}') { depth--; if (depth === 0) break }
    }

    const block = arrayText.substring(objStart, objEnd + 1)
    // Filter: strategy blocks have 'fields:' keyword
    if (block.includes('fields:') && block.length > 100) {
      blocks.push(block)
    }
    pos = objEnd + 1
  }
  return blocks
}

function parseStrategy(block) {
  const g = (pattern) => { const m = block.match(pattern); return m ? m[1] : '' }
  const ga = (pattern) => {
    const m = block.match(pattern)
    return m ? m[1].split(',').map(s => s.trim().replace(/'/g, '').replace(/\s/g, '')).filter(Boolean) : []
  }

  const id = g(/id:\s*'([^']+)'/)
  const label = g(/label:\s*'([^']+)'/)
  const description = g(/description:\s*'([^']*)'/)
  const category = g(/category:\s*'([^']+)'/)
  const liveOnly = /\bliveOnly:\s*true\b/.test(block)
  const crossExchange = /\bcrossExchange:\s*true\b/.test(block)
  const supportedExchanges = ga(/supportedExchanges:\s*\[([^\]]+)\]/)

  const sessionRoles = []
  const rolePattern = /\{\s*name:\s*'([^']+)'[^}]*futures:\s*(true|false)/g
  let rm
  while ((rm = rolePattern.exec(block)) !== null) {
    sessionRoles.push({ name: rm[1], futures: rm[2] === 'true' })
  }

  const fields = []
  const fieldPattern = /\{\s*key:\s*'([^']+)',\s*label:\s*'[^']*',\s*type:\s*'([^']+)'([^}]*)\}/g
  let fm
  while ((fm = fieldPattern.exec(block)) !== null) {
    const fieldBlock = fm[3]
    const required = /\brequired:\s*true\b/.test(fieldBlock)
    const defaultMatch = fieldBlock.match(/default:\s*(?:'([^']*)'|(true|false)|(-?\d+\.?\d*))/)

    let defaultValue
    if (defaultMatch) {
      defaultValue = defaultMatch[1] !== undefined ? defaultMatch[1]
        : defaultMatch[2] !== undefined ? defaultMatch[2]
          : defaultMatch[3]
    }

    fields.push({
      key: fm[1],
      type: fm[2],
      required,
      default: defaultValue,
    })
  }

  return { id, label, description, category, liveOnly, crossExchange, supportedExchanges, sessionRoles, fields }
}

// ── Go code generation ───────────────────────────────────────────────

function toPascal(s) {
  return s.replace(/(^|_)(\w)/g, (_, _sep, c) => c.toUpperCase())
}

function goTypeName(strategyId) {
  return strategyId
    .replace(/_(\w)/g, (_, c) => c.toUpperCase())
    .replace(/^(\w)/, (_, c) => c.toUpperCase()) + 'Config'
}

function goFieldType(fieldType, required) {
  switch (fieldType) {
    case 'number': return required ? 'float64' : '*float64'
    case 'boolean': return required ? 'bool' : '*bool'
    case 'text':
    case 'select': return 'string'
    default: return 'string'
  }
}

function generateStruct(strategy) {
  const name = goTypeName(strategy.id)
  const lines = strategy.fields.map(f => {
    const goType = goFieldType(f.type, f.required)
    const omitempty = f.required ? '' : ',omitempty'
    return `\t${toPascal(f.key)} ${goType} \`json:"${f.key}${omitempty}"\``
  })
  return `type ${name} struct {\n${lines.join('\n')}\n}`
}

function generateRegistry(strategies) {
  const entries = strategies.map(s => {
    const exchanges = s.supportedExchanges.map(e => `"${e}"`).join(', ')
    const roles = s.sessionRoles.map(r =>
      `\t\t\t{Name: "${r.name}", Exchange: "", Futures: ${r.futures}}`
    ).join(',\n')

    const rolesField = s.sessionRoles.length > 0
      ? `\n\t\tSessionRoles: []SessionRoleConfig{\n${roles},\n\t\t},` : ''

    return `\t"${s.id}": {
\t\tID:                "${s.id}",
\t\tLabel:             "${s.label.replace(/"/g, '\\"')}",
\t\tDescription:       "${s.description.replace(/"/g, '\\"')}",
\t\tCategory:          "${s.category}",
\t\tSupportedExchanges: []string{${exchanges}},
\t\tLiveOnly:          ${s.liveOnly},
\t\tCrossExchange:     ${s.crossExchange},${rolesField}
\t},`
  })
  return `var StrategyRegistry = map[string]StrategyMeta{\n${entries.join('\n')}\n}`
}

function generateFactory(strategies) {
  const cases = strategies.map(s =>
    `\tcase "${s.id}":\n\t\treturn &${goTypeName(s.id)}{}`
  ).join('\n')
  return `// NewStrategyConfig returns a typed config instance for the given strategy ID.
func NewStrategyConfig(id string) any {
\tswitch id {
${cases}
\tdefault:
\t\treturn &map[string]any{}
\t}
}`
}

// ── Main ─────────────────────────────────────────────────────────────

const blocks = extractTopLevelBlocks(ts)
const strategies = blocks.map(parseStrategy).filter(s => s.id && s.fields.length > 0)

const categories = {}
for (const s of strategies) {
  if (!categories[s.category]) categories[s.category] = []
  categories[s.category].push(s)
}

const categoryLabels = {
  grid: 'Grid Strategies',
  maker: 'Market Maker Strategies',
  trend: 'Trend Following Strategies',
  'mean-reversion': 'Mean Reversion Strategies',
  dca: 'DCA Strategies',
  volatility: 'Volatility Strategies',
  other: 'Other Strategies',
  indicator: 'Indicator Strategies',
  utility: 'Utility Strategies',
  'cross-exchange': 'Cross-Exchange Strategies',
}

let structsCode = ''
for (const [cat, catStrategies] of Object.entries(categories)) {
  const label = categoryLabels[cat] || cat
  structsCode += `\n// --- ${label} ---\n\n`
  for (const s of catStrategies) {
    structsCode += generateStruct(s) + '\n\n'
  }
}

const header = `// Code generated by web/scripts/gen_strategy_types.mjs. DO NOT EDIT.
// Source: web/src/lib/bbgo/strategies.ts STRATEGY_SCHEMAS
//
// Regenerate: cd web && node scripts/gen_strategy_types.mjs

package main

// StrategyMeta holds metadata for a strategy type.
type StrategyMeta struct {
\tID                string
\tLabel             string
\tDescription       string
\tCategory          string
\tSupportedExchanges []string
\tLiveOnly          bool
\tCrossExchange     bool
\tSessionRoles      []SessionRoleConfig
}

`

const file = header + structsCode + generateRegistry(strategies) + '\n\n' + generateFactory(strategies) + '\n'

writeFileSync(outPath, file, 'utf-8')
console.log(`Generated ${strategies.length} strategy types -> ${outPath}`)
console.log(`Strategies: ${strategies.map(s => s.id).join(', ')}`)
