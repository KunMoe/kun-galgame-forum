import { describe, expect, it } from 'vitest'
import { deletionReasonLabel, entityDeletion } from './deleted'

describe('entityDeletion', () => {
  it('reads the reason and time off an ENTITY_DELETED body', () => {
    expect(
      entityDeletion({
        code: 'ENTITY_DELETED',
        status: 410,
        object: 'work',
        deleted_reason: 'out_of_scope',
        deleted_at: '2026-10-21T08:30:00Z'
      })
    ).toEqual({ reason: 'out_of_scope', at: '2026-10-21T08:30:00Z' })
  })

  it('falls back to other when the body names no reason', () => {
    expect(entityDeletion({ code: 'ENTITY_DELETED' })).toEqual({
      reason: 'other',
      at: null
    })
  })

  it('ignores every other problem', () => {
    expect(entityDeletion({ code: 'ENTITY_MERGED', current_id: '2' })).toBeNull()
    expect(entityDeletion({ code: 'NOT_FOUND' })).toBeNull()
    expect(entityDeletion(undefined)).toBeNull()
  })
})

describe('deletionReasonLabel', () => {
  it('labels a reason catalog adds later as other', () => {
    expect(deletionReasonLabel('spam_or_test')).toBe('这是垃圾或测试条目')
    expect(deletionReasonLabel('duplicate_of_nothing')).toBe('其他原因')
  })
})
