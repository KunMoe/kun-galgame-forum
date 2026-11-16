<script setup lang="ts">
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  KUN_GALGAME_OFFICIAL_LANGUAGE_MAP
} from '~/constants/galgameOfficial'

type CompanyDetail = NonNullable<
  Awaited<ReturnType<typeof useGalgameOfficialDetail>>['data']['value']
>

const props = defineProps<{
  company: CompanyDetail
  workCount: number
  relatedCount: number
}>()

const INTRO_CLAMP_CHARS = 100
const isIntroExpanded = ref(false)

const LINK_PREVIEW = 4
const isLinksExpanded = ref(false)
const hasHiddenLinks = computed(() => props.company.links.length > LINK_PREVIEW)
</script>

<template>
  <KunCard
    :is-transparent="false"
    :is-hoverable="false"
    padding="lg"
    content-class="gap-4 justify-start"
  >
    <div
      :class="
        cn(
          'grid items-start gap-x-4 gap-y-3 sm:gap-x-6',
          company.logo ? 'grid-cols-[auto_minmax(0,1fr)]' : 'grid-cols-1'
        )
      "
    >
      <GalgameOfficialBrandMark
        v-if="company.logo"
        :src="company.logo"
        :name="company.name"
        class="sm:row-span-2"
      />

      <div class="min-w-0 space-y-1 self-center sm:self-start">
        <h1 class="text-xl font-bold break-words sm:text-3xl">
          {{ company.name }}
        </h1>
        <p
          v-if="company.original"
          class="text-default-500 text-sm break-words sm:text-base"
        >
          {{ company.original }}
        </p>
        <p class="text-default-500 text-sm">
          <span class="text-foreground font-semibold tabular-nums">
            {{ workCount }}
          </span>
          部作品
          <template v-if="relatedCount">
            ·
            <span class="text-foreground font-semibold tabular-nums">
              {{ relatedCount }}
            </span>
            家关联会社
          </template>
        </p>
      </div>

      <div class="col-span-full space-y-3 sm:col-span-1 sm:col-start-2">
        <div class="flex flex-wrap items-center gap-2">
          <GalgameOfficialFollowButton
            :company-id="company.id"
            :follower-count="company.follower_count"
          />
          <KunChip color="primary">
            {{
              KUN_GALGAME_OFFICIAL_CATEGORY_MAP[company.category] ||
              company.category
            }}
          </KunChip>
          <KunChip v-if="company.lang" color="secondary">
            {{
              KUN_GALGAME_OFFICIAL_LANGUAGE_MAP[company.lang] || company.lang
            }}
          </KunChip>
          <KunChip v-for="(a, index) in company.alias" :key="index" size="sm">
            {{ a }}
          </KunChip>
        </div>

        <div
          v-if="company.links.length"
          class="flex flex-wrap items-center gap-x-4 gap-y-1"
        >
          <KunLink
            v-for="(link, index) in company.links"
            v-show="isLinksExpanded || index < LINK_PREVIEW"
            :key="link.url"
            :is-show-anchor-icon="true"
            target="_blank"
            rel="noopener noreferrer"
            underline="hover"
            size="sm"
            :to="link.url"
          >
            {{ link.name }}
          </KunLink>
          <KunButton
            v-if="hasHiddenLinks"
            variant="light"
            size="sm"
            color="default"
            @click="isLinksExpanded = !isLinksExpanded"
          >
            <KunIcon
              :name="isLinksExpanded ? 'lucide:chevron-up' : 'lucide:link'"
            />
            {{
              isLinksExpanded ? '收起' : `全部 ${company.links.length} 个链接`
            }}
          </KunButton>
        </div>
      </div>
    </div>

    <div v-if="company.description" class="space-y-1">
      <p
        :class="
          cn(
            'text-default-600 text-sm whitespace-pre-line sm:text-base',
            !isIntroExpanded && 'line-clamp-3'
          )
        "
      >
        {{ company.description }}
      </p>
      <p v-if="company.description_machine" class="text-default-400 text-xs">
        该简介由机器翻译生成
      </p>
      <KunButton
        v-if="company.description.length > INTRO_CLAMP_CHARS"
        variant="light"
        size="sm"
        color="primary"
        @click="isIntroExpanded = !isIntroExpanded"
      >
        <KunIcon
          :name="isIntroExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
        />
        {{ isIntroExpanded ? '收起简介' : '展开简介' }}
      </KunButton>
    </div>
  </KunCard>
</template>
