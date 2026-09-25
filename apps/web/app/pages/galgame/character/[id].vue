<script setup lang="ts">
import {
  getGalgameCharacterLangName,
  getGalgameCharacterIntroCredit
} from '~/constants/galgameCharacter'
import type {
  Appearance,
  AppearanceList,
  Character
} from '#shared/utils/api/schemas'
import { mergedInto } from '#shared/utils/api/merged'
import { settle } from '#shared/utils/api/problem'
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

let movedTo: number | null = null
const { data: character } = await useApi<Character>(
  () => `character:${characterId.value}:${stanceKey.value}`,
  async (client) => {
    const res = await client.GET('/characters/{character_id}', {
      params: {
        path: { character_id: String(characterId.value) },
        query: { include_nsfw: allowsNsfw.value }
      }
    })
    movedTo = mergedInto(res.error)
    return res
  }
)

if (movedTo) {
  await navigateTo(`/galgame/character/${movedTo}`, {
    redirectCode: 301,
    replace: true
  })
} else if (!character.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该角色',
    fatal: true
  })
}

const appearancesQuery = (cursor?: string) => ({
  limit: PAGE_SIZE,
  include_nsfw: allowsNsfw.value,
  ...(cursor ? { cursor } : {})
})

const { data: firstPage } = await useApi<AppearanceList>(
  () => `character-appearances:${characterId.value}:${stanceKey.value}`,
  (client) =>
    client.GET('/characters/{character_id}/appearances', {
      params: {
        path: { character_id: String(characterId.value) },
        query: appearancesQuery()
      }
    })
)

const appearances = ref<Appearance[]>([...(firstPage.value?.items ?? [])])
const nextCursor = ref<string | null>(firstPage.value?.next_cursor ?? null)
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
  appearances.value.push(...res.data.items)
  nextCursor.value = res.data.next_cursor ?? null
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

const voiceLabel = (voice: GalgameDetailCharacterVoice) =>
  voice.lang && voice.lang.toLowerCase() !== 'ja'
    ? `${voice.name}（${getGalgameCharacterLangName(voice.lang)}）`
    : voice.name

const voiceTitle = (voices: GalgameDetailCharacterVoice[]) =>
  voices.map((v) => v.latin || v.name).join(' / ')

const subtitle = computed(() => {
  const parts = [data.value?.name_original, data.value?.latin].filter(
    (part): part is string => !!part && part !== data.value?.name
  )
  return parts.join(' · ')
})

const intros = computed(() => {
  const all = data.value?.intros ?? []
  const lead = all.find((i) => i.intro === data.value?.intro)
  return lead ? [lead, ...all.filter((i) => i !== lead)] : all
})
const introTabs = computed(() =>
  intros.value.map((i) => ({
    value: i.lang,
    textValue: getGalgameCharacterLangName(i.lang)
  }))
)
const introLang = ref(intros.value[0]?.lang ?? '')
watch(intros, (rows) => {
  if (!rows.some((i) => i.lang === introLang.value)) {
    introLang.value = rows[0]?.lang ?? ''
  }
})

const artImages = computed(() =>
  [data.value?.figure, data.value?.image]
    .filter((src): src is string => !!src)
    .map((src) => ({ src, alt: data.value?.name ?? '' }))
)
const bustIndex = computed(() => (data.value?.figure ? 1 : 0))
const isLightboxOpen = ref(false)
const lightboxIndex = ref(0)
const openArt = (index: number) => {
  lightboxIndex.value = index
  isLightboxOpen.value = true
}

const view = data.value
if (view) {
  useKunSeoMeta({
    title: `${view.name} 登场的 Galgame`,
    description:
      view.intro ||
      `角色 ${view.name} 在本站收录的 Galgame 中的登场作品与配音演员一览。`,
    ogCard: { kind: 'character', id: view.id }
  })
}
</script>

