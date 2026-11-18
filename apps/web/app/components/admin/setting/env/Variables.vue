<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import { problemMessage } from '#shared/utils/api/message'
import type { EnvVariable } from '#shared/utils/api/schemas'
import {
  KUN_ENV_ISSUE,
  KUN_ENV_KIND,
  KUN_ENV_SOURCE,
  envValueText
} from '~/constants/setting-env'

const api = useApiClient()
const { data, problem, pending, refresh } = useKunApi(
  'admin-env-config',
  (client, { signal }) => client.GET('/admin/settings/env', { signal })
)

const filterTabs = [
  { value: 'issues', textValue: '有问题' },
  { value: 'all', textValue: '全部' }
]
const filter = ref('issues')

const variables = computed(() => {
  const all = data.value?.variables ?? []
  return filter.value === 'all' ? all : all.filter((v) => v.issues.length)
})

const editing = ref<EnvVariable | null>(null)
const isEditOpen = ref(false)
const edit = (variable: EnvVariable) => {
  editing.value = variable
  isEditOpen.value = true
}

const remove = async (variable: EnvVariable) => {
  const ok = await useComponentMessageStore().alert(
    `删除 ${variable.variable} 的存储值`,
    '重启 kungal-api 后, 它回落到环境变量, 再回落到代码默认值。确认删除吗?'
  )
  if (!ok) return
  const result = await settle(
    api.DELETE('/admin/settings/env/variables/{variable}', {
      params: { path: { variable: variable.variable } }
    })
  )
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('已删除存储值', 'success')
  await refresh()
}
</script>

<template>
  <div class="space-y-4">
    <KunNull v-if="problem" :description="problemMessage(problem)" />
    <KunSkeleton v-else-if="pending && !data" height="20rem" rounded="lg" />

    <template v-else-if="data">
      <KunInfo
        v-if="!data.is_store_enabled"
        color="warning"
        title="配置中心未开启"
        description="kungal-api 启动时没有 KUN_CONFIG_KEK, 不读取也不能写入配置中心。在 Dokploy 面板设置它并重新部署"
      />
      <KunInfo
        v-else
        color="info"
        title="取值顺序"
        description="环境变量 > 配置中心 > 代码默认。kungal-api 只在启动时读取配置中心, 改动在下次重启后生效。启动接线与拓扑变量留在 compose 文件里"
      />

      <div class="flex flex-wrap items-center justify-between gap-2">
        <KunTab
          v-model="filter"
          :items="filterTabs"
          variant="underlined"
          size="sm"
        />
        <span class="text-default-500 text-xs">
          kungal-api 读取配置中心于
          <KunTime :time="data.loaded_at" type="datetime" show-year />
        </span>
      </div>

      <KunNull
        v-if="!variables.length"
        :description="filter === 'all' ? '没有变量' : '没有需要处理的变量'"
      />
      <div
        v-for="variable in variables"
        :key="variable.variable"
        class="border-default-200 space-y-2 rounded-lg border p-3"
      >
        <div class="flex flex-wrap items-center gap-2">
          <span class="font-mono text-sm font-semibold break-all">
            {{ variable.variable }}
          </span>
          <KunChip
            size="xs"
            :color="KUN_ENV_KIND[variable.variable_kind].color"
          >
            {{ KUN_ENV_KIND[variable.variable_kind].label }}
          </KunChip>
          <KunChip v-if="variable.resolved_from" size="xs" variant="flat">
            来自{{ KUN_ENV_SOURCE[variable.resolved_from] }}
          </KunChip>
          <KunTooltip
            v-for="issue in variable.issues"
            :key="issue"
            :text="KUN_ENV_ISSUE[issue].hint"
          >
            <KunChip size="xs" :color="KUN_ENV_ISSUE[issue].color">
              {{ KUN_ENV_ISSUE[issue].label }}
            </KunChip>
          </KunTooltip>
          <div
            v-if="variable.is_storable && data.is_store_enabled"
            class="ml-auto flex gap-1"
          >
            <KunButton size="sm" variant="light" @click="edit(variable)">
              {{ variable.stored ? '修改存储值' : '存入配置中心' }}
            </KunButton>
            <KunButton
              v-if="variable.stored"
              size="sm"
              variant="light"
              color="danger"
              @click="remove(variable)"
            >
              删除存储值
            </KunButton>
          </div>
        </div>
        <dl class="grid gap-x-4 gap-y-1 text-xs sm:grid-cols-[auto_1fr]">
          <dt class="text-default-500">当前生效</dt>
          <dd class="font-mono break-all">
            {{ envValueText(variable.running) }}
          </dd>
          <template v-if="variable.stored">
            <dt class="text-default-500">配置中心</dt>
            <dd class="font-mono break-all">
              {{ envValueText(variable.stored) }}
            </dd>
          </template>
          <template v-if="variable.default_value">
            <dt class="text-default-500">代码默认</dt>
            <dd class="font-mono break-all">{{ variable.default_value }}</dd>
          </template>
        </dl>
      </div>

      <KunCard
        v-if="data.unread_variables.length"
        :is-hoverable="false"
        :is-transparent="false"
        content-class="space-y-2"
      >
        <h3 class="font-semibold">没有代码读取的变量</h3>
        <p class="text-default-500 text-xs">
          kungal-api 的环境里有, 但代码不读: 多半是消费者已删除或名字拼错,
          可以从 compose 和面板里删掉
        </p>
        <div class="flex flex-wrap gap-1.5">
          <KunChip
            v-for="name in data.unread_variables"
            :key="name"
            size="sm"
            variant="flat"
            color="warning"
          >
            {{ name }}
          </KunChip>
        </div>
      </KunCard>
    </template>

    <AdminSettingEnvEditModal
      v-model="isEditOpen"
      :variable="editing"
      @saved="refresh"
    />
  </div>
</template>
