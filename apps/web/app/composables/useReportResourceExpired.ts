import { settle } from '#shared/utils/api/problem'

export type ReportExpireStatus =
  | 'idle'
  | 'checking'
  | 'expired'
  | 'alive'
  | 'unchecked'
  | 'error'

export const useReportResourceExpired = () => {
  const status = ref<ReportExpireStatus>('idle')
  const api = useApiClient()

  const report = async (resourceId: string, onMarked?: () => void) => {
    if (!usePersistUserStore().id) {
      useAuthModal().open()
      return
    }

    const confirmed = await useComponentMessageStore().alert(
      '您确定报告资源链接失效吗？',
      '系统会先用网盘官方接口核验链接是否真的失效: 确认失效才会标记并通知发布者; 若链接仍可访问则不会标记。恶意报告将被处罚。'
    )
    if (!confirmed) return

    status.value = 'checking'
    const result = await settle(
      api.POST('/galgame-resources/{resource_id}/expiry-reports', {
        params: { path: { resource_id: resourceId } }
      })
    )

    if (!result.ok) {
      reportProblem(result.problem)
      status.value = 'error'
      return
    }

    if (result.data.verdict === 'alive') {
      status.value = 'alive'
      return
    }
    if (result.data.verdict === 'unchecked') {
      status.value = 'unchecked'
      onMarked?.()
      return
    }
    status.value = 'expired'
    onMarked?.()
  }

  return { status, report }
}
