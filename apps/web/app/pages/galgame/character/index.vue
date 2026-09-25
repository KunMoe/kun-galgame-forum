<script setup lang="ts">
useKunSeoMeta({
  title: 'Galgame 角色库',
  description:
    '按名字或属性查找 Galgame 角色: 发色、瞳色、发型、服装、性格、身份…… 可以同时组合多个属性, 例如「银发 + 双马尾 + 傲娇」, 找到你喜欢的角色和她们登场的 Galgame。'
})
</script>

<template>
  <div class="flex flex-col gap-6">
    <KunHeader
      name="Galgame 角色库"
      description="按名字或属性查找 Galgame 角色。可以同时选择多个属性, 并切换「全部满足」或「满足任一」。"
    >
      <template #endContent>
        <p class="text-default-500">
          选中一个属性时, 它下面更细的属性也会一起匹配,
          例如「靴子」也会找到穿「过膝靴」的角色。不知道从哪开始?
          点「按分类挑选」, 或去
          <KunLink to="/galgame/trait">角色属性</KunLink>
          页面逛逛。成人向属性需要在设置面板开启 NSFW 开关。如果有数据错误请
          <KunLink to="/doc/contact"> 联系我们 </KunLink>。
        </p>
      </template>
    </KunHeader>

    <GalgameCharacterBrowser />
  </div>
</template>
