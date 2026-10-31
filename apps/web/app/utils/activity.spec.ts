import { describe, expect, it } from 'vitest'
import type { Activity } from '#shared/utils/api/schemas'
import { activitySummaryText, feedTabQuery } from './activity'

describe('activitySummaryText', () => {
  it('masks inline spoilers in a rating timeline summary', () => {
    const activity = {
      activity_type: 'galgame_rating_creation',
      work: {},
      galgame_rating: { short_summary: '音乐很棒||结局\n反转||值得一玩' }
    } as Activity
    expect(activitySummaryText(activity, () => '游戏')).toBe(
      '游戏 · 音乐很棒███值得一玩'
    )
  })
})

describe('feedTabQuery', () => {
  it('keeps the bumped order for a topics-only tab', () => {
    expect(feedTabQuery('TOPIC_NORMAL')).toEqual({
      activity_types: ['topic_creation'],
      topic_sections: 'normal',
      sort: 'bumped_desc'
    })
    expect(feedTabQuery('TOPIC_RESOURCE_HELP').topic_sections).toBe('help')
    expect(
      feedTabQuery('TOPIC_NORMAL,TOPIC_RESOURCE_HELP').topic_sections
    ).toBe('all')
  })

  it('renames the solution echo and drops the upvote echo', () => {
    expect(
      feedTabQuery(
        'TOPIC_NORMAL,TOPIC_REPLY_CREATION,MESSAGE_SOLUTION,MESSAGE_UPVOTE'
      )
    ).toEqual({
      activity_types: [
        'topic_creation',
        'topic_reply_creation',
        'best_answer_set'
      ],
      topic_sections: 'normal'
    })
  })

  it('lowercases the other stored tokens', () => {
    expect(feedTabQuery('GALGAME_CREATION,TODO_CREATION')).toEqual({
      activity_types: ['galgame_creation', 'todo_creation'],
      topic_sections: 'normal'
    })
  })
})
