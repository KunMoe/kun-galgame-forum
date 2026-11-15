import { settle } from '#shared/utils/api/problem'

export type ReportExpireStatus =
  | 'idle'
  | 'checking'
  | 'partial'
  | 'expired'
  | 'alive'
  | 'unchecked'
  | 'error'

export const useReportResourceExpired = () => {
  const status = ref<ReportExpireStatus>('idle')
  const api = useApiClient()

  const report = async (
    resourceId: string,
    onMarked?: (expiredUrls: string[]) => void
  ) => {
    if (!usePersistUserStore().id) {
      useAuthModal().open()
      return
    }

    const confirmed = await useComponentMessageStore().alert(
      '您确定报告资源链接失效吗？',
      '系统会用网盘官方接口逐条核验每个链接: 确认失效的链接会加删除线并通知发布者, 全部链接都失效时资源才会被标记失效; 仍可访问的链接不会被标记。恶意报告将被处罚。'
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

    const { verdict, expired_download_urls } = result.data
    if (verdict === 'alive') {
      status.value = 'alive'
      return
    }
    status.value =
      verdict === 'partial'
        ? 'partial'
        : verdict === 'unchecked'
          ? 'unchecked'
          : 'expired'
    onMarked?.(expired_download_urls)
  }

  return { status, report }
}
