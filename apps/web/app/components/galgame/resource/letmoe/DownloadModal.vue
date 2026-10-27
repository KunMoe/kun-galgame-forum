<script setup lang="ts">
import {
  LETMOE_PROVIDER_LABEL_MAP,
  letmoeFeedbackUrl
} from '~/constants/galgameResource'
import { settle } from '#shared/utils/api/problem'
import type {
  LetmoeResource,
  LetmoeResourceDownload
} from '#shared/utils/api/schemas'

const props = defineProps<{
  resource: LetmoeResource
  mode: 'direct' | 'netdisk'
}>()

const open = defineModel<boolean>({ required: true })

const api = useApiClient()
const download = ref<LetmoeResourceDownload | null>(null)
const isFetching = ref(false)

// Each answer counts one download on letmoe, so the links are asked for once
// per page view, when the reader opens this.
const fetchDownload = async () => {
  if (download.value || isFetching.value) return
  isFetching.value = true
  const result = await settle(
    api.POST('/letmoe-resources/{resource_id}/downloads', {
      params: { path: { resource_id: props.resource.id } }
    })
  )
  isFetching.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    open.value = false
    return
  }
  download.value = result.data
}

watch(open, (isOpen) => {
  if (isOpen) {
    void fetchDownload()
  }
})

const hosted = computed(
  () => download.value?.mirrors.filter((m) => m.is_hosted) ?? []
)
const external = computed(
  () => download.value?.mirrors.filter((m) => !m.is_hosted) ?? []
)
const composites = computed(() => download.value?.composites ?? [])
const hasDirect = computed(
  () => hosted.value.length > 0 || composites.value.length > 0
)
const isEmpty = computed(
  () => download.value && !hasDirect.value && !external.value.length
)
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-2xl w-[92vw] !p-0">
    <div class="flex flex-col">
      <div
        class="bg-primary/10 text-primary-700 dark:text-primary flex items-center justify-between gap-3 px-5 py-3"
      >
        <div class="flex items-center gap-2">
          <KunIcon name="lucide:zap" class="text-xl" />
          <span class="text-base font-medium">
            <GalgameResourceLetmoeLink :to="resource.web_url" />
            提供的下载
          </span>
        </div>
        <KunChip variant="flat" color="primary" size="sm">
          <KunIcon name="lucide:download" />
          {{ resource.download_count }} 次下载
        </KunChip>
      </div>

      <div class="space-y-5 p-5">
        <p class="font-medium break-words">{{ resource.title }}</p>

        <div v-if="isFetching" class="flex justify-center py-8">
          <KunLoading />
        </div>

        <div
          v-else-if="download"
          :class="
            cn(
              'flex gap-5',
              mode === 'direct' ? 'flex-col' : 'flex-col-reverse'
            )
          "
        >
          <div v-if="hasDirect" class="space-y-3">
            <KunHeader name="直链下载" scale="h3" />
            <GalgameResourceLetmoeDirectPicker
              :hosted="hosted"
              :composites="composites"
              :base-languages="resource.languages"
              :base-size="resource.size"
            />
          </div>

          <div v-if="external.length" class="space-y-3">
            <KunHeader name="网盘下载" scale="h3" />
            <KunInfo
              v-for="mirror in external"
              :key="mirror.id"
              color="success"
              variant="flat"
              :title="
                mirror.provider
                  ? (LETMOE_PROVIDER_LABEL_MAP[mirror.provider] ??
                    mirror.provider)
                  : '网盘下载'
              "
            >
              <div class="space-y-2">
                <div class="flex items-start gap-2">
                  <KunIcon
                    name="lucide:external-link"
                    class="text-success mt-1 shrink-0"
                  />
                  <KunLink
                    :to="mirror.url"
                    target="_blank"
                    rel="noopener noreferrer"
                    size="sm"
                    class-name="break-all"
                  >
                    {{ mirror.url }}
                  </KunLink>
                </div>
                <p v-if="mirror.note" class="text-default-500 text-xs">
                  {{ mirror.note }}
                </p>
                <div
                  v-if="mirror.extraction_code || mirror.archive_password"
                  class="flex flex-wrap items-center gap-2"
                >
                  <KunCopy
                    v-if="mirror.extraction_code"
                    variant="solid"
                    color="success"
                    :name="`提取码 ${mirror.extraction_code}`"
                    :text="mirror.extraction_code"
                  />
                  <KunCopy
                    v-if="mirror.archive_password"
                    variant="solid"
                    color="success"
                    :name="`解压码 ${mirror.archive_password}`"
                    :text="mirror.archive_password"
                  />
                </div>
              </div>
            </KunInfo>
          </div>

          <KunNull
            v-if="isEmpty"
            description="这个资源暂时没有可用的下载方式, 请前往 LetMoe 页面查看"
          />
        </div>

        <p class="text-default-500 text-sm">
          下载失败、链接失效、文件损坏或合成版打不开, 请到
          <GalgameResourceLetmoeLink :to="letmoeFeedbackUrl(resource.web_url)">
            LetMoe 资源页的「报错与维护」
          </GalgameResourceLetmoeLink>
          反馈, 不要在鲲 Galgame 论坛报告, 本站无法处理 LetMoe 的资源。
        </p>

        <div class="flex flex-wrap items-center justify-between gap-1">
          <div class="flex flex-wrap items-center gap-1">
            <KunButton
              variant="light"
              color="default"
              target="_blank"
              :href="resource.web_url"
            >
              <KunIcon name="lucide:external-link" />
              前往 LetMoe 资源页
            </KunButton>
            <KunButton
              variant="light"
              color="warning"
              target="_blank"
              :href="letmoeFeedbackUrl(resource.web_url)"
            >
              <KunIcon name="lucide:message-square-warning" />
              到 LetMoe 反馈
            </KunButton>
          </div>
          <KunButton variant="solid" color="default" @click="open = false">
            关闭
          </KunButton>
        </div>
      </div>
    </div>
  </KunModal>
</template>
