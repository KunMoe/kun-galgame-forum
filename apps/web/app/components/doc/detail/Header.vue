<script setup lang="ts">
import type { Doc } from '#shared/utils/api/schemas'
import { KUN_DOC_CATEGORY_MAP } from '~/constants/doc'

defineProps<{
  metadata: Doc
}>()
</script>

<template>
  <KunCard :is-hoverable="false" class-name="border-none">
    <div class="relative mb-6 aspect-video h-full w-full">
      <KunLightboxGallery>
        <KunLightboxGalleryItem
          :src="metadata.banner?.url || '/kungalgame.webp'"
          :alt="metadata.title"
          :wrap="false"
          v-slot="{ open }"
        >
          <KunImage
            :alt="metadata.title"
            class="size-full cursor-zoom-in rounded-lg object-cover"
            :src="metadata.banner?.url || '/kungalgame.webp'"
            loading="eager"
            fetchpriority="high"
            width="100%"
            height="100%"
            @click="open"
          />
        </KunLightboxGalleryItem>
      </KunLightboxGallery>
    </div>

    <div class="flex flex-col gap-3">
      <h1 class="text-2xl font-bold tracking-tight sm:text-4xl">
        {{ metadata.title }}
      </h1>

      <div class="flex flex-wrap items-center gap-3 text-sm">
        <KunChip color="secondary">
          {{ KUN_DOC_CATEGORY_MAP[metadata.doc_category] }}
        </KunChip>
        <div class="text-default-500 flex items-center gap-1">
          <KunIcon name="lucide:eye" class="h-4 w-4" />
          <span>{{ metadata.view_count }} 次浏览</span>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <div class="flex flex-col gap-1">
          <div class="text-default-500 flex items-center gap-2">
            <KunIcon name="lucide:calendar-days" />
            <p class="text-small text-inherit">
              <KunTime
                :time="metadata.published_at"
                type="datetime"
                show-year
              />
            </p>
          </div>
        </div>
      </div>

      <div class="bg-primary/10 text-primary-700 rounded-lg p-3 text-sm">
        {{ metadata.description }}
      </div>
    </div>
  </KunCard>
</template>
