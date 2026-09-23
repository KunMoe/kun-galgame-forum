<script setup lang="ts">
import type { Notification } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { settle } from '#shared/utils/api/problem'
import { useCursorList } from '~/composables/useCursorList'
import { maxDecimalId } from '~/utils/decimalId'

definePageMeta({
  middleware: 'auth'
})

useKunDisableSeo('通知消息')

const client = useApiClient()
const noticeEpoch = useState('message-notice-epoch', () => 0)

const { items, hasMore, problem, status, loadingMore, loadMore, refresh } =
  await useCursorList<Notification>(
    'me-notifications',
    (api, cursor, { signal }) =>
      api.GET('/me/notifications', {
        params: {
          query: {
            limit: 30,
            is_muted: false,
            ...(cursor ? { cursor } : {})
          }
        },
        signal
      })
  )

const isSettingOpen = ref(false)
const didMarkRead = ref(false)

const removeNotification = (id: string) => {
  items.value = items.value.filter((notification) => notification.id !== id)
}

const markLoadedRead = async () => {
  if (didMarkRead.value || !import.meta.client) {
    return
  }
  if (status.value === 'pending' || status.value === 'idle') {
    return
  }
  if (!items.value.some((notification) => !notification.is_read)) {
    return
  }
  const upTo = maxDecimalId(items.value.map((notification) => notification.id))
  if (!upTo) {
    return
  }
  didMarkRead.value = true
  const result = await settle(
    client.PUT('/me/notifications/read-marker', {
      body: { up_to_id: upTo, is_muted: false }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  noticeEpoch.value += 1
}

watch([status, items], () => {
  void markLoadedRead()
})

onMounted(() => {
  void markLoadedRead()
})
</script>

<template>
  <div class="flex w-full flex-col space-y-3">
    <header class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <KunButton
          size="lg"
          :is-icon-only="true"
          variant="light"
          href="/message"
        >
          <KunIcon name="lucide:chevron-left" />
        </KunButton>
        <h2 class="text-lg">通知</h2>
      </div>

      <KunPopover position="bottom-end">
        <template #trigger>
          <KunButton variant="light" size="sm">
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:settings" />设置
            </span>
          </KunButton>
        </template>
        <div class="flex w-36 flex-col gap-1 p-2">
          <KunButton
            variant="light"
            color="default"
            size="sm"
            class-name="w-full justify-start gap-2"
            @click="isSettingOpen = true"
          >
            <KunIcon name="lucide:settings" />通知设置
          </KunButton>
        </div>
      </KunPopover>
    </header>

    <KunModal v-model="isSettingOpen" inner-class-name="max-w-lg w-[92vw]">
      <MessageNotificationPreference v-if="isSettingOpen" />
    </KunModal>

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

    <KunNull v-else-if="status !== 'pending'" />
  </div>
</template>
