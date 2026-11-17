import { describe, expect, it } from 'vitest'
import { pickAgentGreeting } from './agentGreeting'

const seq = (...values: number[]) => {
  let i = 0
  return () => values[i++ % values.length]!
}

const base = {
  greetings: ['你好呀！'],
  name: '鲲',
  hour: 8,
  recent: [{ id: '42', title: '找纯爱游戏' }]
}

describe('pickAgentGreeting', () => {
  it('greets by the time of day with the user name', () => {
    expect(pickAgentGreeting({ ...base, random: seq(0.1, 0) }).text).toBe(
      '早上好，鲲！今天早上想玩哪个游戏？'
    )
    expect(
      pickAgentGreeting({ ...base, hour: 2, random: seq(0.1, 0) }).text
    ).toBe('夜深了，鲲，还在找游戏吗？记得早点休息哦。')
    expect(
      pickAgentGreeting({ ...base, hour: 23, random: seq(0.1, 0) }).text
    ).toBe('夜深了，鲲，还在找游戏吗？记得早点休息哦。')
  })

  it('drops the name cleanly when there is none', () => {
    expect(
      pickAgentGreeting({ ...base, name: '', hour: 20, random: seq(0.1, 0) })
        .text
    ).toBe('晚上好，今晚想玩点什么？')
  })

  it('offers to continue a recent conversation', () => {
    expect(pickAgentGreeting({ ...base, random: seq(0.4, 0) })).toEqual({
      text: '上次聊到「找纯爱游戏」，要接着聊吗？',
      conversationId: '42'
    })
  })

  it('skips untitled conversations and shortens long titles', () => {
    const recent = [
      { id: '1', title: '新对话' },
      { id: '2', title: '一二三四五六七八九十一二三四五六七八九十多出来' }
    ]
    expect(pickAgentGreeting({ ...base, recent, random: seq(0.4, 0) })).toEqual(
      {
        text: '上次聊到「一二三四五六七八九十一二三四五六七八九十…」，要接着聊吗？',
        conversationId: '2'
      }
    )
  })

  it('falls back to the persona greetings, then to a default', () => {
    expect(pickAgentGreeting({ ...base, random: seq(0.9, 0) }).text).toBe(
      '你好呀！'
    )
    expect(
      pickAgentGreeting({
        ...base,
        greetings: [],
        recent: [],
        random: seq(0.4, 0)
      }).text
    ).toBe('今天想聊点什么？找游戏、问资料都可以哦。')
  })

  it('strips the characters typed.js treats as commands', () => {
    expect(
      pickAgentGreeting({
        ...base,
        greetings: ['等^500一下`呀'],
        random: seq(0.9, 0)
      }).text
    ).toBe('等500一下呀')
  })
})
