<script setup lang="ts">
import { KUN_GALGAME_EXTERNAL_RATING_MAP } from '~/constants/galgame-rating'
import type { CatalogLink, WorkExternalRef } from '#shared/utils/api/schemas'
import { catalogLinkLabel } from '#shared/utils/catalogLink'
import { workExternalId } from '#shared/utils/workExternalRef'

const props = defineProps<{
  links: CatalogLink[]
  externalRefs: WorkExternalRef[]
}>()

// The identity ids had exactly one route to the surface: the external rating
// panel, which is only drawn for a source that actually returned a score. A
// work nobody has rated on VNDB therefore showed no VNDB link anywhere, while
// 官方网站 / 维基百科 / Wikidata sat right here — "不知道为什么不带 vndb 玩".
// The link templates are the rating map's, so there is still one definition of
// where a source's id points.
const identityLinks = computed(() =>
  (['vndb', 'bangumi', 'erogamescape'] as const).flatMap((source) => {
    const id = workExternalId(props.externalRefs, source)
    const meta = KUN_GALGAME_EXTERNAL_RATING_MAP[source]
    if (!id || !meta.link) {
      return []
    }
    return [{ label: `${meta.label} ${id}`, link: meta.link(id) }]
  })
)

const catalogLinks = computed(() =>
  props.links.map((link) => ({
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
