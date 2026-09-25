<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import type { TraitSummary } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { traitContext } from '~/utils/galgame/trait'

const query = ref('')
const results = ref<TraitSummary[]>([])
const answered = ref('')
const failed = ref(false)
const api = useApiClient()
const { allowsNsfw } = useContentStance()

let latest = 0

watchDebounced(
  query,
  async (raw) => {
    const current = ++latest
    const q = raw.trim()
    if (!q) {
      results.value = []
      answered.value = ''
      return
    }
    const res = await settle(
      api.GET('/traits', {
        params: { query: { q, limit: 30, include_nsfw: allowsNsfw.value } }
      })
    )
    if (current !== latest) {
      return
    }
    failed.value = !res.ok
    results.value = res.ok ? res.data.items : []
    answered.value = q
  },
  { debounce: 300 }
)

const loading = computed(
  () => !!query.value.trim() && answered.value !== query.value.trim()
)
</script>

<template>
  <div class="space-y-3">
    <KunInput
      v-model="query"
      type="search"
      placeholder="搜索属性, 例如 黑长直、傲娇、眼镜、女仆装"
      aria-label="搜索角色属性"
    />

    <KunLoading v-if="loading" />

    <div
      v-else-if="results.length"
      class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3"
    >
      <KunCard
        v-for="trait in results"
        :key="trait.id"
        :href="`/galgame/trait/${trait.id}`"
        :is-hoverable="true"
        :is-transparent="false"
        padding="none"
      >
        <div class="flex items-center gap-3 px-3 py-2">
          <div class="min-w-0 flex-1">
            <p
              :class="
                cn('truncate font-medium', trait.is_sexual && 'text-danger')
              "
            >
              {{ catalogVocabularyName(trait) }}
            </p>
            <p
              v-if="traitContext(trait)"
              class="text-default-500 truncate text-xs"
            >
              {{ traitContext(trait) }}
            </p>
          </div>
          <span class="text-default-400 shrink-0 text-xs tabular-nums">
            {{ trait.character_count.toLocaleString('en-US') }} 名角色
          </span>
        </div>
      </KunCard>
    </div>

    <p v-else-if="query.trim()" class="text-default-500 text-sm">
      {{
        failed
          ? '属性搜索没能完成, 请稍后重试'
          : allowsNsfw
            ? '没有找到匹配的属性'
            : '没有找到匹配的属性。成人向属性在 SFW 模式下搜不到, 请在设置面板开启 NSFW 开关'
      }}
    </p>
  </div>
</template>
