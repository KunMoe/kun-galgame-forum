<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type {
  WorkMergeFold,
  WorkMergeFoldProblem
} from '#shared/utils/api/schemas'

definePageMeta({ middleware: 'admin' })
useKunDisableSeo('作品合并记录')

const TABLE_LABELS: Record<WorkMergeFoldProblem['table_name'], string> = {
  galgame: '作品本身',
  galgame_resource: '资源',
  galgame_activity: '动态',
  galgame_comment_community_map: '旧评论映射',
  galgame_resource_relocation: '资源迁移记录',
  galgame_like: '点赞',
  galgame_favorite: '收藏',
  galgame_contributor: '贡献者',
  galgame_quiz_galgame: '题目关联',
  galgame_view_daily: '浏览量',
  feed_activity: '首页动态'
}

const CAUSE_LABELS: Record<WorkMergeFoldProblem['cause'], string> = {
  gone: '合并后已被删除, 或它被并入的那一条已被删除',
  elsewhere: '现在挂在另一部作品上',
  conflict: '原作品上已有同一条',
  purged_user: '作者已被清除',
  survivor_folded:
    '保留方又被合并了一次, 原作品带来的发布 / 禁令 / 创建者标记可能留在了最终那部作品上'
}

type FoldPage = { items: WorkMergeFold[]; total: number }

const api = useApiClient()
const page = ref(1)
const limit = 20
const data = ref<FoldPage | null>(null)
const loading = ref(false)

const load = async () => {
  loading.value = true
  const result = await settle(
    api.GET('/admin/work-merge-folds', {
      params: { query: { page: page.value, limit } }
    })
  )
  loading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  data.value = result.data
}

onMounted(load)
watch(page, load)

const totalPage = computed(() =>
  data.value ? Math.max(1, Math.ceil(data.value.total / limit)) : 1
)

const counts = (fold: WorkMergeFold) =>
  [
    ['迁移', fold.moved_count],
    ['同用户重复而移出', fold.dropped_count],
    ['合计入保留方', fold.summed_count],
    ['首页动态改指向', fold.repointed_count]
  ].filter(([, n]) => Number(n) > 0) as [string, number][]
</script>

<template>
  <div class="w-full space-y-4">
    <div>
      <h1 class="text-2xl font-bold">作品合并记录</h1>
      <p class="text-default-500 text-sm">
        鲲 Galgame 论坛数据合并两部作品后,
        本站把被合并作品上的资源、动态、点赞、收藏、贡献者、题目和浏览量迁到保留方,
        每一条都记在这里。合并被撤销或改指向时按记录原样搬回,
        合并后才在保留方产生的数据留在保留方; 搬不回的条目列在下面。 本记录从
        2026-10-21 开始, 之前的合并没有记录, 维持现状。
      </p>
    </div>

    <KunLoading v-if="loading && !data" />
    <KunNull v-else-if="!data?.items.length" description="暂无合并记录" />

    <template v-else>
      <div
        v-for="fold in data.items"
        :key="fold.id"
        class="border-default-200 space-y-2 rounded-lg border p-3"
      >
        <div class="flex flex-wrap items-center gap-2 text-sm">
          <KunChip size="sm" :color="fold.undone_at ? 'warning' : 'primary'">
            {{ fold.undone_at ? '已撤销' : '生效中' }}
          </KunChip>
          <KunLink :to="`/galgame/${fold.old_work_id}`">
            #{{ fold.old_work_id }}
          </KunLink>
          <KunIcon name="lucide:arrow-right" class="text-default-400 size-4" />
          <KunLink :to="`/galgame/${fold.new_work_id}`">
            #{{ fold.new_work_id }}
          </KunLink>
          <KunTime
            :time="fold.folded_at"
            type="datetime"
            show-year
            class="ml-auto text-xs"
          />
        </div>

        <div class="flex flex-wrap gap-1.5">
          <KunChip
            v-for="[label, n] in counts(fold)"
            :key="label"
            size="sm"
            variant="flat"
          >
            {{ label }} {{ n }}
          </KunChip>
          <span v-if="fold.undone_at" class="text-default-500 text-xs">
            撤销于
            <KunTime :time="fold.undone_at" type="datetime" show-year />
          </span>
        </div>

        <div v-if="fold.problems.length" class="space-y-1">
          <p class="text-danger text-xs font-medium">
            {{ `有 ${fold.problems.length} 条没能搬回` }}
          </p>
          <ul class="text-default-600 space-y-0.5 text-xs">
            <li v-for="p in fold.problems" :key="`${p.table_name}:${p.row_id}`">
              {{ TABLE_LABELS[p.table_name] }} #{{ p.row_id }}:
              {{ CAUSE_LABELS[p.cause] }}
              <template v-if="p.current_work_id">
                (现在在
                <KunLink :to="`/galgame/${p.current_work_id}`">
                  #{{ p.current_work_id }}
                </KunLink>
                )
              </template>
            </li>
          </ul>
        </div>
      </div>

      <KunPagination
        v-if="totalPage > 1"
        v-model:current-page="page"
        :total-page="totalPage"
        :is-loading="loading"
      />
    </template>
  </div>
</template>
