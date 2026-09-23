<script setup lang="ts">
import { galgameImageSourceLabel } from '~/constants/galgameImageSource'

const props = defineProps<{ workId: number; covers: GalgameCover[] }>()
const open = defineModel<boolean>({ required: true })

const KIND_LABEL: Record<string, string> = {
  main: '主封面',
  pkgfront: '盒装正面',
  dig: '数字版',
  pkgback: '封底',
  pkgcontent: '内页',
  pkgside: '书脊',
  pkgmed: '碟面',
  '': '其它'
}
const KIND_ORDER = Object.keys(KIND_LABEL)

interface CoverBallot {
  count: number
  voted: boolean
}
const ballots = ref(new Map<number, CoverBallot>())

const seedBallots = () => {
  const seeded = new Map<number, CoverBallot>()
  for (const c of props.covers) {
    if (c.id) {
      seeded.set(c.id, { count: c.vote_count ?? 0, voted: !!c.voted })
    }
  }
  ballots.value = seeded
}
seedBallots()
watch(() => props.covers, seedBallots)

const ballotOf = (cover: GalgameCover): CoverBallot | undefined =>
  cover.id ? ballots.value.get(cover.id) : undefined

const voting = ref(0)

const applyVote = (coverId: number, count: number, voted: boolean) => {
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

const toggleVote = async (cover: GalgameCover) => {
  if (!cover.id || voting.value) return
  if (!requireLogin()) return
  const willUnvote = !!ballotOf(cover)?.voted
  voting.value = cover.id
  const result = await kunFetch<{
    cover_id: number
    vote_count: number
    voted: boolean
  }>(`/galgame/${props.workId}/cover/${cover.id}/vote`, {
    method: willUnvote ? 'DELETE' : 'PUT'
  })
  voting.value = 0
  if (!result) {
    return
  }
  applyVote(result.cover_id, result.vote_count, result.voted)
}

const sorted = computed(() =>
  [...props.covers]
    .filter((c) => !!c.image_hash)
    .sort((a, b) => a.sort_order - b.sort_order)
)

const groups = computed(() => {
  const byKind = new Map<string, GalgameCover[]>()
  for (const c of sorted.value) {
    const k = KIND_LABEL[c.kind ?? ''] !== undefined ? (c.kind ?? '') : ''
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
  () => new Set(sorted.value.map((c) => c.source ?? '')).size > 1
)
const sourceLabel = (cover: GalgameCover) =>
  galgameImageSourceLabel(cover.source)
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
                :key="c.image_hash"
                class="space-y-1.5"
              >
                <KunLightboxGalleryItem
                  :src="galgameImageSrc(c)"
                  :alt="g.label"
                  as="figure"
                  class="border-default/20 bg-default-100 block overflow-hidden rounded-lg border"
                >
                  <KunImage
                    :src="galgameImageSrc(c)"
                    :alt="g.label"
                    loading="lazy"
                    :aspect-ratio="imageAspectRatio(c.width, c.height)"
                    :thumbhash="c.thumbhash"
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
                  v-if="c.id"
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
