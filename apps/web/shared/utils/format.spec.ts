import { describe, it, expect } from 'vitest'
import { formatFileSize, formatNumber, truncateRunes } from './format'

describe('formatFileSize', () => {
  it('B for <1KB', () => {
    expect(formatFileSize(0)).toBe('0 B')
    expect(formatFileSize(1023)).toBe('1023 B')
  })
  it('KB for [1KB, 1MB)', () => {
    expect(formatFileSize(1024)).toBe('1.0 KB')
    expect(formatFileSize(1536)).toBe('1.5 KB')
    expect(formatFileSize(1024 * 1024 - 1)).toMatch(/^1024\.0 KB$/)
  })
  it('MB for [1MB, 1GB)', () => {
    expect(formatFileSize(1024 * 1024)).toBe('1.0 MB')
    expect(formatFileSize(1.5 * 1024 * 1024)).toBe('1.5 MB')
  })
  it('GB for >=1GB', () => {
    expect(formatFileSize(1024 * 1024 * 1024)).toBe('1.00 GB')
    expect(formatFileSize(2.5 * 1024 * 1024 * 1024)).toBe('2.50 GB')
  })
})

describe('formatNumber', () => {
  it('returns string for <1k', () => {
    expect(formatNumber(0)).toBe('0')
    expect(formatNumber(999)).toBe('999')
  })
  it('k suffix for [1k, 10k)', () => {
    expect(formatNumber(1000)).toBe('1.0k')
    expect(formatNumber(9999)).toBe('10.0k')
  })
  it('w (万) suffix for [10k, 1M)', () => {
    expect(formatNumber(10_000)).toBe('1.0w')
    expect(formatNumber(50_000)).toBe('5.0w')
    expect(formatNumber(999_999)).toBe('100.0w')
  })
  it('M suffix for >=1M', () => {
    expect(formatNumber(1_000_000)).toBe('1.0M')
    expect(formatNumber(2_500_000)).toBe('2.5M')
  })
})

describe('truncateRunes', () => {
  it('never leaves half of a surrogate pair', () => {
    const text = '@ #6 你确定访问是  而不是  ？🙃'
    expect(text.slice(0, 20)).toMatch(/[\uD800-\uDBFF]$/)
    expect(truncateRunes(text, 20)).not.toMatch(/[\uD800-\uDBFF]$/)
    expect(truncateRunes(text, 20)).toBe(text)
    expect(truncateRunes(text, 19)).toBe('@ #6 你确定访问是  而不是  ？')
  })
  it('counts an astral char as one', () => {
    expect(truncateRunes('🙃🙃🙃', 2)).toBe('🙃🙃')
  })
  it('returns the input when it is short enough', () => {
    expect(truncateRunes('abc', 3)).toBe('abc')
    expect(truncateRunes('', 5)).toBe('')
  })
})
