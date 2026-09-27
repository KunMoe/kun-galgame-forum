export const directMessagePath = (userId: number | string): string =>
  useRuntimeConfig().public.chatEnabled
    ? `/messages?to=${userId}`
    : `/message/user/${userId}`
