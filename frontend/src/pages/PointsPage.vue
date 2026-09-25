<script setup lang="ts">
import { MapPin, Plus, RadioTower, RefreshCw } from '@lucide/vue'
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
import { OCTAVE_BANDS, type Spectrum } from '../types/common'
import type { CreateMonitoringPoint, MonitoringPoint } from '../types/monitoring-point'
import { formatDate } from '../utils/format'

const store = useMonitoringPointStore()
const { canWritePoints } = useAuth()
const selected = ref<MonitoringPoint | null>(null)
const createOpen = ref(false)
const submitting = ref(false)
const form = reactive<CreateMonitoringPoint>({
  point_code: '', name: '', x_m: 0, y_m: 0, height_m: 1.5, area_type: 'boundary',
  owner_team: 'Occupational Hygiene', background_profile: defaultSpectrum(42),
})
const metrics = computed(() => ({
  total: store.items.length,
  active: store.items.filter((item) => item.point_state === 'active').length,
  measurements: store.items.reduce((sum, item) => sum + item.measurement_count, 0),
  recent: store.items.filter((item) => item.latest_measurement_time).length,
}))

onMounted(async () => { await store.load(); selected.value = store.items[0] ?? null })

function defaultSpectrum(base: number): Spectrum {
  return Object.fromEntries(OCTAVE_BANDS.map((band, index) => [String(band), base - index * 1.4]))
}

async function createPoint() {
  submitting.value = true
  try {
    const created = await store.create({ ...form, background_profile: { ...form.background_profile } })
    selected.value = created; createOpen.value = false; ElMessage.success(`已创建 ${created.point_code}`)
  } catch (error) { ElMessage.error(errorMessage(error)) } finally { submitting.value = false }
}

async function deactivate(item: MonitoringPoint) {
  try { selected.value = await store.deactivate(item); ElMessage.success('监测点已停用') } catch (error) { ElMessage.error(errorMessage(error)) }
}
</script>

<template>
  <AppShell><div class="page-wrap">
    <PageHeader eyebrow="MONITORING GEOMETRY" title="监测点" description="维护厂区坐标、受声区类型和版本化背景谱，所有归因只读取冻结快照。">
      <el-button :icon="RefreshCw" aria-label="刷新监测点" @click="store.load" />
      <el-button v-if="canWritePoints" type="primary" :icon="Plus" @click="createOpen = true">新建监测点</el-button>
    </PageHeader>
    <div class="metric-band">
      <div><span>监测点</span><strong>{{ metrics.total }}</strong></div><div><span>启用中</span><strong>{{ metrics.active }}</strong></div>
      <div><span>关联测量</span><strong>{{ metrics.measurements }}</strong></div><div><span>有近期数据</span><strong>{{ metrics.recent }}</strong></div>
    </div>
    <el-alert v-if="store.error" :title="store.error" type="error" :closable="false" show-icon />
    <el-skeleton v-if="store.loading" :rows="6" animated />
    <div v-else-if="!store.items.length" class="empty-state"><MapPin :size="30" /><h2>尚无监测点</h2><p>创建第一个坐标和完整背景谱后即可导入测量。</p></div>
    <div v-else class="split-workspace">
      <section class="entity-list" aria-label="监测点列表">
        <button v-for="item in store.items" :key="item.id" class="entity-row" :class="{ selected: selected?.id === item.id }" @click="selected = item">
          <RadioTower :size="17" /><span><strong>{{ item.point_code }}</strong><small>{{ item.name }}</small></span><StateBadge :state="item.point_state" />
        </button>
      </section>
      <section v-if="selected" class="entity-detail">
        <div class="detail-heading"><div><p class="eyebrow">{{ selected.area_type }}</p><h2>{{ selected.name }}</h2></div><StateBadge :state="selected.point_state" /></div>
        <dl class="evidence-grid point-grid">
          <div><dt>坐标</dt><dd>{{ selected.x_m }}, {{ selected.y_m }} m</dd></div><div><dt>高度</dt><dd>{{ selected.height_m }} m</dd></div>
          <div><dt>责任团队</dt><dd>{{ selected.owner_team }}</dd></div><div><dt>最近测量</dt><dd>{{ formatDate(selected.latest_measurement_time) }}</dd></div>
          <div><dt>测量质量</dt><dd><QualityBadge :quality="selected.latest_quality" /></dd></div><div><dt>版本</dt><dd>V{{ selected.version }}</dd></div>
        </dl>
        <div class="section-heading"><h2>背景倍频程</h2><p>固定 63-8000 Hz · dB</p></div>
        <OctaveBandChart :series="[{ name: '背景声级', values: selected.background_profile, color: '#60776a' }]" />
        <div class="detail-actions" v-if="canWritePoints && selected.point_state === 'active'"><el-button type="danger" plain @click="deactivate(selected)">停用监测点</el-button></div>
      </section>
    </div>
  </div></AppShell>

  <el-dialog v-model="createOpen" title="新建监测点" width="min(720px, 94vw)" destroy-on-close>
    <el-form label-position="top"><div class="form-grid two">
      <el-form-item label="点位编号"><el-input v-model="form.point_code" placeholder="MP-WEST-04" /></el-form-item>
      <el-form-item label="名称"><el-input v-model="form.name" placeholder="West boundary receptor" /></el-form-item>
      <el-form-item label="X 坐标（m）"><el-input-number v-model="form.x_m" :controls="false" /></el-form-item>
      <el-form-item label="Y 坐标（m）"><el-input-number v-model="form.y_m" :controls="false" /></el-form-item>
      <el-form-item label="高度（m）"><el-input-number v-model="form.height_m" :min="0" :max="100" :step="0.1" /></el-form-item>
      <el-form-item label="受声区"><el-select v-model="form.area_type"><el-option label="厂界" value="boundary" /><el-option label="车间" value="workshop" /><el-option label="办公区" value="office" /><el-option label="居住区" value="residential" /></el-select></el-form-item>
      <el-form-item label="责任团队" class="wide"><el-input v-model="form.owner_team" /></el-form-item>
    </div><div class="band-inputs"><el-form-item v-for="band in OCTAVE_BANDS" :key="band" :label="`${band} Hz`"><el-input-number v-model="form.background_profile[String(band)]" :controls="false" :min="0" :max="180" /></el-form-item></div></el-form>
    <template #footer><el-button @click="createOpen = false">取消</el-button><el-button type="primary" :loading="submitting" @click="createPoint">创建</el-button></template>
  </el-dialog>
</template>
