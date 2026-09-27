export type KunFeedFormat = 'xml' | 'atom' | 'json'

export type KunFeedUrlParams = {
  include_nsfw?: boolean
  section?: string
  lane?: string
  source?: string
}

const PARAM_ORDER = ['include_nsfw', 'section', 'lane', 'source'] as const

const queryValue = (
  key: (typeof PARAM_ORDER)[number],
  params: KunFeedUrlParams
): string | undefined => {
  if (key === 'include_nsfw') {
    return params.include_nsfw ? '1' : undefined
  }
  const value = params[key]
  return value ? value : undefined
}

export const kunFeedUrl = (
  baseUrl: string,
  path: string,
  format: KunFeedFormat = 'xml',
  params: KunFeedUrlParams = {}
): string => {
  const search = new URLSearchParams()
  for (const key of PARAM_ORDER) {
    const value = queryValue(key, params)
    if (value) {
      search.set(key, value)
    }
  }
  const query = search.toString()
  return `${baseUrl}${path}.${format}${query ? `?${query}` : ''}`
}
