<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

const route = useRoute()
const submissionId = computed(() => (route.params as { id: string }).id)

if (!/^\d{1,20}$/.test(submissionId.value)) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该投稿',
    fatal: true
  })
}

const { data: submission } = await useApi(
  () => `my-news-submission:${submissionId.value}`,
  (api, { signal }) =>
    api.GET('/me/news-submissions/{news_submission_id}', {
      params: { path: { news_submission_id: submissionId.value } },
      signal
    })
)

useKunDisableSeo('编辑 Gal 情报')

const isEditable = computed(
  () =>
    submission.value?.state === 'pending' ||
    submission.value?.state === 'published'
)
</script>

<template>
  <div>
    <EditNewsForm
      v-if="submission && isEditable"
      :key="submission.id"
      :initial="submission"
    />
    <KunNull
      v-else-if="submission"
      description="这条投稿已经未通过审核或已撤回，不能再编辑"
    />
    <KunNull v-else description="未找到该投稿" />
  </div>
</template>
