<script setup lang="ts">
import type { Notification } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { markdownToText } from '#shared/utils/markdownToText'
import { toKunUser, deletedUserName } from '~/utils/userRef'
import { getMessageI18n } from '../utils/getMessageI18n'

const props = defineProps<{
  notification: Notification
}>()

const emit = defineEmits<{
  deleted: []
}>()

const api = useApiClient()

const actorName = computed(
  () => props.notification.actor.name ?? deletedUserName
)
const actorUser = computed(() => toKunUser(props.notification.actor))
const actorPath = computed(() =>
  props.notification.actor.name
    ? `/user/${props.notification.actor.id}`
    : undefined
)

const contentPreview = computed(() => {
  const text = markdownToText(props.notification.excerpt_markdown).trim()
  return text || '点击查看详情'
})

const handleDeleteMessage = async () => {
  const res = await useComponentMessageStore().alert(
    '您确定要删除这条消息吗？此操作不可撤销。'
  )
  if (!res) {
    return
  }

  const result = await settle(
    api.DELETE('/me/notifications/{notification_id}', {
      params: { path: { notification_id: props.notification.id } }
    })
  )

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }

  emit('deleted')
  useMessage(10106, 'success')
}
</script>

<template>
  <div
    class="space-y-2 rounded-lg p-2"
    :class="notification.is_read ? 'message-read' : ''"
  >
    <div class="flex items-center gap-2 break-all">
      <div class="flex text-lg">
        <KunIcon
          class="text-secondary"
          v-if="!notification.is_read"
          name="lucide:info"
        />
        <KunIcon
          class="text-default"
          v-if="notification.is_read"
          name="lucide:check-check"
        />
      </div>

      <div>
        <KunLink v-if="actorPath" :to="actorPath">
          {{ actorName }}
        </KunLink>
        <span v-else>{{ actorName }}</span>
        <span>{{ getMessageI18n(notification) }}</span>
      </div>
    </div>

    <div class="flex gap-2">
      <KunAvatar
        :disable-floating="true"
        :user="actorUser"
        :is-navigation="Boolean(notification.actor.name)"
      />

      <KunLink
        color="default"
        underline="none"
        class="hover:text-primary cursor-pointer transition-colors"
        :to="notification.path"
      >
        <pre
          class="break-word text-sm leading-8 whitespace-pre-line text-inherit"
          >{{ contentPreview }}</pre
        >
      </KunLink>
    </div>

    <div class="flex justify-between">
      <span class="text-default-500 text-sm">
        <KunTime :time="notification.created_at" type="datetime" show-year />
      </span>

      <KunButton
        :is-icon-only="true"
        variant="light"
        color="danger"
        size="sm"
        @click="handleDeleteMessage"
      >
        <KunIcon name="lucide:trash-2" />
      </KunButton>
    </div>
  </div>
</template>
