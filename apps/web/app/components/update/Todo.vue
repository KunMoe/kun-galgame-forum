<script setup lang="ts">
import { storeToRefs } from 'pinia'
import {
  KUN_TODO_STATUS,
  KUN_TODO_TYPE_MAP,
  KUN_UPDATE_LOG_STATUS_MAP
} from '~/constants/update'
import type { CreateTodoPayload, UpdateTodoPayload } from './types'

const { id: userId } = storeToRefs(usePersistUserStore())

const canEditUpdateLog = useCan('update_log.edit')
const canReopenTodo = useCan('update_log.reopen')

const iconMap: Record<number, string> = {
  0: 'lucide:circle-divide',
  1: 'lucide:loader',
  2: 'lucide:check',
  3: 'lucide:x'
}

const textMap: Record<number, string> = {
  0: 'text-default',
  1: 'text-primary',
  2: 'text-success',
  3: 'text-danger'
}

const statusFilters = [
  { value: undefined, label: '全部', icon: 'lucide:list-filter', text: '' },
  ...Object.entries(KUN_UPDATE_LOG_STATUS_MAP).map(([key, label]) => ({
    value: Number(key),
    label,
    icon: iconMap[Number(key)]!,
    text: textMap[Number(key)]!
  }))
]

const pageData = ref({
  page: 1,
  limit: 30,
  status: undefined as number | undefined
})

const setStatusFilter = (status?: number) => {
  pageData.value.status = status
  pageData.value.page = 1
}

const { data, status, refresh } = await useKunFetch<UpdateTodoList>(
  '/update/todo',
  { query: pageData }
)

const isCreator = (todo: UpdateTodo) => todo.user.id === userId.value

const isClaimer = (todo: UpdateTodo) =>
  !!todo.claimed_user && todo.claimed_user.id === userId.value

const showTodoModal = ref(false)
const editingTodo = ref<UpdateTodoPayload>({} as UpdateTodoPayload)

const openCreateTodoModal = () => {
  if (!requireLogin()) {
    return
  }
  editingTodo.value = {} as UpdateTodoPayload
  showTodoModal.value = true
}

const openEditTodoModal = (log: UpdateTodo) => {
  if (!data.value) {
    return
  }
  editingTodo.value = {
    type: log.type as UpdateTodoPayload['type'],
    content: log.content,
    todo_id: log.id
  } satisfies UpdateTodoPayload
  showTodoModal.value = true
}

const handleTodoAction = async (
  data: CreateTodoPayload | UpdateTodoPayload
) => {
  const isEdit = 'todo_id' in data
  const result = await kunFetch('/update/todo', {
    method: isEdit ? 'PUT' : 'POST',
    body: data
  })

  if (result) {
    useMessage(isEdit ? '更新成功' : '发布待办成功', 'success')
    refresh()
  }
}

const claimTodo = async (todo: UpdateTodo) => {
  const result = await kunFetch('/update/todo/claim', {
    method: 'POST',
    body: { todo_id: todo.id }
  })

  if (result) {
    useMessage('已认领该待办', 'success')
    refresh()
  }
}

const completeTodo = async (todo: UpdateTodo) => {
  const result = await kunFetch('/update/todo/complete', {
    method: 'POST',
    body: { todo_id: todo.id }
  })

  if (result) {
    useMessage('待办已完成', 'success')
    refresh()
  }
}

const discardTodo = async (todo: UpdateTodo) => {
  const result = await kunFetch('/update/todo/discard', {
    method: 'POST',
    body: { todo_id: todo.id }
  })

  if (result) {
    useMessage('待办已废弃', 'success')
    refresh()
  }
}

const releaseTodo = async (todo: UpdateTodo) => {
  const result = await kunFetch('/update/todo/release', {
    method: 'POST',
    body: { todo_id: todo.id }
  })

  if (result) {
    useMessage('已放弃该待办', 'success')
    refresh()
  }
}

const reopenTodo = async (todo: UpdateTodo) => {
  const result = await kunFetch('/update/todo/reopen', {
    method: 'POST',
    body: { todo_id: todo.id }
  })

  if (result) {
    useMessage('待办已重新启用', 'success')
    refresh()
  }
}
</script>

