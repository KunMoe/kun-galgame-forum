<script setup lang="ts">
import type { ReleaseCalendarMonth } from '#shared/utils/api/schemas'

const props = defineProps<{
  data: ReleaseCalendarMonth
}>()

const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
const cards = useWorkCards(() => props.data.items)

type DayStatus = 'past' | 'today' | 'future'

interface DayRow {
  id: string
  day: number
  weekday: string
  status: DayStatus
  isBucket: boolean
  games: GalgameCard[]
}

const rows = computed<DayRow[]>(() => {
  const [y, mo] = props.data.calendar_month.split('-').map(Number)
  const monthCmp = props.data.calendar_month.localeCompare(
    props.data.today_date.slice(0, 7)
  )
  const todayDay =
    monthCmp === 0 ? Number(props.data.today_date.slice(8, 10)) : -1

  const dayMap = new Map<number, GalgameCard[]>()
  const bucket: GalgameCard[] = []
  for (const game of cards.value) {
    if (game.release_precision === 'month' || !game.release_date) {
      bucket.push(game)
      continue
    }
    const d = Number(game.release_date.slice(8, 10))
    if (!dayMap.has(d)) {
      dayMap.set(d, [])
    }
    dayMap.get(d)!.push(game)
  }

  const statusOf = (d: number): DayStatus =>
    monthCmp < 0
      ? 'past'
      : monthCmp > 0
        ? 'future'
        : d === todayDay
          ? 'today'
          : d < todayDay
            ? 'past'
            : 'future'
  const mkRow = (d: number): DayRow => ({
    id: `${props.data.calendar_month}-${String(d).padStart(2, '0')}`,
    day: d,
    weekday: WEEKDAYS[new Date(y ?? 0, (mo ?? 1) - 1, d).getDay()] ?? '',
    status: statusOf(d),
    isBucket: false,
    games: dayMap.get(d)!
  })

  const days = [...dayMap.keys()].sort((a, b) => a - b)
  const out: DayRow[] = []
  if (todayDay > 0 && dayMap.has(todayDay)) {
    out.push(mkRow(todayDay))
  }
  for (const d of days) {
    if (d !== todayDay) {
      out.push(mkRow(d))
    }
  }
  if (bucket.length) {
    out.push({
      id: `${props.data.calendar_month}-bucket`,
      day: 0,
      weekday: '',
      status: 'future',
      isBucket: true,
      games: bucket
    })
  }
  return out
})

const tileClass = (r: DayRow) => {
  if (!r.isBucket && r.status === 'today') {
    return 'bg-primary text-white'
  }
  if (!r.isBucket && r.status === 'future') {
    return 'border-primary text-primary border-2'
  }
  return 'bg-default-100 text-default-500'
}
</script>

<template>
  <div class="flex flex-col">
    <p v-if="!rows.length" class="text-default-400 py-6 text-center text-sm">
      本月暂无发售的 Galgame
    </p>

    <div
      v-for="(r, i) in rows"
      :id="`cal-day-${r.id}`"
      :key="r.id"
      class="flex scroll-mt-20 flex-col gap-3 py-4 sm:flex-row sm:gap-5"
      :class="{ 'border-default-200 border-t': i > 0 }"
    >
      <div
        class="flex shrink-0 items-center gap-3 sm:w-16 sm:flex-col sm:gap-1.5"
      >
        <div
          class="flex size-14 flex-col items-center justify-center rounded-xl"
          :class="tileClass(r)"
        >
          <KunIcon
            v-if="r.isBucket"
            name="lucide:calendar-clock"
            class="size-6"
          />
          <span v-else class="text-3xl leading-none font-bold">{{
            r.day
          }}</span>
        </div>

        <div class="flex items-center gap-2 sm:flex-col sm:gap-1">
          <span class="text-default-500 text-sm">
            {{ r.isBucket ? '日期待定' : r.weekday }}
          </span>
          <KunChip
            v-if="!r.isBucket && r.status === 'today'"
            variant="solid"
            color="primary"
            size="sm"
          >
            今日
          </KunChip>
        </div>
      </div>

      <div class="min-w-0 flex-1">
        <GalgameCard :galgames="r.games" />
      </div>
    </div>
  </div>
</template>
