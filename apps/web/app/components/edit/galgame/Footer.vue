<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { WorkSubmissionCreate } from '#shared/utils/api/schemas'
import { submitGalgameSchema } from '~/validations/galgame'

const LOCALE_OF = {
  'ja-jp': 'ja',
  'zh-cn': 'zh-Hans',
  'zh-tw': 'zh-Hant',
  'en-us': 'en'
} as const

const ORIGINAL_LANGUAGE_OF: Record<string, string> = {
  ...LOCALE_OF,
  'ko-kr': 'ko'
}

const {
  name,
  content_limit,
  age_limit,
  original_language,
  introduction,
  aliases,
  release_date,
  release_date_tba
} = storeToRefs(usePersistEditGalgameStore())

const api = useApiClient()
const createKey = useIdempotencyKey()
const isPublishing = ref(false)

const submissionBody = (
  bannerHash: string,
  isDuplicateConfirmed: boolean
): WorkSubmissionCreate => {
  const locales = Object.keys(LOCALE_OF) as (keyof typeof LOCALE_OF)[]
  const date = release_date_tba.value ? '' : release_date.value
  return {
    titles: locales
      .filter((l) => name.value[l].trim())
      .map((l) => ({ locale: LOCALE_OF[l], title: name.value[l].trim() })),
    aliases: aliases.value.filter((a) => a.trim()),
    introductions: locales
      .filter((l) => introduction.value[l].trim())
      .map((l) => ({ locale: LOCALE_OF[l], value: introduction.value[l] })),
    original_language:
      ORIGINAL_LANGUAGE_OF[original_language.value] ?? original_language.value,
    content_rating: age_limit.value === 'r18' ? 'r18' : 'all_ages',
    is_nsfw: content_limit.value === 'nsfw',
    ...(date ? { release_date: date } : {}),
    ...(bannerHash ? { banner_hash: bannerHash } : {}),
    is_duplicate_confirmed: isDuplicateConfirmed
  }
}

const handleSubmitGalgame = async () => {
  const banner = await getImage('kun-galgame-publish-banner')
  const result = submitGalgameSchema.safeParse({
    name_en_us: name.value['en-us'],
    name_ja_jp: name.value['ja-jp'],
    name_zh_cn: name.value['zh-cn'],
    name_zh_tw: name.value['zh-tw'],
    intro_en_us: introduction.value['en-us'],
    intro_ja_jp: introduction.value['ja-jp'],
    intro_zh_cn: introduction.value['zh-cn'],
    intro_zh_tw: introduction.value['zh-tw'],
    content_limit: content_limit.value,
    release_date: release_date_tba.value ? '' : release_date.value,
    aliases: aliases.value
  })
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }
  const res = await useComponentMessageStore().alert(
    '确定提交 Galgame 申请吗?',
    '提交后将进入审核队列, 审核通过后才会被公开展示。审核结果会通过站内消息通知您。在审核期间您可以在「我的提交」页继续编辑或撤回。'
  )
  if (!res) {
    return
  }

  if (isPublishing.value) {
    return
  } else {
    isPublishing.value = true
    useMessage(10525, 'info', 7777)
  }

  let bannerHash = ''
  // KunUpload emits what canvas.toBlob produces and getImage returns a Blob, so
  // `banner instanceof File` was false for every submission ever made: no
  // upload, no banner_hash, no cover patch, and the wizard has never once
  // attached a cover.
  if (banner) {
    const uploaded = await uploadWorkEditImage(banner, 'cover')
    if (uploaded) {
      bannerHash = uploaded.hash
    }
  }
  // Catalog refuses a mint whose title matches a live work, but the refusal is
  // soft: the same request with is_duplicate_confirmed mints anyway. Sending
  // it blind would defeat the gate, so ask, then resend once.
  const submit = (isDuplicateConfirmed: boolean) => {
    const body = submissionBody(bannerHash, isDuplicateConfirmed)
    return settle(
      api.POST('/work-submissions', {
        params: {
          header: {
            'Idempotency-Key': createKey.take('/work-submissions', body)
          }
        },
        body
      })
    )
  }

  let created = await submit(false)
  if (!created.ok && created.problem.code === 'DUPLICATE_SUSPECTS') {
    const confirmed = await useComponentMessageStore().alert(
      '资料库中已有同名作品',
      '请先回到「发布 Galgame」搜索确认这不是同一部作品 —— 重复条目会被驳回。确实是不同作品 (例如同名重制版或同人作品) 才继续提交。'
    )
    if (!confirmed) {
      isPublishing.value = false
      return
    }
    created = await submit(true)
  }
  isPublishing.value = false
  if (!created.ok) {
    reportProblem(created.problem)
    return
  }
  createKey.clear()

  if (bannerHash && !created.data.has_banner_attached) {
    useMessage('封面上传失败, 请在「我的提交」中重新添加封面', 'warn', 7777)
  }
  await deleteImage('kun-galgame-publish-banner')
  if (import.meta.client) {
    localStorage.removeItem('kun-galgame-publish-step')
  }

  // Catalog mints straight to live for a submitter it trusts, so the claim
  // state decides where this lands. Announcing "等待审核" for a live entry
  // sends its author to a review list that will never contain it.
  const isLive = created.data.state === 'live'
  useKunLoliInfo(isLive ? 'Galgame 已发布' : 'Galgame 申请已提交, 等待审核', 5)
  await navigateTo(
    isLive ? `/galgame/${created.data.id}` : '/edit/galgame/mine'
  )
  usePersistEditGalgameStore().resetEditGalgameStore()
}
</script>

<template>
  <div class="flex justify-end">
    <KunButton
      :disabled="isPublishing"
      :loading="isPublishing"
      size="lg"
      @click="handleSubmitGalgame"
    >
      提交 Galgame 申请
    </KunButton>
  </div>
</template>
