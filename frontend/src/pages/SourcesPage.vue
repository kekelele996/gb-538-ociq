<script setup lang="ts">
import { Factory, Plus, RefreshCw } from '@lucide/vue'
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '../components/common/AppShell.vue'
import AttributionDetailDrawer from '../components/common/AttributionDetailDrawer.vue'
import OctaveBandChart from '../components/common/OctaveBandChart.vue'
import PageHeader from '../components/common/PageHeader.vue'
import StateBadge from '../components/common/StateBadge.vue'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { useMonitoringPointStore } from '../stores/monitoring-point-store'
import { useSourceProfileStore } from '../stores/source-profile-store'
import { OCTAVE_BANDS, type Spectrum } from '../types/common'
import type { CreateSourceProfile, SourceProfile } from '../types/source-profile'
import { fixed } from '../utils/format'

const store = useSourceProfileStore()
const points = useMonitoringPointStore()
const { canWriteSources } = useAuth()
const selected = ref<SourceProfile | null>(null)
const createOpen = ref(false)
const evidenceOpen = ref(false)
const submitting = ref(false)
const form = reactive<CreateSourceProfile>({
  source_code: '', name: '', x_m: 0, y_m: 0, height_m: 1.2, reference_distance_m: 1,
  octave_power: defaultPower(), directivity: defaultDirectivity(), operating_factor: .8,
})
const active = computed(() => store.items.filter((item) => item.profile_state === 'active').length)

onMounted(async () => { await Promise.all([store.load(), points.load()]); selected.value = store.items[0] ?? null })
function defaultPower(): Spectrum { return Object.fromEntries(OCTAVE_BANDS.map((band, index) => [String(band), 96 - index * 1.7])) }
function defaultDirectivity(): Spectrum { return Object.fromEntries(OCTAVE_BANDS.map((band) => [String(band), 0])) }

async function createProfile() {
  submitting.value = true
  try { const created = await store.create({ ...form, octave_power: { ...form.octave_power }, directivity: { ...form.directivity } }); selected.value = created; createOpen.value = false; ElMessage.success(`已创建 ${created.source_code} V${created.version}`) }
  catch (error) { ElMessage.error(errorMessage(error)) } finally { submitting.value = false }
}

async function transition(toState: 'active' | 'retired') {
  if (!selected.value) return
  try { selected.value = await store.transition(selected.value, toState); ElMessage.success(toState === 'active' ? '声源谱已启用' : '声源谱已废止') }
  catch (error) { ElMessage.error(errorMessage(error)) }
}
</script>

<template>
  <AppShell><div class="page-wrap">
    <PageHeader eyebrow="SOURCE LIBRARY" title="声源谱" description="维护候选设备位置、参考距离、运行系数与版本化倍频程功率谱。">
      <el-button :icon="RefreshCw" aria-label="刷新声源谱" @click="store.load" /><el-button v-if="canWriteSources" type="primary" :icon="Plus" @click="createOpen = true">新建谱版本</el-button>
    </PageHeader>
    <div class="metric-band"><div><span>谱版本</span><strong>{{ store.items.length }}</strong></div><div><span>启用中</span><strong>{{ active }}</strong></div><div><span>候选设备</span><strong>{{ new Set(store.items.map(x => x.source_code)).size }}</strong></div><div><span>固定频带</span><strong>8</strong></div></div>
    <el-alert v-if="store.error" :title="store.error" type="error" :closable="false" show-icon />
    <el-skeleton v-if="store.loading" :rows="6" animated />
    <div v-else-if="!store.items.length" class="empty-state"><Factory :size="30" /><h2>尚无声源谱</h2><p>创建候选设备的第一个频谱版本后才能运行归因。</p></div>
    <div v-else class="split-workspace">
      <section class="entity-list"><button v-for="item in store.items" :key="item.id" class="entity-row" :class="{ selected: selected?.id === item.id }" @click="selected = item"><Factory :size="17" /><span><strong>{{ item.source_code }} · V{{ item.version }}</strong><small>{{ item.name }}</small></span><StateBadge :state="item.profile_state" /></button></section>
      <section v-if="selected" class="entity-detail">
        <div class="detail-heading"><div><p class="eyebrow">VERSION {{ selected.version }}</p><h2>{{ selected.name }}</h2></div><StateBadge :state="selected.profile_state" /></div>
        <dl class="evidence-grid point-grid"><div><dt>位置</dt><dd>{{ selected.x_m }}, {{ selected.y_m }}, {{ selected.height_m }} m</dd></div><div><dt>参考距离</dt><dd>{{ selected.reference_distance_m }} m</dd></div><div><dt>运行系数</dt><dd>{{ fixed(selected.operating_factor * 100, 0) }}%</dd></div><div><dt>并发版本</dt><dd>{{ selected.lock_version }}</dd></div></dl>
        <OctaveBandChart :series="[{ name: '声功率谱', values: selected.octave_power, color: '#2c6b4d' }, { name: '方向性修正', values: selected.directivity, color: '#b0781c' }]" />
        <div class="detail-actions"><el-button @click="evidenceOpen = true">查看冻结字段</el-button><el-button v-if="canWriteSources && selected.profile_state === 'draft'" type="primary" @click="transition('active')">启用版本</el-button><el-button v-if="canWriteSources && selected.profile_state === 'active'" type="danger" plain @click="transition('retired')">废止版本</el-button></div>
      </section>
    </div>
  </div></AppShell>

  <el-dialog v-model="createOpen" title="新建声源谱版本" width="min(820px, 94vw)" destroy-on-close>
    <el-form label-position="top"><div class="form-grid two">
      <el-form-item label="声源编号"><el-input v-model="form.source_code" placeholder="SRC-BLOWER-04" /></el-form-item><el-form-item label="设备名称"><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="X 坐标（m）"><el-input-number v-model="form.x_m" :controls="false" /></el-form-item><el-form-item label="Y 坐标（m）"><el-input-number v-model="form.y_m" :controls="false" /></el-form-item>
      <el-form-item label="高度（m）"><el-input-number v-model="form.height_m" :min="0" :max="100" /></el-form-item><el-form-item label="参考距离（m）"><el-input-number v-model="form.reference_distance_m" :min="0.1" :max="1000" /></el-form-item>
      <el-form-item label="运行系数"><el-input-number v-model="form.operating_factor" :min="0.01" :max="1" :step="0.05" /></el-form-item>
    </div><h3 class="form-subheading">倍频程功率 / 方向性修正</h3><div class="band-pairs"><div v-for="band in OCTAVE_BANDS" :key="band"><strong>{{ band }} Hz</strong><el-input-number v-model="form.octave_power[String(band)]" :controls="false" :min="0" :max="180" /><el-input-number v-model="form.directivity[String(band)]" :controls="false" :min="-30" :max="20" /></div></div></el-form>
    <template #footer><el-button @click="createOpen = false">取消</el-button><el-button type="primary" :loading="submitting" @click="createProfile">创建草稿</el-button></template>
  </el-dialog>
  <AttributionDetailDrawer v-model="evidenceOpen" title="声源谱冻结字段" :subtitle="selected ? `${selected.source_code} · V${selected.version}` : ''" :snapshot="selected" />
</template>
