<script setup lang="ts">
import { getGalgameCharacterVoiceLabel } from '~/constants/galgameCharacter'
import type {
  Appearance,
  AppearanceList,
  Character,
  ListWalkthrough
} from '#shared/utils/api/schemas'
import { mergedInto } from '#shared/utils/api/merged'
import { settle, isMissingProblem } from '#shared/utils/api/problem'
import { workSummaryToCard } from '~/utils/galgame/workCard'
import { characterViewOf } from '~/utils/galgame/entityCards'

const route = useRoute()
const characterId = computed(() => Number((route.params as { id: string }).id))

if (!Number.isInteger(characterId.value) || characterId.value <= 0) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该角色',
    fatal: true
  })
}

const PAGE_SIZE = 50
const api = useApiClient()
const nameOf = useCatalogName()
const { allowsNsfw, stanceKey } = useContentStance()

const appearancesQuery = (cursor?: string) => ({
  limit: PAGE_SIZE,
  include_nsfw: allowsNsfw.value,
  ...(cursor ? { cursor } : {})
})

const { data: firstPage, pending: appearancesPending } =
  useKunApi<AppearanceList>(
    () => `character-appearances:${characterId.value}:${stanceKey.value}`,
    (client, { signal }) =>
      client.GET('/characters/{character_id}/appearances', {
        params: {
          path: { character_id: String(characterId.value) },
          query: appearancesQuery()
        },
        signal
      })
  )

const { data: walkthroughs, pending: walkthroughsPending } =
  useKunApi<ListWalkthrough>(
    () => `character-walkthroughs:${characterId.value}:${stanceKey.value}`,
    (client, { signal }) =>
      client.GET('/walkthroughs', {
        params: {
          query: {
            character_id: String(characterId.value),
            include_nsfw: allowsNsfw.value
          }
        },
        signal
      })
  )

const movedTo = ref<number | null>(null)
const moveTo = (id: number) =>
  navigateTo(`/galgame/character/${id}`, { redirectCode: 301, replace: true })
watch(movedTo, (id) => {
  if (id) {
    moveTo(id)
  }
})

