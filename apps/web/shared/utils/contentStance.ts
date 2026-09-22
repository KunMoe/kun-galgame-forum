export type KunContentStance = 'hide' | 'blur' | 'show'

// The only place this app turns the account's two claims into a stance.
// Upstream retired the age attestation on 2026-09-23 and adult_confirmed is
// constant true from then on, but the claim is still sent and the upstream
// formula still reads it, so this keeps folding on the pair rather than
// trusting nsfw_display alone.
export const foldContentStance = (
  adultConfirmed: boolean | undefined,
  nsfwDisplay: string | undefined
): KunContentStance => {
  if (!adultConfirmed) return 'hide'
  return nsfwDisplay === 'blur' || nsfwDisplay === 'show' ? nsfwDisplay : 'hide'
}
