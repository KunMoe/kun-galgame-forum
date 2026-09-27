import { kungal } from '~/config/kungal'

export const showMoeMessage = () => {
  const asciiArt = String.raw`
    __ ____  ___   __   _________    __
   / //_/ / / / | / /  / ____/   |  / /
  / ,< / / / /  |/ /  / / __/ /| | / /
 / /| / /_/ / /|  /  / /_/ / ___ |/ /___
/_/ |_\____/_/ |_/   \____/_/  |_/_____/
`

  // The `ascii` entry below carries a SANCTIONED EXCEPTION to 铁律 #1 (no
  // gradients): the console startup banner's text gradient. Listed in
  // CLAUDE.md; do NOT remove it in a no-gradient sweep.
  const styles = {
    ascii: `
      font-family: Consolas, Menlo, 'DejaVu Sans Mono', 'Courier New', monospace;
      font-weight: bold;
      font-size: 12px;
      line-height: 1.2;
      color: transparent;
      background: linear-gradient(45deg, #66AAF9 0%, #FF95E1 100%);
      -webkit-background-clip: text;
      background-clip: text;
      text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.1);
    `,

    mainInfo: `
      font-size: 16px;
      padding: 8px 15px;
      border-radius: 8px;
      background: #e6f1fe;
      color: #FF4ECD;
      font-weight: bold;
      line-height: 1.6;
      text-shadow: 1px 1px 2px #F9C97C;
    `,

    secondaryInfo: `
      font-size: 13px;
      color: #7828C8;
      font-style: italic;
      line-height: 1.5;
    `,

    supportInfo: `
      font-size: 14px;
      color: #333;
      font-weight: bold;
    `,

    telegramInvite: `
      font-size: 14px;
      color: #007bff;
    `,

    telegramLink: `
      font-size: 14px;
      background: #e6f1fe;
      color: #007bff;
      padding: 4px 10px;
      border-radius: 5px;
      font-family: 'Courier New', Courier, monospace;
    `,

    r18: `
      font-size: 10px;
      color: #333;
    `
  }

  console.log(`%c${asciiArt}`, styles.ascii)

  console.groupCollapsed(
    '%c🔞 （鲲的秘密花园）点击这里展开更多信息~ 🔞',
    'color: #FF71D7; font-size: 14px; cursor: pointer;'
  )

  console.log('')

  console.log(
    '%c🍭 欸嘿嘿嘿！捕捉到一只可爱的萝莉开发者！恭喜您发现这片秘密花园喵~',
    styles.mainInfo
  )

  console.log(
    `%c ${kungal.titleShort} 能为您带来良好的体验，是我们最大的荣幸~💖谢谢您看到这里！`,
    styles.secondaryInfo
  )

  console.log('')

  console.log(
    '%c 本论坛的所有技术栈, 网站的代码, 子网站, 均为自研, 没有搬运任何的建站框架, 这是一种自豪!',
    styles.supportInfo
  )

  console.log('')

  console.log(
    '%c 如果您对 Web 开发、Galgame 汉化，或是任何计算机技术感兴趣，都热烈欢迎加入我们的计算机大家庭',
    styles.telegramInvite
  )
  console.log(
    '%c 这里开发群，加群需要有头像，最好是可爱的孩子，嗯！',
    styles.telegramInvite
  )
  console.log(
    '%c 🚀 Telegram 群组: https://telegram.me/KUNForum',
    styles.telegramLink
  )

  console.log('')

  console.log(
    '%c 啊嘞~ 杂鱼❤大哥哥想看咱的 R18CG ? 才不会杂鱼看呢, 哼',
    styles.r18
  )

  console.groupEnd()
}
