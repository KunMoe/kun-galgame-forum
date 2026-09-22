<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    active?: boolean
    label?: string
    className?: string
  }>(),
  { active: true, label: '成人向内容已模糊', className: 'rounded-lg' }
)

const revealed = ref(false)
const masked = computed(() => props.active && !revealed.value)

// Every reveal is per item and lasts only as long as this render: the stance
// belongs to the account, so uncovering one image must not quietly turn the
// whole site to 显示.
const reveal = () => {
  revealed.value = true
}

watch(
  () => props.active,
  (active) => {
    if (!active) revealed.value = false
  }
)
</script>

<template>
  <div :class="cn('relative', className)">
    <div
      :class="
        cn(
          'h-full transition-[filter] duration-200',
          masked && 'pointer-events-none scale-[1.02] blur-xl select-none'
        )
      "
    >
      <slot />
    </div>

    <!-- role=button rather than a real <button>: every placement so far sits
         inside a card link or a lightbox trigger, and a <button> nested in a
         <button> is hoisted out of it by the HTML parser, which drops the
         overlay somewhere else on the page entirely. -->
    <div
      v-if="masked"
      role="button"
      tabindex="0"
      class="bg-background/30 hover:bg-background/10 absolute inset-0 flex cursor-pointer flex-col items-center justify-center gap-1 rounded-[inherit] backdrop-blur-[3px] transition-colors outline-none focus-visible:ring-2"
      :aria-label="`${label}，点击查看`"
      @click.stop.prevent="reveal"
      @keydown.enter.stop.prevent="reveal"
      @keydown.space.stop.prevent="reveal"
    >
      <KunIcon name="lucide:eye-off" class="text-danger size-6" />
      <span class="text-default-700 px-2 text-center text-xs font-medium">
        {{ label }}
      </span>
      <span class="text-default-500 text-[0.65rem]">点击查看</span>
    </div>
  </div>
</template>
