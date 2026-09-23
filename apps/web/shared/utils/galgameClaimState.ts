type GalgameClaimStateColor =
  | 'default'
  | 'primary'
  | 'success'
  | 'warning'
  | 'danger'

export interface GalgameClaimStateBadge {
  label: string
  color: GalgameClaimStateColor
}

export const CLAIM_STATE_LIVE = 'live'
export const CLAIM_STATE_DRAFT = 'draft'
export const CLAIM_STATE_PENDING = 'pending'
export const CLAIM_STATE_DECLINED = 'declined'
export const CLAIM_STATE_HIDDEN = 'hidden'
export const CLAIM_STATE_NONE = 'none'
export const galgameClaimStateBadge = (
  state: string | undefined
): GalgameClaimStateBadge => {
  switch (state) {
    case CLAIM_STATE_LIVE:
      return { label: '已发布', color: 'success' }
    case CLAIM_STATE_DRAFT:
      return { label: '草稿', color: 'primary' }
    case CLAIM_STATE_PENDING:
      return { label: '审核中', color: 'warning' }
    case CLAIM_STATE_DECLINED:
      return { label: '已拒绝', color: 'danger' }
    case CLAIM_STATE_HIDDEN:
      return { label: '已下架', color: 'default' }
    case CLAIM_STATE_NONE:
      return { label: '未认领', color: 'primary' }
    default:
      return { label: '未知', color: 'default' }
  }
}

export const isPublicState = (state: string | undefined): boolean =>
  state === CLAIM_STATE_LIVE
