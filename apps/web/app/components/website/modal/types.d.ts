import type { z } from 'zod'
import type {
  websiteFormSchema,
  websiteCategoryFormSchema,
  websiteTagFormSchema,
  websiteTagGroupFormSchema
} from '~/validations/website'

export type WebsiteForm = z.infer<typeof websiteFormSchema>
export type WebsiteCategoryForm = z.infer<typeof websiteCategoryFormSchema>
export type WebsiteTagForm = z.infer<typeof websiteTagFormSchema>
export type WebsiteTagGroupForm = z.infer<typeof websiteTagGroupFormSchema>
