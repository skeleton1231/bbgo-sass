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