<template>
  <div class="space-y-6" v-if="data">
    <KunHeader
      name="待办列表"
      description="这里记录了网站将会实现的功能, 以及更改的功能, 包括 Galgame 以及话题, 以及网站所有方向可能发生的各种更新等等"
    >
      <template #endContent>
        <div class="flex justify-end">
          <KunButton @click="openCreateTodoModal">创建待办</KunButton>
        </div>
      </template>
    </KunHeader>

    <KunCard :is-hoverable="false" :is-transparent="false">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <KunButton
            v-for="filter in statusFilters"
            :key="filter.label"
            size="sm"
            :variant="pageData.status === filter.value ? 'solid' : 'flat'"
            :color="pageData.status === filter.value ? 'primary' : 'default'"
            @click="setStatusFilter(filter.value)"
          >
            <KunIcon
              :name="filter.icon"
              :class="
                cn('h-4 w-4', pageData.status !== filter.value && filter.text)
              "
            />
            {{ filter.label }}
          </KunButton>
        </div>

        <span class="text-default-500 text-sm">共 {{ data.total }} 项</span>
      </div>
    </KunCard>

    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      v-for="todo in data.todos"
      :key="todo.id"
      content-class="space-y-3"
    >
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <KunChip color="primary">
            {{ KUN_TODO_TYPE_MAP[todo.type] }}
          </KunChip>

          <span class="text-default-600 text-sm">
            该企划创建于 <KunTime :time="todo.created" type="datetime" />
          </span>
        </div>

        <div
          v-if="todo.status === KUN_TODO_STATUS.CLAIMED && todo.claimed_user"
          class="text-default-500 flex items-center gap-2 text-sm"
        >
          <KunAvatar
            :user="todo.claimed_user"
            size="xs"
            :is-navigation="false"
          />
          <span>已被 {{ todo.claimed_user.name }} 认领</span>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <KunAvatar :user="todo.user" size="sm" />
        <KunLink
          color="default"
          underline="hover"
          :to="`/user/${todo.user.id}`"
          class-name="text-sm"
        >
          {{ todo.user.name }}
        </KunLink>
      </div>

      <pre class="font-mono break-all whitespace-pre-line">
        {{ todo.content }}
      </pre>

      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2 text-sm">
          <span v-if="todo.completed_time" class="text-default-500">
            <KunTime :time="todo.completed_time" type="datetime" />
          </span>
          <KunIcon
            :name="iconMap[todo.status]"
            :class="cn('h-4 w-4', textMap[todo.status])"
          />
          <span :class="cn(textMap[todo.status])">
            {{ KUN_UPDATE_LOG_STATUS_MAP[todo.status] }}
          </span>
        </div>

        <div class="flex items-center gap-2">
          <KunButton
            v-if="isCreator(todo) && todo.status === KUN_TODO_STATUS.PENDING"
            variant="flat"
            size="sm"
            color="danger"
            @click="discardTodo(todo)"
          >
            废弃
          </KunButton>

          <KunButton
            v-if="isCreator(todo) && todo.status < KUN_TODO_STATUS.DONE"
            variant="flat"
            size="sm"
            @click="openEditTodoModal(todo)"
          >
            编辑
          </KunButton>

          <KunButton
            v-if="canEditUpdateLog && todo.status === KUN_TODO_STATUS.PENDING"
            size="sm"
            color="primary"
            @click="claimTodo(todo)"
          >
            认领此任务
          </KunButton>

          <KunButton
            v-if="
              (isClaimer(todo) || canEditUpdateLog) &&
              todo.status === KUN_TODO_STATUS.CLAIMED
            "
            size="sm"
            color="primary"
            @click="completeTodo(todo)"
          >
            完成
          </KunButton>

          <KunButton
            v-if="
              (isClaimer(todo) || canEditUpdateLog) &&
              todo.status === KUN_TODO_STATUS.CLAIMED
            "
            size="sm"
            color="danger"
            @click="discardTodo(todo)"
          >
            废弃
          </KunButton>

          <KunButton
            v-if="
              isClaimer(todo) && todo.status === KUN_TODO_STATUS.CLAIMED
            "
            variant="flat"
            size="sm"
            color="warning"
            @click="releaseTodo(todo)"
          >
            放弃
          </KunButton>

          <KunButton
            v-if="
              canReopenTodo && todo.status === KUN_TODO_STATUS.DISCARDED
            "
            size="sm"
            color="primary"
            @click="reopenTodo(todo)"
          >
            重新启用
          </KunButton>
        </div>
      </div>
    </KunCard>

    <KunNull v-if="!data.todos.length" description="当前筛选条件下没有待办" />

    <UpdateTodoModal
      v-model="showTodoModal"
      :initial-data="editingTodo"
      @submit="handleTodoAction"
    />

    <KunCard
      v-if="data.total > pageData.limit"
      :is-hoverable="false"
      :is-transparent="false"
    >
      <KunPagination
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </KunCard>
  </div>
</template>
