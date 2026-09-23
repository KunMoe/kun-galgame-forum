import type { Notification, NotificationType } from '#shared/utils/api/schemas'

const messageTemplates: Record<NotificationType, string> = {
  upvoted: ' 推了您!',
  liked: ' 点赞了您!',
  favorited: ' 收藏了您!',
  replied: ' 回复了您!',
  commented: ' 评论了您!',
  mentioned: ' 提到了您！',
  followed_thread_activity: ' 在您关注的评论区发表了新评论',
  best_answer_chosen: '您的回复被标记为最佳答案!',
  reply_pinned: ' 置顶了您的回复!',
  quiz_answered: ' 回答了您的题目!',
  resource_link_reported: ' 报告了您的资源链接已过期！',
  edit_requested: ' 向您提出更新请求！',
  edit_merged: ' 合并了您的更新请求！',
  edit_declined: ' 拒绝了您的更新请求！',
  lottery_won: ' 的抽奖开奖了, 您中奖了!',
  lottery_drawn: ' 的抽奖开奖了',
  lottery_code_expired: ' 的抽奖兑换码已过领取期限',
  poll_closed: ' 的投票已经截止'
}

export const getMessageI18n = (notification: Notification) => {
  if (
    notification.notification_type === 'mentioned' &&
    notification.excerpt_markdown.trim() &&
    notification.origin === 'local'
  ) {
    return messageTemplates.replied
  }
  if (notification.actor_count > 1) {
    if (notification.notification_type === 'liked') {
      return ` 等 ${notification.actor_count} 人点赞了您!`
    }
    if (notification.notification_type === 'followed_thread_activity') {
      return ` 等 ${notification.actor_count} 人在您关注的评论区发表了 ${notification.item_count} 条新评论`
    }
  }
  if (
    notification.notification_type === 'followed_thread_activity' &&
    notification.item_count > 1
  ) {
    return ` 在您关注的评论区发表了 ${notification.item_count} 条新评论`
  }
  return messageTemplates[notification.notification_type]
}
