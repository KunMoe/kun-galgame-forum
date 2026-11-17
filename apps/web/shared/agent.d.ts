interface AgentConversation {
  id: string
  title: string
  mode: 'assistant' | 'search'
  starred: boolean
  active_run_id: string | null
  updated_at: string
  created_at: string
}

interface AgentConversationPage {
  items: AgentConversation[] | null
  next_cursor: string | null
}

interface AgentCardFact {
  label: string
  value: string
}

interface AgentCatalogCover {
  hash: string
  thumbhash: string | null
  sexual: string | null
}

interface AgentEntityCard {
  kind: string
  id: string
  title: string
  localized?: Record<string, { value: string }> | null
  subtitle?: string
  summary?: string
  facts?: AgentCardFact[] | null
  content_limit: 'sfw' | 'nsfw'
  cover?: AgentCatalogCover | string | null
  release_date?: string | null
  content_rating?: string
  tags?: string[] | null
  companies?: string[] | null
  ratings?: AgentCardRating[] | null
  url?: string
}

interface AgentCardRating {
  source: string
  score: number
  votes: number
}

interface AgentCardCover {
  src: string
  thumbhash: string | null
  blurred: boolean
}

type AgentPart =
  | { type: 'text'; id: string; text: string }
  | { type: 'reasoning'; id: string; text: string; done: boolean }
  | { type: 'step'; id: string }
  | {
      type: 'tool'
      id: string
      name: string
      input: unknown
      output: AgentToolOutput | null
      done: boolean
      error?: string
    }
  | { type: 'card'; id: string; card: AgentEntityCard }
  | { type: 'source'; id: string; url: string; title: string }
  | { type: 'browse'; id: string; browse: AgentBrowse }
  | {
      type: 'file'
      filename: string
      mediaType: string
      text: string
      uploadId: string
      url: string
    }

interface AgentBrowse {
  target: string
  ids: string[]
  names: string[]
  sort: string
  sorts: string[]
  total: number
}

interface AgentLink {
  url: string
  title: string
}

interface AgentToolOutput {
  cards: number
  sources: number
  entities: AgentEntityCard[]
  links: AgentLink[]
  total: number | null
  meta: Record<string, unknown>
}

type AgentFeedback = 'up' | 'down'

type AgentMessageStatus = 'complete' | 'streaming' | 'failed' | 'canceled'

interface AgentMessage {
  id: string
  parent_id: string | null
  role: 'user' | 'assistant'
  parts: AgentPart[]
  status: AgentMessageStatus
  run_id: string
  sibling_ids: string[]
  feedback: AgentFeedback | null
  error?: string
}

interface AgentWireMessage {
  id: string
  parent_id: string | null
  role: 'user' | 'assistant'
  parts: Record<string, unknown>[] | null
  status: AgentMessageStatus
  run_id: string
  siblings: number
  sibling_ids: string[] | null
  feedback: AgentFeedback | null
}

interface AgentConversationDetail {
  conversation: AgentConversation
  messages: AgentWireMessage[] | null
  active_run_id: string | null
}

interface AgentSuggestionGroup {
  label: string
  icon: string
  web_search: boolean
  prompts: string[] | null
}

type AgentUploadKind = 'image' | 'text' | 'document'

interface AgentUploadOption {
  kind: AgentUploadKind
  label: string
  icon: string
  accept: string
  max_bytes: number
}

interface AgentUploads {
  max_files: number
  options: AgentUploadOption[] | null
}

interface AgentUpload {
  id: string
  kind: AgentUploadKind
  media_type: string
  filename: string
  size: number
  chars: number
  truncated: boolean
  url: string
}

interface AgentPersona {
  name: string
  avatar: string
  greetings: string[] | null
  suggestions: AgentSuggestionGroup[] | null
  web_search_available: boolean
  web_search_auto?: boolean
  memory_available?: boolean
  uploads?: AgentUploads | null
}

interface AgentPreferences {
  memory: boolean
  history: boolean
  memory_available: boolean
  history_available: boolean
}

interface AgentMemory {
  id: string
  content: string
  origin: 'explicit' | 'auto'
  conversation_id: string | null
  created_at: string
  updated_at: string
}

interface AgentMemoryList {
  items: AgentMemory[] | null
  max: number
}

interface AgentQuota {
  free_left: number
  free_per_day: number
  price_assistant: number
  price_web_surcharge: number
  price_search: number
  paid_enabled: boolean
  moemoepoint_balance: number | null
}

interface AgentSendResult {
  run_id: string
  user_message_id: string
  assistant_message_id: string
}
