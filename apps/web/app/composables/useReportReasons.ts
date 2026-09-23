import { settle } from '#shared/utils/api/problem'
import type { ReportReason } from '#shared/utils/api/schemas'

const reasons = ref<ReportReason[]>([])
const loaded = ref(false)

export const useReportReasons = () => {
  const api = useApiClient()
  const load = async () => {
    if (loaded.value) {
      return
    }
    const result = await settle(api.GET('/report-reasons'))
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    reasons.value = result.data.items
    loaded.value = true
  }
  return { reasons, load }
}
