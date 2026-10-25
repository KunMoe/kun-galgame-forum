<script setup lang="ts">
import type { Walkthrough, Work } from '#shared/utils/api/schemas'
import { KUN_GALGAME_WALKTHROUGH_SOURCE } from '~/constants/galgameWalkthrough'
import { workReleaseText } from '~/utils/galgame/releaseDate'

const props = defineProps<{
  walkthrough: Walkthrough
  work: Work
}>()

const nameOf = useCatalogName()
const workName = useWorkName()

const name = computed(() => workName(props.work))
const original = computed(() => {
  const o = nameOf(props.work).original
  return o && o !== name.value ? o : ''
})

const source = computed(
  () =>
    KUN_GALGAME_WALKTHROUGH_SOURCE[props.walkthrough.source_site] ?? {
      name: props.walkthrough.source_site,
      home: props.walkthrough.source_page_url
    }
)

const stats = computed(() => [
  {
    icon: 'lucide:flag',
    text: `${props.walkthrough.ending_count} 个结局`
  },
  { icon: 'lucide:save', text: `${props.walkthrough.save_count} 个存档点` },
  {
    icon: 'lucide:users',
    text: `${props.walkthrough.chart_characters.length} 名角色`
  }
])
</script>

<template>
  <header class="space-y-4">
    <div class="flex items-start gap-4 sm:gap-6">
      <KunLink
        :to="`/galgame/${work.id}`"
        underline="none"
        class-name="w-24 shrink-0 sm:w-36"
      >
        <KunImage
          v-if="work.cover"
          :src="work.cover.url"
          :alt="name"
          :thumbhash="work.cover.thumbhash ?? undefined"
          loading="eager"
          fetchpriority="high"
          aspect-ratio="3/4"
          object-fit="cover"
          class-name="w-full overflow-hidden rounded-lg"
        />
        <div v-else class="bg-default-100 aspect-[3/4] w-full rounded-lg" />
      </KunLink>

      <div class="flex min-w-0 flex-1 flex-col gap-3">
        <div class="space-y-1">
          <h1 class="text-foreground text-xl font-bold break-words sm:text-3xl">
            {{ name }} 攻略
          </h1>
          <p v-if="original" class="text-default-500 text-sm break-words">
            {{ original }}
          </p>
        </div>

        <dl class="flex flex-wrap gap-x-6 gap-y-2 text-sm">
          <div v-if="work.companies.length" class="flex min-w-0 gap-2">
            <dt class="text-default-500 shrink-0">制作会社</dt>
            <dd class="flex min-w-0 flex-wrap gap-x-2">
              <KunLink
                v-for="company in work.companies"
                :key="company.id"
                :to="`/galgame/official/${company.id}`"
                size="sm"
              >
                {{ nameOf(company).name }}
              </KunLink>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="text-default-500 shrink-0">发售日期</dt>
            <dd class="text-default-700">
              {{
                workReleaseText(work.release_date, work.release_date_precision)
              }}
            </dd>
          </div>
        </dl>

        <ul class="text-default-600 flex flex-wrap gap-x-4 gap-y-1 text-sm">
          <li
            v-for="stat in stats"
            :key="stat.icon"
            class="flex items-center gap-1.5"
          >
            <KunIcon :name="stat.icon" class-name="text-default-400 size-4" />
            {{ stat.text }}
          </li>
        </ul>

        <div class="mt-auto flex flex-wrap gap-2">
          <KunButton :href="`/galgame/${work.id}`" variant="flat" size="sm">
            <KunIcon name="lucide:arrow-left" />
            返回游戏页面
          </KunButton>
          <KunButton
            :href="walkthrough.source_page_url"
            target="_blank"
            variant="light"
            size="sm"
          >
            <KunIcon name="lucide:external-link" />
            查看原攻略页面
          </KunButton>
        </div>
      </div>
    </div>

    <p class="text-default-500 text-xs leading-relaxed">
      选项流程整理自
      <KunLink size="sm" :to="walkthrough.source_page_url" target="_blank">
        {{ source.name }}
      </KunLink>
      <template v-if="walkthrough.source_author">
        (作者 {{ walkthrough.source_author }})</template
      >
      <template v-if="walkthrough.source_updated_date"
        >, 原页面更新于 {{ walkthrough.source_updated_date }}</template
      >。只保留了选项与存档结构, 原文说明请查看原页面。选项的中文为机器翻译,
      游戏内显示的是日文原文; 没有可靠译名的人名保持原文。
    </p>
  </header>
</template>
