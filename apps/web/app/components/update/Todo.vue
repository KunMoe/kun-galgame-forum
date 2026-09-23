<script setup lang="ts">
import {
  KUN_TODO_PROJECT_LABEL,
  KUN_TODO_STATES,
  KUN_TODO_STATE_ICON,
  KUN_TODO_STATE_LABEL,
  KUN_TODO_STATE_TEXT_CLASS
} from '~/constants/update'
import { problemMessage } from '#shared/utils/api/message'
import { settle, type ApiResult } from '#shared/utils/api/problem'
import type { Todo, TodoCreate, TodoState } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const api = useApiClient()
const createKey = useIdempotencyKey()

const stateFilter = ref<TodoState | undefined>(undefined)

const filters = [
  { value: undefined, label: '全部', icon: 'lucide:list-filter', text: '' },
  ...KUN_TODO_STATES.map((state) => ({
    value: state,
    label: KUN_TODO_STATE_LABEL[state],
    icon: KUN_TODO_STATE_ICON[state],
    text: KUN_TODO_STATE_TEXT_CLASS[state]
  }))
]

const {
  items: todos,
  hasMore,
  total,
  loadMore,
  loadingMore,
  status,
  problem,
  refresh
} = await useCursorList<Todo>(
  () => `todos:${stateFilter.value ?? 'all'}`,
  (api, cursor, { signal }) =>
    api.GET('/todos', {
      params: {
        query: {
          include_total: true,
          ...(stateFilter.value ? { state: stateFilter.value } : {}),
          ...(cursor ? { cursor } : {})
        }
      },
      signal
    })
)

const showModal = ref(false)
const editing = ref<Todo | null>(null)
const busy = ref(false)

const initialData = computed<TodoCreate | undefined>(() =>
  editing.value
    ? { project: editing.value.project, text: editing.value.text }
    : undefined
)

const openCreate = () => {
  if (!requireLogin()) {
    return
  }
  editing.value = null
  showModal.value = true
}

const openEdit = (todo: Todo) => {
  editing.value = todo
  showModal.value = true
}

// Another board editor may have moved the task since this page loaded; the
// board is stale, so reload it whatever the message said.
const settleWrite = async (result: ApiResult<unknown>, done: string) => {
  if (!result.ok) {
    reportProblem(result.problem)
    if (result.problem.code === 'INVALID_STATE_TRANSITION') {
      await refresh()
    }
    return false
  }
  useMessage(done, 'success')
  await refresh()
  return true
}

const submit = async (body: TodoCreate) => {
  if (busy.value) {
    return
  }
  busy.value = true
  const target = editing.value
  const result = target
    ? await settle(
        api.PATCH('/todos/{todo_id}', {
          params: { path: { todo_id: target.id } },
          body
        })
      )
    : await settle(
        api.POST('/todos', {
          params: {
            header: { 'Idempotency-Key': createKey.take('/todos', body) }
          },
          body
        })
      )
  busy.value = false
  if (await settleWrite(result, target ? '更新成功' : '发布待办成功')) {
    if (!target) {
      createKey.clear()
    }
    showModal.value = false
  }
}

const move = async (todo: Todo, state: TodoState, done: string) => {
  if (busy.value) {
    return
  }
  busy.value = true
  const result = await settle(
    api.PATCH('/todos/{todo_id}', {
      params: { path: { todo_id: todo.id } },
      body: { state }
    })
  )
  busy.value = false
  await settleWrite(result, done)
}

const remove = async (todo: Todo) => {
  const ok = await useComponentMessageStore().alert('确定删除这条待办吗？')
  if (!ok) {
    return
  }
  const result = await settle(
    api.DELETE('/todos/{todo_id}', {
      params: { path: { todo_id: todo.id } }
    })
  )
  await settleWrite(result, '待办已删除')
}
</script>

