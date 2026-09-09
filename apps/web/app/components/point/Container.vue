<script setup lang="ts">
</script>

<template>
  <div class="space-y-6 pb-12">
    <KunHeader
      name="萌萌点说明"
      description="萌萌点是整个鲲 Galgame 生态通用的社区积分。它不能买东西, 更像是对贡献的感谢: 你帮忙整理内容、认真写点评、热心回帖, 就会得到别人的谢谢(萌萌点); 用来鼓励分享和互动的几个动作, 则需要花一点萌萌点。"
    />

    <KunCard class="space-y-3">
      <h2 class="text-foreground text-lg font-bold">平时怎么赚萌萌点?</h2>
      <ul class="text-default-600 list-disc space-y-1.5 pl-5 text-sm">
        <li>
          <b class="text-foreground">发内容 +3</b>: 发布一个话题、添加一条游戏资源、新建一个工具、给工具传一个资源,
          都是 +3。游戏条目比较特殊——你的投稿在正式通过审核、被收录时, 会给你 +3。
        </li>
        <li>
          <b class="text-foreground">别人回复你 +1</b>: 有人在你的话题下回帖, 你 +1;
          有人在你某条回复下发评论, 你 +1。
        </li>
        <li>
          <b class="text-foreground">被点赞/收藏 +1</b>: 话题、回复、评论、游戏、游戏资源、评分、问答……这些内容被别人点赞/收藏时,
          作者 +1; 取消点赞会把这 1 点收回去(见下面小贴士)。
        </li>
        <li>
          <b class="text-foreground">认真写评分 +3 / +5 / +10</b>: 游戏评分按"短评"长短给奖励——写够 233
          字奖励 +5, 写够 666 字奖励 +10, 更短的也有 +3。
        </li>
        <li>
          <b class="text-foreground">出题 +2</b>: 给游戏出一道理性问答题目, 出题人 +2。
        </li>
        <li>
          <b class="text-foreground">被选为最佳回答 +7</b>: 你的回复被楼主设为"最佳回答", +7。
        </li>
        <li>
          <b class="text-foreground">话题被推荐 +5</b>: 有人花萌萌点把你的话题顶上热榜, 楼主收到 +5。
        </li>
        <li>
          <b class="text-foreground">编辑被采纳 +1</b>: 你对游戏条目提交的修改被合并, 提案人 +1。
        </li>
        <li>
          <b class="text-foreground">每天签到</b>: 每天签到拿随机 0~7 点。
        </li>
        <li>部分站内活动(比如抽奖)也会发放萌萌点, 以活动说明为准。</li>
      </ul>
    </KunCard>

    <KunCard class="space-y-3">
      <h2 class="text-foreground text-lg font-bold">什么时候会花掉萌萌点?</h2>
      <ul class="text-default-600 list-disc space-y-1.5 pl-5 text-sm">
        <li>
          <b class="text-foreground">推话题 -10</b>: 想帮别人把话题顶起来, 每次要花 10 点(同时楼主收到
          +5)。余额不足 10 点时不能推, 也不能推自己的话题。
        </li>
        <li>
          <b class="text-foreground">发到"需要消耗"的板块 -10</b>: 这类板块发话题前会先扣 10 点,
          而且这种帖子不再给你 +3 的发帖奖励。余额不足时不能发。
        </li>
        <li>
          <b class="text-foreground">改名 -17</b>: 修改一次用户名花 17 点, 余额不足时改不了。
        </li>
        <li>
          <b class="text-foreground">删除内容会被"回扣"</b>: 站里不鼓励发完就删, 所以删除会按内容类型收回之前给过你的奖励,
          删得越"热"的扣得越多——
          <ul class="text-default-600 mt-1 list-disc space-y-1 pl-5">
            <li>
              <b class="text-foreground">删评分</b>: 退回当初拿到的奖励(按原短评长度 3 / 5 / 10 点);
            </li>
            <li>
              <b class="text-foreground">删问答题目</b>: 收回出题奖励, -2;
            </li>
            <li>
              <b class="text-foreground">删自己的回复</b>: 扣 <b class="text-foreground">3 × (被评论数 + 被赞数 +
              1)</b>——这条回复收到的每条评论、每个赞, 当初都给过你 +1, 删掉时一并收回;
            </li>
            <li>
              <b class="text-foreground">删自己的评论</b>: 扣 <b class="text-foreground">3 × (被赞数 +
              1)</b>;
            </li>
            <li>
              <b class="text-foreground">删一条游戏资源</b>: 扣 <b class="text-foreground">(被赞数 +
              5)</b>;
            </li>
            <li>
              <b class="text-foreground">删工具 / 删工具资源</b>: 各 -3。
            </li>
          </ul>
          删除时如果余额不够扣, 系统会提示"萌萌点不足"而不让删。
        </li>
        <li>
          <b class="text-foreground">最佳回答被撤销 -7</b>: 你被设为"最佳回答"时 +7;
          如果楼主之后取消了这条最佳回答, 会把这 7 点收回去。
        </li>
        <li>
          <b class="text-foreground">取消点赞 / 取消收藏</b>: 会把之前送给对方的那 1 点从对方那里收回来(对方
          -1), 所以点下去之前想清楚, 别反复横跳。
        </li>
        <li>
          <b class="text-foreground">内容被平台处理</b>: 违规内容被移除时, 之前因此拿到的奖励可能一并收回, 以审核结果为准。
        </li>
      </ul>
    </KunCard>

    <KunCard class="space-y-3">
      <h2 class="text-foreground text-lg font-bold">几条小贴士</h2>
      <ul class="text-default-600 list-disc space-y-1.5 pl-5 text-sm">
        <li>
          点赞是"把 1 点送给对方", 取消点赞会把那 1 点从对方那里拿回来——所以点下去之前想一下, 别反复横跳。
        </li>
        <li>
          你当天能发的话题数跟余额挂钩: 大约 <b class="text-foreground">每 10 点萌萌点 + 1 篇</b>
          上限(比如 100 点 = 一天最多 11 篇)。
        </li>
        <li>被管理员处理的内容可能被扣回之前的奖励; 具体以审核结果为准。</li>
        <li>
          规则会随版本更新, 本页只是把当前规则说清楚; 具体每一笔收支都能在右上角 <b class="text-foreground">萌萌点明细</b> 里看到。
        </li>
      </ul>
    </KunCard>
  </div>
</template>
