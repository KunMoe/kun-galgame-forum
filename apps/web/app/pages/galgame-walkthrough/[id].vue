<script setup lang="ts">
import type { Walkthrough } from '#shared/utils/api/schemas'
import { isTransientProblem } from '#shared/utils/api/problem'

const route = useRoute()
const walkthroughId = String((route.params as { id: string }).id)

const { data, problem } = await useKunApi<Walkthrough>(
  () => `walkthrough:${walkthroughId}`,
  (api, { signal }) =>
    api.GET('/walkthroughs/{walkthrough_id}', {
      params: { path: { walkthrough_id: walkthroughId } },
      signal
    }),
  { critical: true }
)

const redirect = (walkthrough: Walkthrough) =>
  navigateTo(
    {
      path: `/galgame/${walkthrough.work_id}/walkthrough/${walkthroughId}`,
      query: route.query,
      hash: route.hash
    },
    { redirectCode: 301, replace: true }
  )
const unavailable = () =>
  createError({
    statusCode: problem.value && isTransientProblem(problem.value) ? 503 : 404,
    statusMessage: '未找到这份攻略',
    fatal: true
  })

if (data.value) {
  await redirect(data.value)
} else if (import.meta.server) {
  throw unavailable()
}
watch(data, (walkthrough) => {
  if (walkthrough) {
    redirect(walkthrough)
  }
})
watch(problem, (failed) => {
  if (failed) {
    showError(unavailable())
  }
})
</script>

<template>
  <div />
</template>
