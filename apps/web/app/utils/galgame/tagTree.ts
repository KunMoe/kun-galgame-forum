import type { TagNode, TagRef } from '#shared/utils/api/schemas'

export const tagPathsUnder = (rootId: string, nodes: TagNode[]) => {
  const byId = new Map(nodes.map((n) => [n.id, n]))
  const paths = new Map<string, TagNode[]>([[rootId, []]])
  for (const node of nodes) {
    let best: TagNode[] | undefined
    for (const parentId of node.parent_ids) {
      const above = paths.get(parentId)
      const parent = byId.get(parentId)
      if (!above) {
        continue
      }
      const via = parent ? [...above, parent] : above
      if (!best || via.length < best.length) {
        best = via
      }
    }
    paths.set(node.id, best ?? [])
  }
  return paths
}

export const ancestorTrail = (parents: TagRef[], ancestors: TagNode[]) => {
  const byId = new Map(ancestors.map((n) => [n.id, n]))
  const trail: TagNode[] = []
  const seen = new Set<string>()
  let next = parents[0] ? byId.get(parents[0].id) : undefined
  while (next && !seen.has(next.id)) {
    seen.add(next.id)
    trail.unshift(next)
    next = next.parent_ids[0] ? byId.get(next.parent_ids[0]) : undefined
  }
  return trail
}
