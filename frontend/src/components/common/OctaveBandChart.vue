<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { OCTAVE_BANDS, type Spectrum } from '../../types/common'

interface ChartSeries { name: string; values: Spectrum; color?: string }
const props = withDefaults(defineProps<{ series: ChartSeries[]; unit?: string; height?: number }>(), { unit: 'dB', height: 280 })
const container = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null
let observer: ResizeObserver | null = null
const hasData = computed(() => props.series.some((item) => OCTAVE_BANDS.some((band) => Number.isFinite(item.values[String(band)]))))

function render() {
  if (!container.value || !hasData.value) return
  chart ??= echarts.init(container.value, undefined, { renderer: 'canvas' })
  chart.setOption({
    animation: !window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    color: props.series.map((item) => item.color).filter(Boolean),
    tooltip: { trigger: 'axis', valueFormatter: (value) => `${Number(value).toFixed(2)} ${props.unit}` },
    legend: { top: 4, left: 8, textStyle: { color: '#44534b', fontSize: 11 } },
    grid: { left: 48, right: 20, top: props.series.length > 1 ? 42 : 24, bottom: 40 },
    xAxis: { type: 'category', data: OCTAVE_BANDS.map(String), name: 'Hz', nameLocation: 'end', axisLine: { lineStyle: { color: '#8d9c93' } }, axisLabel: { color: '#4d5c54' } },
    yAxis: { type: 'value', name: props.unit, scale: true, splitLine: { lineStyle: { color: '#e0e6e2' } }, axisLabel: { color: '#4d5c54' } },
    series: props.series.map((item) => ({
      name: item.name, type: 'bar', barMaxWidth: 30,
      data: OCTAVE_BANDS.map((band) => item.values[String(band)] ?? null),
      emphasis: { focus: 'series' }, itemStyle: { borderRadius: [2, 2, 0, 0] },
    })),
  }, true)
}

watch(() => props.series, () => nextTick(render), { deep: true })
onMounted(() => {
  render()
  observer = new ResizeObserver(() => chart?.resize())
  if (container.value) observer.observe(container.value)
})
onBeforeUnmount(() => { observer?.disconnect(); chart?.dispose(); chart = null })
</script>

<template>
  <div class="chart-frame" :style="{ height: `${height}px` }">
    <div v-if="hasData" ref="container" class="chart-canvas" role="img" aria-label="倍频程频谱柱状图" />
    <div v-else class="chart-empty">暂无可绘制的倍频程数据</div>
  </div>
</template>
