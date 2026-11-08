// @vitest-environment nuxt
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { reactive } from 'vue'
import ChatPane from '~/components/chat/Pane.vue'
import ChatEmojiPicker from '~/components/chat/EmojiPicker.vue'

const { store, act } = vi.hoisted(() => ({
  store: { value: null as unknown as Record<string, unknown> },
  act: {
    send: vi.fn(),
    uploadPhoto: vi.fn(),
    saveDraft: vi.fn(),
    read: vi.fn(),
    sendTyping: vi.fn()
  }
}))

mockNuxtImport('useChatStore', () => () => store.value)
mockNuxtImport('useChatActions', () => () => act)

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
mockNuxtImport('useApiClient', () => () => ({ GET: get }))

const photo = {
  object: 'chat_image',
  type: 'photo',
  image_hash: 'a'.repeat(64),
  width: 10,
  height: 10,
  thumbhash: 'th',
  url: 'https://img/a.webp'
}

const deferred = <T>() => {
  let resolve!: (v: T) => void
  const promise = new Promise<T>((r) => (resolve = r))
  return { promise, resolve }
}

beforeEach(() => {
  store.value = reactive({
    model: {
      me: '1',
      users: {},
      conversations: {
        c1: {
          id: 'c1',
          kind: 'direct',
          peer_id: null,
          peer_read_seq: 0,
          pinned_messages: [],
          last_message: null,
          me: { accepted: true, last_read_seq: 0, draft: null }
        }
      },
      windows: { c1: { items: [], hasOlder: false, hasNewer: false } }
    },
    typing: {},
    users: [],
    reactions: [],
    open: vi.fn(async () => {}),
    loadReactions: vi.fn()
  })
  act.send.mockResolvedValue({ ok: true, data: {} })
  act.saveDraft.mockResolvedValue({ ok: true, data: {} })
  URL.createObjectURL = vi.fn(() => 'blob:preview')
  URL.revokeObjectURL = vi.fn()
})

afterEach(() => {
  Object.values(act).forEach((fn) => fn.mockReset())
})

const mountPane = async () => {
  const wrapper = await mountSuspended(ChatPane, {
    props: { id: 'c1' },
    global: {
      stubs: {
        KunChatMessageList: true,
        KunChatHeader: true,
        ChatDialogMenu: true,
        ChatDeleteModal: true,
        ChatReportModal: true
      }
    }
  })
  await flushPromises()
  return wrapper
}

const composerOf = (wrapper: Awaited<ReturnType<typeof mountPane>>) =>
  wrapper.findComponent({ name: 'KunChatComposer' })

const sendButton = (wrapper: Awaited<ReturnType<typeof mountPane>>) =>
  composerOf(wrapper)
    .findAll('button')
    .find((b) => /发送|send/i.test(b.attributes('aria-label') ?? ''))!

const image = () => new File(['x'], 'a.png', { type: 'image/png' })

describe('chat pane sending a photo', () => {
  it('waits for an upload still in flight instead of sending an empty message', async () => {
    const upload = deferred<unknown>()
    act.uploadPhoto.mockReturnValue(upload.promise)
    const wrapper = await mountPane()

    composerOf(wrapper).vm.$emit('attach', [image()])
    await flushPromises()
    await sendButton(wrapper).trigger('click')
    await flushPromises()
    expect(act.send).not.toHaveBeenCalled()

    upload.resolve({ ok: true, data: photo })
    await flushPromises()
    expect(act.send).toHaveBeenCalledTimes(1)
    const [id, body, opts] = act.send.mock.calls[0]!
    expect(id).toBe('c1')
    expect(body.text).toBe('')
    expect(opts.media).toMatchObject({
      type: 'photo',
      image_hash: photo.image_hash,
      url: photo.url
    })
  })

  it('sends a photo-only message once its upload has finished', async () => {
    act.uploadPhoto.mockResolvedValue({ ok: true, data: photo })
    const wrapper = await mountPane()

    composerOf(wrapper).vm.$emit('attach', [image()])
    await flushPromises()
    await sendButton(wrapper).trigger('click')
    await flushPromises()

    expect(act.send).toHaveBeenCalledTimes(1)
    expect(act.send.mock.calls[0]![2].media.image_hash).toBe(photo.image_hash)
  })

  it('keeps a failed upload in the composer and sends nothing empty', async () => {
    act.uploadPhoto.mockResolvedValue({
      ok: false,
      problem: { kind: 'problem', status: 500, code: null, errors: [] }
    })
    const wrapper = await mountPane()

    composerOf(wrapper).vm.$emit('attach', [image()])
    await flushPromises()
    await sendButton(wrapper).trigger('click')
    await flushPromises()

    expect(act.send).not.toHaveBeenCalled()
    const attachments = composerOf(wrapper).props('attachments') as {
      error?: boolean
    }[]
    expect(attachments).toHaveLength(1)
    expect(attachments[0]!.error).toBe(true)
  })

  it('sends the caption on the last photo of an album', async () => {
    act.uploadPhoto.mockResolvedValue({ ok: true, data: photo })
    const wrapper = await mountPane()

    composerOf(wrapper).vm.$emit('attach', [image(), image()])
    await flushPromises()
    await composerOf(wrapper).find('textarea').setValue('caption')
    await sendButton(wrapper).trigger('click')
    await flushPromises()

    expect(act.send).toHaveBeenCalledTimes(2)
    expect(act.send.mock.calls[0]![1].text).toBe('')
    expect(act.send.mock.calls[1]![1].text).toBe('caption')
  })
})

