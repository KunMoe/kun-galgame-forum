// @vitest-environment node
import { describe, expect, it } from 'vitest'
import type { TagNode, TagRef } from '#shared/utils/api/schemas'
import { ancestorTrail, tagPathsUnder } from './tagTree'

const node = (id: string, parentIds: string[]) =>
  ({
    object: 'tag',
    id,
    display_name: `t${id}`,
    latin: null,
    localized: {},
    is_sexual: false,
    parent_ids: parentIds
  }) as TagNode

const ids = (nodes: TagNode[] | undefined) => nodes?.map((n) => n.id)

describe('tagPathsUnder', () => {
  it('names the tags between the root and each node along a shortest path', () => {
    const paths = tagPathsUnder('1', [
      node('2', ['1']),
      node('3', ['1']),
      node('4', ['2', '3']),
      node('5', ['4', '1']),
      node('6', ['4'])
    ])
    expect(ids(paths.get('2'))).toEqual([])
    expect(ids(paths.get('4'))).toEqual(['2'])
    expect(ids(paths.get('5'))).toEqual([])
    expect(ids(paths.get('6'))).toEqual(['2', '4'])
  })
})

describe('ancestorTrail', () => {
  it('walks first parents up to a root, root first', () => {
    const parent = { id: '3' } as TagRef
    const trail = ancestorTrail(
      [parent],
      [node('3', ['2', '9']), node('2', ['1']), node('9', []), node('1', [])]
    )
    expect(ids(trail)).toEqual(['1', '2', '3'])
  })

  it('is empty for a root and stops on a loop', () => {
    expect(ancestorTrail([], [])).toEqual([])
    const trail = ancestorTrail(
      [{ id: '2' } as TagRef],
      [node('2', ['3']), node('3', ['2'])]
    )
    expect(ids(trail)).toEqual(['3', '2'])
  })
})
