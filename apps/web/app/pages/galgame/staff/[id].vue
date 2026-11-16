<script setup lang="ts">
import type { CreditList, CreditName, Credit } from '#shared/utils/api/schemas'
import { mergedInto } from '#shared/utils/api/merged'
import { settle, isMissingProblem } from '#shared/utils/api/problem'
import { workSummaryToCard } from '~/utils/galgame/workCard'

const route = useRoute()
const staffId = computed(() => Number((route.params as { id: string }).id))

if (!Number.isInteger(staffId.value) || staffId.value <= 0) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该制作人员',
    fatal: true
  })
}

const PAGE_SIZE = 50
const api = useApiClient()
const nameOf = useCatalogName()
const { allowsNsfw, stanceKey } = useContentStance()

const creditsQuery = (cursor?: string) => ({
  limit: PAGE_SIZE,
  include_nsfw: allowsNsfw.value,
  ...(cursor ? { cursor } : {})
})

const { data: firstPage, pending: creditsPending } = useKunApi<CreditList>(
  () => `credit-name-credits:${staffId.value}:${stanceKey.value}`,
  (client, { signal }) =>
    client.GET('/credit-names/{credit_name_id}/credits', {
      params: {
        path: { credit_name_id: String(staffId.value) },
        query: creditsQuery()
      },
      signal
    })
)

const movedTo = ref<number | null>(null)
const moveTo = (id: number) =>
  navigateTo(`/galgame/staff/${id}`, { redirectCode: 301, replace: true })
watch(movedTo, (id) => {
  if (id) {
    moveTo(id)
  }
})

const {
  data: person,
  problem,
  pending
} = await useKunApi<CreditName>(
  () => `credit-name:${staffId.value}`,
  async (client, { signal }) => {
    const res = await client.GET('/credit-names/{credit_name_id}', {
      params: { path: { credit_name_id: String(staffId.value) } },
      signal
    })
    movedTo.value = mergedInto(res.error)
    return res
  },
  { critical: true }
)

if (import.meta.server && movedTo.value) {
  await moveTo(movedTo.value)
}
useKunNotFound(
  () => isMissingProblem(problem.value) && movedTo.value === null,
  '未找到该制作人员'
)

const extraCredits = ref<Credit[]>([])
const moreCursor = ref<string | null>()
watch(firstPage, () => {
  extraCredits.value = []
  moreCursor.value = undefined
})
const credits = computed(() => [
  ...(firstPage.value?.items ?? []),
  ...extraCredits.value
])
const nextCursor = computed(() =>
  moreCursor.value === undefined
    ? (firstPage.value?.next_cursor ?? null)
    : moreCursor.value
)
const loadingMore = ref(false)

const loadMore = async () => {
  if (!nextCursor.value || loadingMore.value) {
    return
  }
  loadingMore.value = true
  const res = await settle(
    api.GET('/credit-names/{credit_name_id}/credits', {
      params: {
        path: { credit_name_id: String(staffId.value) },
        query: creditsQuery(nextCursor.value)
      }
    })
  )
  loadingMore.value = false
  if (!res.ok) {
    return
  }
  extraCredits.value.push(...res.data.items)
  moreCursor.value = res.data.next_cursor ?? null
}

const works = computed(() =>
  credits.value.map((c) => ({
    ...workSummaryToCard(c.work_summary, nameOf),
    roles: c.credit_roles.map((r) => r.display_name),
    characters: c.characters.map((ch) => ch.display_name)
  }))
)

const data = computed(() => {
  const p = person.value
  if (!p) {
    return null
  }
  const { name, original } = nameOf(p)
  const intro = pickCatalogIntro(p.intros)
  return {
    id: Number(p.id),
    name,
    name_original: original,
    latin: p.latin ?? '',
    photo: p.photo?.url ?? '',
    intro: intro?.value ?? '',
    intro_machine: intro?.is_machine ?? false,
    siblings: p.siblings.map((s) => ({
      id: Number(s.id),
      name: nameOf(s).name
    })),
    links: p.links.map((l) => ({
      url: l.url,
      name: catalogLinkLabel(l.site, l.url)
    })),
    roles: [
      ...new Set(
        credits.value.flatMap((c) => c.credit_roles.map((r) => r.display_name))
      )
    ]
  }
})

const genderText = computed(() =>
  person.value?.gender === 'female'
    ? '女'
    : person.value?.gender === 'male'
      ? '男'
      : ''
)

const birthdayText = computed(() =>
  formatFuzzyDate(
    person.value?.birth_year,
    person.value?.birth_month,
    person.value?.birth_day
  )
)

const subtitle = computed(() => {
  const parts = [data.value?.name_original, data.value?.latin].filter(
    (part): part is string => !!part && part !== data.value?.name
  )
  return parts.join(' · ')
})

