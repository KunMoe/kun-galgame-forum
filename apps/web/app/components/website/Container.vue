<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { WebsiteSummary } from '#shared/utils/api/schemas'
import type { WebsiteForm } from '~/components/website/modal/types'

const api = useApiClient()
const createKey = useIdempotencyKey()
const { data, refresh } = await useWebsiteList()
const { data: categories } = await useWebsiteCategories()

const canCreateWebsite = useCan('website.create')
const canManageTaxonomy = useCan('website.edit')
const searchQuery = ref('')
const showWebsiteModal = ref(false)
const isSubmitting = ref(false)

const matchedWebsites = computed(() => {
  const sites = data.value ?? []
  const query = searchQuery.value.toLowerCase()
  if (query === '') {
    return sites
  }
  return sites.filter(
    (site) =>
      site.title.toLowerCase().includes(query) ||
      site.description.toLowerCase().includes(query) ||
      site.host.includes(query)
  )
})

const closedWebsites = computed(() =>
  matchedWebsites.value.filter((site) => site.state === 'closed').sort(byScore)
)

const categorizedWebsites = computed(() => {
  const byCategory = new Map<string, WebsiteSummary[]>()
  for (const site of matchedWebsites.value) {
    if (site.state === 'closed') {
      continue
    }
    const key = site.website_category.id
    byCategory.set(key, [...(byCategory.get(key) ?? []), site])
  }

  return (categories.value ?? [])
    .map((category) => ({
      key: category.id,
      name: category.label || category.slug,
      sites: (byCategory.get(category.id) ?? []).sort(byScore)
    }))
    .filter((group) => group.sites.length > 0)
})

const openCreateWebsiteModal = () => {
  showWebsiteModal.value = true
}

// The modal deliberately does not close itself: a rejected submit (a duplicate
// name, a failed upload) used to wipe the whole form on its way out, so the
// only way to retry was to fill it in again from scratch.
const handleCreateWebsite = async (form: WebsiteForm) => {
  isSubmitting.value = true
  const result = await settle(
    api.POST('/admin/websites', {
      params: {
        header: { 'Idempotency-Key': createKey.take('/admin/websites', form) }
      },
      body: form
    })
  )
  isSubmitting.value = false

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  createKey.clear()
  useMessage('创建网站成功', 'success')
  showWebsiteModal.value = false
  await refresh()
}
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="Galgame 网站资料库"
      description="世界上最全的 Galgame 资源网站, 社区网站, 论坛网站, 资讯网站, Galgame 资料库, Telegram 社群 等 Galgame 网站汇总, 仅收录 Galgame 相关的网站。本资料库仅会收录免费网站, 并且严格禁止任何有付费行为的网站。"
    >
      <template #endContent>
        <div class="space-y-3">
          <p class="text-default-500">
            当前本页面正在不断更新中, 默认仅显示 SFW 的网站, 查看 NSFW
            网站请在设置面板打开 NSFW 开关。
          </p>
          <p class="text-default-500">
            关于 Galgame 网站资料库请查看
            <KunLink to="/doc/galgame-website-wiki">
              Galgame 网站资料库
            </KunLink>
            , 如果有数据错误请
            <KunLink to="/doc/contact"> 联系我们 </KunLink>。
          </p>
          <KunInput
            v-model="searchQuery"
            type="text"
            placeholder="搜索 Galgame 网站"
          />

          <div class="flex flex-wrap justify-end gap-3">
            <KunButton
              v-if="canManageTaxonomy"
              variant="light"
              href="/admin/website"
            >
              分类与标签管理
            </KunButton>
            <KunButton v-if="canCreateWebsite" @click="openCreateWebsiteModal">
              创建新网站
            </KunButton>
          </div>
        </div>
      </template>
    </KunHeader>

    <WebsiteModalWebsite
      v-model="showWebsiteModal"
      :is-editing="false"
      :loading="isSubmitting"
      @submit="handleCreateWebsite"
    />

    <div v-for="categoryGroup in categorizedWebsites" :key="categoryGroup.key">
      <div class="mb-3 flex items-center space-x-3">
        <h2 class="text-default-900 text-2xl">
          {{ categoryGroup.name }}
        </h2>
        <KunChip> {{ categoryGroup.sites.length }} 个网站 </KunChip>
      </div>

      <div
        class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      >
        <WebsiteCard
          v-for="website in categoryGroup.sites"
          :key="website.id"
          :website="website"
        />
      </div>
    </div>

    <div v-if="closedWebsites.length" class="border-default-200 border-t pt-6">
      <div class="mb-3 flex items-center space-x-3">
        <h2 class="text-default-500 text-2xl">已关站</h2>
        <KunChip color="danger"> {{ closedWebsites.length }} 个网站 </KunChip>
      </div>
      <p class="text-default-500 mb-3 text-sm">
        这些网站已经停止运营, 仅作存档保留, 链接大概率已经失效。
      </p>

      <div
        class="grid grid-cols-1 gap-3 opacity-75 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      >
        <WebsiteCard
          v-for="website in closedWebsites"
          :key="website.id"
          :website="website"
        />
      </div>
    </div>
  </div>
</template>
