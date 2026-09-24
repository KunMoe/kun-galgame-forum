import type {
  EditAmendment as KitAmendment,
  EditProposal as KitProposal,
  EditRevision as KitRevision,
  EditSchemaField,
  EditUser,
  EditVocabularyMap
} from '@nextmoe/edit-ui-core'
import type { components } from '#shared/types/api/v1'
import { toKunUser } from '~/utils/userRef'

export type EditForm = components['schemas']['EditForm']
export type EditField = components['schemas']['EditField']
export type EditProposal = components['schemas']['EditProposal']
export type EditProposalSummary = components['schemas']['EditProposalSummary']
export type EditRevision = components['schemas']['EditRevision']
export type EditRevisionDiff = components['schemas']['EditRevisionDiff']
type UserRef = components['schemas']['UserRef']

// The editing kit predates /api/v1 and speaks numeric ids with *_uid names.
// Catalog ids stay far below 2^53, so the conversion is exact.

export const toKitField = (field: EditField): EditSchemaField => ({
  key: field.key,
  kind: field.field_type,
  diff_hint: field.diff_hint,
  deprecated: field.is_deprecated,
  locked: false,
  can_propose: !field.is_deprecated,
  can_review: true,
  would_automerge: false,
  max_elements: field.max_elements,
  max_suppressed: field.max_suppressed,
  vocabulary: field.vocabulary || undefined,
  encoding: field.encoding ?? undefined,
  base: field.base,
  nullable: field.is_nullable,
  element: field.element
    ? {
        type: field.element.element_type,
        members: field.element.members.map((m) => ({
          key: m.key,
          type: m.member_type,
          vocabulary: m.vocabulary || undefined,
          base: m.base,
          nullable: m.is_nullable
        }))
      }
    : null
})

export const toKitVocabularies = (form: EditForm): EditVocabularyMap =>
  Object.fromEntries(
    form.vocabularies.map((v) => [
      v.vocabulary,
      {
        name: v.vocabulary,
        closed: v.is_closed,
        values: v.values.map((it) => ({
          value: it.value,
          display_name: it.display_name || undefined,
          description: it.description || undefined
        }))
      }
    ])
  )

const toKitAmendment = (
  a: EditProposal['amendments'][number]
): KitAmendment => ({
  id: Number(a.id),
  seq: a.seq,
  amender_uid: Number(a.amender.id),
  note: a.note ?? '',
  created_at: a.created_at
})

export const toKitProposal = (
  p: EditProposal | EditProposalSummary
): KitProposal => ({
  id: Number(p.id),
  entity_type: 'catalog.work',
  entity_id: Number(p.work_id),
  base_revision_seq: p.base_revision_seq,
  patch: 'patch' in p ? p.patch : {},
  effective_patch: 'effective_patch' in p ? p.effective_patch : undefined,
  proposer_uid: Number(p.proposer.id),
  note: p.note ?? '',
  site: 'kungal',
  status: p.state,
  decided_by_uid: p.decider ? Number(p.decider.id) : undefined,
  decided_at: p.decided_at ?? undefined,
  created_at: p.created_at,
  updated_at: p.updated_at,
  amendments: 'amendments' in p ? p.amendments.map(toKitAmendment) : undefined
})

export const toKitRevision = (r: EditRevision): KitRevision => ({
  id: Number(r.id),
  seq: r.seq,
  action: r.revision_action,
  changed_fields: r.changed_fields,
  snapshot: {},
  actor_uid: Number(r.actor.id),
  amender_uid: r.last_amender ? Number(r.last_amender.id) : undefined,
  proposal_id: r.proposal_id ? Number(r.proposal_id) : undefined,
  site: 'kungal',
  created_at: r.created_at
})

export const kitUsers = (
  refs: (UserRef | null | undefined)[]
): Record<number, EditUser> => {
  const out: Record<number, EditUser> = {}
  for (const ref of refs) {
    if (ref) {
      const user = toKunUser(ref)
      out[user.id] = user
    }
  }
  return out
}

export const proposalUsers = (
  items: (EditProposal | EditProposalSummary)[]
): Record<number, EditUser> =>
  kitUsers(
    items.flatMap((p) => [
      p.proposer,
      p.decider,
      ...('amendments' in p ? p.amendments.map((a) => a.amender) : [])
    ])
  )
