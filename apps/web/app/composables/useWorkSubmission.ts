import { settle, type ApiResult } from '#shared/utils/api/problem'
import type { components } from '#shared/types/api/v1'
import type { WorkSubmission } from '#shared/utils/api/schemas'

type WorkSubmissionTarget =
  components['schemas']['WorkSubmissionPatch']['state']

export const useWorkSubmission = () => {
  const api = useApiClient()

  const versionOf = async (
    workId: string
  ): Promise<ApiResult<string | undefined>> => {
    let etag: string | undefined
    const read = await settle(
      api
        .GET('/work-submissions/{work_id}', {
          params: { path: { work_id: workId } }
        })
        .then((result) => {
          etag = result.response.headers.get('ETag') ?? undefined
          return result
        })
    )
    return read.ok ? { ok: true, data: etag } : read
  }

  const move = async (
    workId: string,
    state: WorkSubmissionTarget,
    note?: string
  ): Promise<ApiResult<WorkSubmission>> => {
    const version = await versionOf(workId)
    if (!version.ok) {
      return version
    }
    return settle(
      api.PATCH('/work-submissions/{work_id}', {
        params: {
          path: { work_id: workId },
          header: { 'If-Match': version.data }
        },
        body: note ? { state, note } : { state }
      })
    )
  }

  const remove = async (workId: string): Promise<ApiResult<undefined>> => {
    const version = await versionOf(workId)
    if (!version.ok) {
      return version
    }
    return settle(
      api.DELETE('/work-submissions/{work_id}', {
        params: {
          path: { work_id: workId },
          header: { 'If-Match': version.data }
        }
      })
    )
  }

  return { move, remove }
}
