<script setup lang="ts">
import type { GalgameTag } from '#shared/utils/api/schemas'

const props = defineProps<{ tag: GalgameTag; name: string; sfw: boolean }>()

const { blockOf } = useTagBlocks()
const { open } = useSettingPanel()

const block = computed(() =>
  blockOf(
    Number(props.tag.id),
    props.tag.ancestors.map((a) => Number(a.id))
  )
)
const own = computed(() =>
  block.value?.tagId === Number(props.tag.id) ? block.value.mode : null
)
const blockedAncestor = computed(() =>
  own.value
    ? undefined
    : props.tag.ancestors.find((a) => Number(a.id) === block.value?.tagId)
)
</script>

<template>
  <KunInfo v-if="own" color="warning" title="该标签已被屏蔽">
    <p>
      带有「{{ name }}」{{
        own === 'tree' && tag.descendant_count ? '或其子标签' : ''
      }}的 Galgame 不会出现在各处列表中, 所以这里也不会显示。
    </p>
    <div class="mt-2">
      <KunButton
        size="xs"
        variant="flat"
        color="warning"
        @click="open('content')"
      >
        <KunIcon name="lucide:ban" />
        管理屏蔽标签
      </KunButton>
    </div>
  </KunInfo>

  <KunInfo v-else-if="blockedAncestor" color="warning" title="上级标签已被屏蔽">
    <p>
      您屏蔽了「{{ catalogVocabularyName(blockedAncestor) }}」及其子标签, 「{{
        name
      }}」也在其中, 带有它的 Galgame 不会出现在各处列表中, 所以这里也不会显示。
    </p>
    <div class="mt-2">
      <KunButton
        size="xs"
        variant="flat"
        color="warning"
        @click="open('content')"
      >
        <KunIcon name="lucide:ban" />
        管理屏蔽标签
      </KunButton>
    </div>
  </KunInfo>

  <KunInfo
    v-else-if="sfw"
    color="warning"
    title="部分 Galgame 已隐藏"
    description="当前为 SFW 模式，该标签含 NSFW 内容的 Galgame 不会显示。如需查看，请在设置面板开启 NSFW 开关。"
  />
</template>
