<script setup lang="ts">
import { AudioLines, FileUp, RefreshCw } from '@lucide/vue'
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '../components/common/AppShell.vue'
import OctaveBandChart from '../components/common/OctaveBandChart.vue'
import PageHeader from '../components/common/PageHeader.vue'
import QualityBadge from '../components/common/QualityBadge.vue'
import StateBadge from '../components/common/StateBadge.vue'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { useMonitoringPointStore } from '../stores/monitoring-point-store'
import { useNoiseMeasurementStore } from '../stores/noise-measurement-store'
import { OCTAVE_BANDS, type Spectrum } from '../types/common'
import { measurementStateLabels, measurementTransitions, type MeasurementState } from '../types/enums/measurement-quality'
import type { CreateNoiseMeasurement, NoiseMeasurement } from '../types/noise-measurement'
import { fixed, formatDate, shortHash } from '../utils/format'

const store = useNoiseMeasurementStore()
const points = useMonitoringPointStore()
const { canImport, canFlowMeasurement } = useAuth()
const selected = ref<NoiseMeasurement | null>(null)
const importOpen = ref(false)
const submitting = ref(false)
const form = reactive<CreateNoiseMeasurement>({
  monitoring_point_id: 0, measured_at: new Date().toISOString(), duration_s: 900,
  octave_bands: fixtureSpectrum(), overall_dba: 72.4, background_dba: 43.1,
  weather_note: 'Dry, wind below 2 m/s', quality_reason: '',
})
const nextStates = computed(() => selected.value ? measurementTransitions[selected.value.measurement_state] ?? [] : [])

onMounted(async () => {
  await Promise.all([store.load(), points.load()]); selected.value = store.items[0] ?? null; form.monitoring_point_id = points.items[0]?.id ?? 0
})

function fixtureSpectrum(): Spectrum {
  return { '63': 66.4, '125': 70.1, '250': 72.3, '500': 69.7, '1000': 65.2, '2000': 61.1, '4000': 56.8, '8000': 51.2 }
}

async function importMeasurement() {
  submitting.value = true
  try { const created = await store.create({ ...form, measured_at: new Date(form.measured_at).toISOString(), octave_bands: { ...form.octave_bands } }); selected.value = created; importOpen.value = false; ElMessage.success('测量频谱已导入') }
  catch (error) { ElMessage.error(errorMessage(error)) } finally { submitting.value = false }
}

async function transition(toState: MeasurementState) {
  if (!selected.value) return
  try { selected.value = await store.transition(selected.value, toState); ElMessage.success(`状态已更新为${measurementStateLabels[toState]}`) }
  catch (error) { ElMessage.error(errorMessage(error)) }
}
</script>

