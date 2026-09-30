export const KUN_NEWS_SUBMISSION_STATE_MAP: Record<
  KunNewsSubmissionState,
  { label: string; color: 'warning' | 'success' | 'danger' | 'default' }
> = {
  pending: { label: '待审核', color: 'warning' },
  published: { label: '已发布', color: 'success' },
  rejected: { label: '未通过', color: 'danger' },
  withdrawn: { label: '已撤回', color: 'default' }
}
