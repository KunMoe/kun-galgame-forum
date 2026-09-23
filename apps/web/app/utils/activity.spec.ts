import { describe, expect, it } from 'vitest'
import { feedTabQuery } from './activity'

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
