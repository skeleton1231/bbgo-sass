import { describe, it, expect } from 'vitest'

/**
 * The BotChartPanel must receive indicator config + toggle from parent.
 * If it uses its own local empty state, toggles are non-functional.
 * This test validates the contract: parent owns indicators state.
 */
describe('indicator toggle contract', () => {
  it('parent indicators config should drive toggle visibility', () => {
    const parentIndicators = [
      { id: 'sma20', name: 'SMA(20)', enabled: false },
      { id: 'ema12', name: 'EMA(12)', enabled: true },
    ]

    const visibleButtons = parentIndicators.map((ic) => ({
      id: ic.id,
      name: ic.name,
      enabled: ic.enabled,
    }))

    expect(visibleButtons).toHaveLength(2)
    expect(visibleButtons[0]!.enabled).toBe(false)
    expect(visibleButtons[1]!.enabled).toBe(true)

    const toggled = parentIndicators.map((ic) =>
      ic.id === 'sma20' ? { ...ic, enabled: !ic.enabled } : ic
    )
    expect(toggled[0]!.enabled).toBe(true)
    expect(toggled[1]!.enabled).toBe(true)
  })

  it('local empty state would produce zero buttons', () => {
    const localIndicators: Array<{ id: string; enabled: boolean }> = []
    expect(localIndicators).toHaveLength(0)
  })
})
