import { describe, expect, it } from 'vitest'
import { relationLabel, relationLabels } from './relationLabel'

const rel = (phrase: string) => ({ relation_type: 'x', phrase })

describe('relationLabel', () => {
  it('names what the related work is, in both directions', () => {
    expect(relationLabel(rel('是…的续作'))).toBe('前作')
    expect(relationLabel(rel('有续作'))).toBe('续作')
    expect(relationLabel(rel('有 Fandisc'))).toBe('FD')
    expect(relationLabel(rel('同系列'))).toBe('同系列')
  })

  it('never shows a bare ellipsis for a phrase it does not know', () => {
    expect(relationLabel(rel('是…的前传'))).toBe('本作是它的前传')
  })

  it('folds directions that land on one label', () => {
    expect(
      relationLabels([rel('是…的外传'), rel('是…的 Fandisc'), rel('同系列')])
    ).toEqual(['本篇', '同系列'])
  })
})
