<script setup lang="ts">
import { showMoeMessage } from '~/widget/showMoeMessage'
import { kunFeedUrl } from '#shared/utils/feedUrl'

const {
  showKUNGalgamePageTransparency,
  showKUNGalgameBackgroundBlur,
  showKUNGalgameRounded
} = storeToRefs(usePersistSettingsStore())

const { isOpen: isAuthModalOpen } = useAuthModal()

const route = useRoute()
const config = useRuntimeConfig()

const siteImage = kunOgSiteImage(
  config.public.ogCardEnabled,
  kungal.domain.main
)

useHead({
  htmlAttrs: { lang: 'zh-Hans' },
  meta: [
    {
      name: 'image',
      content: kungal.images[0] ? kungal.images[0].fullUrl : '/kungalgame.webp'
    },
    { property: 'og:image:type', content: 'image/webp' },
    {
      name: 'yandex-verification',
      content: config.public.KUN_VISUAL_NOVEL_FORUM_YANDEX_VERIFICATION
    }
  ],
  templateParams: {
    schemaOrg: {
      host: kungal.domain.main,
      path: route.path,
      inLanguage: 'zh-Hans'
    }
  },
  link: [
    {
      rel: 'alternate',
      type: 'application/rss+xml',
      title: `${kungal.titleShort}话题订阅`,
      href: kunFeedUrl(config.public.KUN_GALGAME_URL || '', '/rss/topic')
    },
    {
      rel: 'alternate',
      type: 'application/rss+xml',
      title: `${kungal.titleShort} Galgame 资源订阅`,
      href: kunFeedUrl(config.public.KUN_GALGAME_URL || '', '/rss/galgame')
    },
    {
      rel: 'alternate',
      type: 'application/rss+xml',
      title: `${kungal.titleShort} Gal 情报订阅`,
      href: kunFeedUrl(config.public.KUN_GALGAME_URL || '', '/rss/news')
    },
    {
      rel: 'me',
      href: `https://mastodon.social/@kungal`,
      type: 'text/html',
      title: 'Mastodon'
    },
    { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' },
    { rel: 'canonical', href: kungal.domain.main }
  ]
})

useSeoMeta({
  titleTemplate: () => kungal.titleTemplate,
  charset: 'utf-8',
  viewport: 'width=device-width, initial-scale=1',
  formatDetection: 'telephone=no',
  description: kungal.description,
  themeColor: kungal.themeColor,

  ogDescription: kungal.description,
  ogLocale: 'zh_CN',
  ogTitle: kungal.title,
  ogSiteName: kungal.title,
  ogType: 'website',

  ogImage: siteImage.url,
  ogImageAlt: kungal.title,

  // These used to claim 1920x1080 image/png for a 1672x941 WebP, which no crawler could
  // reconcile with the bytes it fetched. og:image:type goes through useHead because
  // unhead's ogImageType only types jpeg/gif/png, and every image we serve here is WebP.
  ogImageWidth: siteImage.width,
  ogImageHeight: siteImage.height,

  twitterCard: 'summary_large_image',
  twitterImage: siteImage.url
})

useSchemaOrg([
  defineOrganization({
    name: kungal.titleShort,
    url: kungal.domain.main,
    sameAs: [kungal.github]
  }),
  defineWebSite({ name: kungal.titleShort, description: kungal.description }),
  defineWebPage()
])

onMounted(() => {
  usePersistSettingsStore().setKUNGalgameTransparency(
    showKUNGalgamePageTransparency.value
  )

  usePersistSettingsStore().setKUNGalgameBackgroundBlur(
    showKUNGalgameBackgroundBlur.value
  )

  usePersistSettingsStore().setKUNGalgameRounded(showKUNGalgameRounded.value)

  if (process.env.NODE_ENV === 'development') {
    localStorage.setItem(
      '__VUE_DEVTOOLS_NEXT_PLUGIN_SETTINGS__dev.esm.pinia__',
      '{"logStoreChanges":false}'
    )
    localStorage.setItem('umami.disabled', '1')
  } else {
    showMoeMessage()
  }
})
</script>

<template>
  <div class="kun">
    <KunAlertProvider />
    <KunMessageProvider />
    <KunLoliProvider />

    <KunAuthModal v-model="isAuthModalOpen" />

    <KunCapture />

    <LazyKunTopBarMoemoepointLog />
    <LazyKunTopBarLogout />
    <LazyKunTopBarCreatorApply />

    <LazyReportModal />

    <LazyTopicUpvoteModal />

    <KunFloatingBar />

    <LazyTopicReplyPanel />

    <LazyKunSettingPanel />

    <NuxtLoadingIndicator color="var(--color-primary)" />

    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </div>
</template>

<style>
.kun-page-enter-active,
.kun-page-leave-active {
  transition: all 0.2s ease;
  will-change: transform, opacity;
}

.kun-page-enter-from {
  opacity: 0;
  transform: translateY(20px);
}
.kun-page-enter-to {
  opacity: 1;
  transform: translateY(0);
}

.kun-page-leave-from {
  opacity: 1;
  transform: translateY(0);
}
.kun-page-leave-to {
  opacity: 0;
  transform: translateY(20px);
}
</style>
