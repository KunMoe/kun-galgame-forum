<script setup lang="ts">
import { KUN_GALGAME_EXTERNAL_RATING_MAP } from '~/constants/galgame-rating'

const props = defineProps<{ refs?: Record<string, string> }>()

const route = useRoute()
const gid = computed(() => parseInt((route.params as { gid: string }).gid))

const { data } = await useKunFetch<GalgameLink[]>(
  `/galgame/${gid.value}/link/all`,
  {
    lazy: true,
    method: 'GET',
    query: { galgame_id: gid.value },
    watch: false
  }
)

// The identity ids had exactly one route to the surface: the external rating
// panel, which is only drawn for a source that actually returned a score. A
// work nobody has rated on VNDB therefore showed no VNDB link anywhere, while
// 官方网站 / 维基百科 / Wikidata sat right here — "不知道为什么不带 vndb 玩".
// The link templates are the rating map's, so there is still one definition of
// where a source's id points.
const identityLinks = computed(() =>
  (['vndb', 'bangumi', 'erogamescape'] as const).flatMap((source) => {
    const ref = props.refs?.[source]
    const meta = KUN_GALGAME_EXTERNAL_RATING_MAP[source]
    if (!ref || !meta.link) {
      return []
    }
    return [{ label: `${meta.label} ${ref}`, link: meta.link(ref) }]
  })
)
</script>

<template>
  <div
    v-if="identityLinks.length || data?.length"
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
      v-for="(link, index) in data"
      :key="index"
      :to="link.link"
      target="_blank"
      rel="noopener noreferrer"
      size="sm"
      color="default"
      class-name="text-default-500 hover:text-default-700"
    >
      {{ link.name }}
    </KunLink>
  </div>
</template>
