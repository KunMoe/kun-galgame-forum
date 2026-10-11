<template>
  <KunAccordion variant="bordered">
    <KunAccordionItem value="notice" title="首次发布资源必看！">
      <div class="space-y-4">
        <KunInfo title="相关文档" color="info" icon="lucide:book-open">
          <ul class="list-none space-y-1">
            <li>
              <KunLink to="/doc/about-galgame-resource">
                关于 Galgame 资源的盗链界定
              </KunLink>
            </li>
            <li>
              <KunLink to="/doc/galgame-publish-rule">
                Galgame 资源发布规定
              </KunLink>
            </li>
          </ul>
        </KunInfo>

        <KunInfo
          title="萌萌点奖励"
          color="secondary"
          icon="lucide:lollipop"
          description="每发布一个有效的资源链接，您可以获得 3 萌萌点。若链接被删除，通过该链接获得的萌萌点也会被扣除。"
        />

        <KunInfo
          title="其它注意事项"
          color="warning"
          icon="lucide:triangle-alert"
        >
          <ul class="list-disc space-y-2 pl-5">
            <li>
              所有链接为用户添加，
              <strong> 鲲 Galgame 论坛不提供任何下载功能 </strong>
              。
            </li>
            <li>
              所有资源链接必须与该页面游戏相关，<strong>必须免费</strong>，不得需要积分或付费下载。
            </li>
            <li>
              可直接粘贴网盘分享文本，系统会自动识别链接、网盘种类和提取码。提交时链接需为纯链接。
            </li>
            <li>
              <strong>禁止使用网盘的二维码分享</strong
              >：请填写网盘的纯文本分享链接（如
              https://pan.baidu.com/s/…），不要填写二维码图片、二维码图片的图床链接或微信
              / QQ 小程序链接，这类链接会被直接拒绝。
            </li>
            <li>
              允许重复贡献不同类型的资源链接（如不同网盘），但同类型网盘链接每人仅限一次。
            </li>
            <li>
              允许指向其它网址，但该网址必须开放直接下载（禁止需要登录才能下载）。
            </li>
            <li>
              <strong>不要添加网盘下载站的页面链接</strong>，详见
              <KunLink
                to="https://www.kungal.com/zh-cn/galgame/819"
                size="sm"
                target="_blank"
              >
                此 Galgame
              </KunLink>
              下的评论区原因。
            </li>
            <li>
              违规链接可举报。若有其它需求，请在“其它 -> 论坛相关”版块发布话题。
            </li>
          </ul>
        </KunInfo>
      </div>
    </KunAccordionItem>
  </KunAccordion>
</template>
