<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'
import type {
  Notification,
  NotificationPreferences,
  NotificationType
} from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { localNotificationCategories } from '~/constants/notification'
import { useCursorList } from '~/composables/useCursorList'

definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('已静音的消息')

const { data: prefs } = await useApi<NotificationPreferences>(
  'me-notification-preferences',
  (api, { signal }) =>
    api.GET('/me/notification-preferences', { signal })
)
const mutedCategories = computed(() => {
  const muted = new Set(prefs.value?.muted_types ?? [])
  return localNotificationCategories.filter((c) => muted.has(c.key))
})
const tabItems = computed(() => [
  { value: 'all', textValue: '全部' },
  ...mutedCategories.value.map((c) => ({ value: c.key, textValue: c.label }))
])
const activeTab = useRouteQuery<string>('tab', 'all', { mode: 'replace' })

const notificationType = computed((): NotificationType | undefined => {
  if (activeTab.value === 'all') {
    return undefined
  }
  return activeTab.value as NotificationType
})

const { items, hasMore, problem, status, loadingMore, loadMore, refresh } =
  await useCursorList<Notification>(
    () => `me-notifications-muted:${activeTab.value}`,
    (api, cursor, { signal }) =>
      api.GET('/me/notifications', {
        params: {
          query: {
            limit: 30,
            is_muted: true,
            ...(cursor ? { cursor } : {}),
            ...(notificationType.value
              ? { notification_type: notificationType.value }
              : {})
          }
        },
        signal
      })
  )

const removeNotification = (id: string) => {
  items.value = items.value.filter((notification) => notification.id !== id)
}
</script>

<template>
  <div class="flex w-full flex-col space-y-3">
    <header class="flex items-center gap-2">
      <KunButton size="lg" :is-icon-only="true" variant="light" href="/message">
        <KunIcon name="lucide:chevron-left" />
      </KunButton>
      <h2 class="text-lg">已静音的消息</h2>
    </header>

    <KunTab
      v-model="activeTab"
      :items="tabItems"
      variant="underlined"
      color="primary"
      size="sm"
    />

    <KunDivider />

    <div
      v-if="status === 'pending' && !items.length"
      class="flex justify-center py-8"
    >
      <KunLoading />
    </div>

    <template v-else-if="problem">
      <KunNull :description="problemMessage(problem)" />
      <div class="flex justify-center">
        <KunButton variant="flat" size="sm" @click="() => refresh()">
          重试
        </KunButton>
      </div>
    </template>

    <template v-else-if="items.length">
      <KunOverlayScroll class="h-full">
        <MessageAsideNotice
          v-for="notification in items"
          :key="notification.id"
          :notification="notification"
          @deleted="removeNotification(notification.id)"
        />
      </KunOverlayScroll>

      <div v-if="hasMore" class="flex justify-center pt-2">
        <KunButton
          variant="light"
          :loading="loadingMore"
          @click="loadMore"
        >
          加载更多
        </KunButton>
      </div>
    </template>

    <KunNull
      v-else-if="status !== 'pending'"
      description="没有已静音的消息"
    />
  </div>
</template>
