import { describe, expect, it } from 'vitest'
import { agentBrowseLink } from './agentBrowse'

const browse = (over: Partial<AgentBrowse>): AgentBrowse => ({
  target: 'works',
  ids: ['118', '638'],
  names: ['治愈', '温馨'],
  sort: 'classic',
  sorts: ['classic', 'popular'],
  total: 38,
  ...over
})

describe('agentBrowseLink', () => {
  it('opens the library on the same tags', () => {
    expect(agentBrowseLink(browse({}))).toEqual({
      key: '118,638:classic',
      label: '去 Galgame 数据库按「治愈」「温馨」筛选作品',
      href: '/gallib?tags=118%2C638'
    })
    expect(agentBrowseLink(browse({ sort: 'newest' }))?.href).toBe(
      '/gallib?tags=118%2C638&sortField=release_date'
    )
    expect(agentBrowseLink(browse({ sort: 'popular' }))?.label).toBe(
      '去 Galgame 数据库按「治愈」「温馨」筛选热门作品'
    )
  })

  it('has no page for characters or unusable ids', () => {
    expect(agentBrowseLink(browse({ target: 'characters' }))).toBeNull()
    expect(agentBrowseLink(browse({ ids: ['x', '0'] }))).toBeNull()
  })
})
