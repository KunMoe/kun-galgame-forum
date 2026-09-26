// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import { KunLightbox } from '@kungal/ui-vue'
import type { BlockNode, ContentDocument } from '#shared/utils/api/schemas'
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
})
