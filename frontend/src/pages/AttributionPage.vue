<script setup lang="ts">
import { CircleGauge, Eye, Play, RefreshCw, ShieldCheck } from '@lucide/vue'
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AppShell from '../components/common/AppShell.vue'
import AttributionDetailDrawer from '../components/common/AttributionDetailDrawer.vue'
import PageHeader from '../components/common/PageHeader.vue'
import StateBadge from '../components/common/StateBadge.vue'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { useAttributionRun } from '../hooks/useAttributionRun'
import { useNoiseMeasurementStore } from '../stores/noise-measurement-store'
import { useSourceProfileStore } from '../stores/source-profile-store'
import type { AttributionRun } from '../types/attribution-run'
import { fixed, formatDate, shortHash } from '../utils/format'

const { store, canReviewSelected, canConfirmSelected } = useAttributionRun()
const measurements = useNoiseMeasurementStore()
const sources = useSourceProfileStore()
const { canRun } = useAuth()
const runOpen = ref(false)
const detailOpen = ref(false)
const reviewOpen = ref(false)
const submitting = ref(false)
const reviewNote = ref('Independent review: frozen inputs, residual and source ranking checked.')
const form = reactive({ measurement_ids: [] as number[], source_profile_ids: [] as number[] })
const topContribution = computed(() => store.selected?.contributions[0])

onMounted(async () => {
  await Promise.all([store.load(), measurements.load(), sources.load()])
  if (store.items[0]) await selectRun(store.items[0])
  form.measurement_ids = measurements.items.filter((item) => item.measurement_state === 'ready').map((item) => item.id)
  form.source_profile_ids = sources.items.filter((item) => item.profile_state === 'active').map((item) => item.id)
})

async function selectRun(run: AttributionRun) {
  try { await store.select(run.id) } catch (error) { ElMessage.error(errorMessage(error)) }
}

async function createRun() {
  submitting.value = true
  try { await store.create({ measurement_ids: [...form.measurement_ids], source_profile_ids: [...form.source_profile_ids] }); runOpen.value = false; detailOpen.value = true; ElMessage.success('归因运行已完成并冻结证据') }
  catch (error) { ElMessage.error(errorMessage(error)) } finally { submitting.value = false }
}

async function review() {
  if (!store.selected) return
  try { await store.review(store.selected, reviewNote.value); reviewOpen.value = false; ElMessage.success('独立复核已记录') } catch (error) { ElMessage.error(errorMessage(error)) }
}

async function confirm() {
  if (!store.selected) return
  try { await store.confirm(store.selected); ElMessage.success('归因结果已由独立角色确认') } catch (error) { ElMessage.error(errorMessage(error)) }
}
</script>

