<script setup lang="ts">
import type { NotificationPreferences } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { notificationCategoryGroups } from '~/constants/notification'

const tabs = notificationCategoryGroups

const tabItems = tabs.map(({ value, textValue, icon }) => ({
  value,
  textValue,
  icon
}))
const activeTab = ref('interaction')

const activeItems = computed(
  () => tabs.find((t) => t.value === activeTab.value)?.items ?? []
)

const allKeys = tabs.flatMap((t) => t.items.map((i) => i.key))

const enabled = reactive<Record<string, boolean>>(
  Object.fromEntries(allKeys.map((k) => [k, true]))
)
const isLoading = ref(true)

const api = useApiClient()

const applyMuted = (muted: NotificationPreferences['muted_types']) => {
  const mutedSet = new Set(muted)
  for (const key of allKeys) {
    enabled[key] = !mutedSet.has(key)
  }
}

const collectMuted = (): NotificationPreferences['muted_types'] =>
  allKeys.filter((key) => !enabled[key])

const load = async () => {
  const result = await settle(api.GET('/me/notification-preferences'))
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  applyMuted(result.data.muted_types)
}

const isSaving = ref(false)
let queued = false

const persist = async () => {
  if (isSaving.value) {
    queued = true
    return
  }
  isSaving.value = true
  try {
    do {
      queued = false
      const result = await settle(
        api.PUT('/me/notification-preferences', {
          body: { muted_types: collectMuted() }
        })
      )
      if (!result.ok) {
        reportProblem(result.problem)
        await load()
        return
      }
    } while (queued)
  } finally {
    isSaving.value = false
  }
}

const onToggle = (key: string, value: boolean) => {
  enabled[key] = value
  void persist()
}

onMounted(async () => {
  await load()
  isLoading.value = false
})
</script>

<template>
  <div class="space-y-4">
    <div>
      <span class="text-xl">消息通知</span>
      <p class="text-default-500 text-sm">
        关闭某类通知后，它不会再点亮顶栏红点，也不会出现在通知列表里，而是收进「已静音的消息」，你随时可以去那里查看。
      </p>
    </div>

    <KunTab
      v-model="activeTab"
      :items="tabItems"
      variant="underlined"
      color="primary"
      size="sm"
    />

    <div class="divide-default-100 divide-y">
      <div
        v-for="item in activeItems"
        :key="item.key"
        class="flex items-center justify-between py-2"
      >
        <span class="text-default-700 text-sm">{{ item.label }}</span>
        <KunSwitch
          :model-value="enabled[item.key] ?? true"
          :disabled="isLoading"
          @update:model-value="(v: boolean) => onToggle(item.key, v)"
        />
      </div>
    </div>
  </div>
</template>
