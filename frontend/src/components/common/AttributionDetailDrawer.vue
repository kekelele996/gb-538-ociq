<script setup lang="ts">
import { Binary, CircleCheck, FileJson, TriangleAlert } from '@lucide/vue'
import { computed } from 'vue'
import type { AttributionRun } from '../../types/attribution-run'
import type { Spectrum } from '../../types/common'
import { fixed, shortHash } from '../../utils/format'
import OctaveBandChart from './OctaveBandChart.vue'
import StateBadge from './StateBadge.vue'

const props = withDefaults(defineProps<{
  modelValue: boolean
  run?: AttributionRun | null
  title?: string
  subtitle?: string
  snapshot?: unknown
}>(), { run: null, title: '证据详情', subtitle: '', snapshot: undefined })
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const contributionSeries = computed(() => props.run?.contributions.map((source, index) => ({
  name: source.source_code,
  color: ['#2c6b4d', '#b0781c', '#3f6b8d', '#8b4d5e'][index % 4],
  values: Object.fromEntries(source.bands.map((band) => [String(band.band_hz), band.predicted_db])) as Spectrum,
})) ?? [])
const renderedSnapshot = computed(() => JSON.stringify(props.snapshot ?? props.run?.input_snapshot ?? {}, null, 2))
</script>

<template>
  <el-drawer :model-value="modelValue" size="min(780px, 94vw)" :with-header="false" @update:model-value="emit('update:modelValue', $event)">
    <div class="drawer-heading">
      <div><span class="drawer-icon"><Binary :size="18" /></span><span><small>{{ subtitle || 'IMMUTABLE EVIDENCE' }}</small><strong>{{ run?.run_code || title }}</strong></span></div>
      <StateBadge v-if="run" :state="run.attribution_state" />
    </div>

    <template v-if="run">
      <div class="explanation-lead"><p>{{ run.explanation }}</p></div>
      <dl class="evidence-grid">
        <div><dt>算法版本</dt><dd>{{ run.algorithm_version }}</dd></div>
        <div><dt>输入哈希</dt><dd class="hash-text" :title="run.input_hash">{{ shortHash(run.input_hash) }}</dd></div>
        <div><dt>相对残差</dt><dd>{{ fixed(run.residual_error, 6) }}</dd></div>
        <div><dt>矩阵规模</dt><dd>{{ run.evidence.matrix_rows }} × {{ run.evidence.matrix_columns }}</dd></div>
        <div><dt>迭代 / 收敛</dt><dd><CircleCheck v-if="run.evidence.converged" :size="13" />{{ run.evidence.iterations }} / {{ run.evidence.converged ? '是' : '否' }}</dd></div>
        <div><dt>列相关性提示</dt><dd>{{ fixed(run.evidence.condition_hint, 4) }}</dd></div>
      </dl>
      <section class="drawer-section"><h3>逐频带预测贡献</h3><OctaveBandChart :series="contributionSeries" /></section>
      <section class="drawer-section">
        <h3>贡献排序</h3>
        <div v-for="(source, index) in run.contributions" :key="source.source_profile_id" class="contribution-row">
          <span>{{ String(index + 1).padStart(2, '0') }}</span><strong>{{ source.source_name }}</strong>
          <small>{{ source.source_code }}</small><b>{{ fixed(source.contribution_pct, 2) }}%</b>
        </div>
      </section>
      <div v-if="run.evidence.warnings.length" class="warning-list">
        <strong><TriangleAlert :size="14" /> 可解释性提示</strong>
        <span v-for="warning in run.evidence.warnings" :key="warning">{{ warning }}</span>
      </div>
    </template>

    <section class="drawer-section">
      <h3><FileJson :size="14" /> 冻结快照</h3>
      <pre class="snapshot-code">{{ renderedSnapshot }}</pre>
    </section>
  </el-drawer>
</template>
