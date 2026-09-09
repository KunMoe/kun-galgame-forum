// The avatar fallback pool for this site.
//
// Why this file exists: KunUI's `avatarFallbackPool` defaults to `[]`, and an
// empty pool makes `pickAvatarFallback` return the single bundled
// `KUN_AVATAR_FALLBACK` data URI for every seed. Nothing throws and nothing
// 404s -- every account without an avatar image simply renders the SAME
// picture. 94.3% of NextMoe accounts have no avatar, so that is most of the
// faces here: measured on the live forum before this file existed, one page
// carried 68 avatar <img> elements belonging to 47 distinct users and rendered
// 1 distinct image.
//
// The pick is `hash(user.name) % pool.length`, so this array is an INDEX
// SPACE, not a ranking: replacing an entry in place moves only the users who
// land on it, while changing the array's LENGTH re-assigns everybody. Keep the
// baked fallback below the same length as the live pool.
//
// See @kungal/ui-vue `docs/INTEGRATION.md` 7.1.
interface KunAvatarPoolResponse {
  code: number
  data: { version: string; variant: string; urls: string[] } | null
}

// Baked snapshot of pool version `cd41af02cb0050ea`, so a failed or offline fetch
// degrades to a STALE pool rather than back to one repeated image. Safe to bake
// only because these URLs are content-addressed (`<sha256>_128.webp`): unlike
// the positional `/stickers/{pack}/{n}.webp` path this API replaced, a snapshot
// of them cannot rot.
//
// To refresh (rarely needed -- only to show newer pictures):
//
//   curl -s https://sticker.kungal.com/api/v1/avatar-pool | jq -r '.data.urls[]'
export const AVATAR_POOL_FALLBACK: string[] = [
  'https://image.kungal.iloveren.link/c6/97/c697c3b779ad6c034a538516395a678f068afc1b24203e57d516f71bb09f070f_128.webp',
  'https://image.kungal.iloveren.link/29/13/2913c14bb5ffcde877a936520e2c1585a84dbba0afaf2d388058c22203d62b6c_128.webp',
  'https://image.kungal.iloveren.link/dc/8f/dc8fd453f42d3edd7d51ac8a4ae1c7b698fb24d3166628d3560c5d4386bbf320_128.webp',
  'https://image.kungal.iloveren.link/dc/f1/dcf153b812c8b62c8b93a75077dd38e2272745217dc7a1629c66a715e15660e5_128.webp',
  'https://image.kungal.iloveren.link/a5/38/a538294053a3a87ec08b21d07871d83b1e633c597cfd20188e954b205da457b1_128.webp',
  'https://image.kungal.iloveren.link/d6/01/d601f9745db81e72aeb0e153052ef1faf21427e90d956c2be55ab093fa817cee_128.webp',
  'https://image.kungal.iloveren.link/90/d3/90d3ab0d3b04f5ff814d134c784cffb7b3942484ade8de19c8a11c68246f1c68_128.webp',
  'https://image.kungal.iloveren.link/2c/67/2c6787e02437965dcc0588918b9fd9fe8246f8ab359259f2754743af0e66f716_128.webp',
  'https://image.kungal.iloveren.link/60/16/6016317952f8a945543208666069388cde91014ce07675050adb2557c542838c_128.webp',
  'https://image.kungal.iloveren.link/5e/a8/5ea8108bcab620e35ddb9dea37c19e9f1483cd2613b575d347b4e1afe040b9db_128.webp',
  'https://image.kungal.iloveren.link/54/0b/540b51c8ebab0b47a24d47f2373b9e1a38bc07d2a8bb91ad2e46f632d30fa7a4_128.webp',
  'https://image.kungal.iloveren.link/bf/0a/bf0ac1af8f1a1bc8983b187b66a93c1ccac43a35900d490ae01946fabfa3948d_128.webp',
  'https://image.kungal.iloveren.link/0d/a6/0da696d45c54aa2c8dcac3b26ab83c6774b655af1ab45faba5aec160b008f774_128.webp',
  'https://image.kungal.iloveren.link/54/9a/549a84f2fa2b1d9c92b90183f0d33930292fd9b705c6ceb109f9f7cca61b139d_128.webp',
  'https://image.kungal.iloveren.link/0e/63/0e63093d0255a90960f8fd29d80c80c35647e4ddc3e5c0ee433a760e0e890904_128.webp',
  'https://image.kungal.iloveren.link/e2/6a/e26a73b9c36e9f19a25da7525ced393b1091bf81fa02fcc9c0ff4911b21f6810_128.webp',
  'https://image.kungal.iloveren.link/94/b3/94b3ecb8a46f65584fea5e81d701c99bc498b5c1ee496e68ab7d97ceb35cb44f_128.webp',
  'https://image.kungal.iloveren.link/b1/90/b190356e1faf2a2a26e6e39acae4a4a411e88178ecc4511245ed9380963eb6d6_128.webp',
  'https://image.kungal.iloveren.link/09/70/09706ce22ba70fca249740aecb6fc0e1cdc95554688b70d26fed9d63192e074e_128.webp',
  'https://image.kungal.iloveren.link/f9/24/f924c4fcef2d1c68aaa255a83f1a6c3129256a5e13bc137df6197abcc10f7422_128.webp',
  'https://image.kungal.iloveren.link/f1/d8/f1d8a8e734bedc4810d31a1df9b752df6b0f857cd93065c6411b22cd956c3613_128.webp',
  'https://image.kungal.iloveren.link/78/35/78356e14f95a941e27fcad3c338940f2911e987c28347625a14b50e9c457436a_128.webp',
  'https://image.kungal.iloveren.link/b8/a8/b8a889468a1c6c0062157e0ad5c19a47c80d14cd23ad911cf6ccffa9f67d4fc2_128.webp',
  'https://image.kungal.iloveren.link/37/7d/377d3e657cd8304876ff2a019e4094a6ed845e0d65bd24c88ab038a69e6b5316_128.webp',
  'https://image.kungal.iloveren.link/96/81/9681d8bde368a520468bf8708f56b5738ceeab8762b0e05dc822f38b228efd11_128.webp',
  'https://image.kungal.iloveren.link/86/b9/86b9fe00a8564ed36f89575d5740c1d7c5dd4a174d8807499c479f7c705d6132_128.webp',
  'https://image.kungal.iloveren.link/7e/22/7e22c431d10ddac463ba32960c178eaec9268a267ee5e5c61e3dd4972476fa50_128.webp',
  'https://image.kungal.iloveren.link/d5/eb/d5ebfa226aeaef306a1ebf6a17cf4d4e8b16571df17839d74fe66f62edcfad98_128.webp',
  'https://image.kungal.iloveren.link/ca/f4/caf455d1dcaacfd2819f4a41e91b14778d92c2ca952001ada084de6cb03412e4_128.webp',
  'https://image.kungal.iloveren.link/1a/05/1a05a35370fc2264a1ae53c1feca16278ab932585052e516630e2411abe40399_128.webp',
  'https://image.kungal.iloveren.link/72/d8/72d89db6890d79908a78632a1405ade6497063d10d1c67c5a8d3af70d8bdb0ee_128.webp',
  'https://image.kungal.iloveren.link/b2/a2/b2a2d9bbd7517b70e0a148c73a5499c3c860588f816b65949d587594bebbee60_128.webp',
  'https://image.kungal.iloveren.link/e4/f6/e4f6008a1fc9f970b06e33d91a2bf1613737d52711a31afae281a5503d028af0_128.webp',
  'https://image.kungal.iloveren.link/b7/b7/b7b70f9b4f5332dce6fee9447fb2d7e303795b6f4ea3ea8d31c5cbc40e934d73_128.webp',
  'https://image.kungal.iloveren.link/a1/73/a1732838d54ecf4258d495e93af6c7c88118a624d7073124e32ca5e7bfbdf53a_128.webp',
  'https://image.kungal.iloveren.link/82/ba/82ba7425100eccec3dbc6347ca74efe200d1bf80170c37031f7a43279754e1a6_128.webp',
  'https://image.kungal.iloveren.link/ed/94/ed9465a6b545e16eef6df62fdc8ecfee4295e4ec0d9bae3214294456ac6b92df_128.webp',
  'https://image.kungal.iloveren.link/77/1b/771b0d99e97ca392588ff09bba231c60fbed847ebcdd75fc2d5f339b5ddc42d2_128.webp',
  'https://image.kungal.iloveren.link/36/51/365100c2907c038269c4fb51fa54bc77f07a157bc93f38d428c2f4b057896487_128.webp',
  'https://image.kungal.iloveren.link/72/ee/72ee4cd2b1bb26054ae6f9e62f42f012ea26a26d2352d5ee03a478bccd643c81_128.webp',
  'https://image.kungal.iloveren.link/e9/3e/e93e3198bc55d47931150492660a7ac1584eb110c127e3d0306dee55f36ab99a_128.webp',
  'https://image.kungal.iloveren.link/5f/40/5f4003c2ca61252e8a04a64015579041d8ae713d774dd8461fdb8712781c1703_128.webp',
  'https://image.kungal.iloveren.link/29/a4/29a4fc319249ef3d40c3036d3e386fcf2308e4f85bc33546fe52c0da344848e7_128.webp',
  'https://image.kungal.iloveren.link/49/63/4963541e788beca8acaffc115d5f8e3ad4bfd1404dde4ca986fbc9c76dc912a2_128.webp',
  'https://image.kungal.iloveren.link/ff/d7/ffd7f32c2d43848047d7373a66187c31b05ba64a396b6adc17a9baab9ada49f2_128.webp',
  'https://image.kungal.iloveren.link/83/af/83af6a3fe04e534cf50b984633d8c0087ed03f6f927148377f79a3076ff7e3d2_128.webp',
  'https://image.kungal.iloveren.link/29/8f/298ff6242a4f80e4290363660a1375d1d4949c3961e6427ef60b37377b2a0cff_128.webp',
  'https://image.kungal.iloveren.link/18/d3/18d3223bb64bc1a2a38b60c519bb666dc0d222b3ee2a48f115d4089060186cc9_128.webp',
  'https://image.kungal.iloveren.link/72/e4/72e4f2adfdb5301325a51a89fdfc59b8c0adfc93f6ade4939f329bcb6edcdd88_128.webp',
  'https://image.kungal.iloveren.link/ec/6a/ec6a1a324aced75ea5a28f4c3cfc813db8c06289e77e0078c3251e7a9cdaa052_128.webp',
  'https://image.kungal.iloveren.link/2f/16/2f160616d59a51b294c578191f657ec49ea5ba78e6cee7d63387a52c4a54bba4_128.webp',
  'https://image.kungal.iloveren.link/91/5c/915caccac223dbbab3829d593fdde0b07cab6af776ffe51b32ff4560c3cd19db_128.webp',
  'https://image.kungal.iloveren.link/4b/3f/4b3f944f4b262dc8dd152733db791f7ba4f4197c1587c17d6cecdde21e984389_128.webp',
  'https://image.kungal.iloveren.link/c4/4c/c44c91cb036a7ee197acaa27cce5d9b5eb5d42c40b887e2e113a371f8ca8f974_128.webp',
  'https://image.kungal.iloveren.link/97/cf/97cf96ba7c9e05797dbdcd012e257e4f659fb74527f631b91d57ffe2be7f21e7_128.webp',
  'https://image.kungal.iloveren.link/f4/ff/f4ff43261e1d4754d797410d3657fbdcc586c7a5294d6f88485fe6daf302969b_128.webp',
  'https://image.kungal.iloveren.link/d7/7d/d77d2e1edd96760faa4319ae727e0a77532ff4e2367b73ab6b853d2ee1788d39_128.webp',
  'https://image.kungal.iloveren.link/ea/71/ea71f5394ec86b525f00b1289048fca98a05b0396a4502fcf4965ad581359792_128.webp',
  'https://image.kungal.iloveren.link/66/20/6620f4d2cf32ae46b84481d5dd8c42fd85eb89394b494166c96e1d8395ab9657_128.webp',
  'https://image.kungal.iloveren.link/12/8d/128deb5e43fbfd1aaf1b47f18e9a6b7497c4620cba90129b8cd1094348c36120_128.webp',
  'https://image.kungal.iloveren.link/8e/41/8e415e9945d6c5113dd10bcc6687f91e38c883c1d4479552ae5b86fc82e54d02_128.webp',
  'https://image.kungal.iloveren.link/16/3a/163a237ae4590e2c8c62d0df0144e3e2c1df1704819e0bc2b06449300a0a1235_128.webp',
  'https://image.kungal.iloveren.link/b7/f2/b7f255c80c0aaecaff00eff4e3155dca05335d1ab71f6b7c5c21798273eef1da_128.webp',
  'https://image.kungal.iloveren.link/3e/60/3e609eaf0c40913f5e1273acbbad7833740d105f31e1a8eb5c546c2426bcd10c_128.webp',
]

// One fetch per process, NOT per render. Module scope is safe here precisely
// because the list is immutable and not user-specific -- module-scoped state
// that IS user-visible is shared across SSR requests and leaks between users.
let poolPromise: Promise<string[]> | null = null

export const fetchAvatarPool = (): Promise<string[]> =>
  (poolPromise ??= (async () => {
    try {
      const res = await $fetch<KunAvatarPoolResponse>(
        'https://sticker.kungal.com/api/v1/avatar-pool',
        { timeout: 5000, retry: 1 }
      )
      const urls = res?.data?.urls
      return Array.isArray(urls) && urls.length > 0 ? urls : AVATAR_POOL_FALLBACK
    } catch {
      // Never fail a render over the default avatar.
      return AVATAR_POOL_FALLBACK
    }
  })())
