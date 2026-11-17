export interface AgentGreeting {
  text: string
  conversationId?: string
}

export interface AgentGreetingInput {
  greetings: string[]
  name: string
  hour: number
  recent: Pick<AgentConversation, 'id' | 'title'>[]
  random?: () => number
}

const TIME_CHANCE = 0.35
const HISTORY_CHANCE = 0.25
const RECENT_PICK = 5
const TITLE_MAX = 20
const FALLBACK = '今天想聊点什么？找游戏、问资料都可以哦。'

const TIME_LINES: { from: number; lines: string[] }[] = [
  {
    from: 0,
    lines: [
      '夜深了，{name}，还在找游戏吗？记得早点休息哦。',
      '这么晚还没睡呀，{name}，想找一部睡前玩的作品吗？'
    ]
  },
  {
    from: 5,
    lines: [
      '早上好，{name}！今天早上想玩哪个游戏？',
      '早安，{name}，新的一天想从哪部作品开始？'
    ]
  },
  {
    from: 9,
    lines: [
      '上午好，{name}，来找点新作吗？',
      '上午好呀，{name}，今天想聊哪部作品？'
    ]
  },
  {
    from: 11,
    lines: [
      '中午好，{name}，午休时间来点轻松的作品？',
      '午饭吃了吗，{name}？顺便找部游戏吧。'
    ]
  },
  {
    from: 13,
    lines: [
      '下午好，{name}，想找一部什么样的游戏？',
      '下午好呀，{name}，有想补的作品吗？'
    ]
  },
  {
    from: 17,
    lines: [
      '傍晚好，{name}，今晚打算推哪部？',
      '辛苦啦，{name}，下班路上想看看什么作品？'
    ]
  },
  {
    from: 19,
    lines: [
      '晚上好，{name}，今晚想玩点什么？',
      '晚上好呀，{name}，来找一部今晚的游戏吧。'
    ]
  },
  {
    from: 23,
    lines: ['夜深了，{name}，还在找游戏吗？记得早点休息哦。']
  }
]

const pick = <T>(list: T[], random: () => number): T | undefined =>
  list[Math.floor(random() * list.length)]

// typed.js reads ^N as a pause and backticks as an instant run.
const plain = (s: string) => s.replace(/[\^`]/g, '').trim()

const shorten = (s: string) =>
  [...s].length > TITLE_MAX ? `${[...s].slice(0, TITLE_MAX).join('')}…` : s

const timeLine = (hour: number, name: string, random: () => number) => {
  const slot = TIME_LINES.findLast((t) => hour >= t.from) ?? TIME_LINES[0]!
  const line = pick(slot.lines, random) ?? ''
  return name
    ? line.replaceAll('{name}', name)
    : line.replace(/，\{name\}/g, '').replace(/\{name\}/g, '')
}

export const pickAgentGreeting = ({
  greetings,
  name,
  hour,
  recent,
  random = Math.random
}: AgentGreetingInput): AgentGreeting => {
  const titled = recent
    .slice(0, RECENT_PICK)
    .map((c) => ({ id: c.id, title: plain(c.title) }))
    .filter((c) => c.title && c.title !== '新对话')

  const roll = random()
  if (roll < TIME_CHANCE) return { text: timeLine(hour, plain(name), random) }
  if (roll < TIME_CHANCE + HISTORY_CHANCE && titled.length) {
    const c = pick(titled, random)!
    return {
      text: `上次聊到「${shorten(c.title)}」，要接着聊吗？`,
      conversationId: c.id
    }
  }
  const own = greetings.map(plain).filter(Boolean)
  return { text: pick(own, random) ?? FALLBACK }
}

export const AGENT_ASK_MAX = 200

export const agentAskHref = (q: string) => {
  const ask = q.trim().slice(0, AGENT_ASK_MAX)
  return ask ? `/chat?ask=${encodeURIComponent(ask)}` : '/chat'
}
