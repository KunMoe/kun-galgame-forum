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
</script>

<template>
  <KunLink
    :class-name="cn(props.className)"
    target="_blank"
    :to="kungal.ad[0]!.link"
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
