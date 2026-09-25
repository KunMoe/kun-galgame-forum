<script setup lang="ts">
import type { Trait } from '#shared/utils/api/schemas'
import { traitFullName, traitSections, traitSide } from '~/utils/galgame/trait'

const route = useRoute()
const traitId = computed(() => Number((route.params as { id: string }).id))

if (!Number.isInteger(traitId.value) || traitId.value <= 0) {
  throw createError({
    statusCode: 404,
    statusMessage: '未找到该角色属性',
    fatal: true
  })
}

const { allowsNsfw, stanceKey } = useContentStance()

const { data: trait } = await useApi<Trait>(
  () => `trait:${traitId.value}:${stanceKey.value}`,
  (api) =>
    api.GET('/traits/{trait_id}', {
      params: {
        path: { trait_id: String(traitId.value) },
        query: { include_nsfw: allowsNsfw.value }
      }
    })
)

if (!trait.value) {
  throw createError({
    statusCode: 404,
    statusMessage: allowsNsfw.value
      ? '未找到该角色属性'
      : '未找到该角色属性, 如果它是成人向属性, 请在设置面板开启 NSFW 开关',
    fatal: true
  })
}

const name = computed(() => catalogVocabularyName(trait.value!))
const fullName = computed(() => traitFullName(trait.value!))
const group = computed(() => catalogVocabularyName(trait.value!.trait_group))
const isRoot = computed(() => !trait.value!.parents.length)
const isGroupParent = computed(() =>
  trait.value!.parents.some((p) => p.id === trait.value!.trait_group_id)
)
const sections = computed(() =>
  traitSections(trait.value!.id, trait.value!.subtraits)
)
const intro = computed(
  () => pickCatalogIntro(trait.value!.intros)?.value || trait.value!.description
)
const aliases = computed(() =>
  trait.value!.aliases.filter((a) => a && a !== name.value)
)

const t = trait.value
if (!t.is_sexual) {
  useKunSeoMeta({
    title: `含有属性 ${fullName.value} 的 Galgame 角色`,
    description: `拥有「${fullName.value}」属性的 Galgame 角色一览, 共 ${t.character_count} 名。可以继续叠加其他属性或按名字筛选, 找到你喜欢的角色。`
  })
} else {
  useKunDisableSeo(`含有属性 ${fullName.value} 的 Galgame 角色`)
}
</script>

<template>
  <div v-if="trait" class="flex flex-col gap-6">
    <KunHeader
      :name="`含有属性「${name}」的 Galgame 角色`"
      :description="
        isRoot
          ? `属性分组 · 共 ${trait.character_count.toLocaleString('en-US')} 名角色`
          : `${group} · 共 ${trait.character_count.toLocaleString('en-US')} 名角色`
      "
    >
      <template #endContent>
        <div class="space-y-3">
          <div
            v-if="trait.parents.length"
            class="flex flex-wrap items-center gap-1.5 text-sm"
          >
            <template v-if="!isGroupParent">
              <span class="text-default-500">分组</span>
              <KunLink
                underline="none"
                :to="`/galgame/trait/${trait.trait_group_id}`"
              >
                <KunChip
                  size="xs"
                  variant="flat"
                  color="primary"
                  class-name="cursor-pointer"
                >
                  {{ group }}
                </KunChip>
              </KunLink>
            </template>
            <span class="text-default-500">上级属性</span>
            <KunLink
              v-for="parent in trait.parents"
              :key="parent.id"
              underline="none"
              :to="`/galgame/trait/${parent.id}`"
            >
              <KunChip size="xs" variant="flat" class-name="cursor-pointer">
                {{ catalogVocabularyName(parent) }}
                <span v-if="traitSide(parent)" class="opacity-60">
                  · {{ traitSide(parent) }}
                </span>
              </KunChip>
            </KunLink>
          </div>

          <p v-if="aliases.length" class="text-default-500 text-sm">
            别名: {{ aliases.join(' / ') }}
          </p>

          <p v-if="intro" class="text-default-600 text-sm whitespace-pre-line">
            {{ intro }}
          </p>

          <p class="text-default-500 text-sm">
            拥有更细分属性的角色也算在内,
            例如「靴子」包括穿「过膝靴」的角色。剧透性质的属性不参与匹配。想组合更多属性?
            在下面继续添加, 或去
            <KunLink to="/galgame/character">角色库</KunLink>
            。数据来自 NextMoe 目录, 如果有错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>
        </div>
      </template>
    </KunHeader>

    <section v-if="sections.length" class="space-y-2">
      <h2 class="text-default-700 text-sm font-semibold">细分属性</h2>
      <KunScrollShadow axis="vertical" class-name="max-h-64">
        <div class="flex flex-wrap gap-x-4 gap-y-2">
          <div
            v-for="section in sections"
            :key="section.trait.id"
            class="flex flex-wrap items-center gap-1.5"
          >
            <GalgameTraitChip :trait="section.trait" is-heading />
            <GalgameTraitChip
              v-for="child in section.children"
              :key="child.id"
              :trait="child"
            />
          </div>
        </div>
      </KunScrollShadow>
    </section>

    <GalgameCharacterBrowser :pinned="trait" />
  </div>
</template>
