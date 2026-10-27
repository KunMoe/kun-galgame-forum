// Call it after an awaited useKunApi: the server has the result by then and
// throws, so the response is a real 404. On client navigation the fetch is
// still in flight, so the check follows it instead.
export const useKunNotFound = (
  missing: () => boolean,
  statusMessage: string
) => {
  const notFound = () =>
    createError({ statusCode: 404, statusMessage, fatal: true })
  if (import.meta.server) {
    if (missing()) {
      throw notFound()
    }
    return
  }
  watch(
    missing,
    (gone) => {
      if (gone) {
        showError(notFound())
      }
    },
    { immediate: true }
  )
}