const {
  data: character,
  problem,
  pending
} = await useKunApi<Character>(
  () => `character:${characterId.value}:${stanceKey.value}`,
  async (client, { signal }) => {
    const res = await client.GET('/characters/{character_id}', {
      params: {
        path: { character_id: String(characterId.value) },
        query: { include_nsfw: allowsNsfw.value }
      },
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
  '未找到该角色'
)

const extraAppearances = ref<Appearance[]>([])
const moreCursor = ref<string | null>()
watch(firstPage, () => {
  extraAppearances.value = []
  moreCursor.value = undefined
})
const appearances = computed(() => [
  ...(firstPage.value?.items ?? []),
  ...extraAppearances.value
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
    api.GET('/characters/{character_id}/appearances', {
      params: {
        path: { character_id: String(characterId.value) },
        query: appearancesQuery(nextCursor.value)
      }
    })
  )
  loadingMore.value = false
  if (!res.ok) {
    return
  }
  extraAppearances.value.push(...res.data.items)
  moreCursor.value = res.data.next_cursor ?? null
}

const works = computed(() =>
  appearances.value.map((a) => ({
    ...workSummaryToCard(a.work_summary, nameOf),
    voices: a.voices.map((v) => ({
      id: Number(v.id),
      name: nameOf(v).name,
      lang: v.lang ?? '',
      latin: v.latin ?? ''
    }))
  }))
)

const data = computed(() =>
  character.value ? characterViewOf(character.value, nameOf) : null
)

const voices = computed(() => {
  const seen = new Map<number, GalgameDetailCharacterVoice>()
  for (const work of works.value) {
    for (const voice of work.voices) {
      if (!seen.has(voice.id)) {
        seen.set(voice.id, voice)
      }
    }
  }
  return [...seen.values()]
})

const voiceTitle = (voices: GalgameDetailCharacterVoice[]) =>
  voices.map((v) => v.latin || v.name).join(' / ')

const workCountText = computed(
  () => `${works.value.length}${nextCursor.value !== null ? '+' : ''}`
)

const artImages = computed(() =>
  [data.value?.figure, data.value?.image]
    .filter((src): src is string => !!src)
    .map((src) => ({ src, alt: data.value?.name ?? '' }))
)
const isLightboxOpen = ref(false)
const lightboxIndex = ref(0)
const openArt = (kind: 'image' | 'figure') => {
  lightboxIndex.value = kind === 'image' && data.value?.figure ? 1 : 0
  isLightboxOpen.value = true
}

const hasAside = computed(
  () =>
    !!data.value &&
    (!!data.value.traits.length || (!!data.value.image && !!data.value.figure))
)

useKunSeoMeta(() => {
  const view = data.value
  return view
    ? {
        title: `${view.name} 登场的 Galgame`,
        description:
          view.intro ||
          `角色 ${view.name} 在本站收录的 Galgame 中的登场作品与配音演员一览。`,
        ogCard: { kind: 'character', id: view.id }
      }
    : {}
})
</script>

<template>
  <div>
    <div v-if="data && !walkthroughsPending" class="space-y-6">
      <GalgameCharacterHero
        :character="data"
        :voices="voices"
        :work-count-text="workCountText"
        @open-art="openArt"
      />

      <div
        :class="
          cn(
            'grid grid-cols-1 items-start gap-6',
            hasAside && 'lg:grid-cols-[20rem_minmax(0,1fr)]'
          )
        "
      >
        <aside v-if="hasAside" class="order-last space-y-4 lg:order-none">
          <div
            v-if="data.image && data.figure"
            class="hidden justify-center lg:flex"
          >
            <GalgameCharacterArt
              :src="data.figure"
              :alt="data.name"
              :label="`查看 ${data.name} 的立绘`"
              :meta="data.figure_meta"
              :max-width="320"
              :max-height="440"
              @open="openArt('figure')"
            />
          </div>

          <KunCard
            v-if="data.traits.length"
            :is-transparent="false"
            :is-hoverable="false"
            padding="md"
            content-class="gap-3 justify-start"
          >
            <h2 class="text-lg font-semibold">角色属性</h2>
            <GalgameCharacterTraits :traits="data.traits" />
          </KunCard>
        </aside>

        <section class="flex min-w-0 flex-col gap-4">
          <GalgameCharacterWalkthroughs
            :character-id="String(characterId)"
            :walkthroughs="walkthroughs?.items ?? []"
          />

          <KunHeader
            :name="`${data.name}登场的全部 Galgame 作品`"
            description=""
            scale="h2"
          />

          <KunInfo
            v-if="!allowsNsfw"
            color="warning"
            title="部分 Galgame 已隐藏"
            description="当前为 SFW 模式，含 NSFW 内容的 Galgame 不会显示，也不计入数量。如需查看，请在设置面板开启 NSFW 开关。"
          />

          <GalgameCardGridSkeleton v-if="appearancesPending" />

          <GalgameCard
            v-else-if="works.length"
            :galgames="works"
            :is-transparent="false"
          >
            <template #meta="{ galgame }">
              <p
                v-if="galgame.voices?.length"
                class="text-default-500 mt-1 line-clamp-2 text-xs"
                :title="voiceTitle(galgame.voices)"
              >
                CV
                {{
                  galgame.voices.map(getGalgameCharacterVoiceLabel).join(' / ')
                }}
              </p>
            </template>
          </GalgameCard>

          <KunNull v-else description="暂无该角色登场的 Galgame" />

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
            资料来自 鲲 Galgame 论坛的角色图谱。如果有数据错误请
            <KunLink to="/doc/contact" size="sm">联系我们</KunLink>。
          </p>
        </section>
      </div>

      <KunLightbox
        v-model:is-open="isLightboxOpen"
        :images="artImages"
        :initial-index="lightboxIndex"
      />
    </div>
    <GalgameEntityDetailSkeleton
      v-else-if="pending || (data && walkthroughsPending)"
    />
  </div>
</template>
