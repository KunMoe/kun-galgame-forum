export const maskSpoilers = (text: string, mask = '███') =>
  text ? text.replace(/\|\|(.*?)\|\|/gs, mask) : ''
