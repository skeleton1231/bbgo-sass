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

export const CATEGORY_LABELS: Record<string, string> = {
  grid: 'Grid',
  maker: 'Market Maker',
  trend: 'Trend Following',
  'mean-reversion': 'Mean Reversion',
  dca: 'DCA',
  volatility: 'Volatility',
  indicator: 'Indicator',
  'cross-exchange': 'Cross-Exchange',
  utility: 'Utility',
  other: 'Other',
}

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
