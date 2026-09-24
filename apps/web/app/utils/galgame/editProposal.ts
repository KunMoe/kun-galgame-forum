import type { ApiClient } from '#shared/utils/api/client'
import { settle } from '#shared/utils/api/problem'
import type { components } from '#shared/types/api/v1'

type ProposalPatch = components['schemas']['EditProposalPatch']

const ifMatch = (etag: string | null | undefined) =>
  etag ? { 'If-Match': etag } : {}

export const patchEditProposal = async (
  api: ApiClient,
  id: string,
  body: ProposalPatch,
  etag?: string | null
) => {
  let validator = etag
  if (validator === undefined) {
    const { response } = await api.GET('/edit-proposals/{proposal_id}', {
      params: { path: { proposal_id: id } }
    })
    validator = response.ok ? response.headers.get('ETag') : null
  }
  return settle(
    api.PATCH('/edit-proposals/{proposal_id}', {
      params: { path: { proposal_id: id }, header: ifMatch(validator) },
      body
    })
  )
}

export const amendEditProposal = async (
  api: ApiClient,
  id: string,
  body: components['schemas']['EditAmendmentCreate'],
  etag: string | null,
  idempotencyKey: string
) => {
  const call = api.POST('/edit-proposals/{proposal_id}/amendments', {
    params: {
      path: { proposal_id: id },
      header: { 'Idempotency-Key': idempotencyKey, ...ifMatch(etag) }
    },
    body
  })
  const [result, raw] = await Promise.all([settle(call), call])
  return { result, etag: raw.response.headers.get('ETag') }
}
