// @vitest-environment node
import { describe, expect, it } from 'vitest'
import type { TraitRef } from '#shared/utils/api/schemas'
import { traitFullName, traitSide } from './trait'

const name = (zh: string) => ({
  display_name: zh,
  latin: null,
  localized: { 'zh-Hans': { value: zh, is_machine: false } }
})

const ref = (id: string, zh: string, groupId: string, group: string) =>
  ({
    object: 'trait',
    id,
    ...name(zh),
    trait_group: name(group),
    trait_group_id: groupId
  }) as TraitRef

describe('traitSide', () => {
  it('tells the two sides of a paired act apart', () => {
    const passive = ref('2279', '女同强奸', '625', '被动(性)')
    const active = ref('2312', '女同强奸', '2709', '主动(性)')
    expect(traitFullName(passive)).toBe('女同强奸 · 被动(性)')
    expect(traitFullName(active)).toBe('女同强奸 · 主动(性)')
    expect(traitSide(ref('1612', '强奸', '2689', '被动'))).toBe('被动')
  })

  it('leaves other groups and the group roots alone', () => {
    expect(traitSide(ref('781', '靴子', '2373', '服装'))).toBe('')
    expect(traitFullName(ref('625', '被动(性)', '625', '被动(性)'))).toBe(
      '被动(性)'
    )
  })
})
