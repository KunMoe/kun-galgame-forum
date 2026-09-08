import tailwindcss from '@tailwindcss/vite'
import { ICON_COLLECTIONS, ICON_NAMES } from './lib/icon'
import type { TSConfig } from 'pkg-types'

const sharedTsConfig: TSConfig = {
  exclude: ['**/backup/**', '**/dist/**', '**/node_modules/**']
}

export default defineNuxtConfig({
  extends: ['@kungal/ui-nuxt', '@kungal/editor-nuxt'],

  devtools: { enabled: false },

  app: {
    pageTransition: { name: 'kun-page', mode: 'out-in' },
    layoutTransition: { name: 'kun-page', mode: 'out-in' },

    baseURL: '/'
  },

  experimental: {
    scanPageMeta: true,
    typescriptPlugin: true
  },

  compatibilityDate: '2025-07-15',

  devServer: {
    host: '127.0.0.1',
    port: 2333
  },

  modules: [
    '@nuxt/image',
    '@nuxt/icon',
    '@nuxt/eslint',
    '@nuxtjs/color-mode',
    '@pinia/nuxt',
    '@dxup/nuxt',
    'pinia-plugin-persistedstate/nuxt',
    'nuxt-schema-org',
    '@nuxtjs/sitemap',
    'nuxt-umami',
    '@nextmoe/edit-ui-nuxt'
  ],

  // The module's default prefix is 'Edit'; this repo's templates were written
  // against the local copy's <Editkit*> names, so the prefix is the contract.
  editUi: {
    prefix: 'Editkit'
  },

  runtimeConfig: {
    apiBaseUrl: process.env.API_BASE_URL || 'http://127.0.0.1:2334',

    imageCdnBase:
      process.env.IMAGE_CDN_BASE || 'https://image.kungal.iloveren.link',

    // Where /api/sticker-packs proxies from. Server-side only: the browser
    // never reaches across to the sticker site.
    stickerBaseUrl:
      process.env.STICKER_BASE_URL || 'https://sticker.kungal.com',

    // The og renderer's per-site HMAC key. Set at runtime as NUXT_OG_SITE_KEY — the image
    // is built by CI without build args, so a process.env default here is empty in prod.
    ogSiteKey: '',
    ogBaseUrl: 'https://og.nextmoe.dev',

    public: {
      // NUXT_PUBLIC_OG_CARD_ENABLED. Off until og.nextmoe.dev is deployed with a forum key:
      // pages must not advertise a 1200x630 og:image the /og routes cannot actually produce.
      ogCardEnabled: false,

      KUN_GALGAME_URL: process.env.KUN_GALGAME_URL,
      KUN_VISUAL_NOVEL_FORUM_YANDEX_VERIFICATION:
        process.env.KUN_VISUAL_NOVEL_FORUM_YANDEX_VERIFICATION,

      apiBaseUrl: process.env.API_BASE_URL || 'http://127.0.0.1:2334',

      oauthServerUrl:
        process.env.OAUTH_SERVER_URL || 'http://127.0.0.1:9277/api/v1',
      oauthFrontendUrl:
        process.env.OAUTH_FRONTEND_URL || 'https://oauth.kungal.com',
      oauthClientId: process.env.OAUTH_CLIENT_ID || '',
      oauthRedirectUri:
        process.env.OAUTH_REDIRECT_URI || 'http://127.0.0.1:2333/auth/callback',

      imageCdnBase:
        process.env.IMAGE_CDN_BASE || 'https://image.kungal.iloveren.link'
    }
  },

  routeRules: {
    '/emoji/**': { headers: { 'cache-control': 'public, max-age=2592000' } },
    // Without this the retired path falls through to /galgame/[gid] and
    // answers "未找到这个 Galgame" instead of 404ing or moving.
    '/galgame/library': { redirect: { to: '/gallib', statusCode: 301 } },
    '/galgame-resource': {
      redirect: { to: '/galgame/resource', statusCode: 301 }
    },
    '/galgame-resource/**': {
      redirect: { to: '/galgame/resource/**', statusCode: 301 }
    }
  },

  imports: {
    dirs: ['./composables', './config', './utils'],
    presets: [
      {
        from: '@kungal/ui-core',
        imports: ['cn', 'randomNum']
      }
    ]
  },

  site: {
    url: process.env.KUN_GALGAME_URL || 'https://www.kungal.com'
  },

  sitemap: {
    exclude: [
      '/admin',
      '/admin/**',
      '/auth/**',
      '/edit/**',
      '/message',
      '/message/**',
      '/report',
      '/user',
      '/rss'
    ],
    defaultSitemapsChunkSize: 1000,
    sitemaps: {
      kungal: {
        includeAppSources: true,
        sources: ['/api/__sitemap__/urls'],
        chunks: true
      }
    },
    cacheMaxAgeSeconds: 60 * 60 * 6,
    defaults: { changefreq: 'daily', priority: 0.7 }
  },

  umami: {
    id:
      process.env.KUN_VISUAL_NOVEL_FORUM_UMAMI_ID ||
      '2fc714ed-ed3b-459a-b52c-65f7e1621834',
    host: 'https://umami.kungal.org/',
    autoTrack: true,
    ignoreLocalhost: true
  },

  css: ['~/styles/index.css'],

  icon: {
    mode: 'svg',
    localApiEndpoint: '/_nuxt_icon',
    serverBundle: {
      collections: ICON_COLLECTIONS
    },
    clientBundle: {
      icons: ICON_NAMES,
      scan: false
    }
  },

  typescript: {
    tsConfig: {
      ...sharedTsConfig
    },
    nodeTsConfig: {
      ...sharedTsConfig
    },
    sharedTsConfig: {
      ...sharedTsConfig
    }
  },

  vite: {
    plugins: [tailwindcss()]
  },

  eslint: {
    config: {
      stylistic: false
    }
  },

  pinia: {
    storesDirs: ['./store/**']
  },

  piniaPluginPersistedstate: {
    cookieOptions: {
      maxAge: 60 * 60 * 24 * 7,
      sameSite: 'strict'
    }
  },

  colorMode: {
    preference: 'system',
    fallback: 'light',
    globalName: '__KUNGALGAME_COLOR_MODE__',
    componentName: 'ColorScheme',
    classPrefix: 'kun-',
    classSuffix: '-mode',
    storageKey: 'kungalgame-color-mode'
  },

  nitro: {
    typescript: {
      tsConfig: {
        ...sharedTsConfig
      }
    }
  }
})
