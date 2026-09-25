<script setup lang="ts">
import { Eye, RefreshCw, ScrollText, Search } from '@lucide/vue'
import { computed, onMounted, reactive, ref } from 'vue'
import AppShell from '../components/common/AppShell.vue'
import AttributionDetailDrawer from '../components/common/AttributionDetailDrawer.vue'
import PageHeader from '../components/common/PageHeader.vue'
import { auditApi } from '../api/audit-api'
import { errorMessage } from '../api/client'
import type { AuditLog } from '../types/audit'
import { formatDate, shortHash } from '../utils/format'

const items = ref<AuditLog[]>([])
const loading = ref(false)
const error = ref('')
const selected = ref<AuditLog | null>(null)
const detailOpen = ref(false)
const filters = reactive({ entity_type: '', request_id: '', actor: '' })
const entityCounts = computed(() => new Set(items.value.map((item) => item.entity_type)).size)

async function load() {
  loading.value = true; error.value = ''
  try { items.value = await auditApi.list({ ...filters, limit: 300 }) } catch (reason) { error.value = errorMessage(reason) } finally { loading.value = false }
}
function inspect(item: AuditLog) { selected.value = item; detailOpen.value = true }
onMounted(load)
</script>

<template>
  <AppShell><div class="page-wrap">
    <PageHeader eyebrow="IMMUTABLE CHANGE STREAM" title="审计中心" description="按实体、request ID 和操作者核对写入前后快照与算法元数据。"><el-button :icon="RefreshCw" aria-label="刷新审计" @click="load" /></PageHeader>
    <div class="metric-band"><div><span>审计事件</span><strong>{{ items.length }}</strong></div><div><span>实体类型</span><strong>{{ entityCounts }}</strong></div><div><span>含 request ID</span><strong>{{ items.filter(x => x.request_id).length }}</strong></div><div><span>不可变记录</span><strong>ON</strong></div></div>
    <div class="audit-filters"><el-select v-model="filters.entity_type" clearable placeholder="全部实体"><el-option v-for="name in ['MonitoringPoint','NoiseMeasurement','SourceProfile','AttributionRun']" :key="name" :label="name" :value="name" /></el-select><el-input v-model="filters.request_id" clearable placeholder="Request ID" :prefix-icon="Search" /><el-input v-model="filters.actor" clearable placeholder="操作者" /><el-button type="primary" @click="load">筛选</el-button></div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
    <el-skeleton v-if="loading" :rows="7" animated />
    <div v-else-if="!items.length" class="empty-state"><ScrollText :size="30" /><h2>没有匹配的审计事件</h2><p>调整筛选条件，或完成一次写操作后刷新。</p></div>
    <div v-else class="audit-table-wrap"><el-table :data="items" stripe><el-table-column prop="id" label="#" width="64" /><el-table-column label="事件" min-width="220"><template #default="scope"><div class="primary-cell"><strong>{{ scope.row.action }}</strong><span>{{ scope.row.entity_type }} #{{ scope.row.entity_id }}</span></div></template></el-table-column><el-table-column label="操作者" min-width="150"><template #default="scope"><span>{{ scope.row.actor_name }}</span></template></el-table-column><el-table-column label="Request ID" min-width="180"><template #default="scope"><code class="request-code" :title="scope.row.request_id">{{ shortHash(scope.row.request_id) }}</code></template></el-table-column><el-table-column label="时间" min-width="140"><template #default="scope">{{ formatDate(scope.row.created_at) }}</template></el-table-column><el-table-column label="证据" width="82" fixed="right"><template #default="scope"><el-tooltip content="查看前后快照"><el-button text circle aria-label="查看审计证据" @click="inspect(scope.row)"><Eye :size="16" /></el-button></el-tooltip></template></el-table-column></el-table></div>
  </div></AppShell>
  <AttributionDetailDrawer v-model="detailOpen" :title="selected ? `${selected.entity_type} #${selected.entity_id}` : '审计证据'" :subtitle="selected?.action" :snapshot="selected ? { before: selected.before, after: selected.after, metadata: selected.metadata, request_id: selected.request_id, actor: selected.actor_name } : {}" />
</template>
