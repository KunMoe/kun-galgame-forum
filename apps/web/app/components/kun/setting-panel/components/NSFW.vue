<script setup lang="ts">
import type { KunRadioOption } from '@kungal/ui-vue'

const {
  isSignedIn,
  stance,
  adultConfirmed,
  setStance,
  setAnonymousNsfw,
  openAccountSettings
} = useContentStance()
const { flush } = useCloudPreferences()

// Signed out this is the same binary cookie switch it has always been: one
// value, no round trip, and the reload that reloads the server-filtered lists.
const anonymousOption = ref(stance.value !== 'hide')

watch(anonymousOption, (enabled) => {
  setAnonymousNsfw(enabled)
  location.reload()
})

const stanceOptions: KunRadioOption<KunContentStance>[] = [
  {
    value: 'hide',
    label: '隐藏',
    description: '不显示 R18 等有内容限制的游戏、话题与图片',
    icon: 'lucide:eye-off'
  },
  {
    value: 'blur',
    label: '模糊',
    description: '显示条目与标题，封面和截图打码，点击单张即可查看',
    icon: 'lucide:eye-closed'
  },
  {
    value: 'show',
    label: '显示',
    description: '直接显示成人向内容',
    icon: 'lucide:eye'
  }
]

const selected = ref<KunContentStance>(stance.value)
watch(stance, (value) => {
  selected.value = value
})

const pending = ref(false)
const hint = ref('')
const isAttestationOpen = ref(false)

const onSelect = async (next: KunContentStance) => {
  if (pending.value || next === stance.value) return

  pending.value = true
  hint.value = ''
  selected.value = next
  const result = await setStance(next)
  pending.value = false

  if (result === 'ok') {
    // The gate that matters is server-side, so the lists have to be fetched
    // again — but a debounced cloud write scheduled a moment ago would be lost
    // to the navigation, so it goes out first.
    await flush()
    location.reload()
    return
  }

  selected.value = stance.value
  if (result === 'attestation-required') {
    isAttestationOpen.value = true
    return
  }
  if (result === 'unavailable') {
    hint.value = '本站还没有拿到保存该设置的授权，请退出登录后重新登录再试。'
  }
}

const goToAccountCentre = () => {
  isAttestationOpen.value = false
  openAccountSettings()
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="isSignedIn" class="space-y-3">
      <div class="space-y-0.5">
        <p class="text-default-700 font-medium">成人向内容显示方式</p>
        <p class="text-default-500 text-sm">
          这项设置跟着您的 NextMoe
          账号走，在所有站点生效。更改后会自动刷新页面以重新加载内容。
        </p>
      </div>

      <KunRadioGroup
        :model-value="selected"
        :options="stanceOptions"
        :disabled="pending"
        variant="card"
        aria-label="成人向内容显示方式"
        @update:model-value="onSelect"
      />

      <p v-if="!adultConfirmed" class="text-default-500 text-sm">
        选择「模糊」或「显示」需要先在账号中心完成年龄确认。
      </p>
      <p v-if="hint" class="text-warning-600 text-sm">{{ hint }}</p>
    </div>

    <div v-else class="flex items-start justify-between gap-4">
      <div class="space-y-0.5">
        <p class="text-default-700 font-medium">启用网站 NSFW 模式</p>
        <p class="text-default-500 text-sm">
          显示 R18
          等有内容限制的游戏与图片。开启或关闭后会自动刷新页面以重新加载内容。
        </p>
      </div>
      <KunSwitch v-model="anonymousOption" class="shrink-0" />
    </div>

    <KunModal v-model="isAttestationOpen" inner-class-name="max-w-sm">
      <div class="space-y-3">
        <p class="text-foreground font-medium">需要先完成年龄确认</p>
        <p class="text-default-500 text-sm">
          年龄确认属于账号信息，只能在 NextMoe
          账号中心完成，本站不会代为记录。确认后回到这里即可选择「模糊」或「显示」。
        </p>
        <div class="flex justify-end gap-2">
          <KunButton variant="light" @click="isAttestationOpen = false">
            取消
          </KunButton>
          <KunButton @click="goToAccountCentre">前往账号中心</KunButton>
        </div>
      </div>
    </KunModal>
  </div>
</template>
