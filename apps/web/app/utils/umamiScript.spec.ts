import { describe, expect, it } from 'vitest'
import { umamiTrackerSrc } from './umamiScript'

describe('umamiTrackerSrc', () => {
  it('omits the tracker when the host or the website id is empty', () => {
    expect(umamiTrackerSrc('', 'site', false)).toBe('')
    expect(umamiTrackerSrc('https://umami.example', '', false)).toBe('')
    expect(umamiTrackerSrc(undefined, undefined, false)).toBe('')
  })

  it('builds script.js from the origin and drops a trailing slash', () => {
    expect(
      umamiTrackerSrc('https://umami.example/', 'site-id', false)
    ).toBe('https://umami.example/script.js')
  })

  it('rejects a host that is not a URL', () => {
    expect(umamiTrackerSrc('not a host', 'site-id', false)).toBe('')
  })

  it('stays off in development even when both values are set', () => {
    expect(
      umamiTrackerSrc('https://umami.example', 'site-id', true)
    ).toBe('')
  })
})
