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
          <b class="text-foreground">推话题 -10</b>: 想把别人的话题顶起来, 需要花 10 点(对方得 5)。萌萌点不够时不能推。
        </li>
        <li>
          <b class="text-foreground">发到"需要消耗"的板块 -10</b>: 这类板块发话题要先扣 10 点, 而且不会给你发帖奖励。
        </li>
        <li>
          <b class="text-foreground">改名 -17</b>: 修改用户名要花 17 点。
        </li>
        <li>
          <b class="text-foreground">删除内容会扣回</b>: 站里不鼓励"发完就删"。删得越"热"的内容扣得越多——删自己一条回复要扣
          <b class="text-foreground">3 × (被评论数 + 被赞数 + 1)</b>, 删自己一条评论扣
          <b class="text-foreground">3 × (被赞数 + 1)</b>。删评分会退回当初拿到的奖励(3/5/10),
          删一条游戏资源会扣 <b class="text-foreground">(被赞数 + 5)</b>, 删工具 / 工具资源各 -3, 所以删之前想清楚。
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
