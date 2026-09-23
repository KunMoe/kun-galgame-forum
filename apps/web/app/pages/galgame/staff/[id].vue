<script setup lang="ts">
import type { CreditList, CreditName, Credit } from '#shared/utils/api/schemas'
import { mergedInto } from '#shared/utils/api/merged'
import { settle } from '#shared/utils/api/problem'
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
const { allowsNsfw } = useContentStance()

let movedTo: number | null = null
const { data: person } = await useApi<CreditName>(
  () => `credit-name:${staffId.value}`,
  async (client) => {
    const res = await client.GET('/credit-names/{credit_name_id}', {
      params: { path: { credit_name_id: String(staffId.value) } }
    })
    movedTo = mergedInto(res.error)
    return res
  }
)

if (movedTo) {
  await navigateTo(`/galgame/staff/${movedTo}`, {
    redirectCode: 301,
    replace: true
  })
} else if (!person.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该制作人员',
    fatal: true
  })
}

const creditsQuery = (cursor?: string) => ({
  limit: PAGE_SIZE,
  include_nsfw: allowsNsfw.value,
  ...(cursor ? { cursor } : {})
})

const { data: firstPage } = await useApi<CreditList>(
  () => `credit-name-credits:${staffId.value}:${allowsNsfw.value}`,
  (client) =>
    client.GET('/credit-names/{credit_name_id}/credits', {
      params: {
        path: { credit_name_id: String(staffId.value) },
        query: creditsQuery()
      }
    })
)

const credits = ref<Credit[]>([...(firstPage.value?.items ?? [])])
const nextCursor = ref<string | null>(firstPage.value?.next_cursor ?? null)
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
  credits.value.push(...res.data.items)
  nextCursor.value = res.data.next_cursor ?? null
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
    siblings: p.siblings.map((s) => ({ id: Number(s.id), name: nameOf(s).name })),
    links: p.links.map((l) => ({
      source: l.site,
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

const staff = data.value
if (staff) {
  useKunSeoMeta({
    title: `${staff.name} 参与制作的 Galgame`,
    description:
      staff.intro ||
      `${staff.name} 在本站收录的 Galgame 中担任 ${staff.roles.join(' / ')} 等职位的作品一览。`,
    ogCard: { kind: 'staff', id: staff.id }
  })
}
</script>

<template>
  <div v-if="data" class="space-y-6">
    <KunHeader :name="data.name" :description="subtitle">
      <template v-if="data.photo" #headerEndContent>
        <KunImage
          :src="data.photo"
          :alt="data.name"
          loading="eager"
          object-fit="cover"
          class-name="w-28 shrink-0 rounded-lg sm:w-32"
          :style="{ aspectRatio: '3/4' }"
        />
      </template>

      <template #endContent>
        <div class="space-y-3">
          <div
            v-if="genderText || birthdayText"
            class="text-default-500 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm"
          >
            <span v-if="genderText">性别 {{ genderText }}</span>
            <span v-if="birthdayText">生日 {{ birthdayText }}</span>
          </div>

          <div v-if="data.intro" class="space-y-1">
            <p class="text-default-600">{{ data.intro }}</p>
            <p v-if="data.intro_machine" class="text-default-400 text-xs">
              该简介由机器翻译生成
            </p>
          </div>

          <div v-if="data.roles.length" class="flex flex-wrap gap-2">
            <KunChip v-for="role in data.roles" :key="role" color="primary">
              {{ role }}
            </KunChip>
          </div>

          <div v-if="data.siblings.length" class="space-y-1">
            <p class="text-default-500 text-sm">同一人的其他名义</p>
            <div class="flex flex-wrap gap-x-4 gap-y-1">
              <KunLink
                v-for="sibling in data.siblings"
                :key="sibling.id"
                :to="`/galgame/staff/${sibling.id}`"
                underline="none"
                class-name="text-foreground hover:text-primary font-medium"
              >
                {{ sibling.name }}
              </KunLink>
            </div>
          </div>

          <div
            v-if="data.links.length"
            class="flex flex-wrap items-center gap-3"
          >
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
                <KunIcon name="lucide:external-link" class="inline size-3" />
              </KunLink>
              <span v-else class="text-default-400 text-sm">{{
                link.name
              }}</span>
            </template>
          </div>

          <p class="text-default-500 text-sm">
            资料来自 NextMoe 目录的署名图谱。本页是一个「署名名义」而非人物档案,
            同一位创作者可能以多个名义署名。默认仅显示 SFW 的 Galgame, 查看 NSFW
            Galgame 请在设置面板打开 NSFW 开关。如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>
        </div>
      </template>
    </KunHeader>

    <GalgameCard v-if="works.length" :galgames="works" :is-transparent="false">
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
        加载更多作品
      </KunButton>
    </div>
  </div>
</template>