describe('chat pane sticker', () => {
  it('sends a picked sticker as sticker media with no text', async () => {
    const hash = 'ef'.repeat(32)
    const src = `https://img.example/ef/ef/${hash}_320.webp`
    useState('kun-sticker-images').value = {
      [hash]: { width: 640, height: 480, thumbhash: 'th' }
    }
    const wrapper = await mountPane()

    wrapper.findComponent(ChatEmojiPicker).vm.$emit('sticker', src)
    await flushPromises()

    expect(act.send).toHaveBeenCalledTimes(1)
    const [id, body, opts] = act.send.mock.calls[0]!
    expect(id).toBe('c1')
    expect(body).toEqual({ text: '', entities: [] })
    expect(opts.media).toEqual({
      type: 'sticker',
      image_hash: hash,
      width: 640,
      height: 480,
      thumbhash: 'th',
      url: src
    })
  })

  it('sends nothing for an image that is not a known sticker', async () => {
    useState('kun-sticker-images').value = {}
    const wrapper = await mountPane()

    wrapper
      .findComponent(ChatEmojiPicker)
      .vm.$emit('sticker', `/image/${'ef'.repeat(32)}_320`)
    await flushPromises()

    expect(act.send).not.toHaveBeenCalled()
  })
})

describe('chat sticker panel', () => {
  it('shows the official packs and emits the picked sticker', async () => {
    const hash = '12'.repeat(32)
    useState('kun-sticker-packs').value = []
    get.mockResolvedValue({
      response: new Response(null, { status: 200 }),
      data: {
        items: [
          {
            display_name: 'pack',
            stickers: [
              {
                display_name: 's',
                image: { hash, width: 640, height: 480, thumbhash: null }
              }
            ]
          }
        ]
      }
    })
    const wrapper = await mountSuspended(ChatEmojiPicker, {
      attachTo: document.body
    })
    await wrapper.find('button').trigger('click')
    const tab = await vi.waitFor(() => {
      const found = [...document.querySelectorAll('button')].find(
        (b) => b.textContent?.trim() === '贴纸'
      )
      expect(found).toBeDefined()
      return found!
    })
    tab.click()
    const cell = await vi.waitFor(() => {
      const found = document.querySelector(
        `.kun-sticker-panel__cell img[src*="${hash}_320"]`
      )
      expect(found).not.toBeNull()
      return found!.closest('button') ?? (found as HTMLElement)
    })
    ;(cell as HTMLElement).click()
    await flushPromises()

    expect(wrapper.emitted('sticker')?.[0]?.[0]).toContain(`${hash}_320`)
    wrapper.unmount()
  })
})

describe('chat pane emoji', () => {
  it('inserts a picked emoji into the composer', async () => {
    const wrapper = await mountPane()

    wrapper.findComponent(ChatEmojiPicker).vm.$emit('emoji', '😀')
    await flushPromises()

    expect(
      (composerOf(wrapper).find('textarea').element as HTMLTextAreaElement)
        .value
    ).toContain('😀')
  })

  it('emits the emoji picked in the shared sticker panel', async () => {
    const wrapper = await mountSuspended(ChatEmojiPicker, {
      attachTo: document.body
    })
    await wrapper.find('button').trigger('click')
    await vi.waitFor(
      () => {
        expect(
          document.querySelector('.kun-sticker-panel__cell')
        ).not.toBeNull()
      },
      { timeout: 5000 }
    )
    ;(document.querySelector('.kun-sticker-panel__cell') as HTMLElement).click()
    await flushPromises()

    const emitted = wrapper.emitted('emoji')
    expect(emitted?.[0]?.[0]).toBeTypeOf('string')
    expect(document.querySelector('.kun-sticker-panel')).not.toBeNull()
    wrapper.unmount()
  })
})
