// @vitest-environment nuxt
import { describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import DualEditorProvider from '~/components/kun/milkdown/DualEditorProvider.vue'

const hash = 'ab'.repeat(32)

const { get } = vi.hoisted(() => ({ get: vi.fn() }))

mockNuxtImport('useApiClient', () => () => ({ GET: get }))

describe('editor sticker size', () => {
  it('marks a sticker already in loaded markdown once the packs arrive', async () => {
    useState('kun-sticker-packs').value = []
    let respond!: (v: unknown) => void
    get.mockReturnValue(new Promise((r) => (respond = r)))
    const packs = {
      response: new Response(null, { status: 200 }),
      data: {
        items: [
          {
            display_name: 'pack',
            stickers: [{ display_name: 's', image: { hash } }]
          }
        ]
      }
    }

    const wrapper = await mountSuspended(DualEditorProvider, {
      props: {
        valueMarkdown: `text ![s](/image/${hash}_320 "s") and ![p](/image/${'cd'.repeat(32)})`
      },
      attachTo: document.body
    })

    await vi.waitFor(
      () => expect(wrapper.findAll('.ProseMirror img').length).toBe(2),
      { timeout: 10_000 }
    )
    expect(wrapper.findAll('.ProseMirror img.kun-sticker').length).toBe(0)

    respond(packs)
    await vi.waitFor(
      () => {
        expect(wrapper.findAll('.ProseMirror img.kun-sticker').length).toBe(1)
      },
      { timeout: 10_000 }
    )
    expect(get).toHaveBeenCalledWith('/sticker-packs')
    expect(
      wrapper.find('.ProseMirror img.kun-sticker').attributes('src')
    ).toContain(hash)
    wrapper.unmount()
  })
})
