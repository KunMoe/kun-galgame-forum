import Cookies from 'js-cookie'

interface OAuthFlowOptions {
  prompt?: string
  loginHint?: string
  returnTo?: string
}

const buildAuthorizeUrl = async (
  opts: OAuthFlowOptions = {}
): Promise<string> => {
  const config = useRuntimeConfig()

  const codeVerifier = generateCodeVerifier()
  const codeChallenge = await generateCodeChallenge(codeVerifier)
  const state = generateState()

  const oauthCookieOptions = {
    expires: 10 / 1440,
    sameSite: 'lax' as const,
    secure: location.protocol === 'https:',
    path: '/'
  }
  Cookies.set('oauth_code_verifier', codeVerifier, oauthCookieOptions)
  Cookies.set('oauth_state', state, oauthCookieOptions)
  if (opts.returnTo) {
    Cookies.set('oauth_return_to', opts.returnTo, oauthCookieOptions)
  }

  const params = new URLSearchParams({
    client_id: config.public.oauthClientId as string,
    redirect_uri: config.public.oauthRedirectUri as string,
    response_type: 'code',
    // catalog:read is what the user lane on /v2/catalog/* demands. Dropping
    // it does not fail loudly: the cover-vote read 403s for every signed-in
    // reader and falls back to the app-key lane, which answers tallies but
    // never `voted`, so the "I voted for this cover" mark was dead site-wide
    // from the day the user lane shipped until 2026-09-07 and nobody
    // reported it. The client's allowed_scopes is the other half.
    // folder:read / folder:write are the one scoped family on /v2/me: the
    // collection face reads and writes the user's own folders, and a token
    // minted without them is 403 SCOPE_REQUIRED on every collection call.
    // The grant is fixed at authorization and a refresh cannot widen it, so a
    // session from before this line has to log in again.
    scope:
      'openid profile catalog:read catalog:edit playtime:read playtime:write folder:read folder:write',
    state,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256'
  })
  if (opts.prompt) params.set('prompt', opts.prompt)
  if (opts.loginHint) params.set('login_hint', opts.loginHint)
  return `${config.public.oauthServerUrl}/oauth/authorize?${params}`
}

export const startOAuthLogin = async (
  opts: OAuthFlowOptions = {}
): Promise<void> => {
  const authorizeUrl = await buildAuthorizeUrl(opts)
  window.location.href = authorizeUrl
}

export const startOAuthSwitchAccount = async (
  loginHint: string,
  returnTo?: string
): Promise<void> => {
  await startOAuthLogin({ prompt: 'select_account', loginHint, returnTo })
}

export const startOAuthAddAccount = async (
  returnTo?: string
): Promise<void> => {
  await startOAuthLogin({ prompt: 'login', returnTo })
}

export const startOAuthRegister = async (): Promise<void> => {
  const config = useRuntimeConfig()
  const authorizeUrl = await buildAuthorizeUrl()
  const registerUrl = `${config.public.oauthFrontendUrl}/auth/register?redirect=${encodeURIComponent(authorizeUrl)}`
  window.location.href = registerUrl
}

export const startOAuthLogout = (): void => {
  const config = useRuntimeConfig()
  const params = new URLSearchParams({
    client_id: config.public.oauthClientId as string,
    redirect: `${window.location.origin}/`
  })
  window.location.href = `${config.public.oauthServerUrl}/oauth/logout?${params}`
}

export const consumeOAuthReturnTo = (): string | null => {
  const value = Cookies.get('oauth_return_to')
  if (value) Cookies.remove('oauth_return_to', { path: '/' })
  if (!value) return null
  try {
    const url = new URL(value, window.location.origin)
    if (url.origin === window.location.origin) {
      return url.pathname + url.search + url.hash
    }
  } catch {
    // A malformed stored return-to must not throw on the auth path; treat it as
    // absent and send the user to the default landing page.
  }
  return null
}
