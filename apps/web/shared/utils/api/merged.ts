// ENTITY_MERGED carries the survivor's id as an extension member, which the
// shared problem settle() does not keep; read it off the raw error body.
export const mergedInto = (error: unknown): number | null => {
  if (
    error &&
    typeof error === 'object' &&
    (error as { code?: unknown }).code === 'ENTITY_MERGED'
  ) {
    const id = Number((error as { current_id?: unknown }).current_id)
    return Number.isInteger(id) && id > 0 ? id : null
  }
  return null
}
