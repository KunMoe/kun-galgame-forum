<script setup lang="ts">
import { ENTITY_KINDS, entityHref, entityKind } from '~/utils/entityTarget'

const { state, close } = useEntityPreviewPanel()
const route = useRoute()

const isOpen = computed({
  get: () => state.value.isOpen,
  set: (open) => {
    if (!open) close()
  }
})

const target = computed(() => state.value.target)
const title = computed(() =>
  target.value ? `${ENTITY_KINDS[entityKind(target.value)].noun}预览` : '预览'
)

watch(() => route.fullPath, close)

const openDetail = async () => {
  if (!target.value) return
  const href = entityHref(target.value)
  close()
  await navigateTo(href)
}
</script>

<template>
  <KunDrawer
    v-model="isOpen"
    placement="right"
    responsive
    size="lg"
    :title="title"
  >
    <div v-if="target" class="flex h-full flex-col">
      <div class="scrollbar-hide flex-1 overflow-y-auto pb-4">
        <ContentEntityPreviewBody
          :key="entityHref(target)"
          :target="target"
          @navigate="close"
        />
      </div>
      <div class="border-default-200 shrink-0 border-t pt-4">
        <KunButton class-name="w-full justify-center" @click="openDetail">
          查看详情
          <KunIcon name="lucide:arrow-right" />
        </KunButton>
      </div>
    </div>
  </KunDrawer>
</template>
