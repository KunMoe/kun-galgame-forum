<script setup lang="ts">
import type { Activity } from '#shared/utils/api/schemas'

defineProps<{ activity: Activity }>()
</script>

<template>
  <ActivityCardTopic
    v-if="activity.activity_type === 'topic_creation' && activity.topic"
    :activity="activity"
    :topic="activity.topic"
  />
  <ActivityCardTopicUpvote
    v-else-if="activity.activity_type === 'topic_upvote' && activity.topic"
    :activity="activity"
    :topic="activity.topic"
  />
  <ActivityCardReply
    v-else-if="
      activity.activity_type === 'topic_reply_creation' && activity.reply
    "
    :activity="activity"
    :reply="activity.reply"
  />
  <ActivityCardSolution
    v-else-if="activity.activity_type === 'best_answer_set' && activity.reply"
    :activity="activity"
    :reply="activity.reply"
  />
  <ActivityCardTopicComment
    v-else-if="
      activity.activity_type === 'topic_comment_creation' && activity.comment
    "
    :activity="activity"
    :comment="activity.comment"
  />
  <ActivityCardGalgame
    v-else-if="activity.activity_type === 'galgame_creation' && activity.work"
    :activity="activity"
    :work="activity.work"
  />
  <ActivityCardGalgameEdit
    v-else-if="activity.activity_type === 'galgame_edit' && activity.work"
    :activity="activity"
    :work="activity.work"
  />
  <ActivityCardGalgamePr
    v-else-if="
      activity.activity_type === 'galgame_pr_creation' && activity.work
    "
    :activity="activity"
    :work="activity.work"
  />
  <ActivityCardGalgameRating
    v-else-if="
      activity.activity_type === 'galgame_rating_creation' &&
      activity.work &&
      activity.galgame_rating
    "
    :activity="activity"
    :work="activity.work"
    :rating="activity.galgame_rating"
  />
  <ActivityCardGalgameComment
    v-else-if="
      activity.activity_type === 'galgame_comment_creation' && activity.work
    "
    :activity="activity"
    :work="activity.work"
  />
  <ActivityCardGalgameResource
    v-else-if="
      activity.activity_type === 'galgame_resource_creation' &&
      activity.work &&
      activity.resource
    "
    :activity="activity"
    :work="activity.work"
    :resource="activity.resource"
  />
  <ActivityCardNote
    v-else-if="
      activity.activity_type === 'todo_creation' ||
      activity.activity_type === 'update_log_creation'
    "
    :activity="activity"
  />
  <ActivityCardToolset
    v-else-if="activity.activity_type === 'toolset_creation'"
    :activity="activity"
  />
  <ActivityCardWebsite
    v-else-if="activity.activity_type === 'galgame_website_creation'"
    :activity="activity"
  />
  <ActivityCardToolsetResource
    v-else-if="
      activity.activity_type === 'toolset_resource_creation' && activity.toolset
    "
    :activity="activity"
    :toolset="activity.toolset"
  />
  <ActivityCardQuiz
    v-else-if="
      activity.activity_type === 'galgame_quiz_creation' && activity.quiz
    "
    :activity="activity"
    :quiz="activity.quiz"
  />
  <ActivityCardGeneric v-else :activity="activity" />
</template>
