import { describe, expect, it } from 'vitest'
import { galgameClaimStateBadge, isPublicState } from './galgameClaimState'

describe('galgameClaimStateBadge', () => {
  it('names every state in the registry vocabulary', () => {
    expect(galgameClaimStateBadge('live')).toEqual({
      label: '已发布',
      color: 'success'
    })
    expect(galgameClaimStateBadge('draft')).toEqual({
      label: '草稿',
      color: 'primary'
    })
    expect(galgameClaimStateBadge('pending')).toEqual({
      label: '审核中',
      color: 'warning'
    })
    expect(galgameClaimStateBadge('declined')).toEqual({
      label: '已拒绝',
      color: 'danger'
    })
    expect(galgameClaimStateBadge('hidden')).toEqual({
      label: '已下架',
      color: 'default'
    })
    expect(galgameClaimStateBadge('none')).toEqual({
      label: '未认领',
      color: 'primary'
    })
  })

  it('falls through to 未知 rather than guessing', () => {
    expect(galgameClaimStateBadge(undefined)).toEqual({
      label: '未知',
      color: 'default'
    })
    expect(galgameClaimStateBadge('nonsense')).toEqual({
      label: '未知',
      color: 'default'
    })
  })

  it('only a live entry has a public page to link to', () => {
    expect(isPublicState('live')).toBe(true)
    expect(isPublicState('draft')).toBe(false)
    expect(isPublicState('hidden')).toBe(false)
  })
})
