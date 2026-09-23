<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { KnownAccount } from '~/composables/useKnownAccounts'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'

const emit = defineEmits<{ close: [] }>()

const api = useApiClient()
const checkInKey = useIdempotencyKey()

const { id, sub, name, moemoepoint, isCheckIn } = storeToRefs(
  usePersistUserStore()
)
const { canModerate, isCreator } = useRole()
const { accounts } = useKnownAccounts()
const route = useRoute()
const {
  messageStatus,
  showKUNGalgameMoemoepointLog,
  showKUNGalgameLogout,
  showKUNGalgameCreatorApply
} = storeToRefs(useTempSettingStore())

const isShowMessageDot = computed(() => messageStatus.value === 'new')

const showCreatorApply = computed(() => !canModerate.value && !isCreator.value)

const { entryPath: adminEntryPath } = useAdminNav()

const showAccountSwitch = ref(false)
const switchableAccounts = computed(() =>
  accounts.value.filter((a) => a.sub !== sub.value)
)

const onSwitchAccount = (account: KnownAccount) => {
  emit('close')
  startOAuthSwitchAccount(account.sub, route.fullPath)
}
const onAddAccount = () => {
  emit('close')
  startOAuthAddAccount(route.fullPath)
}
const needsReauth = (account: KnownAccount) =>
  (account.roles ?? []).some((r) => r === 'admin' || r === 'ren')

const openCreatorApply = () => {
  emit('close')
  showKUNGalgameCreatorApply.value = true
}

const openMoemoepointLog = () => {
  emit('close')
  showKUNGalgameMoemoepointLog.value = true
}

const handleCheckIn = async () => {
  emit('close')
  isCheckIn.value = true

  const result = await settle(
    api.POST('/me/check-ins', {
      params: {
        header: {
          'Idempotency-Key': checkInKey.take('/me/check-ins', {})
        }
      }
    })
  )

  if (!result.ok) {
    if (result.problem.code === 'ALREADY_EXISTS') {
      return
    }
    isCheckIn.value = false
    reportProblem(result.problem)
    return
  }
  checkInKey.clear()
  moemoepoint.value = result.data.moemoepoint
  const awarded = result.data.moemoepoint_awarded

  if (awarded === 0) {
    useKunLoliInfo(
      '杂~~~鱼~♡杂鱼~♡ 臭杂鱼♡. 签到成功，您今日什么也没获得...',
      5000
    )
  } else if (awarded === 7) {
    useKunLoliInfo('杂鱼~♡♡♡♡♡. 签到成功, 您今日好运获得了 7 萌萌点哦!', 5000)
  } else {
    useKunLoliInfo(`杂~~~鱼~♡. 签到成功，您今日获得了 ${awarded} 萌萌点`, 5000)
  }
}

const openLogout = () => {
  emit('close')
  showKUNGalgameLogout.value = true
}
</script>

<template>
  <div class="flex flex-col gap-1">
    <div class="px-2 py-1">
      <p class="truncate font-semibold">{{ name }}</p>
    </div>

    <button
      type="button"
      class="hover:bg-default-100 flex w-full items-center justify-between rounded-lg px-2 py-2 text-sm transition-colors"
      @click="openMoemoepointLog"
    >
      <span class="flex items-center gap-2">
        <KunIcon class="text-secondary size-4" name="lucide:lollipop" />
        萌萌点
      </span>
      <span class="flex items-center gap-1">
        <span class="text-secondary font-bold tabular-nums">
          {{ moemoepoint }}
        </span>
        <KunIcon
          class="text-foreground/40 size-4"
          name="lucide:chevron-right"
        />
      </span>
    </button>

    <KunButton
      v-if="!isCheckIn"
      variant="light"
      color="secondary"
      size="sm"
      :full-width="true"
      rounded="md"
      class-name="justify-between"
      @click="handleCheckIn"
    >
      <span class="flex items-center gap-2">
        <KunIcon class="size-4" name="lucide:calendar-check" />
        每日签到
      </span>
      <KunIcon class="text-secondary-500 size-5" name="lucide:sparkles" />
    </KunButton>

    <NuxtLink
      :to="`/user/${id}`"
      class="hover:bg-default-100 flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors"
      @click="emit('close')"
    >
      <KunIcon class="size-4" name="lucide:user-round" />
      个人主页
    </NuxtLink>

    <NuxtLink
      to="/message"
      class="hover:bg-default-100 flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors"
      @click="emit('close')"
    >
      <KunIcon class="size-4" name="lucide:mail" />
      我的消息
      <span
        v-if="isShowMessageDot"
        class="bg-secondary-500 ml-auto size-2 rounded-full"
      />
    </NuxtLink>

    <NuxtLink
      v-if="adminEntryPath"
      :to="adminEntryPath"
      class="hover:bg-default-100 flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors"
      @click="emit('close')"
    >
      <KunIcon class="size-4" name="lucide:shield-check" />
      管理系统
    </NuxtLink>

    <button
      v-if="showCreatorApply"
      type="button"
      class="text-primary hover:bg-primary-50 flex w-full items-center gap-2 rounded-lg px-2 py-2 text-sm font-medium transition-colors"
      @click="openCreatorApply"
    >
      <KunIcon class="size-4" name="lucide:sparkles" />
      创作者申请
      <KunIcon
        class="text-primary/50 ml-auto size-4"
        name="lucide:chevron-right"
      />
    </button>

    <button
      type="button"
      class="hover:bg-default-100 flex w-full items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors"
      @click="showAccountSwitch = !showAccountSwitch"
    >
      <KunIcon class="size-4" name="lucide:users-round" />
      账号切换
      <KunIcon
        class="text-foreground/40 ml-auto size-4 transition-transform"
        :class="showAccountSwitch ? 'rotate-90' : ''"
        name="lucide:chevron-right"
      />
    </button>

    <div v-if="showAccountSwitch" class="flex flex-col gap-1 pl-2">
      <button
        v-for="account in switchableAccounts"
        :key="account.sub"
        type="button"
        class="hover:bg-default-100 flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors"
        @click="onSwitchAccount(account)"
      >
        <KunAvatar
          :user="{ id: account.id, name: account.name, avatar: account.avatar }"
          size="sm"
          :is-navigation="false"
          :disable-floating="true"
        />
        <span class="flex min-w-0 flex-col items-start">
          <span class="max-w-40 truncate">{{ account.name }}</span>
          <span v-if="needsReauth(account)" class="text-default-400 text-xs">
            切换需重新登录
          </span>
        </span>
      </button>

      <button
        type="button"
        class="text-primary hover:bg-primary-50 flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-sm font-medium transition-colors"
        @click="onAddAccount"
      >
        <KunIcon class="size-4" name="lucide:plus" />
        添加新账号
      </button>
    </div>

    <KunButton
      variant="light"
      color="danger"
      size="sm"
      :full-width="true"
      rounded="md"
      class-name="justify-start"
      @click="openLogout"
    >
      <KunIcon class="size-4" name="lucide:log-out" />
      退出登录
    </KunButton>
  </div>
</template>