<template>
  <div class="space-y-6">
    <KunHeader
      name="待办列表"
      description="这里记录了网站将会实现的功能, 以及更改的功能, 包括 Galgame 以及话题, 以及网站所有方向可能发生的各种更新等等"
    >
      <template #endContent>
        <div class="flex justify-end">
          <KunButton @click="openCreate">创建待办</KunButton>
        </div>
      </template>
    </KunHeader>

    <KunCard :is-hoverable="false" :is-transparent="false">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <KunButton
            v-for="filter in filters"
            :key="filter.label"
            size="sm"
            :variant="stateFilter === filter.value ? 'solid' : 'flat'"
            :color="stateFilter === filter.value ? 'primary' : 'default'"
            @click="stateFilter = filter.value"
          >
            <KunIcon
              :name="filter.icon"
              :class="
                cn('h-4 w-4', stateFilter !== filter.value && filter.text)
              "
            />
            {{ filter.label }}
          </KunButton>
        </div>

        <span v-if="total !== undefined" class="text-default-500 text-sm">
          共 {{ total }} 项
        </span>
      </div>
    </KunCard>

    <div
      v-if="status === 'pending' && !todos.length"
      class="flex justify-center py-8"
    >
      <KunLoading description="正在加载待办..." />
    </div>

    <KunNull v-else-if="problem" :description="problemMessage(problem)" />

    <KunNull v-else-if="!todos.length" description="当前筛选条件下没有待办" />

    <template v-else>
      <KunCard
        :is-hoverable="false"
        :is-transparent="false"
        v-for="todo in todos"
        :key="todo.id"
        content-class="space-y-3"
      >
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <KunChip color="primary">
              {{ KUN_TODO_PROJECT_LABEL[todo.project] }}
            </KunChip>

            <span class="text-default-600 text-sm">
              该企划创建于 <KunTime :time="todo.created_at" type="datetime" />
            </span>
          </div>

          <div
            v-if="todo.state === 'in_progress' && todo.claimer"
            class="text-default-500 flex items-center gap-2 text-sm"
          >
            <KunAvatar
              :user="toKunUser(todo.claimer)"
              size="xs"
              :is-navigation="false"
            />
            <span>已被 {{ toKunUser(todo.claimer).name }} 认领</span>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <KunAvatar :user="toKunUser(todo.author)" size="sm" />
          <KunLink
            color="default"
            underline="hover"
            :to="`/user/${todo.author.id}`"
            class-name="text-sm"
          >
            {{ toKunUser(todo.author).name }}
          </KunLink>
        </div>

        <pre class="font-mono break-all whitespace-pre-line">{{
          todo.text
        }}</pre>

        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2 text-sm">
            <span v-if="todo.completed_at" class="text-default-500">
              <KunTime :time="todo.completed_at" type="datetime" />
            </span>
            <KunIcon
              :name="KUN_TODO_STATE_ICON[todo.state]"
              :class="cn('h-4 w-4', KUN_TODO_STATE_TEXT_CLASS[todo.state])"
            />
            <span :class="KUN_TODO_STATE_TEXT_CLASS[todo.state]">
              {{ KUN_TODO_STATE_LABEL[todo.state] }}
            </span>
          </div>

          <div v-if="todo.viewer" class="flex flex-wrap items-center gap-2">
            <KunButton
              v-if="todo.viewer.can_edit"
              variant="flat"
              size="sm"
              @click="openEdit(todo)"
            >
              编辑
            </KunButton>

            <KunButton
              v-if="todo.viewer.can_claim"
              size="sm"
              color="primary"
              @click="move(todo, 'in_progress', '已认领该待办')"
            >
              认领此任务
            </KunButton>

            <KunButton
              v-if="todo.viewer.can_complete"
              size="sm"
              color="primary"
              @click="move(todo, 'done', '待办已完成')"
            >
              完成
            </KunButton>

            <KunButton
              v-if="todo.viewer.can_discard"
              variant="flat"
              size="sm"
              color="danger"
              @click="move(todo, 'discarded', '待办已废弃')"
            >
              废弃
            </KunButton>

            <KunButton
              v-if="todo.viewer.can_release"
              variant="flat"
              size="sm"
              color="warning"
              @click="move(todo, 'pending', '已放弃该待办, 任务回到待处理')"
            >
              放弃
            </KunButton>

            <KunButton
              v-if="todo.viewer.can_reopen"
              size="sm"
              color="primary"
              @click="move(todo, 'pending', '待办已重新启用')"
            >
              重新启用
            </KunButton>

            <KunButton
              v-if="todo.viewer.can_delete"
              variant="light"
              size="sm"
              color="danger"
              @click="remove(todo)"
            >
              删除
            </KunButton>
          </div>
        </div>
      </KunCard>

      <div v-if="hasMore" class="text-center">
        <KunButton variant="flat" :loading="loadingMore" @click="loadMore">
          加载更多
        </KunButton>
      </div>
    </template>

    <UpdateTodoModal
      v-model="showModal"
      :initial-data="initialData"
      :is-editing="!!editing"
      @submit="submit"
    />
  </div>
</template>
