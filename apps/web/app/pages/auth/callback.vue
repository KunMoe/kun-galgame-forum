<script setup lang="ts">
import Cookies from 'js-cookie'

definePageMeta({ layout: false })

const route = useRoute()
const error = ref('')

useKunDisableSeo('OAuth 登录回调')

onMounted(async () => {
  const code = route.query.code as string
  const returnedState = route.query.state as string
  const savedState = Cookies.get('oauth_state')
  const codeVerifier = Cookies.get('oauth_code_verifier')

  Cookies.remove('oauth_state', { path: '/' })
  Cookies.remove('oauth_code_verifier', { path: '/' })

  if (!code) {
    error.value = '未收到授权码'
    redirectToLogin()
    return
  }

  if (returnedState !== savedState) {
    error.value = 'State 不匹配，可能存在安全风险'
    redirectToLogin()
    return
  }

  if (!codeVerifier) {
    error.value = 'PKCE 验证器丢失，请重新登录'
    redirectToLogin()
    return
  }

  const result = await kunFetch<{
    id: number
    sub: string
    name: string
    avatar: string
    roles: string[]
    moemoepoint: number
    bio: string
    adult_confirmed: boolean
    nsfw_display: string
  }>('/auth/oauth/callback', {
    method: 'POST',
    body: { code, code_verifier: codeVerifier }
  })

  if (result) {
    const userStore = usePersistUserStore()
    userStore.setUserInfo({
      id: result.id,
      sub: result.sub,
      name: result.name,
      avatar: result.avatar,
      avatarMin: result.avatar ? withImageVariant(result.avatar, '100') : '',
      moemoepoint: result.moemoepoint,
      roles: result.roles ?? [],
      isCheckIn: false,
      dailyToolsetUploadBytes: 0,
      adultConfirmed: result.adult_confirmed,
      nsfwDisplay: result.nsfw_display
    })

    useKnownAccounts().rememberUser({
      sub: result.sub,
      id: result.id,
      name: result.name,
      avatar: result.avatar,
      roles: result.roles
    })

    useKunLoliInfo(`登录成功! 欢迎来到 ${kungal.name}`)
    await navigateTo(consumeOAuthReturnTo() ?? '/')
  } else {
    error.value = '登录失败，请重试'
    redirectToLogin()
  }
})

const redirectToLogin = () => {
  setTimeout(() => navigateTo('/'), 2000)
}
</script>

<template>
  <div
    class="bg-background flex min-h-dvh w-full flex-col items-center justify-center gap-5 px-6 text-center"
  >
    <template v-if="!error">
      <KunIcon
        class="text-primary size-10 animate-spin"
        name="lucide:loader-circle"
      />
      <div class="space-y-1">
        <p class="text-foreground text-lg font-medium">正在登录...</p>
        <p class="text-default-500 text-sm">正在验证您的身份，请稍候</p>
      </div>
    </template>
    <template v-else>
      <KunIcon class="text-danger size-10" name="lucide:circle-alert" />
      <div class="space-y-1">
        <p class="text-danger text-lg font-medium">{{ error }}</p>
        <p class="text-default-500 text-sm">即将返回首页…</p>
      </div>
    </template>
  </div>
</template>
