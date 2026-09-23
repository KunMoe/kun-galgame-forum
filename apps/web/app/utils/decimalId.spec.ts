import { describe, expect, it } from 'vitest'
import { compareDecimalIds, maxDecimalId } from './decimalId'

describe('compareDecimalIds', () => {
  it('orders by length then lexicographically so 99 is less than 100', () => {
    expect(compareDecimalIds('99', '100')).toBeLessThan(0)
    expect(compareDecimalIds('100', '99')).toBeGreaterThan(0)
    expect(compareDecimalIds('100', '100')).toBe(0)
  })

  it('orders same-length ids lexicographically', () => {
    expect(compareDecimalIds('9', '8')).toBeGreaterThan(0)
    expect(compareDecimalIds('12', '13')).toBeLessThan(0)
  })
})

describe('maxDecimalId', () => {
  it('returns the numerically greatest id among mixed lengths', () => {
    expect(maxDecimalId(['99', '100', '9'])).toBe('100')
  })

  it('returns the sole id or undefined for an empty list', () => {
    expect(maxDecimalId(['7'])).toBe('7')
    expect(maxDecimalId([])).toBeUndefined()
  })
})
