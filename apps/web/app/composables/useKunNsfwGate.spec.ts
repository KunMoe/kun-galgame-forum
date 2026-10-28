// @vitest-environment nuxt
import { describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { ref } from 'vue'
import type { ApiResult } from '#shared/utils/api/problem'
import { readWithNsfwProbe, useKunNsfwGate } from './useKunNsfwGate'

const stance = vi.hoisted(() => ({ allows: false }))
mockNuxtImport('useContentStance', () => () => ({
  allowsNsfw: ref(stance.allows)
}))

const missing: ApiResult<never> = {
  ok: false,
  problem: {
    kind: 'problem',
    status: 404,
    code: 'NOT_FOUND',
    errors: [],
    requestId: null
  }
}

describe('readWithNsfwProbe', () => {
  it('asks again with include_nsfw only after a hidden reader gets 404', async () => {
    const read = vi.fn(async (includeNsfw: boolean) =>
      includeNsfw ? { ok: true as const, data: 'adult' } : missing
    )
    await expect(readWithNsfwProbe(read, false)).resolves.toEqual({
      ok: true,
      data: 'adult'
    })
    expect(read.mock.calls).toEqual([[false], [true]])

    read.mockClear()
    await readWithNsfwProbe(read, true)
    expect(read.mock.calls).toEqual([[true]])
  })

  it('keeps the 404 when the record is missing either way', async () => {
    const read = vi.fn(async () => missing)
    await expect(readWithNsfwProbe(read, false)).resolves.toBe(missing)
  })
})

describe('useKunNsfwGate', () => {
  it('gates adult content for a hidden reader until revealed', () => {
    stance.allows = false
    const adult = ref(true)
    const { gated, includeNsfw, reveal } = useKunNsfwGate(() => adult.value)
    expect(gated.value).toBe(true)
    expect(includeNsfw.value).toBe(false)
    reveal()
    expect(gated.value).toBe(false)
    expect(includeNsfw.value).toBe(true)
  })

  it('never gates for a reader who shows adult content, or SFW content', () => {
    stance.allows = true
    expect(useKunNsfwGate(() => true).gated.value).toBe(false)
    stance.allows = false
    expect(useKunNsfwGate(() => false).gated.value).toBe(false)
  })
})
