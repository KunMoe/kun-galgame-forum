<script setup lang="ts">
import { useRouteQuery } from '@vueuse/router'

useKunSeoMeta({
  title: 'Galgame 角色属性',
  description:
    'Galgame 角色的属性分类一览: 发色、瞳色、体型、服装、饰品、性格、身份与经历。点击属性查看拥有它的全部角色, 或在角色库里组合多个属性筛选角色。'
})

const group = useRouteQuery<string>('group', '', { mode: 'replace' })
</script>

<template>
  <div class="flex flex-col gap-6">
    <KunHeader
      name="Galgame 角色属性"
      description="角色的外貌、服装、性格与身份都被整理成了层层细分的属性, 例如 头发 › 发色 › 银发。点击任意属性, 查看拥有该属性的全部角色。"
    >
      <template #endContent>
        <div class="space-y-3">
          <p class="text-default-500">
            想同时满足多个属性? 去
            <KunLink to="/galgame/character">Galgame 角色库</KunLink>
            组合筛选, 例如「银发 + 双马尾 + 傲娇」。属性数据来自 NextMoe 目录,
            如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>

          <GalgameTraitSearch />
        </div>
      </template>
    </KunHeader>

    <GalgameTraitGroupBrowser v-model:group="group" />
  </div>
</template>
