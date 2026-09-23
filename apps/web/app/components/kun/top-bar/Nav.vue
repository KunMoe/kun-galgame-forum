<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const route = useRoute()
const api = useApiClient()

const { showKUNGalgameHamburger, messageStatus } = storeToRefs(
  useTempSettingStore()
)
const { id, moemoepoint, isCheckIn, dailyToolsetUploadBytes } = storeToRefs(
  usePersistUserStore()
)

const { isSnowing, toggleSnow, startSnow } = useKunSnowEffect()

const router = useRouter()
const canGoBack = ref(false)

const updateCanGoBack = () => {
  canGoBack.value = window.history.length > 2
}

watch(
  () => route.name,
  () => {
    useTempSettingStore().reset()
  }
)

onMounted(async () => {
  if (id.value) {
    const result = await settle(api.GET('/me'))
    if (result.ok) {
      isCheckIn.value = result.data.has_checked_in_today
      moemoepoint.value = result.data.moemoepoint
      messageStatus.value = result.data.has_unread_messages ? 'new' : 'online'
      dailyToolsetUploadBytes.value = result.data.toolset_upload_today_bytes
    }
  }

  updateCanGoBack()
  startSnow()
  router.afterEach(() => {
    updateCanGoBack()
  })
})
</script>

<template>
  <div class="flex items-center gap-1">
    <KunButton
      :is-icon-only="true"
      color="default"
      size="xl"
      variant="light"
      @click="showKUNGalgameHamburger = true"
      class-name="flex desktop-nav:hidden"
    >
      <KunIcon name="lucide:menu" />
    </KunButton>

    <KunTooltip :text="canGoBack ? '返回上一页' : '返回主页'" position="bottom">
      <KunButton
        :is-icon-only="true"
        color="default"
        size="xl"
        variant="light"
        class-name="hidden sm:block mr-6"
        @click="canGoBack ? router.back() : navigateTo('/')"
      >
        <KunIcon :name="canGoBack ? 'lucide:arrow-left' : 'lucide:home'" />
      </KunButton>
    </KunTooltip>

    <KunTooltip
      text="本网站完全开源, 代码完全自主编写, 点击访问 GitHub 仓库为我们点亮 star ⭐"
      position="bottom"
      class-name="hidden md:inline-block"
      v-if="!id"
    >
      <KunButton
        variant="light"
        color="default"
        size="xl"
        target="_blank"
        :href="kungal.github"
        class-name="text-xl"
      >
        <KunIcon name="ant-design:github-filled" />
        <span class="text-sm sm:text-base">GitHub</span>
      </KunButton>
    </KunTooltip>

    <KunAdAIFYIcon />

    <LazyKunTopBarHamburger />
  </div>
</template>
