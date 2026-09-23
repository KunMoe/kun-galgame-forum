import type { AppPlatform } from '#shared/utils/api/schemas'
import { kungal } from '~/config/kungal'

export const KUN_APP_DOWNLOAD_PAGE = `${kungal.domain.main}/app`

export const KUN_APP_PLATFORMS: {
  key: AppPlatform
  label: string
  icon: string
  hint: string
}[] = [
  {
    key: 'android',
    label: 'Android',
    icon: 'mdi:android',
    hint: '下载 APK 后安装，首次安装需允许「安装未知来源应用」'
  },
  {
    key: 'ios',
    label: 'iOS',
    icon: 'mdi:apple',
    hint: '以 IPA 形式提供，需要自行侧载安装'
  },
  {
    key: 'windows',
    label: 'Windows',
    icon: 'mdi:microsoft-windows',
    hint: '适用于 Windows 10 及以上版本'
  },
  {
    key: 'linux',
    label: 'Linux',
    icon: 'mdi:linux',
    hint: '适用于主流 x64 发行版'
  }
]
