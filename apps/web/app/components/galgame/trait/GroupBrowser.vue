<script setup lang="ts">
import type { Trait, TraitPage, TraitSummary } from '#shared/utils/api/schemas'
import { traitSections } from '~/utils/galgame/trait'

const props = withDefaults(
  defineProps<{
    /** Set, a chip picks its trait instead of opening its page. */
    pickedIds?: number[] | null
  }>(),
  { pickedIds: null }
)

const emit = defineEmits<{ pick: [trait: TraitSummary] }>()

const group = defineModel<string>('group', { default: '' })

const { allowsNsfw, stanceKey } = useContentStance()

const { data: roots } = await useApi<TraitPage>(
  () => `trait-roots:${stanceKey.value}`,
  (api) =>
    api.GET('/traits', {
      params: { query: { include_nsfw: allowsNsfw.value } }
    })
)

const activeId = computed(() => {
  const items = roots.value?.items ?? []
  return items.some((r) => r.id === group.value)
    ? group.value
    : (items[0]?.id ?? '')
})

const tabs = computed(() =>
  (roots.value?.items ?? []).map((r) => ({
    value: r.id,
    textValue: catalogVocabularyName(r)
  }))
)

const { data: tree, status } = await useApi<Trait>(
  () => `trait-tree:${activeId.value}:${stanceKey.value}`,
  (api) =>
    api.GET('/traits/{trait_id}', {
      params: {
        path: { trait_id: activeId.value },
        query: { include_nsfw: allowsNsfw.value }
      }
    })
)

const sections = computed(() =>
  tree.value ? traitSections(tree.value.id, tree.value.subtraits) : []
)
const branches = computed(() => sections.value.filter((s) => s.children.length))
const leaves = computed(() =>
  sections.value.filter((s) => !s.children.length).map((s) => s.trait)
)

const isPickMode = computed(() => props.pickedIds !== null)
const isPicked = (t: TraitSummary) => !!props.pickedIds?.includes(Number(t.id))
</script>

<template>
  <div class="space-y-3">
    <KunTab
      v-if="tabs.length"
      :model-value="activeId"
      :items="tabs"
      variant="underlined"
      color="primary"
      size="sm"
      @update:model-value="group = String($event)"
    />

    <KunLoading :loading="status === 'pending'">
      <div v-if="sections.length" class="space-y-4">
        <div v-if="leaves.length" class="flex flex-wrap gap-1.5">
          <GalgameTraitChip
            v-for="trait in leaves"
            :key="trait.id"
            :trait="trait"
            :pick-mode="isPickMode"
            :picked="isPicked(trait)"
            @pick="emit('pick', trait)"
          />
        </div>

        <div
          class="grid grid-cols-1 items-start gap-3 md:grid-cols-2 xl:grid-cols-3"
          v-if="branches.length"
        >
          <div
            v-for="section in branches"
            :key="section.trait.id"
            class="border-default-200 space-y-2 rounded-lg border p-3"
          >
            <GalgameTraitChip
              :trait="section.trait"
              :pick-mode="isPickMode"
              :picked="isPicked(section.trait)"
              is-heading
              @pick="emit('pick', section.trait)"
            />
            <div class="flex flex-wrap gap-1.5">
              <GalgameTraitChip
                v-for="trait in section.children"
                :key="trait.id"
                :trait="trait"
                :pick-mode="isPickMode"
                :picked="isPicked(trait)"
                @pick="emit('pick', trait)"
              />
            </div>
          </div>
        </div>
      </div>

      <KunNull v-else description="这个分组下暂无属性" />
    </KunLoading>

    <p v-if="!allowsNsfw" class="text-default-400 text-xs">
      当前为 SFW 模式, 成人向属性与「主动(性) / 被动(性)」分组已隐藏。如需查看,
      请在设置面板开启 NSFW 开关。
    </p>
  </div>
</template>
