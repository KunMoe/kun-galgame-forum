const siteLabels: Record<string, string> = {
  official_site: '官方网站',
  twitter: 'X',
  cien: 'Ci-en',
  steam: 'Steam',
  pixiv: 'pixiv',
  vndb: 'VNDB',
  bangumi: 'Bangumi',
  erogamescape: '批评空间',
  dlsite: 'DLsite',
  dmm: 'DMM',
  getchu: 'Getchu'
}

const hostLabels: [string, string][] = [
  ['ci-en.dlsite.com', 'Ci-en'],
  ['wikipedia.org', '维基百科'],
  ['wikidata.org', 'Wikidata'],
  ['youtube.com', 'YouTube'],
  ['nicovideo.jp', 'niconico'],
  ['x.com', 'X'],
  ['twitter.com', 'X'],
  ['pixiv.net', 'pixiv'],
  ['fanbox.cc', 'pixivFANBOX'],
  ['fantia.jp', 'Fantia'],
  ['patreon.com', 'Patreon'],
  ['booth.pm', 'BOOTH'],
  ['melonbooks.co.jp', 'Melonbooks'],
  ['steampowered.com', 'Steam'],
  ['dlsite.com', 'DLsite'],
  ['dmm.co.jp', 'DMM'],
  ['getchu.com', 'Getchu'],
  ['gamefaqs.gamespot.com', 'GameFAQs'],
  ['mobygames.com', 'MobyGames'],
  ['anidb.net', 'AniDB'],
  ['myanimelist.net', 'MyAnimeList'],
  ['bgm.tv', 'Bangumi'],
  ['bangumi.tv', 'Bangumi'],
  ['vndb.org', 'VNDB'],
  ['erogamescape.dyndns.org', '批评空间'],
  ['facebook.com', 'Facebook'],
  ['instagram.com', 'Instagram'],
  ['twitch.tv', 'Twitch'],
  ['soundcloud.com', 'SoundCloud'],
  ['tumblr.com', 'Tumblr'],
  ['itch.io', 'itch.io'],
  ['github.com', 'GitHub']
]

const hostOf = (url: string): string => {
  try {
    return new URL(url).hostname.toLowerCase().replace(/^www\./, '')
  } catch {
    return ''
  }
}

export const catalogLinkLabel = (site: string, url: string): string => {
  const known = siteLabels[site]
  if (known) {
    return known
  }
  const host = hostOf(url)
  if (!host) {
    return site
  }
  const match = hostLabels.find(
    ([suffix]) => host === suffix || host.endsWith(`.${suffix}`)
  )
  return match ? match[1] : host
}
