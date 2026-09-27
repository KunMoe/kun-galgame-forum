import type {
  KunChatEntity,
  KunChatMessage,
  KunChatReaction,
  KunChatReactionOption,
  KunChatUser
} from '@kungal/ui-vue'

// NextMoe chat's wire shapes (/v2/chat, relayed at /api/v1/chat). Nullability
// follows infra's internal/platform/chat/dto, not docs/chat/openapi.yaml: the
// generated schema marks every list nullable (none ever is) and no object
// nullable (last_message, draft, media and the rest are).

export type ChatMessage = KunChatMessage
export type ChatUser = KunChatUser

export interface ChatDraft {
  text: string
  entities: KunChatEntity[]
  reply_to_seq: number | null
  updated_at: string
}

export interface ChatDialog {
  role: 'owner' | 'admin' | 'member'
  accepted: boolean
  last_read_seq: number
  unread_count: number
  marked_unread: boolean
  cleared_through_seq: number
  visible_from_seq: number
  muted_until: string | null
  archived: boolean
  pinned_rank: number | null
  draft: ChatDraft | null
}

export interface ChatConversation {
  object: 'conversation'
  id: string
  kind: 'direct' | 'group'
  title: string | null
  about: string | null
  photo_image_hash: string | null
  peer_id: string | null
  member_count: number
  last_seq: number
  peer_read_seq: number
  created_at: string
  me: ChatDialog
  last_message: ChatMessage | null
  pinned_seqs?: number[]
  pinned_messages?: ChatMessage[]
}

export interface ChatConversationDetail extends ChatConversation {
  members: { user_id: string; role: ChatDialog['role']; joined_at: string }[]
  pinned_seqs: number[]
  pinned_messages: ChatMessage[]
  users: ChatUser[]
}

export type ChatFolder = 'inbox' | 'archive' | 'requests'

export interface ChatConversationPage {
  object: 'list'
  items: ChatConversation[]
  users: ChatUser[]
  next_cursor?: string
}

export interface ChatMessagePage {
  object: 'list'
  items: ChatMessage[]
  users: ChatUser[]
  has_more_before: boolean
  has_more_after: boolean
}

export interface ChatState {
  object: 'chat_state'
  last_update_seq: number
  unread_conversation_count: number
  unread_message_count: number
  request_count: number
}

export type ChatUpdateKind =
  | 'new_message'
  | 'edit_message'
  | 'delete_messages'
  | 'read_inbox'
  | 'read_outbox'
  | 'message_reactions'
  | 'pinned_messages'
  | 'dialog'
  | 'hide_messages'
  | 'clear_history'
  | 'member'
  | 'conversation'

export interface ChatUpdate {
  object: 'update'
  update_seq: number
  kind: ChatUpdateKind
  conversation_id: string
  data: Record<string, unknown>
  created_at: string
}

export interface ChatUpdatesPage {
  object: 'update_list'
  updates: ChatUpdate[]
  messages: ChatMessage[]
  conversations: ChatConversation[]
  users: ChatUser[]
  last_update_seq: number
  has_more: boolean
  too_long: boolean
}

export type ChatAllow = 'all' | 'following' | 'none'

export interface ChatSettings {
  object: 'chat_settings'
  allow_incoming: ChatAllow
  accept_requests: boolean
  allow_group_invites: ChatAllow
}

export interface ChatRealtimeToken {
  object: 'realtime_token'
  token: string
  expires_at: string
  url: string
}

export interface ChatPhoto {
  object: 'chat_image'
  type: 'photo'
  image_hash: string
  width: number
  height: number
  thumbhash?: string
  url: string
}

export type ChatReportReason =
  | 'spam'
  | 'harassment'
  | 'sexual'
  | 'violence'
  | 'illegal'
  | 'other'

export interface ChatReactionList {
  object: 'list'
  items: KunChatReactionOption[]
}

export interface ChatReactionsBody {
  object: 'message_reactions'
  message_id: string
  reactions: KunChatReaction[]
}

export interface ChatReadState {
  object: 'read_state'
  conversation_id: string
  last_read_seq: number
  unread_count: number
}

export interface ChatDeleted {
  object: 'deleted_messages'
  seqs: number[]
  for_everyone: boolean
}

export type WithUsers<T> = T & { users?: ChatUser[] }

export type ChatPush =
  | {
      type: 'update'
      update: ChatUpdate
      message?: ChatMessage
      reactions?: KunChatReaction[]
    }
  | { type: 'typing'; conversation_id: string; user_id: string }
