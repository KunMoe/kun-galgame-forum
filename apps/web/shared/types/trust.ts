export interface ReviewItemView {
  id: number
  site: string
  subject_kind: string
  subject_id: string
  source: number
  severity?: number
  classifier_score?: number
  report_weight_sum?: number
  subject_reach?: number
  context_note?: string
  priority: number
  status: number
  claimed_by?: number
  claimed_at?: string
  decided_by?: number
  decided_at?: string
  created_at: string
}

export interface ReportView {
  id: number
  site: string
  subject_kind: string
  subject_id: string
  reporter_id: number
  reason_id: number
  note?: string
  subject_snapshot?: string
  subject_url?: string
  weight: number
  review_item_id?: number
  status: number
  created_at: string
}

export interface ReviewItemDetail {
  item: ReviewItemView
  reports: ReportView[]
}

export interface ReviewItemPage {
  items: ReviewItemView[]
  total: number
}