<template>
  <div v-if="data" class="space-y-8">
    <div class="flex flex-col gap-8 lg:flex-row lg:items-start">
      <aside
        v-if="data.figure || data.image"
        class="hidden shrink-0 flex-col items-center gap-3 lg:sticky lg:top-24 lg:flex"
      >
        <GalgameCharacterArt
          v-if="data.figure"
          :src="data.figure"
          :alt="data.name"
          :label="`查看 ${data.name} 的立绘`"
          :meta="data.figure_meta"
          :max-width="280"
          :max-height="520"
          @open="openArt(0)"
        />
        <GalgameCharacterArt
          v-if="data.image"
          :src="data.image"
          :alt="data.name"
          :label="`查看 ${data.name} 的头像`"
          :meta="data.image_meta"
          :max-width="data.figure ? 112 : 240"
          :max-height="data.figure ? 132 : 300"
          @open="openArt(bustIndex)"
        />
      </aside>

      <div class="min-w-0 flex-1 space-y-8">
        <div class="flex items-start gap-4">
          <GalgameCharacterArt
            v-if="data.image"
            class="lg:hidden"
            :src="data.image"
            :alt="data.name"
            :label="`查看 ${data.name} 的头像`"
            :meta="data.image_meta"
            :max-width="96"
            :max-height="128"
            @open="openArt(bustIndex)"
          />

          <div class="min-w-0 flex-1">
            <KunHeader :name="data.name" :description="subtitle">
              <template #endContent>
                <div class="flex flex-wrap items-center gap-3">
                  <KunButton
                    v-if="data.figure"
                    class-name="lg:hidden"
                    variant="flat"
                    color="primary"
                    size="sm"
                    @click="openArt(0)"
                  >
                    <KunIcon name="lucide:user-round" />
                    查看立绘
                  </KunButton>
                  <template v-for="link in data.links" :key="link.source">
                    <KunLink
                      v-if="link.url"
                      :to="link.url"
                      target="_blank"
                      rel="noopener noreferrer"
                      size="sm"
                      color="default"
                      class-name="text-default-500 hover:text-default-700"
                    >
                      {{ link.name }}
                      <KunIcon
                        name="lucide:external-link"
                        class="inline size-3"
                      />
                    </KunLink>
                    <span v-else class="text-default-400 text-sm">
                      {{ link.name }}
                    </span>
                  </template>
                </div>
              </template>
            </KunHeader>
          </div>
        </div>

        <section v-if="intros.length" class="space-y-3">
          <div class="flex flex-wrap items-end justify-between gap-2">
            <h2 class="text-foreground text-lg font-semibold">角色简介</h2>
            <KunTab
              v-if="introTabs.length > 1"
              v-model="introLang"
              :items="introTabs"
              name="character-intro"
              variant="underlined"
              size="sm"
            />
          </div>
          <KunTabPanels v-model="introLang" name="character-intro">
            <KunTabPanel
              v-for="intro in intros"
              :key="intro.lang"
              :value="intro.lang"
              class-name="space-y-2"
            >
              <p class="text-default-700 leading-relaxed whitespace-pre-line">
                {{ intro.intro }}
              </p>
              <p
                v-if="getGalgameCharacterIntroCredit(intro)"
                class="text-default-400 text-xs"
              >
                {{ getGalgameCharacterIntroCredit(intro) }}
              </p>
            </KunTabPanel>
          </KunTabPanels>
        </section>

        <section v-if="data.traits.length" class="space-y-3">
          <h2 class="text-foreground text-lg font-semibold">角色属性</h2>
          <GalgameCharacterTraits :traits="data.traits" />
        </section>

        <p class="text-default-500 text-sm">
          资料来自 NextMoe 目录的角色图谱。默认仅显示 SFW 的 Galgame, 查看 NSFW
          Galgame 请在设置面板打开 NSFW 开关。如果有数据错误请
          <KunLink to="/doc/contact"> 联系我们 </KunLink>。
        </p>
      </div>
    </div>

    <section class="space-y-3">
      <h2 class="text-foreground text-lg font-semibold">登场作品</h2>

      <GalgameCard
        v-if="works.length"
        :galgames="works"
        :is-transparent="false"
      >
        <template #meta="{ galgame }">
          <p
            v-if="galgame.voices?.length"
            class="text-default-500 mt-1 line-clamp-2 text-xs"
            :title="voiceTitle(galgame.voices)"
          >
            CV {{ galgame.voices.map(voiceLabel).join(' / ') }}
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
          加载更多作品
        </KunButton>
      </div>
    </section>

    <KunLightbox
      v-model:is-open="isLightboxOpen"
      :images="artImages"
      :initial-index="lightboxIndex"
    />
  </div>
</template>
