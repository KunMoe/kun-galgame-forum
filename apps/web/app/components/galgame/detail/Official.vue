<script setup lang="ts">
import {
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP,
  KUN_GALGAME_OFFICIAL_LANGUAGE_MAP,
  KUN_GALGAME_OFFICIAL_ROLE_CATEGORY_SYNONYM,
  KUN_GALGAME_OFFICIAL_ROLE_MAP
} from '~/constants/galgameOfficial'
import type { WorkCompany } from '#shared/utils/api/schemas'

defineProps<{
  companies: WorkCompany[]
}>()

const nameOf = useCatalogName()

const getCategoryText = (category: string) =>
  KUN_GALGAME_OFFICIAL_CATEGORY_MAP[category] || category

const getRoleText = (role: string) =>
  KUN_GALGAME_OFFICIAL_ROLE_MAP[role] || role

const showCategory = (item: WorkCompany) =>
  !item.attribution_roles.some(
    (role) =>
      KUN_GALGAME_OFFICIAL_ROLE_CATEGORY_SYNONYM[role] === item.company_kind
  )

const officialSite = (item: WorkCompany) =>
  item.links.find((link) => link.site === 'official_site')?.url ?? ''
</script>

<template>
  <div>
    <dt class="text-default-500 text-sm font-medium">制作方</dt>
    <dd class="mt-1.5 space-y-3">
      <div class="space-y-2" v-for="item in companies" :key="item.id">
        <KunLink
          :to="`/galgame/official/${item.id}`"
          underline="none"
          class-name="text-foreground hover:text-primary text-base font-semibold"
        >
          {{ nameOf(item).name }}
          <KunTooltip
            v-if="item.catalog_work_count > 0"
            :text="`该会社制作了 ${item.catalog_work_count} 个 Galgame`"
          >
            <KunChip size="xs">
              {{ `+ ${item.catalog_work_count}` }}
            </KunChip>
          </KunTooltip>
        </KunLink>

        <div class="mt-1 flex items-center justify-between">
          <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
            <KunTooltip
              v-for="role in item.attribution_roles"
              :key="role"
              text="该会社在本作中担任的角色"
            >
              <KunChip size="xs" class-name="rounded-md" color="primary">
                {{ getRoleText(role) }}
              </KunChip>
            </KunTooltip>
            <KunTooltip v-if="showCategory(item)" text="该会社自身的类型">
              <KunChip size="xs" class-name="rounded-md" color="default">
                {{ getCategoryText(item.company_kind) }}
              </KunChip>
            </KunTooltip>
            <span class="text-default-500 dark:text-default-400 text-xs">
              {{
                KUN_GALGAME_OFFICIAL_LANGUAGE_MAP[item.lang ?? ''] ||
                item.lang
              }}
            </span>
          </div>

          <KunLink
            v-if="officialSite(item)"
            :is-show-anchor-icon="true"
            target="_blank"
            :to="officialSite(item)"
            size="sm"
            underline="hover"
            rel="noopener noreferrer"
          >
            官方网站
          </KunLink>
          <span v-else class="text-default-400 text-xs"> 暂无官网 </span>
        </div>

        <div
          v-if="item.aliases.length"
          class="text-default-500 flex flex-wrap gap-2"
        >
          <KunChip
            size="xs"
            color="success"
            v-for="(a, index) in item.aliases"
            :key="index"
          >
            {{ a }}
          </KunChip>
        </div>
      </div>
    </dd>
  </div>
</template>
