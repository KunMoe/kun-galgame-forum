<script setup lang="ts">
import type { Walkthrough, Work } from '#shared/utils/api/schemas'

definePageMeta({ key: (route) => route.path })

const route = useRoute()
const { allowsNsfw, stanceKey } = useContentStance()
const workName = useWorkName()

const params = route.params as { id: string; wid: string }
const workId = String(params.id)
const walkthroughId = String(params.wid)

const { data: walkthrough, problem } = await useApi<Walkthrough>(
  () => `walkthrough:${walkthroughId}`,
  (api, { signal }) =>
    api.GET('/walkthroughs/{walkthrough_id}', {
      params: { path: { walkthrough_id: walkthroughId } },
      signal
    })
)

if (problem.value?.status === 404 || (!walkthrough.value && !problem.value)) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到这份攻略',
    fatal: true
  })
}
if (walkthrough.value && walkthrough.value.work_id !== workId) {
  await navigateTo(
    `/galgame/${walkthrough.value.work_id}/walkthrough/${walkthroughId}`,
    {
      redirectCode: 301,
      replace: true
    }
  )
}

const { data: work } = await useApi<Work>(
  () => `work:${workId}:${stanceKey.value}`,
  (api, { signal }) =>
    api.GET('/works/{work_id}', {
      params: {
        path: { work_id: workId },
        query: { include_nsfw: allowsNsfw.value }
      },
      signal
    })
)

const revealed = ref(false)
const isShowWalkthrough = computed(
  () => !walkthrough.value?.is_nsfw || allowsNsfw.value || revealed.value
)

if (work.value && !work.value.is_nsfw && work.value.is_published) {
  const name = workName(work.value)
  useKunSeoMeta({
    title: `${name} 攻略`,
    description: `《${name}》的选项攻略流程: 按角色整理的选项、存档点与 ${walkthrough.value?.ending_count ?? 0} 个结局。`
  })
} else {
  useKunDisableSeo(work.value ? `${workName(work.value)} 攻略` : '攻略')
}
</script>

<template>
  <div class="container mx-auto my-4 max-w-6xl px-3">
    <template v-if="walkthrough && work">
      <GalgameWalkthroughContainer
        v-if="isShowWalkthrough"
        :walkthrough="walkthrough"
        :work="work"
      />
      <KunNsfwGate v-else noun="攻略" @reveal="revealed = true" />
    </template>
    <KunNull v-else description="攻略暂时无法加载, 请稍后再试" />
  </div>
</template>
