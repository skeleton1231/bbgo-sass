export class FifoQueue {
  private items: Array<{ price: number; quantity: number }> = []
  private head = 0

  push(price: number, quantity: number): void {
    this.items.push({ price, quantity })
  }

  match(quantity: number): { costBasis: number; remaining: number } {
    let remaining = quantity
    let costBasis = 0
    while (remaining > 1e-12 && this.head < this.items.length) {
      const item = this.items[this.head]!
      if (item.quantity <= remaining) {
        costBasis += item.quantity * item.price
        remaining -= item.quantity
        this.head++
      } else {
        costBasis += remaining * item.price
        item.quantity -= remaining
        remaining = 0
      }
    }
    return { costBasis, remaining }
  }
}

const QUOTE_CURRENCIES = ['USDT', 'FDUSD', 'BUSD', 'USDC', 'TUSD', 'DAI', 'BTC', 'ETH', 'BNB']
  .sort((a, b) => b.length - a.length)

export function extractBaseCurrency(symbol: string): string {
  for (const q of QUOTE_CURRENCIES) {
    if (symbol.endsWith(q)) return symbol.slice(0, -q.length)
  }
  return symbol
}
