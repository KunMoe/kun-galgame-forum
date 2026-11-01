<script setup lang="ts">
import type { ReleaseCalendarMonth } from '#shared/utils/api/schemas'

const props = defineProps<{
  data: ReleaseCalendarMonth
}>()

const emit = defineEmits<{
  month: [month: string]
  date: [date: string]
}>()

const minDate = computed(() =>
  props.data.min_month ? `${props.data.min_month}-01` : undefined
)
const maxDate = computed(() => {
  if (!props.data.max_month) {
    return undefined
  }
  const [y, m] = props.data.max_month.split('-').map(Number)
  return new Date(y ?? 0, m ?? 1, 0)
})

const monthValue = computed({
  get: () => props.data.calendar_month,
  set: (v) => {
    if (typeof v === 'string' && v) {
      emit('month', v)
    }
  }
})

const dateValue = ref<string | null>(null)
watch(dateValue, (v) => {
  if (v) {
    emit('date', v)
  }
})
watch(
  () => props.data.calendar_month,
  (m) => {
    if (dateValue.value?.slice(0, 7) !== m) {
      dateValue.value = null
    }
  }
)
</script>

<template>
  <div class="grid grid-cols-2 gap-2">
    <KunDatePicker
      v-model="monthValue"
      precision="month"
      format="yyyy 年 M 月"
      size="sm"
      :min-date="minDate"
      :max-date="maxDate"
    />
    <KunDatePicker
      v-model="dateValue"
      precision="day"
      placeholder="跳转到日期"
      size="sm"
      :min-date="minDate"
      :max-date="maxDate"
    />
  </div>
</template>
