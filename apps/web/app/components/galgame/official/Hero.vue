<script setup lang="ts">
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  KUN_GALGAME_OFFICIAL_LANGUAGE_MAP
} from '~/constants/galgameOfficial'

type CompanyDetail = NonNullable<
  Awaited<ReturnType<typeof useGalgameOfficialDetail>>['data']['value']
>

defineProps<{
  company: CompanyDetail
  workCount: number
  relatedCount: number
}>()
</script>

<template>
  <GalgameEntityHero :name="company.name" :subtitle="company.original">
    <template v-if="company.logo" #media>
      <GalgameOfficialBrandMark :src="company.logo" :name="company.name" />
    </template>

    <template #meta>
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
    </template>

    <template #headerEnd>
      <GalgameOfficialFollowButton
        :company-id="company.id"
        :follower-count="company.follower_count"
      />
    </template>

    <template #actions>
      <div class="flex flex-wrap items-center gap-2">
        <KunChip color="primary">
          {{
            KUN_GALGAME_OFFICIAL_CATEGORY_MAP[company.category] ||
            company.category
          }}
        </KunChip>
        <KunChip v-if="company.lang" color="secondary">
          {{ KUN_GALGAME_OFFICIAL_LANGUAGE_MAP[company.lang] || company.lang }}
        </KunChip>
        <KunChip v-for="(a, index) in company.alias" :key="index" size="sm">
          {{ a }}
        </KunChip>
      </div>

      <GalgameEntityLinks :links="company.links" />
    </template>

    <GalgameEntityIntro
      v-if="company.description"
      :text="company.description"
      :note="company.description_machine ? '该简介由机器翻译生成' : ''"
    />
  </GalgameEntityHero>
</template>
