import { settle, type ApiResult } from '#shared/utils/api/problem'
import type { components } from '#shared/types/api/v1'
import type { WorkSubmission } from '#shared/utils/api/schemas'

type WorkSubmissionTarget =
  components['schemas']['WorkSubmissionPatch']['state']

interface MoveOptions {
  version?: string
  note?: string
}

export const useWorkSubmission = () => {
  const api = useApiClient()

  const ifMatch = (version?: string) =>
    version ? { 'If-Match': version } : undefined

  const move = (
    workId: string,
    state: WorkSubmissionTarget,
    { version, note }: MoveOptions = {}
  ): Promise<ApiResult<WorkSubmission>> =>
    settle(
      api.PATCH('/work-submissions/{work_id}', {
        params: { path: { work_id: workId }, header: ifMatch(version) },
        body: note ? { state, note } : { state }
      })
    )

  const remove = (
    workId: string,
    version?: string
  ): Promise<ApiResult<undefined>> =>
    settle(
      api.DELETE('/work-submissions/{work_id}', {
        params: { path: { work_id: workId }, header: ifMatch(version) }
      })
    )

  return { move, remove }
}