<template>
  <AppShell><div class="page-wrap">
    <PageHeader eyebrow="MEASUREMENT QUALITY" title="测量工作台" description="导入完整倍频程，核对 checksum、背景裕量和质量状态，再逐步推进到可归因。">
      <el-button :icon="RefreshCw" aria-label="刷新测量" @click="store.load" /><el-button v-if="canImport" type="primary" :icon="FileUp" @click="importOpen = true">导入频谱</el-button>
    </PageHeader>
    <div class="metric-band">
      <div><span>测量总数</span><strong>{{ store.items.length }}</strong></div><div><span>可归因</span><strong>{{ store.items.filter(x => x.measurement_state === 'ready').length }}</strong></div>
      <div><span>质量有效</span><strong>{{ store.items.filter(x => x.measurement_quality === 'valid').length }}</strong></div><div><span>固定频带</span><strong>8</strong></div>
    </div>
    <el-alert v-if="store.error" :title="store.error" type="error" :closable="false" show-icon />
    <el-skeleton v-if="store.loading" :rows="6" animated />
    <div v-else-if="!store.items.length" class="empty-state"><AudioLines :size="30" /><h2>尚无测量</h2><p>导入 8 个固定倍频程后，系统将自动计算 checksum 和质量判断。</p></div>
    <div v-else class="split-workspace">
      <section class="entity-list">
        <button v-for="item in store.items" :key="item.id" class="entity-row measurement-row" :class="{ selected: selected?.id === item.id }" @click="selected = item">
          <span class="run-index">M{{ String(item.id).padStart(3, '0') }}</span><span><strong>{{ item.point_code }}</strong><small>{{ formatDate(item.measured_at) }}</small></span><QualityBadge :quality="item.measurement_quality" /><StateBadge :state="item.measurement_state" />
        </button>
      </section>
      <section v-if="selected" class="entity-detail">
        <div class="detail-heading"><div><p class="eyebrow">{{ selected.point_code }}</p><h2>{{ selected.point_name }}</h2></div><QualityBadge :quality="selected.measurement_quality" /></div>
        <div class="quality-band"><div><span>总声级</span><strong>{{ fixed(selected.overall_dba) }} dBA</strong></div><div><span>背景声级</span><strong>{{ fixed(selected.background_dba) }} dBA</strong></div><div><span>时长</span><strong>{{ selected.duration_s }} s</strong></div><div><span>状态</span><StateBadge :state="selected.measurement_state" /></div></div>
        <OctaveBandChart :series="[{ name: '原始测量', values: selected.octave_bands, color: '#3f6b8d' }, ...(selected.normalized_bands ? [{ name: '扣背景后', values: selected.normalized_bands, color: '#2c6b4d' }] : [])]" />
        <dl class="evidence-grid"><div><dt>来源校验值</dt><dd class="hash-text" :title="selected.source_checksum">{{ shortHash(selected.source_checksum) }}</dd></div><div><dt>质量理由</dt><dd>{{ selected.quality_reason }}</dd></div><div><dt>天气记录</dt><dd>{{ selected.weather_note || '—' }}</dd></div><div><dt>版本</dt><dd>V{{ selected.version }}</dd></div></dl>
        <div v-if="canFlowMeasurement && nextStates.length" class="detail-actions"><span>状态推进</span><el-button v-for="state in nextStates" :key="state" :type="state === 'rejected' || state === 'superseded' ? 'danger' : 'primary'" plain @click="transition(state)">{{ measurementStateLabels[state] }}</el-button></div>
      </section>
    </div>
  </div></AppShell>

  <el-dialog v-model="importOpen" title="导入倍频程测量" width="min(760px, 94vw)" destroy-on-close>
    <el-form label-position="top"><div class="form-grid two">
      <el-form-item label="监测点"><el-select v-model="form.monitoring_point_id"><el-option v-for="point in points.items.filter(x => x.point_state === 'active')" :key="point.id" :label="`${point.point_code} · ${point.name}`" :value="point.id" /></el-select></el-form-item>
      <el-form-item label="测量时间"><el-date-picker v-model="form.measured_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" /></el-form-item>
      <el-form-item label="时长（s）"><el-input-number v-model="form.duration_s" :min="10" :max="86400" /></el-form-item>
      <el-form-item label="总声级（dBA）"><el-input-number v-model="form.overall_dba" :min="0" :max="180" :step="0.1" /></el-form-item>
      <el-form-item label="背景声级（dBA）"><el-input-number v-model="form.background_dba" :min="0" :max="180" :step="0.1" /></el-form-item>
      <el-form-item label="天气记录"><el-input v-model="form.weather_note" /></el-form-item>
    </div><div class="band-inputs"><el-form-item v-for="band in OCTAVE_BANDS" :key="band" :label="`${band} Hz`"><el-input-number v-model="form.octave_bands[String(band)]" :controls="false" :min="0" :max="180" /></el-form-item></div></el-form>
    <template #footer><el-button @click="importOpen = false">取消</el-button><el-button type="primary" :loading="submitting" @click="importMeasurement">导入并校验</el-button></template>
  </el-dialog>
</template>