const isPhotoOpen = ref(false)

const workCountText = computed(
  () => `${works.value.length}${nextCursor.value !== null ? '+' : ''}`
)

useKunSeoMeta(() => {
  const staff = data.value
  return staff
    ? {
        title: `${staff.name} 参与制作的 Galgame`,
        description:
          staff.intro ||
          `${staff.name} 在本站收录的 Galgame 中担任 ${staff.roles.join(' / ')} 等职位的作品一览。`,
        ogCard: { kind: 'staff', id: staff.id }
      }
    : {}
})
</script>

<template>
  <div>
    <div v-if="data" class="space-y-6">
      <GalgameEntityHero :name="data.name" :subtitle="subtitle">
        <template v-if="data.photo" #media>
          <button
            type="button"
            class="block cursor-zoom-in"
            :aria-label="`查看 ${data.name} 的照片`"
            @click="isPhotoOpen = true"
          >
            <KunImage
              :src="data.photo"
              :alt="data.name"
              loading="eager"
              object-fit="cover"
              class-name="w-20 shrink-0 rounded-xl sm:w-28 lg:w-36"
              :style="{ aspectRatio: '3/4' }"
            />
          </button>
          <KunLightbox
            v-model:is-open="isPhotoOpen"
            :images="[{ src: data.photo, alt: data.name }]"
          />
        </template>

        <template #meta>
          <p>
            <span class="text-foreground font-semibold tabular-nums">
              {{ workCountText }}
            </span>
            部作品
            <template v-if="genderText"> · 性别 {{ genderText }}</template>
            <template v-if="birthdayText"> · 生日 {{ birthdayText }}</template>
          </p>
        </template>

        <template v-if="data.roles.length || data.links.length" #actions>
          <div v-if="data.roles.length" class="flex flex-wrap gap-2">
            <KunChip v-for="role in data.roles" :key="role" color="primary">
              {{ role }}
            </KunChip>
          </div>
          <GalgameEntityLinks :links="data.links" />
        </template>

        <GalgameEntityIntro
          v-if="data.intro"
          :text="data.intro"
          :note="data.intro_machine ? '该简介由机器翻译生成' : ''"
        />
      </GalgameEntityHero>

      <div
        :class="
          cn(
            'grid grid-cols-1 items-start gap-6',
            data.siblings.length && 'lg:grid-cols-[20rem_minmax(0,1fr)]'
          )
        "
      >
        <aside
          v-if="data.siblings.length"
          class="order-last lg:sticky lg:top-20 lg:order-none lg:max-h-[calc(100dvh-6rem)] lg:overflow-y-auto"
        >
          <GalgameStaffSiblings :siblings="data.siblings" />
        </aside>

        <section class="flex min-w-0 flex-col gap-4">
          <KunHeader
            :name="`${data.name}参与制作的全部 Galgame 作品`"
            description=""
            scale="h2"
          />

          <KunInfo
            v-if="!allowsNsfw"
            color="warning"
            title="部分 Galgame 已隐藏"
            description="当前为 SFW 模式，含 NSFW 内容的 Galgame 不会显示，也不计入数量。如需查看，请在设置面板开启 NSFW 开关。"
          />

          <GalgameCardGridSkeleton v-if="creditsPending" />

          <GalgameCard
            v-else-if="works.length"
            :galgames="works"
            :is-transparent="false"
          >
            <template #meta="{ galgame }">
              <div class="mt-2 flex flex-wrap gap-1">
                <KunChip v-for="role in galgame.roles" :key="role" size="xs">
                  {{ role }}
                </KunChip>
              </div>

              <p
                v-if="galgame.characters?.length"
                class="text-default-500 mt-1 line-clamp-2 text-xs"
              >
                {{ galgame.characters.join(' / ') }}
              </p>
            </template>
          </GalgameCard>

          <KunNull v-else description="暂无该制作人员参与的 Galgame" />

          <div v-if="nextCursor !== null" class="flex justify-center">
            <KunButton
              variant="flat"
              color="primary"
              :is-loading="loadingMore"
              @click="loadMore"
            >
              <KunIcon name="lucide:chevron-down" />
              加载更多作品
            </KunButton>
          </div>

          <p class="text-default-400 text-xs">
            资料来自 鲲 Galgame 论坛的署名图谱。<template
              v-if="!data.siblings.length"
              >本页是一个「署名名义」而非人物档案,
              同一位创作者可能以多个名义署名。</template
            >如果有数据错误请
            <KunLink to="/doc/contact" size="sm">联系我们</KunLink>。
          </p>
        </section>
      </div>
    </div>
    <GalgameEntityDetailSkeleton v-else-if="pending" />
  </div>
</template>
