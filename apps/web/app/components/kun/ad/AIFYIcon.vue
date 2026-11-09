<script setup lang="ts">
import { kungal } from '~/config/kungal'
import { storeToRefs } from 'pinia'

const showAd = useAdVisible()
const aifyLink = useAifyLink()
const { id } = storeToRefs(usePersistUserStore())
</script>

<template>
  <KunTooltip
    :class-name="cn('flex', id && 'hidden desktop-nav:flex')"
    text="听说现在的 AI 比人还要 H ?"
    position="bottom"
    v-if="showAd"
  >
    <KunLink target="_blank" :to="aifyLink">
      <KunImage
        class-name="block h-11"
        :src="kungal.ad[0]!.icon"
        :width="500"
        :height="193"
        aspect-ratio="500 / 193"
      />
    </KunLink>
  </KunTooltip>
</template>
