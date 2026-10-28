// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import { KunLightbox } from '@kungal/ui-vue'
import type {
  BlockNode,
  ContentDocument,
  EntityTarget,
  Image,
  TopicPreview
} from '#shared/utils/api/schemas'
import Document from './Document.vue'

const doc = (children: BlockNode[]): ContentDocument => ({
  object: 'document',
  children
})

describe('ContentDocument', () => {
  it('merges compact and className into the article classes', async () => {
    const plain = await mountSuspended(Document, {
      props: { document: doc([]) }
    })
    expect(plain.get('article').classes()).toEqual(['kun-prose'])
    const styled = await mountSuspended(Document, {
      props: { document: doc([]), compact: true, className: 'pt-3' }
    })
    expect(styled.get('article').classes()).toEqual([
      'kun-prose',
      'kun-prose-compact',
      'pt-3'
    ])
  })

  it('reveals a spoiler when its hidden surface is clicked', async () => {
    const wrapper = await mountSuspended(Document, {
      props: {
        document: doc([
          {
            object: 'spoiler',
            children: [
              {
                object: 'paragraph',
                children: [{ object: 'text', value: 'secret' }]
              }
            ]
          }
        ])
      }
    })
    const spoiler = wrapper.get('.kun-spoiler-hidden')
    await spoiler.trigger('click')
    expect(wrapper.find('.kun-spoiler-hidden').exists()).toBe(false)
    expect(wrapper.get('.kun-spoiler').text()).toContain('secret')
  })

  it('opens the lightbox when an image is clicked', async () => {
    const wrapper = await mountSuspended(Document, {
      props: {
        document: doc([
          {
            object: 'paragraph',
            children: [
              {
                object: 'image',
                url: 'https://cdn.example/small.webp',
                alt: 'cover',
                image: {
                  hash: 'ab',
                  url: 'https://cdn.example/full.webp',
                  width: 800,
                  height: 600,
                  thumbhash: 'thumb',
                  sexual: 'safe'
                },
                is_sticker: false
              }
            ]
          }
        ])
      }
    })
    await wrapper.get('img').trigger('click')
    const lightbox = wrapper.getComponent(KunLightbox)
    expect(lightbox.props('isOpen')).toBe(true)
    expect(lightbox.props('images')).toEqual([
      { src: 'https://cdn.example/small.webp', alt: 'cover' }
    ])
  })

  const cover: Image = {
    hash: 'cd',
    url: 'https://cdn.example/cover.webp',
    width: 800,
    height: 450,
    thumbhash: null,
    sexual: 'safe'
  }

  const topic = (over: Partial<TopicPreview> = {}): TopicPreview => ({
    object: 'topic',
    id: '7',
    title: '话题标题',
    cover_images: [cover],
    author: {
      object: 'user',
      id: '3',
      name: 'Alice',
      avatar: null,
      avatar_frame: null
    },
    view_count: 1200,
    like_count: 2,
    reply_count: 5,
    is_nsfw: false,
    created_at: '2026-10-01T00:00:00Z',
    ...over
  })

  const fallback = (url: string) => ({
    object: 'link' as const,
    url,
    children: [{ object: 'text' as const, value: url }]
  })

  const card = (target: EntityTarget): BlockNode => ({
    object: 'entity_card',
    target,
    children: [
      {
        object: 'paragraph',
        children: [fallback('https://www.kungal.com/topic/7')]
      }
    ]
  })

  it('renders an entity card that links to its target', async () => {
    const wrapper = await mountSuspended(Document, {
      props: { document: doc([card(topic())]) }
    })
    const link = wrapper.get('.kun-content-entity a')
    expect(link.attributes('href')).toBe('/topic/7')
    expect(link.text()).toContain('话题标题')
    expect(link.text()).toContain('Alice')
    expect(link.text()).toContain('1.2k 浏览 · 5 回复 · 2 点赞')
  })

  it('does not open the lightbox for a card cover', async () => {
    const wrapper = await mountSuspended(Document, {
      props: { document: doc([card(topic())]) }
    })
    await wrapper.get('.kun-content-entity img').trigger('click')
    expect(wrapper.getComponent(KunLightbox).props('isOpen')).toBe(false)
  })

  it('draws no cover for an NSFW topic under the default stance', async () => {
    const wrapper = await mountSuspended(Document, {
      props: { document: doc([card(topic({ is_nsfw: true }))]) }
    })
    expect(wrapper.find('.kun-content-entity').html()).not.toContain(cover.url)
    expect(wrapper.get('.kun-content-entity').text()).toContain('NSFW')
  })

  it('renders an unavailable target as a muted card that still links', async () => {
    const wrapper = await mountSuspended(Document, {
      props: {
        document: doc([
          card({
            object: 'unavailable_entity',
            entity_type: 'topic',
            entity_id: '9'
          })
        ])
      }
    })
    expect(wrapper.get('.kun-content-entity a').attributes('href')).toBe(
      '/topic/9'
    )
    expect(wrapper.get('.kun-content-entity').text()).toContain(
      '无法预览的话题'
    )
  })

  it('renders an entity link as an inline chip', async () => {
    const wrapper = await mountSuspended(Document, {
      props: {
        document: doc([
          {
            object: 'paragraph',
            children: [
              { object: 'text', value: '看 ' },
              {
                object: 'entity_link',
                target: topic(),
                children: [fallback('https://www.kungal.com/topic/7')]
              }
            ]
          }
        ])
      }
    })
    const chip = wrapper.get('p a.kun-content-entity-chip')
    expect(chip.attributes('href')).toBe('/topic/7')
    expect(chip.text()).toBe('话题标题')
  })

  it('falls back to the plain link for a target it does not know', async () => {
    const wrapper = await mountSuspended(Document, {
      props: {
        document: doc([
          card({ object: 'moyu_patch', id: '1' } as unknown as EntityTarget)
        ])
      }
    })
    expect(wrapper.find('.kun-content-entity').exists()).toBe(false)
    expect(wrapper.get('p a').attributes('href')).toBe(
      'https://www.kungal.com/topic/7'
    )
  })

  it('opens the preview on a plain click instead of navigating', async () => {
    const wrapper = await mountSuspended(Document, {
      props: { document: doc([card(topic())]) }
    })
    const { state, close } = useEntityPreviewPanel()
    close()
    const link = wrapper.get('.kun-content-entity a')
    const plain = new MouseEvent('click', {
      bubbles: true,
      cancelable: true,
      button: 0
    })
    link.element.dispatchEvent(plain)
    expect(plain.defaultPrevented).toBe(true)
    expect(state.value.isOpen).toBe(true)
    expect(state.value.target).toMatchObject({ object: 'topic', id: '7' })

    close()
    const withCtrl = new MouseEvent('click', {
      bubbles: true,
      cancelable: true,
      button: 0,
      ctrlKey: true
    })
    link.element.dispatchEvent(withCtrl)
    expect(state.value.isOpen).toBe(false)
  })
})
