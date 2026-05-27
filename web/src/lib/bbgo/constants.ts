export const EXCHANGES = [
  'binance',
  'okex',
  'kucoin',
  'bybit',
  'bitget',
  'max',
  'coinbase',
  'bitfinex',
] as const

export type Exchange = (typeof EXCHANGES)[number]

export const EXCHANGES_REQUIRING_PASSPHRASE: string[] = ['okex', 'kucoin', 'bitget']

export const CATEGORY_KEYS = [
  'grid',
  'maker',
  'trend',
  'mean-reversion',
  'dca',
  'volatility',
  'indicator',
  'cross-exchange',
  'utility',
  'other',
] as const

export const EXCHANGE_OPTIONS = [
  { id: 'binance', label: 'Binance' },
  { id: 'okex', label: 'OKX' },
  { id: 'bybit', label: 'Bybit' },
  { id: 'bitget', label: 'Bitget' },
  { id: 'kucoin', label: 'KuCoin' },
  { id: 'max', label: 'MAX' },
  { id: 'coinbase', label: 'Coinbase' },
  { id: 'bitfinex', label: 'Bitfinex' },
] as const
