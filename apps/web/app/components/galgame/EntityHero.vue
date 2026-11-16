<script setup lang="ts">
defineProps<{
  name: string
  subtitle?: string
}>()

const slots = defineSlots<{
  media?: () => unknown
  meta?: () => unknown
  headerEnd?: () => unknown
  actions?: () => unknown
  default?: () => unknown
}>()
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
          slots.media
            ? 'grid-cols-[auto_minmax(0,1fr)] sm:grid-rows-[auto_1fr]'
            : 'grid-cols-1'
        )
      "
    >
      <div v-if="slots.media" class="sm:row-span-2">
        <slot name="media" />
      </div>

      <div
        class="flex min-w-0 flex-wrap items-start justify-between gap-3 self-center sm:self-start"
      >
        <div class="min-w-0 space-y-1">
          <h1 class="text-xl font-bold break-words sm:text-3xl">{{ name }}</h1>
          <p
            v-if="subtitle"
            class="text-default-500 text-sm break-words sm:text-base"
          >
            {{ subtitle }}
          </p>
          <div v-if="slots.meta" class="text-default-500 text-sm">
            <slot name="meta" />
          </div>
        </div>
        <slot name="headerEnd" />
      </div>

      <div
        v-if="slots.actions"
        :class="
          cn(
            'space-y-3',
            slots.media && 'col-span-full sm:col-span-1 sm:col-start-2'
          )
        "
      >
        <slot name="actions" />
      </div>
    </div>

    <slot />
  </KunCard>
</template>
