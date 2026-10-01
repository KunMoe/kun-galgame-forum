<script setup lang="ts">
const {
  showPlatform,
  showRating,
  showViewLike,
  showUpdateTime,
  showNsfwBadge,
  showCompany,
  showJapaneseName,
  isOpenInNewTab
} = storeToRefs(usePersistGalgameCardStore())

const {
  showKUNGalgameNoResource,
  showKUNGalgameAllOriginalLanguages,
  showKUNGalgamePreferOriginalName,
  showKUNGalgamePhoneCardColumns
} = storeToRefs(usePersistSettingsStore())

const phoneColumnOptions = [
  { value: 2, label: '2 张' },
  { value: 3, label: '3 张' }
] as const

// Names are elected by the API, so the switch only takes effect on the next
// request — the same reason the NSFW switch reloads.
const preferOriginalName = ref(showKUNGalgamePreferOriginalName.value)
watch(preferOriginalName, (value) => {
  showKUNGalgamePreferOriginalName.value = value
  location.reload()
})
</script>

<template>
  <div class="space-y-4">
    <div class="space-y-2.5">
      <p class="text-default-700 text-sm font-semibold">名称显示</p>
      <div class="flex items-start justify-between gap-4">
        <div class="space-y-0.5">
          <p class="text-default-700">优先显示日语原名</p>
          <p class="text-default-500 text-sm">
            游戏标题、角色、声优与制作人员、会社、系列等所有名称都改为显示条目本身的原名，中文名退到副标题。开启或关闭后会自动刷新页面。角色属性等译名词表不受影响。
          </p>
        </div>
        <KunSwitch v-model="preferOriginalName" class="shrink-0" />
      </div>
    </div>

    <p class="text-default-500 text-sm">
      以下选项控制 Galgame 列表卡片的展示内容，更改后即时生效于全站所有 Galgame
      列表。
    </p>

    <div class="space-y-2.5">
      <p class="text-default-700 text-sm font-semibold">手机上每行的卡片</p>
      <div class="grid grid-cols-2 gap-2">
        <KunButton
          v-for="opt in phoneColumnOptions"
          :key="opt.value"
          size="sm"
          :variant="
            showKUNGalgamePhoneCardColumns === opt.value ? 'solid' : 'flat'
          "
          :color="
            showKUNGalgamePhoneCardColumns === opt.value ? 'primary' : 'default'
          "
          @click="showKUNGalgamePhoneCardColumns = opt.value"
        >
          {{ opt.label }}
        </KunButton>
      </div>
    </div>

    <div class="space-y-2.5">
      <p class="text-default-700 text-sm font-semibold">封面四角</p>
      <div class="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
        <KunSwitch v-model="showPlatform" label="左上角 · 游戏平台" />
        <KunSwitch v-model="showNsfwBadge" label="右上角 · NSFW 角标" />
        <KunSwitch v-model="showViewLike" label="左下角 · 浏览 / 点赞" />
        <KunSwitch v-model="showUpdateTime" label="右下角 · 更新时间" />
      </div>
    </div>

    <div class="space-y-2.5">
      <p class="text-default-700 text-sm font-semibold">卡片底部</p>
      <div class="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
        <KunSwitch v-model="showCompany" label="左侧 · 制作会社" />
        <KunSwitch v-model="showRating" label="右侧 · 总评分" />
      </div>
    </div>

    <div class="space-y-2.5">
      <p class="text-default-700 text-sm font-semibold">其它</p>
      <div class="grid grid-cols-1 gap-2.5">
        <KunSwitch v-model="showJapaneseName" label="标题下显示另一个名称" />
        <KunSwitch v-model="isOpenInNewTab" label="在新页面打开卡片" />
        <KunSwitch
          v-model="showKUNGalgameNoResource"
          label="排行与动态包含无资源的 Galgame"
        />
        <KunSwitch
          v-model="showKUNGalgameAllOriginalLanguages"
          label="列表显示原语言不是日语或中文的 Galgame"
        />
      </div>
    </div>
  </div>
</template>
