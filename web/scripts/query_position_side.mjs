import { createClient } from '@supabase/supabase-js'
import { readFileSync } from 'node:fs'

const ENV = Object.fromEntries(
  readFileSync('D:/git_projects/bbgo/saas/.env', 'utf-8')
    .split('\n')
    .filter(l => l && !l.startsWith('#') && l.includes('='))
    .map(l => {
      const i = l.indexOf('=')
      return [l.slice(0, i).trim(), l.slice(i + 1).trim()]
    })
)

const URL = ENV.NEXT_PUBLIC_SUPABASE_URL || ENV.SUPABASE_URL
const KEY = ENV.SUPABASE_SERVICE_KEY

if (!URL || !KEY) {
  console.error('missing URL or service key')
  process.exit(1)
}

const sb = createClient(URL, KEY, { auth: { persistSession: false } })

async function count(tbl) {
  const { data, error } = await sb.from(tbl).select('position_side', { count: 'exact', head: false })
  if (error) { console.error(`${tbl}: ${error.message}`); return }
  const groups = {}
  for (const r of data ?? []) {
    const k = r.position_side || '(empty)'
    groups[k] = (groups[k] || 0) + 1
  }
  console.log(`${tbl}: total=${data.length}`)
  for (const [k, v] of Object.entries(groups).sort()) console.log(`  side='${k}': ${v}`)
}

await count('paper_futures_position_risks')
await count('futures_position_risks')
