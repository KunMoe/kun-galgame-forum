import type { WorkRelation } from '#shared/utils/api/schemas'

// Catalog words a relation from the page's own work ("是…的续作": this work is
// the sequel of that one) and sends no direction, so a card printed "是…的续作"
// under the earlier game and readers saw only "…". These labels name what the
// related work is to this one, keyed by catalog's seeded phrases.
const LABELS: Record<string, string> = {
  改编自: '原作',
  被改编为: '改编',
  '是…的续作': '前作',
  有续作: '续作',
  '是…的外传': '本篇',
  有外传: '外传',
  '是…的 Fandisc': '本篇',
  '有 Fandisc': 'FD',
  收录: '收录作',
  被收录于: '合集',
  重制自: '原版',
  被重制为: '重制版',
  同系列: '同系列',
  同世界观: '同世界观',
  联动: '联动',
  角色出演: '角色出演',
  不同世界观: '不同世界观',
  不同演绎: '不同演绎'
}

const LINEAGE = new Set(['前作', '续作', '本篇', '外传', 'FD'])

export const relationLabel = (relation: WorkRelation) =>
  LABELS[relation.phrase] ?? `本作${relation.phrase.replace('…', '它')}`

export const relationLabels = (relations: WorkRelation[]) => [
  ...new Set(relations.map(relationLabel))
]

export const isLineageLabel = (label: string) => LINEAGE.has(label)
