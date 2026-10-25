<script setup lang="ts">
import type { Walkthrough } from '#shared/utils/api/schemas'

const route = useRoute()
const walkthroughId = String((route.params as { id: string }).id)

const { data, problem } = await useApi<Walkthrough>(
  () => `walkthrough:${walkthroughId}`,
  (api, { signal }) =>
    api.GET('/walkthroughs/{walkthrough_id}', {
      params: { path: { walkthrough_id: walkthroughId } },
      signal
    })
)

if (data.value) {
  await navigateTo(
    {
      path: `/galgame/${data.value.work_id}/walkthrough/${walkthroughId}`,
      query: route.query,
      hash: route.hash
    },
    { redirectCode: 301, replace: true }
  )
} else {
  throw createError({
    statusCode: (problem.value?.status ?? 0) >= 500 ? 503 : 404,
    statusMessage: '未找到这份攻略',
    fatal: true
  })
}
</script>

<template>
  <div />
</template>
