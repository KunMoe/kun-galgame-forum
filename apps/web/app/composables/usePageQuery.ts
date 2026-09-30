// A paginated list's page belongs in the URL. As a component-local ref it dies
// with the component: 「多标签搜索里点击游戏都会返回第一页」— opening a game from
// page 3 of the multi-tag results and pressing back re-mounted the list at page
// 1, and a copied link never reopened where it was made.
//
// `replace` so paging does not fill the history: back is for leaving the list,
// not for undoing one page step. Only call this where every other input that
// shapes the list is in the URL too — a restored page against a filter that
// reset to its default points at a list the reader never saw.
export const usePageQuery = (key = 'page') =>
  useRouteQuery(key, 1, {
    mode: 'replace',
    transform: (value: unknown) => {
      const page = Number(value)
      return Number.isInteger(page) && page > 0 ? page : 1
    }
  })
