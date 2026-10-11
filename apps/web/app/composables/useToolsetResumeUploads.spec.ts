// @vitest-environment nuxt
import { beforeEach, describe, expect, it } from 'vitest'
import { useToolsetResumeUploads } from './useToolsetResumeUploads'

const KEY = 'kungal:toolset-upload-resume:7'

beforeEach(() => localStorage.clear())

describe('useToolsetResumeUploads', () => {
  it('keeps a completed upload, marked completed, until it is removed', () => {
    const store = useToolsetResumeUploads('7')
    store.upsert({
      id: 'u1',
      name: 'tool.7z',
      size: 2048,
      last_modified: 0,
      progress: 100,
      updated_at: 1,
      completed: true
    })
    expect(store.list()).toEqual([
      expect.objectContaining({ id: 'u1', completed: true, size: 2048 })
    ])
    store.remove('u1')
    expect(store.list()).toEqual([])
  })

  it('reads a record written before completion was stored as still uploading', () => {
    localStorage.setItem(
      KEY,
      JSON.stringify([
        {
          id: 'u2',
          name: 'old.zip',
          size: 1,
          last_modified: 5,
          progress: 40,
          updated_at: 2
        }
      ])
    )
    expect(useToolsetResumeUploads('7').list()[0]?.completed).toBe(false)
  })
})
