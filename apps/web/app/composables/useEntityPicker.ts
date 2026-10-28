import type { EntityLinkRef, EntityRefType } from '~/utils/entityUrl'

export type EntityPickDisplay = 'card' | 'chip'

let settlePick: ((ref: EntityLinkRef | null) => void) | null = null

export const useEntityPicker = () => {
  const state = useState('kun-entity-picker', () => ({
    isOpen: false,
    type: 'work' as EntityRefType,
    display: 'card' as EntityPickDisplay
  }))

  // KunModal keeps the page behind it inert until its close is applied, and
  // the editor focuses its caret as soon as this settles: settled any earlier,
  // that focus landed on <body>. The modal itself returns focus nowhere.
  const finish = async (ref: EntityLinkRef | null) => {
    state.value.isOpen = false
    const resolve = settlePick
    settlePick = null
    await nextTick()
    resolve?.(ref)
  }

  const pick = (type: EntityRefType, display: EntityPickDisplay) =>
    new Promise<EntityLinkRef | null>((resolve) => {
      settlePick?.(null)
      settlePick = resolve
      state.value = { isOpen: true, type, display }
    })

  return { state, pick, finish }
}
