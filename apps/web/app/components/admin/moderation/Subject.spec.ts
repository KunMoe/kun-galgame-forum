// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { ReviewAuthor, ReviewSubject } from '#shared/utils/api/schemas'
import Subject from './Subject.vue'

const subject: ReviewSubject = {
  object: 'review_subject',
  state: 'hidden',
  title: '4 楼',
  content: {
    object: 'document',
    children: [
      {
        object: 'paragraph',
        children: [{ object: 'text', value: '被举报的正文' }]
      }
    ]
  },
  page_path: '/topic/7?reply=4',
  parent_title: '某个话题',
  parent_path: '/topic/7',
  authored_at: '2026-09-01T12:00:00Z'
}

const author: ReviewAuthor = {
  object: 'review_author',
  is_account_active: false,
  past_actioned_count: 2,
  past_dismissed_count: 5,
  profile: {
    object: 'user',
    id: '42',
    name: '作者甲',
    avatar: null,
    bio: '',
    about_html: null,
    avatar_frame: null,
    roles: [],
    created_at: '2025-01-01T00:00:00Z',
    moemoepoint: 7,
    counts: {
      topic_count: 1,
      poll_count: 0,
      lottery_count: 0,
      reply_count: 3,
      topic_comment_count: 0,
      community_comment_count: null,
      published_galgame_count: 0,
      contributed_galgame_count: 0,
      published_galgame_today_count: 0,
      galgame_rating_count: 0,
      galgame_resource_count: 0,
      toolset_count: 0,
      toolset_resource_count: 0,
      received_upvote_count: 0,
      received_like_count: 0,
      received_dislike_count: 0,
      topic_today_count: 0,
      follower_count: null,
      following_count: null
    }
  }
}

describe('AdminModerationSubject', () => {
  it('shows the hidden content, where it lives and who wrote it', async () => {
    const wrapper = await mountSuspended(Subject, {
      props: { kind: 'forum_reply', subject, author }
    })
    const text = wrapper.text()
    for (const want of [
      '已隐藏',
      '某个话题',
      '4 楼',
      '被举报的正文',
      '作者',
      '作者甲',
      '已封禁或已注销',
      '回复 3',
      '萌萌点 7'
    ]) {
      expect(text).toContain(want)
    }
    expect(text).not.toContain('社区评论')
  })

  it("shows the author's past decisions and asks for the rest of their items", async () => {
    const wrapper = await mountSuspended(Subject, {
      props: { kind: 'forum_reply', subject, author }
    })
    expect(wrapper.text()).toContain('处置 2')
    expect(wrapper.text()).toContain('驳回 5')
    const button = wrapper
      .findAll('button')
      .find((b) => b.text().includes('查看该用户的全部条目'))
    await button!.trigger('click')
    expect(wrapper.emitted('showAuthorHistory')).toEqual([[author]])
  })

  it('shows what a deleted post said, and a dash when history is unknown', async () => {
    const wrapper = await mountSuspended(Subject, {
      props: {
        kind: 'community_post',
        subject: { ...subject, state: 'deleted' },
        author: { ...author, past_actioned_count: null }
      }
    })
    const text = wrapper.text()
    expect(text).toContain('已删除（原文保留）')
    expect(text).toContain('被举报的正文')
    expect(text).toContain('处置 —')
  })

  it('names a reported user as the user under review', async () => {
    const wrapper = await mountSuspended(Subject, {
      props: { kind: 'user', subject: { ...subject, state: 'visible' }, author }
    })
    expect(wrapper.text()).toContain('被审核用户')
  })

  it('says when the forum cannot read the content', async () => {
    const wrapper = await mountSuspended(Subject, {
      props: { kind: 'chat_message', subject: null, author: null }
    })
    expect(wrapper.text()).toContain('论坛无法读取这类内容')
  })
})
