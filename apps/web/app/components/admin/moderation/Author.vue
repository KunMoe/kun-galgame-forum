<script setup lang="ts">
import type { ReviewAuthor } from '#shared/utils/api/schemas'
import { KUN_USER_ROLE_MAP } from '~/constants/user'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  author: ReviewAuthor
}>()

const emit = defineEmits<{
  showHistory: []
}>()

const profile = computed(() => props.author.profile)

const counts = computed(() => {
  const c = profile.value.counts
  return [
    { label: '话题', value: c.topic_count },
    { label: '回复', value: c.reply_count },
    { label: '话题评论', value: c.topic_comment_count },
    { label: '社区评论', value: c.community_comment_count },
    { label: '资源', value: c.galgame_resource_count },
    { label: '评价', value: c.galgame_rating_count },
    { label: '工具', value: c.toolset_count },
    { label: '关注者', value: c.follower_count }
  ].filter((item) => item.value !== null)
})
</script>

<template>
  <div class="bg-default-100 space-y-2 rounded-lg p-3 text-sm">
    <div class="flex flex-wrap items-center gap-2">
      <UserHoverCard :user-id="Number(profile.id)">
        <KunUserChip :user="toKunUser(profile)" size="sm" />
      </UserHoverCard>
      <span class="text-default-400 text-xs">#{{ profile.id }}</span>
      <KunChip v-if="!author.is_account_active" color="danger" size="xs">
        已封禁或已注销
      </KunChip>
      <KunChip
        v-for="role in profile.roles"
        :key="role"
        color="primary"
        variant="flat"
        size="xs"
      >
        {{ KUN_USER_ROLE_MAP[role] ?? role }}
      </KunChip>
      <KunLink
        :to="`/user/${profile.id}`"
        target="_blank"
        class-name="ml-auto text-xs"
      >
        个人主页
      </KunLink>
    </div>

    <div class="text-default-500 flex flex-wrap gap-x-3 gap-y-1 text-xs">
      <span class="flex items-center gap-1">
        注册于
        <KunTime :time="profile.created_at" type="datetime" show-year />
      </span>
      <span>萌萌点 {{ profile.moemoepoint }}</span>
      <span v-for="item in counts" :key="item.label">
        {{ item.label }} {{ item.value }}
      </span>
    </div>

    <div class="flex flex-wrap items-center gap-2 text-xs">
      <span
        class="text-default-500"
        title="只统计 Trust 平台记录了作者的条目，不含当前条目"
      >
        过往审核
      </span>
      <KunChip
        :color="author.past_actioned_count ? 'danger' : 'default'"
        variant="flat"
        size="xs"
      >
        处置 {{ author.past_actioned_count ?? '—' }}
      </KunChip>
      <KunChip variant="flat" size="xs">
        驳回 {{ author.past_dismissed_count ?? '—' }}
      </KunChip>
      <KunButton
        variant="light"
        color="primary"
        size="xs"
        @click="emit('showHistory')"
      >
        查看该用户的全部条目
      </KunButton>
    </div>

    <p
      v-if="profile.bio"
      class="text-default-600 text-xs break-words whitespace-pre-wrap"
    >
      {{ profile.bio }}
    </p>
  </div>
</template>
