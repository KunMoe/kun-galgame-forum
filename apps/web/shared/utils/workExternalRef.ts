import type { WorkExternalRef } from '#shared/utils/api/schemas'

const vndbWorkId = /^v[0-9]+$/

// A VNDB work carries its release ids too, and catalog's order is not the anchor:
// taking the first vndb row linked release-first works to a release page (the
// legacy refs map had the same fix, vndb_anchor_test.go).
export const workExternalId = (
  refs: WorkExternalRef[],
  site: string
): string | undefined => {
  const rows = refs.filter((ref) => ref.site === site)
  const anchor =
    site === 'vndb'
      ? rows.find((ref) => vndbWorkId.test(ref.external_id))
      : undefined
  return (anchor ?? rows[0])?.external_id
}
