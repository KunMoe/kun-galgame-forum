<script setup lang="ts">
defineProps<{
  siblings: { id: number; name: string }[]
}>()
</script>

<template>
  <KunCard
    :is-transparent="false"
    :is-hoverable="false"
    padding="md"
    content-class="gap-4 justify-start"
  >
    <div class="space-y-1">
      <h2 class="text-lg font-semibold">同一人的其他名义</h2>
      <p class="text-default-500 text-xs">
        本页是一个「署名名义」而非人物档案, 同一位创作者可能以多个名义署名。
      </p>
    </div>

    <div class="space-y-2">
      <KunCard
        v-for="sibling in siblings"
        :key="sibling.id"
        :is-transparent="false"
        :is-hoverable="true"
        :href="`/galgame/staff/${sibling.id}`"
        padding="sm"
        content-class="flex-row items-center justify-start gap-2"
      >
        <KunIcon
          name="lucide:user-round"
          class-name="text-default-400 shrink-0"
        />
        <span class="line-clamp-2 min-w-0 text-sm font-medium break-words">
          {{ sibling.name }}
        </span>
      </KunCard>
    </div>
  </KunCard>
</template>
