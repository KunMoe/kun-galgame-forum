<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

const props = defineProps<{
  user: UserProfile
}>()

const displayName = props.user.name ?? ''

useKunSeoMeta({
  title: `${displayName} 的主页`,
  description: props.user.bio
    ? `${displayName} 的个人主页 —— ${props.user.bio}`
    : `${displayName} 在 ${kungal.titleShort} 的个人主页, 查看 TA 发布的话题、评论、收藏与 Galgame 评分。`,
  ...(props.user.avatar?.url ? { ogImage: props.user.avatar.url } : {})
})

useHead({
  link: [
    {
      rel: 'canonical',
      href: `${kungal.domain.main}/user/${props.user.id}/info`
    }
  ]
})
</script>

<template>
  <UserInfo :user="user" />
</template>
