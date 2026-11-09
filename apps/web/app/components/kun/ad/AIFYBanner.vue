<script setup lang="ts">
import { kungal } from '~/config/kungal'

const props = withDefaults(
  defineProps<{
    className?: string
  }>(),
  {
    className: ''
  }
)

const showAd = useAdVisible()
const aifyLink = useAifyLink()
</script>

<template>
  <KunLink
    :class-name="cn('block', props.className)"
    target="_blank"
    :to="aifyLink"
    v-if="showAd"
  >
    <KunImage
      class-name="w-full rounded-lg"
      :src="kungal.ad[0]!.banner"
      :width="1920"
      :height="300"
      aspect-ratio="1920 / 300"
    />
  </KunLink>
</template>
