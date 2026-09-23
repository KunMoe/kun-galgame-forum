export const replyPermalink = (topicLink: string, floor?: number) =>
  floor && floor > 0 ? `${topicLink}?reply=${floor}` : topicLink

export const commentPermalink = (
  topicLink: string,
  commentId?: number | string
) =>
  commentId != null && Number(commentId) > 0
    ? `${topicLink}?comment=${commentId}`
    : topicLink
