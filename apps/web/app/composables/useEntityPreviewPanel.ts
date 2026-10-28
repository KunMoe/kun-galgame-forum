import type { EntityTarget } from '#shared/utils/api/schemas'

export const useEntityPreviewPanel = () => {
  const state = useState('kun-entity-preview-panel', () => ({
    isOpen: false,
    target: null as EntityTarget | null
  }))

  const open = (target: EntityTarget) => {
    state.value = { isOpen: true, target }
  }

  const close = () => {
    state.value.isOpen = false
  }

  return { state, open, close }
}
