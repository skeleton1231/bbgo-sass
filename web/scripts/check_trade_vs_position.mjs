import { createClient } from '@supabase/supabase-js'
import { readFileSync } from 'node:fs'

const ENV = Object.fromEntries(
  readFileSync('D:/git_projects/bbgo/saas/.env', 'utf-8')
    .split('\n').filter(l => l && !l.startsWith('#') && l.includes('='))
    .map(l => { const i = l.indexOf('='); return [l.slice(0,i).trim(), l.slice(i+1).trim()] })
)
const sb = createClient(ENV.NEXT_PUBLIC_SUPABASE_URL || ENV.SUPABASE_URL, ENV.SUPABASE_SERVICE_KEY, { auth: { persistSession: false } })

const { data: trades } = await sb.from('paper_trades')
  .select('side,quantity,created_at')
  .eq('symbol', 'BTCUSDT')
  .order('created_at', { ascending: true })
  .limit(10)

console.log('Latest 10 paper_trades (chronological):')
let pos = 0
for (const t of trades ?? []) {
  const signed = (t.side === 'BUY' ? 1 : -1) * Number(t.quantity)
  pos += signed
  console.log(`  ${t.created_at.replace('T',' ').replace(/\.\d+Z$/,'Z')}  ${t.side.padEnd(4)} qty=${t.quantity}  →  running_amt=${pos.toFixed(4)}`)
}

const { data: risks } = await sb.from('paper_futures_position_risks')
  .select('position_amount,position_side,updated_at')
  .eq('symbol', 'BTCUSDT')
  .order('updated_at', { ascending: false })
  .limit(5)

console.log('\nLatest 5 position snapshots:')
for (const r of risks ?? []) {
  const amt = Number(r.position_amount)
  const dir = amt > 0 ? 'long' : amt < 0 ? 'short' : 'flat'
  console.log(`  ${r.updated_at.replace('T',' ').replace(/\.\d+Z$/,'Z')}  amt=${amt}  dir=${dir}  position_side=${r.position_side}`)
}
