<script setup lang="ts">
import { submitGalgameSchema } from '~/validations/galgame'

const CODE_DUPLICATE_SUSPECTS = 236

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

const isPublishing = ref(false)

const handleSubmitGalgame = async () => {
  const banner = await getImage('kun-galgame-publish-banner')
  const data: Record<
    string,
    number | string | string[] | Blob | boolean | null
  > = {
    name_en_us: name.value['en-us'],
    name_ja_jp: name.value['ja-jp'],
    name_zh_cn: name.value['zh-cn'],
    name_zh_tw: name.value['zh-tw'],
    intro_en_us: introduction.value['en-us'],
    intro_ja_jp: introduction.value['ja-jp'],
    intro_zh_cn: introduction.value['zh-cn'],
    intro_zh_tw: introduction.value['zh-tw'],
    content_limit: content_limit.value,
    age_limit: age_limit.value,
    original_language: original_language.value,
    release_date: release_date.value,
    release_date_tba: release_date_tba.value,
    banner
  }
  const result = submitGalgameSchema.safeParse(data)
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

  const { banner: _bannerBlob, ...jsonFields } = data
  let bannerHash = ''
  // KunUpload emits what canvas.toBlob produces and getImage returns a Blob, so
  // `banner instanceof File` was false for every submission ever made: no
  // upload, no banner_hash, no cover patch, and the wizard has never once
  // attached a cover.
  if (banner) {
    const uploaded = await uploadGalgameImage(banner, 'galgame_banner')
    if (uploaded) {
      bannerHash = uploaded.hash
    }
  }
  // Catalog refuses a mint whose title matches a live work, but the refusal is
  // soft: the same request with confirm_duplicates mints anyway. Sending it
  // blind would defeat the gate, so ask, then resend once.
  let duplicates = false
  const submit = (confirmDuplicates: boolean) =>
    kunFetch<{
      gid: number
      claim_state: string
      banner_attached: boolean
    }>('/galgame/submit', {
      method: 'POST',
      body: {
        ...jsonFields,
        aliases: aliases.value,
        banner_hash: bannerHash,
        confirm_duplicates: confirmDuplicates
      },
      onApiError: (envelope: { code: number }) => {
        if (envelope.code !== CODE_DUPLICATE_SUSPECTS) {
          return false
        }
        duplicates = true
        return true
      }
    })

  let created = await submit(false)
  if (!created && duplicates) {
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

  if (created?.gid) {
    if (bannerHash && !created.banner_attached) {
      useMessage('封面上传失败, 请在「我的提交」中重新添加封面', 'warn', 7777)
    }
    await deleteImage('kun-galgame-publish-banner')
    if (import.meta.client) {
      localStorage.removeItem('kun-galgame-publish-step')
    }

    // Catalog mints straight to live for a submitter it trusts, so the claim
    // state decides where this lands. Announcing "等待审核" for a live entry
    // sends its author to a review list that will never contain it.
    const isLive = created.claim_state === 'live'
    useKunLoliInfo(
      isLive ? 'Galgame 已发布' : 'Galgame 申请已提交, 等待审核',
      5
    )
    await navigateTo(isLive ? `/galgame/${created.gid}` : '/edit/galgame/mine')
    usePersistEditGalgameStore().resetEditGalgameStore()
  }
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
