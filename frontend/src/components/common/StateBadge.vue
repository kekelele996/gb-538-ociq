<script setup lang="ts">
import { computed } from 'vue'
import { attributionStateLabels } from '../../types/enums/attribution-state'
import { measurementStateLabels } from '../../types/enums/measurement-quality'

const props = defineProps<{ state: string }>()
const labels: Record<string, string> = {
  ...attributionStateLabels, ...measurementStateLabels,
  active: '已启用', inactive: '已停用', draft: '草稿', retired: '已废止',
}
const label = computed(() => labels[props.state] ?? props.state)
</script>

<template><span class="state-badge" :class="state"><i aria-hidden="true" />{{ label }}</span></template>
