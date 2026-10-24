import type { Section } from '#shared/utils/api/schemas'

export interface SectionForm {
  label: string
  description: string
  icon_name: Section['icon_name']
  color: Section['color']
  banner_image_hash: string
  mascot_image_hashes: string[]
}
