// @vitest-environment nuxt
import { afterEach, describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import type { ContentDocument, Rating } from '#shared/utils/api/schemas'
import Detail from './Detail.vue'

let wrapper: VueWrapper | undefined

const document: ContentDocument = {
  object: 'document',
  children: [
    {
      object: 'paragraph',
      children: [
        {
          object: 'strong',
          children: [{ object: 'text', value: '音乐很好，' }]
        },
        {
          object: 'inline_spoiler',
          children: [{ object: 'text', value: '秘密' }]
        },
        { object: 'text', value: '，推荐' }
      ]
    }
  ]
}

const mountReview = async (level: string, summaryContent?: ContentDocument) => {
  wrapper = await mountSuspended(Detail, {
    props: {
      data: {
        object: 'rating',
        id: '31',
        overall: 8,
        recommend: 'yes',
        author: {
          object: 'user',
          id: '7',
          name: 'rater',
          avatar: null,
          avatar_frame: null
        },
        aspect_scores: {
          art: 8,
          story: 8,
          music: 8,
          character: 8,
          route: 8,
          system: 8,
          voice: 8,
          replay_value: 8
        },
        short_summary: '**音乐很好，**||秘密||，推荐',
        ...(summaryContent ? { summary_content: summaryContent } : {}),
        spoiler_level: level,
        play_status: 'done_all',
        game_types: ['plot'],
        work_summary: { id: '7' },
        work: null,
        likers: [],
        viewer: null,
        like_count: 0,
        view_count: 0,
        comment_count: 0,
        created_at: '2026-11-03T10:00:00Z',
        updated_at: '2026-11-03T10:00:00Z'
      } as unknown as Rating,
      refresh: () => {}
    },
    global: {
      stubs: {
        GalgameRatingDetailGalgame: true,
        GalgameRatingRadar: true,
        GalgameRatingDetailLike: true,
        GalgameRatingCommentCommunityContainer: true
      }
    }
  })
  return wrapper
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

describe('rating detail summaries', () => {
  it('renders an older API response without a parsed content field', async () => {
    const review = await mountReview('portion')

    expect(review.get('article').text()).toContain('音乐很好，')
    expect(review.findAll('.kun-spoiler-hidden')).toHaveLength(1)
  })

  it.each(['none', 'portion'])(
    'renders Markdown without a whole-review gate at level %s',
    async (level) => {
      const review = await mountReview(level, document)

      expect(review.get('article strong').text()).toBe('音乐很好，')
      expect(review.get('article').classes()).not.toContain('blur-sm')
      const spoiler = review.get('.kun-spoiler-hidden')
      await spoiler.trigger('click')
      expect(spoiler.attributes('aria-expanded')).toBe('true')
    }
  )

  it('requires confirmation for a serious review and keeps its inline spoilers masked', async () => {
    const review = await mountReview('serious', document)
    expect(review.get('article').classes()).toContain('blur-sm')
    const confirm = review
      .findAll('button')
      .find((button) => button.text().includes('该评分含严重剧透'))!

    await confirm.trigger('click')

    expect(review.get('article').classes()).not.toContain('blur-sm')
    expect(review.text()).not.toContain('该评分含严重剧透，点击查看')
    expect(review.findAll('.kun-spoiler-hidden')).toHaveLength(1)
  })
})
