import { createClient } from '@supabase/supabase-js'
import { readFileSync } from 'node:fs'

const ENV = Object.fromEntries(
  readFileSync('D:/git_projects/bbgo/saas/.env', 'utf-8')
    .split('\n').filter(l => l && !l.startsWith('#') && l.includes('='))
    .map(l => { const i = l.indexOf('='); return [l.slice(0,i).trim(), l.slice(i+1).trim()] })
)
const sb = createClient(ENV.NEXT_PUBLIC_SUPABASE_URL || ENV.SUPABASE_URL, ENV.SUPABASE_SERVICE_KEY, { auth: { persistSession: false } })

const since = process.argv[2]
const symbol = process.argv[3] || 'BTCUSDT'

let q = sb.from('paper_futures_position_risks')
  .select('position_side,position_amount,updated_at,strategy_instance_id,symbol')
  .eq('symbol', symbol)
  .order('updated_at', { ascending: false })
  .limit(15)
if (since) q = q.gte('updated_at', since)

const { data, error } = await q
if (error) { console.error(error.message); process.exit(1) }

console.log(`Latest ${data.length} paper_futures_position_risks rows for ${symbol}:`)
for (const r of data) {
  const ts = r.updated_at.replace('T',' ').replace(/\.\d+Z$/,'Z')
  console.log(`  ${ts}  side='${r.position_side}'  amt=${r.position_amount}  inst=${r.strategy_instance_id}`)
}
