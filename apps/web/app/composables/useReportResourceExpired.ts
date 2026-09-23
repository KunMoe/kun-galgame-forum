export type ReportExpireStatus =
  | 'idle'
  | 'checking'
  | 'expired'
  | 'alive'
  | 'error'

export const useReportResourceExpired = () => {
  const status = ref<ReportExpireStatus>('idle')
  const nuxtApp = useNuxtApp()

  const report = async (
    workId: number,
    resourceId: number,
    onMarked?: () => void
  ) => {
    if (!usePersistUserStore().id) {
      useAuthModal().open()
      return
    }

    const confirmed = await useComponentMessageStore().alert(
      '您确定报告资源链接失效吗？',
      '系统会先用网盘官方接口核验链接是否真的失效: 确认失效才会标记并通知发布者; 若链接仍可访问则不会标记。若 17 天内资源发布者没有更换有效链接, 该链接会被删除。恶意报告将被处罚。'
    )
    if (!confirmed) return

    status.value = 'checking'
    const result = await nuxtApp.runWithContext(() =>
      kunFetch<{ verdict: string; marked: boolean }>(
        `/galgame/${workId}/resource/expired`,
        { method: 'PUT', body: { galgame_resource_id: resourceId } }
      )
    )

    if (!result) {
      status.value = 'error'
      return
    }
    const stillAlive = typeof result === 'object' && result.marked === false
    if (stillAlive) {
      status.value = 'alive'
    } else {
      status.value = 'expired'
      nuxtApp.runWithContext(() => onMarked?.())
    }
  }

  return { status, report }
}
