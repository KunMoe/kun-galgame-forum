<script setup lang="ts">
import { galgameImageSourceLabel } from '~/constants/galgameImageSource'
import { settle } from '#shared/utils/api/problem'
import type { WorkCover } from '#shared/utils/api/schemas'

const props = defineProps<{ workId: number; covers: WorkCover[] }>()
const open = defineModel<boolean>({ required: true })

const KIND_LABEL: Record<string, string> = {
  main: '主封面',
  pkgfront: '盒装正面',
  dig: '数字版',
  pkgback: '封底',
  pkgcontent: '内页',
  pkgside: '书脊',
  pkgmed: '碟面',
  other: '其它'
}
const KIND_ORDER = Object.keys(KIND_LABEL)

interface CoverBallot {
  count: number
  voted: boolean
}
const ballots = ref(new Map<string, CoverBallot>())

const seedBallots = () => {
  const seeded = new Map<string, CoverBallot>()
  for (const c of props.covers) {
    seeded.set(c.id, {
      count: c.vote_count,
      voted: c.viewer?.has_voted ?? false
    })
  }
  ballots.value = seeded
}
seedBallots()
watch(() => props.covers, seedBallots)

const ballotOf = (cover: WorkCover): CoverBallot | undefined =>
  ballots.value.get(cover.id)

const api = useApiClient()
const voting = ref('')

const applyVote = (coverId: string, count: number, voted: boolean) => {
  const next = new Map(ballots.value)
  for (const [id, ballot] of next) {
    if (id === coverId) continue
    if (voted && ballot.voted) {
      next.set(id, { count: Math.max(0, ballot.count - 1), voted: false })
    }
  }
  next.set(coverId, { count, voted })
  ballots.value = next
}

const toggleVote = async (cover: WorkCover) => {
  if (voting.value) return
  if (!requireLogin()) return
  const previous = ballots.value
  const willUnvote = !!ballotOf(cover)?.voted
  voting.value = cover.id
  const params = {
    params: {
      path: {
        work_id: String(props.workId),
        cover_id: cover.id
      }
    }
  }
  const result = await settle(
    willUnvote
      ? api.DELETE('/works/{work_id}/covers/{cover_id}/vote', params)
      : api.PUT('/works/{work_id}/covers/{cover_id}/vote', params)
  )
  voting.value = ''
  if (!result.ok) {
    ballots.value = previous
    reportProblem(result.problem)
    return
  }
  applyVote(
    result.data.cover_id,
    result.data.vote_count,
    result.data.viewer?.has_voted ?? !willUnvote
  )
}

const coverSrc = (cover: WorkCover) => cover.image?.url ?? ''

const sorted = computed(() =>
  [...props.covers]
    .filter((c) => !!c.image)
    .sort((a, b) => a.sort_order - b.sort_order)
)

const groups = computed(() => {
  const byKind = new Map<string, WorkCover[]>()
  for (const c of sorted.value) {
    const k = KIND_LABEL[c.cover_slot] ? c.cover_slot : 'other'
    if (!byKind.has(k)) byKind.set(k, [])
    byKind.get(k)!.push(c)
  }
  return KIND_ORDER.filter((k) => byKind.has(k)).map((k) => ({
    kind: k,
    label: KIND_LABEL[k],
    covers: byKind.get(k)!
  }))
})

const showSource = computed(
  () => new Set(sorted.value.map((c) => c.site)).size > 1
)
const sourceLabel = (cover: WorkCover) => galgameImageSourceLabel(cover.site)
</script>

<template>
  <KunModal v-model="open" inner-class-name="max-w-3xl w-full">
    <div class="space-y-4">
      <div class="flex items-center gap-3">
        <div class="bg-primary h-6 w-1 rounded" />
        <h2 class="text-xl font-bold">所有封面</h2>
      </div>

      <KunNull v-if="!sorted.length" description="该 Galgame 暂无封面" />

      <KunLightboxGallery v-else>
        <div class="space-y-5">
          <section v-for="g in groups" :key="g.kind" class="space-y-2">
            <h3 class="text-default-600 text-sm font-medium">
              {{ g.label }}
              <span class="text-default-400">({{ g.covers.length }})</span>
            </h3>
            <div class="grid grid-cols-1 items-start gap-3 sm:grid-cols-2">
              <div
                v-for="c in g.covers"
                :key="c.id"
                class="space-y-1.5"
              >
                <KunLightboxGalleryItem
                  :src="coverSrc(c)"
                  :alt="g.label"
                  as="figure"
                  class="border-default/20 bg-default-100 block overflow-hidden rounded-lg border"
                >
                  <KunImage
                    :src="coverSrc(c)"
                    :alt="g.label"
                    loading="lazy"
                    :aspect-ratio="
                      imageAspectRatio(c.image?.width ?? undefined, c.image?.height ?? undefined)
                    "
                    :thumbhash="c.image?.thumbhash ?? undefined"
                    class-name="bg-default-100"
                  />
                </KunLightboxGalleryItem>

                <KunChip
                  v-if="showSource"
                  size="sm"
                  color="default"
                  variant="flat"
                >
                  {{ sourceLabel(c) }}
                </KunChip>

                <KunButton
                  variant="flat"
                  size="sm"
                  :color="ballotOf(c)?.voted ? 'primary' : 'default'"
                  :loading="voting === c.id"
                  full-width
                  @click="toggleVote(c)"
                >
                  <KunIcon
                    :name="ballotOf(c)?.voted ? 'lucide:heart' : 'lucide:plus'"
                    class="size-4"
                  />
                  {{ ballotOf(c)?.voted ? '已选为最佳封面' : '选为最佳封面' }}
                  <span class="text-default-500">
                    {{ ballotOf(c)?.count ?? 0 }}
                  </span>
                </KunButton>
              </div>
            </div>
          </section>
        </div>
      </KunLightboxGallery>
    </div>
  </KunModal>
</template>
