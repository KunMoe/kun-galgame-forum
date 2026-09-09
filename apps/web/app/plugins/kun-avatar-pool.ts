import { installKunUIConfig } from '@kungal/ui-vue'
import { AVATAR_POOL_FALLBACK, fetchAvatarPool } from '~/utils/avatarPool'

// Hands KunUI the images it picks a default avatar from. Without this the pool
// is `[]` and every avatar-less account on the site renders one identical
// picture -- see `utils/avatarPool.ts` for the measurement.
//
// `useState` is what keeps server and client picking the SAME image: the list
// rides the Nuxt payload, so hydration sees the same array -- and therefore the
// same `hash(name) % pool.length` -- as the server did. Refetching in the
// browser could return a pool of a different length and reshuffle every avatar
// on hydration.
//
// `installKunUIConfig` merges over whatever a previous call installed, so this
// plugin and the @kungal/ui-nuxt layer's own config plugin may run in either
// order.
export default defineNuxtPlugin(async (nuxtApp) => {
  const pool = useState<string[]>('kun-avatar-pool', () => AVATAR_POOL_FALLBACK)
  if (import.meta.server) pool.value = await fetchAvatarPool()
  installKunUIConfig(nuxtApp.vueApp, { avatarFallbackPool: pool.value })
})
