// h3 hands handlers the decoded path, and Node's setHeader throws
// ERR_INVALID_CHAR on anything above U+00FF: /galgame-resource/12342%E2%80%8C
// (a link copied with a trailing U+200C) answered 500 through a '/**'
// redirect rule. A `%XX` already in the string is kept as it is, since h3
// leaves %25 and %2F encoded.
const UNSAFE_RE = /%(?![0-9A-Fa-f]{2})|[^A-Za-z0-9\-._~:/?#[\]@!$&'()*+,;=%]/gu

export const encodeLocation = (location: string) =>
  location.replace(UNSAFE_RE, (c) => encodeURIComponent(c))

const FORMAT_CHAR_RE = /\p{Cf}/gu
const LEGACY_RESOURCE_RE = /^\/galgame-resource(?=\/|$)/

export const canonicalPath = (path: string) => {
  const to = path
    .replace(FORMAT_CHAR_RE, '')
    .replace(LEGACY_RESOURCE_RE, '/galgame/resource')
  return to.length > 1 ? to.replace(/\/+$/, '') || '/' : to
}
