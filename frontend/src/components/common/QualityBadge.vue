<script setup lang="ts">
import { Ban, CircleCheck, CircleHelp, TriangleAlert } from '@lucide/vue'
import { computed } from 'vue'
import type { MeasurementQuality } from '../../types/enums/measurement-quality'
import { measurementQualityLabels } from '../../types/enums/measurement-quality'

const props = defineProps<{ quality?: MeasurementQuality | string }>()
const normalized = computed<MeasurementQuality>(() => ['valid', 'contaminated', 'clipped', 'missing'].includes(props.quality ?? '') ? props.quality as MeasurementQuality : 'missing')
const icon = computed(() => ({ valid: CircleCheck, contaminated: TriangleAlert, clipped: Ban, missing: CircleHelp })[normalized.value])
</script>

<template>
  <span class="quality-badge" :class="normalized">
    <component :is="icon" :size="13" aria-hidden="true" />{{ measurementQualityLabels[normalized] }}
  </span>
</template>
