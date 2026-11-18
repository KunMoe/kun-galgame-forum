<script setup lang="ts">
const { data, problem, pending, form, isSaving, save } = useAdminSettingsForm({
  dlsite_link_template: '',
  dlsite_coupon_url: ''
})
</script>

<template>
  <AdminSettingSection
    title="DLsite 补票链接"
    description="Galgame 页面补票提示里的购买链接与优惠券落地页"
    :problem="problem"
    :pending="pending"
    :is-ready="!!data"
    :is-saving="isSaving"
    @save="save"
  >
    <KunInput v-model="form.dlsite_link_template" label="联盟链接模板" />
    <p class="text-default-500 text-xs">
      {workno} 会替换成作品编号。商品还没有铸好的短链时使用它;
      留空则关闭购买链接
    </p>
    <KunInput v-model="form.dlsite_coupon_url" label="优惠券落地页" />
    <p class="text-default-500 text-xs">
      必须是短链 (合作方要求)。store 面报告有活动时会被活动链接顶掉;
      留空则不展示
    </p>
  </AdminSettingSection>
</template>
