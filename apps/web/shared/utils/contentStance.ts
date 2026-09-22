export type KunContentStance = 'hide' | 'blur' | 'show'

// The only place this app turns the account's two claims into a stance. The
// upstream migration backfilled nsfw_display to 'blur' for every existing
// account while leaving adult_confirmed_at null, so folding on nsfw_display
// alone uncensors the site for every reader who has never attested their age.
export const foldContentStance = (
  adultConfirmed: boolean | undefined,
  nsfwDisplay: string | undefined
): KunContentStance => {
  if (!adultConfirmed) return 'hide'
  return nsfwDisplay === 'blur' || nsfwDisplay === 'show' ? nsfwDisplay : 'hide'
}

export const KUN_ACCOUNT_SETTINGS_URL = 'https://account.nextmoe.com/settings'
