<script setup lang="ts">
import {
  KUN_FEED_KIND_GROUPS,
  isFollowingFeedTab,
  isNewsFeedTab
} from '~/constants/activity'

const settings = usePersistSettingsStore()
const { feedTabs } = storeToRefs(settings)

const editingId = ref(feedTabs.value[0]?.id ?? '')
const tabOptions = computed(() =>
  feedTabs.value.map((t) => ({ value: t.id, label: t.name }))
)
const editingTab = computed(() =>
  feedTabs.value.find((t) => t.id === editingId.value)
)

watchEffect(() => {
  if (
    feedTabs.value.length &&
    !feedTabs.value.some((t) => t.id === editingId.value)
  ) {
    editingId.value = feedTabs.value[0]!.id
  }
})

const toggleKind = (kind: string) => {
  const tab = editingTab.value
  if (!tab) return
  const i = tab.kinds.indexOf(kind)
  if (i >= 0) {
    tab.kinds.splice(i, 1)
  } else {
    tab.kinds.push(kind)
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-0.5">
        <p class="text-default-700 font-medium">主页动态标签</p>
        <p class="text-default-500 text-sm">
          选择一个标签，勾选它要展示的动态种类。
        </p>
      </div>
      <KunButton
        size="sm"
        variant="light"
        class="shrink-0"
        @click="settings.resetKUNGalgameFeedTabs()"
      >
        恢复默认
      </KunButton>
    </div>

    <KunSelect v-model="editingId" :options="tabOptions" />

    <KunInfo
      v-if="isNewsFeedTab(editingTab)"
      color="info"
      description="Gal 情报来自合作站点转载的情报流，不是站内动态，因此没有可勾选的种类。"
    />

    <div v-else-if="editingTab" class="space-y-3">
      <KunInfo
        v-if="isFollowingFeedTab(editingTab)"
        color="info"
        description="关注标签只显示你关注的人的动态，下面勾选的种类同样生效。"
      />
      <div
        v-for="group in KUN_FEED_KIND_GROUPS"
        :key="group.label"
        class="space-y-1.5"
      >
        <p class="text-default-400 text-xs">{{ group.label }}</p>
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="kind in group.kinds"
            :key="kind.value"
            type="button"
            :class="
              cn(
                'flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs transition-colors',
                editingTab?.kinds.includes(kind.value)
                  ? 'border-primary bg-primary/10 text-primary'
                  : 'border-default-200 text-default-600 hover:bg-default-100'
              )
            "
            @click="toggleKind(kind.value)"
          >
            <KunIcon :name="kind.icon" class="text-sm" />
            {{ kind.label }}
          </button>
        </div>
      </div>

      <p v-if="!editingTab?.kinds.length" class="text-warning text-xs">
        该标签未选择任何种类，将不会显示内容。
      </p>
    </div>
  </div>
</template>
