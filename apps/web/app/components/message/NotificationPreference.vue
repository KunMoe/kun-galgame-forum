<script setup lang="ts">
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

const applyMuted = (muted: string[]) => {
  const mutedSet = new Set(muted)
  for (const key of allKeys) {
    enabled[key] = !mutedSet.has(key)
  }
}

const collectMuted = () => allKeys.filter((key) => !enabled[key])

const load = async () => {
  const result = await kunFetch<NotificationPreference>(
    '/user/notification-preferences'
  )
  if (result?.muted_types) {
    applyMuted(result.muted_types)
  }
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
      const result = await kunFetch<NotificationPreference>(
        '/user/notification-preferences',
        { method: 'PUT', body: { muted_types: collectMuted() } }
      )
      if (!result) {
        throw new Error('save failed')
      }
    } while (queued)
  } catch {
    useMessage('保存失败，请稍后重试', 'error')
    await load()
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
