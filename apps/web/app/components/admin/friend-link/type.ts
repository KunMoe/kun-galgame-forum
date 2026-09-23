import type {
  FriendLinkCreate,
  FriendLinkPatch
} from '#shared/utils/api/schemas'

export type FriendLinkSubmit =
  | { id: null; body: FriendLinkCreate }
  | { id: string; body: FriendLinkPatch }
