<script setup lang="ts">
import type { FollowedCompanyList } from '#shared/utils/api/schemas'
import { problemMessage } from '#shared/utils/api/message'
import { companyItemOf } from '~/utils/galgame/entityCards'

const props = defineProps<{
  userId: string
  isOwner: boolean
}>()

const LIMIT = 24
const pageData = reactive({ page: usePageQuery(), limit: LIMIT })
const nameOf = useCatalogName()

const { data, status, pending, refresh, problem } =
  useKunApi<FollowedCompanyList>(
    () =>
      `user-followed-companies:${props.userId}:${pageData.page}:${pageData.limit}`,
    (api, { signal }) =>
      api.GET('/users/{user_id}/followed-companies', {
        params: {
          path: { user_id: props.userId },
          query: { page: pageData.page, limit: pageData.limit }
        },
        signal
      })
  )

const isPrivate = computed(
  () => problem.value?.code === 'FOLLOWED_COMPANIES_PRIVATE'
)

const privacy = useKunApi(
  () => `me-privacy-settings:${props.userId}`,
  (client, { signal }) => client.GET('/me/privacy-settings', { signal }),
  { immediate: props.isOwner }
)
const ownPublic = computed(
  () => privacy.data.value?.is_followed_companies_public ?? null
)
const settingHref = computed(
  () => `/user/${props.userId}/setting#followed-companies`
)

const officials = computed(() =>
  (data.value?.items ?? []).map((item) => companyItemOf(item.company, nameOf))
)

const onFollowChange = async (delta: number) => {
  if (delta >= 0) {
    return
  }
  if ((data.value?.items.length ?? 0) <= 1 && pageData.page > 1) {
    pageData.page -= 1
    return
  }
  await refresh()
}

const listStart = useTemplateRef('listStart')
const revealList = useRevealListStart(
  listStart,
  () => pageData.page,
  () => status.value === 'pending'
)
</script>

<template>
  <div ref="listStart" class="scroll-mt-36 space-y-3 lg:scroll-mt-20">
    <h2 class="sr-only">关注的会社<KunPageLabel :page="pageData.page" /></h2>
    <template v-if="isOwner && ownPublic !== null">
      <p v-if="ownPublic" class="text-default-500 text-sm">
        你关注的会社对所有人公开。
        <KunLink :to="settingHref" size="sm" underline="hover">
          在设置里关闭
        </KunLink>
      </p>
      <p v-else class="text-default-500 text-sm">
        你关注的会社只有自己能看到。
        <KunLink :to="settingHref" size="sm" underline="hover">
          在设置里公开
        </KunLink>
      </p>
    </template>
    <KunNull v-if="isPrivate" description="TA 未公开关注的会社" />
    <UserListSkeleton v-else-if="pending" shape="media" />
    <template v-else>
      <div v-if="officials.length" class="flex flex-col gap-3">
        <div
          v-for="official in officials"
          :key="official.id"
          class="flex flex-wrap items-center gap-3"
        >
          <div class="min-w-0 flex-1">
            <GalgameOfficialCard :official="official" />
          </div>
          <GalgameOfficialFollowButton
            v-if="isOwner"
            :company-id="official.id"
            @change="onFollowChange"
          />
        </div>

        <KunPagination
          v-if="data && data.total > pageData.limit"
          v-model:current-page="pageData.page"
          :total-page="Math.ceil(data.total / pageData.limit)"
          :is-loading="status === 'pending'"
          @update:current-page="revealList"
        />
      </div>

      <KunNull v-else-if="problem" :description="problemMessage(problem)" />

      <KunNull
        v-else-if="status !== 'pending'"
        :description="isOwner ? '还没有关注任何会社' : 'TA 还没有关注会社'"
      />
    </template>
  </div>
</template>
