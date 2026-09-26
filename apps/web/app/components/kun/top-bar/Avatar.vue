<script setup lang="ts">
const { id, name, avatar } = storeToRefs(usePersistUserStore())
const { showKUNGalgamePanel, messageStatus } = storeToRefs(
  useTempSettingStore()
)

const { open: openAuthModal } = useAuthModal()

const userMenu = ref<{ close: () => void } | null>(null)

const statusClasses = computed(() => {
  if (messageStatus.value === 'admin') {
    return 'bg-danger'
  } else if (messageStatus.value === 'new') {
    return 'bg-secondary'
  } else if (messageStatus.value === 'online') {
    return 'bg-success'
  } else {
    return 'bg-primary'
  }
})
</script>

<template>
  <div class="flex items-center space-x-1 leading-none">
    <SearchPalette />

    <KunButton
      :is-icon-only="true"
      variant="light"
      color="default"
      size="xl"
      @click="showKUNGalgamePanel = !showKUNGalgamePanel"
    >
      <KunIcon name="lucide:settings" />
    </KunButton>

    <KunPopover ref="userMenu" position="bottom-end" inner-class="p-2 min-w-60">
      <template v-if="id" #trigger>
        <div class="relative inline-flex">
          <KunAvatar
            size="lg"
            :is-navigation="false"
            :user="{ id, name, avatar }"
          />
          <div
            class="absolute right-0 bottom-0 size-2 rounded-full"
            :class="cn(statusClasses, messageStatus)"
          />
        </div>
      </template>

      <Suspense>
        <LazyKunTopBarUserInfo @close="userMenu?.close()" />
        <template #fallback>
          <div class="flex min-h-48 min-w-56 items-center justify-center py-6">
            <KunLoading />
          </div>
        </template>
      </Suspense>
    </KunPopover>

    <template v-if="!id">
      <KunButton size="lg" color="primary" @click="openAuthModal()">
        登录
      </KunButton>
    </template>
  </div>
</template>

<style scoped>
.new {
  animation: kun-pulse 1s infinite;
}

.admin {
  animation: kun-pulse 1s infinite;
}

@keyframes kun-pulse {
  0% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.5);
    opacity: 0.5;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}
</style>
