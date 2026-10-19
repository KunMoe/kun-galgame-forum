<script setup lang="ts">
import type { CatalogLink } from '#shared/utils/api/schemas'
import { catalogLinkLabel } from '#shared/utils/catalogLink'

const props = defineProps<{
  links: CatalogLink[]
}>()

// The identity ids had exactly one route to the surface: the external rating
// panel, which is only drawn for a source that actually returned a score. A
// work nobody has rated on VNDB therefore showed no VNDB link anywhere, while
// 官方网站 / 维基百科 / Wikidata sat right here — "不知道为什么不带 vndb 玩".
const identityLinks = computed(() =>
  props.links
    .filter((link) => link.is_identity)
    .map((link) => ({
      label: [catalogLinkLabel(link.site, link.url), link.page_id]
        .filter(Boolean)
        .join(' '),
      link: link.url
    }))
)

const catalogLinks = computed(() =>
  props.links
    .filter((link) => !link.is_identity)
    .map((link) => ({
      label: catalogLinkLabel(link.site, link.url),
      link: link.url
    }))
)
</script>

<template>
  <div
    v-if="identityLinks.length || catalogLinks.length"
    class="flex flex-wrap gap-x-3 gap-y-1"
  >
    <KunLink
      v-for="item in identityLinks"
      :key="item.link"
      :to="item.link"
      target="_blank"
      rel="noopener noreferrer"
      size="sm"
      color="default"
      class-name="text-default-600 hover:text-primary font-medium"
    >
      {{ item.label }}
    </KunLink>

    <KunLink
      v-for="item in catalogLinks"
      :key="item.link"
      :to="item.link"
      target="_blank"
      rel="noopener noreferrer"
      size="sm"
      color="default"
      class-name="text-default-500 hover:text-default-700"
    >
      {{ item.label }}
    </KunLink>
  </div>
</template>
