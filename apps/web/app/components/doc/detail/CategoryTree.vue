<script setup lang="ts">
import type { DocCategory, DocSummary } from '#shared/utils/api/schemas'
import { KUN_DOC_CATEGORIES, KUN_DOC_CATEGORY_MAP } from '~/constants/doc'

const route = useRoute()

const { docs } = await useAllDocs('published_desc')

const expandedCategories = ref<Record<DocCategory, boolean>>({
  galgame: true,
  notice: true,
  kun: true,
  other: true
})

const docsByCategory = computed(() => {
  const grouped: Partial<Record<DocCategory, DocSummary[]>> = {}
  for (const doc of docs.value) {
    ;(grouped[doc.doc_category] ??= []).push(doc)
  }
  return grouped
})

const toggleCategory = (category: DocCategory) => {
  expandedCategories.value[category] = !expandedCategories.value[category]
}
</script>

<template>
  <div class="fixed hidden shrink-0 space-y-1 lg:w-64 xl:block">
    <h3 class="p-3 text-xl font-semibold">文档索引</h3>
    <div class="scrollbar-hide max-h-[calc(100dvh-10rem)] overflow-y-auto">
      <div
        class="space-y-1"
        v-for="category in KUN_DOC_CATEGORIES"
        :key="category"
      >
        <KunButton
          :full-width="true"
          variant="light"
          size="lg"
          @click="toggleCategory(category)"
          class-name="justify-between mb-2"
        >
          <span class="text-foreground">
            {{ KUN_DOC_CATEGORY_MAP[category] }}
          </span>
          <KunIcon
            :name="
              expandedCategories[category]
                ? 'lucide:chevron-down'
                : 'lucide:chevron-right'
            "
          />
        </KunButton>

        <div v-if="expandedCategories[category]" class="ml-4 space-y-1">
          <KunButton
            :full-width="true"
            :variant="
              route.fullPath === `/doc/${article.slug}` ? 'flat' : 'light'
            "
            v-for="article in docsByCategory[category] || []"
            :key="article.id"
            :href="`/doc/${article.slug}`"
            class-name="justify-start text-start"
          >
            <span
              :class="
                cn(
                  'gap-2',
                  route.fullPath === `/doc/${article.slug}`
                    ? ''
                    : 'text-foreground'
                )
              "
            >
              {{ article.title }}
            </span>
          </KunButton>
        </div>
      </div>
    </div>
  </div>
</template>
