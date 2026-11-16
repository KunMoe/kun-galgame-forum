// The API refuses page × limit > 10000 on every page-number collection, while
// total is exact past that depth, so ceil(total / limit) alone names pages
// that answer 400.
const PAGE_DEPTH = 10000

export const pageCountOf = (total: number, limit: number) =>
  Math.min(Math.ceil(total / limit), Math.floor(PAGE_DEPTH / limit))
