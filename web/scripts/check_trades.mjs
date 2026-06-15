import { createClient } from '@supabase/supabase-js'
import { readFileSync } from 'node:fs'
const ENV = Object.fromEntries(
  readFileSync('D:/git_projects/bbgo/saas/.env', 'utf-8')
    .split('\n').filter(l => l && !l.startsWith('#') && l.includes('='))
    .map(l => { const i = l.indexOf('='); return [l.slice(0,i).trim(), l.slice(i+1).trim()] })
)
const sb = createClient(ENV.NEXT_PUBLIC_SUPABASE_URL || ENV.SUPABASE_URL, ENV.SUPABASE_SERVICE_KEY, { auth: { persistSession: false } })
const { data, error } = await sb.from('paper_trades')
  .select('*')
  .eq('symbol', 'BTCUSDT')
  .order('id', { ascending: false })
  .limit(5)
if (error) { console.error(error.message); process.exit(1) }
console.log('cols:', data.length ? Object.keys(data[0]).join(',') : '(empty)')
console.log(JSON.stringify(data, null, 2))