<template>
  <AppShell><div class="page-wrap">
    <PageHeader eyebrow="NON-NEGATIVE ATTRIBUTION" title="贡献归因" description="冻结 ready 测量与 active 声源版本，输出贡献排序、逐频带证据、残差和不可辨识提示。">
      <el-button :icon="RefreshCw" aria-label="刷新归因运行" @click="store.load" /><el-button v-if="canRun" type="primary" :icon="Play" @click="runOpen = true">运行归因</el-button>
    </PageHeader>
    <div class="metric-band"><div><span>历史运行</span><strong>{{ store.items.length }}</strong></div><div><span>已确认</span><strong>{{ store.items.filter(x => x.attribution_state === 'confirmed').length }}</strong></div><div><span>当前残差</span><strong>{{ fixed(store.selected?.residual_error, 4) }}</strong></div><div><span>首要来源</span><strong class="source-code">{{ topContribution?.source_code ?? '—' }}</strong></div></div>
    <el-alert v-if="store.error" :title="store.error" type="error" :closable="false" show-icon />
    <el-skeleton v-if="store.loading" :rows="6" animated />
    <div v-else-if="!store.items.length" class="empty-state"><CircleGauge :size="30" /><h2>尚无归因运行</h2><p>选择 ready 测量与 active 声源后执行第一条冻结计算。</p></div>
    <div v-else class="split-workspace attribution-workspace">
      <section class="entity-list"><button v-for="item in store.items" :key="item.id" class="entity-row run-row" :class="{ selected: store.selected?.id === item.id }" @click="selectRun(item)"><span class="run-index">{{ item.id }}</span><span><strong>{{ item.run_code }}</strong><small>{{ formatDate(item.finished_at) }}</small></span><StateBadge :state="item.attribution_state" /></button></section>
      <section v-if="store.selected" class="entity-detail">
        <div class="detail-heading"><div><p class="eyebrow">{{ store.selected.algorithm_version }}</p><h2>{{ store.selected.run_code }}</h2></div><StateBadge :state="store.selected.attribution_state" /></div>
        <div class="run-summary"><div><span>输入哈希</span><strong :title="store.selected.input_hash">{{ shortHash(store.selected.input_hash) }}</strong></div><div><span>测量 / 声源</span><strong>{{ store.selected.measurement_ids.length }} / {{ store.selected.source_profile_ids.length }}</strong></div><div><span>矩阵</span><strong>{{ store.selected.evidence.matrix_rows }} × {{ store.selected.evidence.matrix_columns }}</strong></div><div><span>迭代</span><strong>{{ store.selected.evidence.iterations }}</strong></div></div>
        <div class="ranking-table"><div class="ranking-head"><span>排名</span><span>候选声源</span><span>总贡献</span><span>预测总级</span></div><div v-for="(source, index) in store.selected.contributions" :key="source.source_profile_id" class="ranking-row"><span>{{ String(index + 1).padStart(2, '0') }}</span><span><strong>{{ source.source_name }}</strong><small>{{ source.source_code }}</small></span><b>{{ fixed(source.contribution_pct, 2) }}%</b><span>{{ fixed(source.overall_db, 2) }} dB</span></div></div>
        <div v-if="store.selected.evidence.warnings.length" class="warning-list"><strong>解释性提示</strong><span v-for="warning in store.selected.evidence.warnings" :key="warning">{{ warning }}</span></div>
        <div class="detail-actions"><el-button :icon="Eye" @click="detailOpen = true">计算证据</el-button><el-button v-if="canReviewSelected" type="primary" plain @click="reviewOpen = true">记录复核</el-button><el-button v-if="canConfirmSelected" type="primary" :icon="ShieldCheck" @click="confirm">独立确认</el-button></div>
      </section>
    </div>
  </div></AppShell>

  <el-dialog v-model="runOpen" title="运行离线归因" width="min(680px, 94vw)" destroy-on-close>
    <el-alert title="只接受 ready 测量和 active 声源谱；服务端会按 ID 重载并冻结完整输入，不信任客户端频谱。" type="info" :closable="false" show-icon />
    <el-form label-position="top"><el-form-item label="测量"><el-select v-model="form.measurement_ids" multiple collapse-tags collapse-tags-tooltip><el-option v-for="item in measurements.items.filter(x => x.measurement_state === 'ready')" :key="item.id" :label="`${item.point_code} · ${formatDate(item.measured_at)}`" :value="item.id" /></el-select></el-form-item><el-form-item label="候选声源"><el-select v-model="form.source_profile_ids" multiple collapse-tags collapse-tags-tooltip><el-option v-for="item in sources.items.filter(x => x.profile_state === 'active')" :key="item.id" :label="`${item.source_code} · V${item.version}`" :value="item.id" /></el-select></el-form-item></el-form>
    <template #footer><el-button @click="runOpen = false">取消</el-button><el-button type="primary" :loading="submitting" :disabled="!form.measurement_ids.length || !form.source_profile_ids.length" @click="createRun">冻结并计算</el-button></template>
  </el-dialog>
  <el-dialog v-model="reviewOpen" title="记录独立复核" width="min(560px, 94vw)"><el-form label-position="top"><el-form-item label="复核说明"><el-input v-model="reviewNote" type="textarea" :rows="4" maxlength="1000" show-word-limit /></el-form-item></el-form><template #footer><el-button @click="reviewOpen = false">取消</el-button><el-button type="primary" @click="review">提交复核</el-button></template></el-dialog>
  <AttributionDetailDrawer v-model="detailOpen" :run="store.selected" />
</template>
