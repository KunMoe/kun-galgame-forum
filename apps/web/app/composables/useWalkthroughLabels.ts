import type {
  WalkthroughChart,
  WalkthroughSegment
} from '#shared/utils/api/schemas'
import type { GalgameWalkthroughGraphNode } from '#shared/types/galgame-walkthrough'
import type { KunWalkthroughTextMode } from '~/constants/galgameWalkthrough'

type LabeledNode = Pick<
  GalgameWalkthroughGraphNode,
  'kind' | 'label' | 'localized' | 'contextKey'
>

export const useWalkthroughLabels = () => {
  const mode = inject<Ref<KunWalkthroughTextMode>>(
    'walkthroughTextMode',
    ref('both')
  )
  const routeName = inject<(key: string) => string>(
    'walkthroughRouteName',
    (key) => key
  )

  const point = (label: string, localized: LabeledNode['localized']) =>
    walkthroughPointLabel(label, localized, mode.value)

  const short = (text: string, localized: LabeledNode['localized']) =>
    walkthroughText(text, localized, mode.value === 'ja' ? 'ja' : 'zh').main

  const node = (n: LabeledNode) => {
    const main =
      n.kind === 'ending'
        ? short(n.label, n.localized)
        : point(n.label, n.localized)
    return n.contextKey ? `${main} · ${routeName(n.contextKey)}` : main
  }

  const start = (chart: WalkthroughChart, segment: WalkthroughSegment) =>
    point(
      nodeLabel(chart, segment.from_node),
      nodeLocalized(chart, segment.from_node)
    )

  const ending = (chart: WalkthroughChart, key: string) => {
    const e = chart.chart_endings.find((x) => x.key === key)
    return e ? short(e.ending_name, e.localized) : key
  }

  const goal = (chart: WalkthroughChart, segment: WalkthroughSegment) => {
    const to = segment.to_node
    if (to.node_type === 'collect') return '回收 CG / 场景'
    return to.node_type === 'ending' && to.node_key
      ? ending(chart, to.node_key)
      : '本段结束'
  }

  return { point, short, node, start, ending, goal }
}
