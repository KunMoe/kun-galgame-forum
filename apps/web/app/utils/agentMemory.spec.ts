import { describe, expect, it } from 'vitest'
import { agentMemoryChange, agentPastConversations } from './agentMemory'

describe('agentMemoryChange', () => {
  it('reads the memory a tool changed', () => {
    expect(
      agentMemoryChange({ memory: { op: 'add', id: '3', content: ' 喜欢猫 ' } })
    ).toEqual({ op: 'add', content: '喜欢猫' })
  })

  it('ignores meta it does not understand', () => {
    expect(agentMemoryChange(undefined)).toBeNull()
    expect(agentMemoryChange({ memory: [] })).toBeNull()
    expect(
      agentMemoryChange({ memory: { op: 'merge', content: 'x' } })
    ).toBeNull()
    expect(agentMemoryChange({ memory: { op: 'add', content: '' } })).toBeNull()
  })
})

describe('agentPastConversations', () => {
  it('keeps conversations with a usable id', () => {
    expect(
      agentPastConversations({
        conversations: [
          { id: '41', title: '催泪', updated_at: '2026-10-20T08:00:00Z' },
          { id: '42', title: ' ' },
          { id: '../x', title: '坏的' },
          { id: 7, title: '数字' },
          null
        ]
      })
    ).toEqual([
      { id: '41', title: '催泪', updated_at: '2026-10-20T08:00:00Z' },
      { id: '42', title: '新对话', updated_at: '' }
    ])
    expect(agentPastConversations({ conversations: 'x' })).toEqual([])
    expect(agentPastConversations(undefined)).toEqual([])
  })
})
