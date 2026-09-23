<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

definePageMeta({
  middleware: 'auth'
})

defineProps<{
  user: UserProfile
}>()

useKunDisableSeo('信息设置')
</script>

<template>
  <UserSetting :user="user" />
</template>
