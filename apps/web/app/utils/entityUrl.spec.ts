import { describe, expect, it } from 'vitest'
import { entityFromUrl } from './entityUrl'

const site = 'https://www.kungal.com'

describe('entityFromUrl', () => {
  it.each([
    [`${site}/topic/1`, 'topic', '1'],
    [`${site}/topic/1/`, 'topic', '1'],
    ['https://kungal.com/topic/1', 'topic', '1'],
    ['http://www.kungal.com/topic/1', 'topic', '1'],
    ['/topic/1', 'topic', '1'],
    [`${site}/galgame/3`, 'work', '3'],
    [`${site}/galgame/resource/4`, 'galgame_resource', '4'],
    [`${site}/user/7`, 'user', '7']
  ])('%s is a %s', (raw, type, id) => {
    expect(entityFromUrl(raw)).toEqual({
      type,
      id,
      url: `${site}${{ topic: '/topic/', work: '/galgame/', galgame_resource: '/galgame/resource/', user: '/user/' }[type]}${id}`
    })
  })

  it.each([
    `${site}/topic/1?page=2`,
    `${site}/topic/1?`,
    `${site}/topic/1#k3`,
    `${site}/topic/1/edit`,
    `${site}/topic/0`,
    `${site}/Topic/1`,
    `${site}/topic/%31`,
    'https://www.kungal.com:443/topic/1',
    'https://example.com/topic/1',
    'https://user@www.kungal.com/topic/1',
    'www.kungal.com/topic/1',
    '//www.kungal.com/topic/1',
    'topic/1',
    'kungal-user:1',
    'ftp://www.kungal.com/topic/1'
  ])('%s is not an entity', (raw) => {
    expect(entityFromUrl(raw)).toBeNull()
  })
})
