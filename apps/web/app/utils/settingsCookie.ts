// A store's cookie options replace the global ones, so sameSite is restated.
// The global seven days only renew when the store changes, so a setting left
// alone for a week reverted to its default.
export const settingsCookieStorage = () =>
  piniaPluginPersistedstate.cookies({
    maxAge: 60 * 60 * 24 * 365,
    sameSite: 'lax'
  })
