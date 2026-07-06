<template>
  <div class="drawer-tab-content">
    <div class="section-header">
      <div class="section-title">Histórico de Jobs</div>
      <p class="section-desc">Visualize as execuções passadas e selecione para ver detalhes.</p>
    </div>

    <div v-if="loading && jobs.length === 0" class="loading-state">
      <div class="spinner-small"></div>
      <span>Carregando histórico...</span>
    </div>

    <div v-else-if="jobs.length === 0" class="empty-state">
      <p>Nenhum job registrado para esta empresa.</p>
    </div>

    <div v-else class="jobs-list">
      <div 
        v-for="j in jobs" 
        :key="j.id" 
        class="job-item" 
        :class="{ 'job-item--selected': selectedJobId === j.id }"
        @click="$emit('selectJob', j.id)"
      >
        <div class="job-main">
          <div class="job-meta">
            <span class="job-type td-mono">
              {{ j.tipo }}
              <span v-if="j.executor" class="executor-tag">({{ j.executor }})</span>
            </span>
            <span class="job-date td-mono">{{ fmtDate(j.iniciado_at) }}</span>
          </div>
          <div class="job-status">
            <span :class="['pill-small', statusCls(j.status)]">{{ j.status }}</span>
          </div>
        </div>
        
        <div class="job-footer">
          <span class="job-duration td-mono">⏱ {{ duration(j) }}</span>
          <span v-if="j.erro" class="job-error td-mono" :title="j.erro">{{ j.erro }}</span>
        </div>
      </div>

      <button 
        v-if="hasMore" 
        class="btn-load-more" 
        @click="loadMore" 
        :disabled="loading"
      >
        <span v-if="loading" class="spinner-small"></span>
        <span v-else>CARREGAR MAIS</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import api from "@/api/client"

interface SyncJob {
  id: string
  tipo: string
  status: string
  erro: string
  iniciado_at: string | null
  concluido_at: string | null
  executor?: string
}

const props = defineProps<{
  empresaId: string
  selectedJobId: string
}>()

const emit = defineEmits<{
  (e: 'selectJob', jobId: string): void
}>()

const jobs = ref<SyncJob[]>([])
const loading = ref(false)
const page = ref(1)
const hasMore = ref(true)
const perPage = 10

async function fetchJobs(reset = false) {
  if (reset) {
    page.value = 1
    jobs.value = []
    hasMore.value = true
  }
  
  if (!hasMore.value || loading.value) return

  loading.value = true
  try {
    const r = await api.get(`/sync/${props.empresaId}/jobs?page=${page.value}&per_page=${perPage}`)
    const newJobs = r.data.data ?? []
    
    if (newJobs.length < perPage) {
      hasMore.value = false
    }
    
    jobs.value = [...jobs.value, ...newJobs]
  } catch (err) {
    console.error('Erro ao buscar histórico:', err)
    hasMore.value = false
  } finally {
    loading.value = false
  }
}

function loadMore() {
  page.value++
  fetchJobs()
}

onMounted(() => {
  fetchJobs(true)
})

watch(() => props.empresaId, () => {
  fetchJobs(true)
})

function fmtDate(d: string | null) {
  if (!d) return "-"
  const dt = new Date(d)
  return dt.toLocaleString("pt-BR", { 
    day: '2-digit', 
    month: '2-digit', 
    hour: '2-digit', 
    minute: '2-digit' 
  })
}

function duration(j: SyncJob) {
  if (!j.iniciado_at || !j.concluido_at) return "-"
  const s = Math.round((new Date(j.concluido_at).getTime() - new Date(j.iniciado_at).getTime()) / 1000)
  return s > 60 ? `${Math.floor(s / 60)}m ${s % 60}s` : `${s}s`
}

function statusCls(s: string) {
  switch (s) {
    case 'concluido': return 'text-green'
    case 'erro': return 'text-red'
    case 'rodando': return 'text-blue'
    case 'pendente': return 'text-gray'
    default: return 'text-gray'
  }
}
</script>

<style scoped>
.drawer-tab-content { padding: 24px; }

.section-header { margin-bottom: 20px; }
.section-title { font-family: var(--mono); font-size: 12px; font-weight: 700; color: var(--accent); text-transform: uppercase; letter-spacing: 1px; margin-bottom: 4px; }
.section-desc { font-size: 11px; color: var(--text3); margin: 0; }

.loading-state, .empty-state { padding: 48px 24px; text-align: center; color: var(--text3); font-size: 12px; display: flex; flex-direction: column; align-items: center; gap: 12px; }

.jobs-list { display: flex; flex-direction: column; gap: 10px; }

.job-item { 
  background: rgba(255,255,255,0.02); 
  border: 1px solid var(--border); 
  border-radius: 8px; 
  padding: 12px 16px; 
  cursor: pointer; 
  transition: var(--trans);
}
.job-item:hover { background: rgba(255,255,255,0.04); border-color: var(--border2); }
.job-item--selected { border-color: var(--accent); background: rgba(0, 229, 255, 0.03); }

.job-main { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.job-meta { display: flex; flex-direction: column; gap: 2px; }
.job-type { font-size: 12px; font-weight: 700; color: var(--text); text-transform: uppercase; }
.executor-tag { color: var(--accent); font-size: 10px; margin-left: 4px; }
.job-date { font-size: 10px; color: var(--text3); }

.job-footer { display: flex; justify-content: space-between; align-items: center; gap: 16px; }
.job-duration { font-size: 10px; color: var(--text2); }
.job-error { font-size: 10px; color: var(--red); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex: 1; text-align: right; }

.btn-load-more { 
  margin-top: 16px; 
  background: var(--bg3); 
  border: 1px solid var(--border2); 
  color: var(--text2); 
  border-radius: 6px; 
  padding: 10px; 
  font-family: var(--mono); 
  font-size: 10px; 
  font-weight: 700; 
  cursor: pointer; 
  transition: var(--trans); 
}
.btn-load-more:hover:not(:disabled) { border-color: var(--accent); color: var(--accent); }
.btn-load-more:disabled { opacity: 0.5; cursor: not-allowed; }

.td-mono { font-family: var(--mono); }
.pill-small { font-family: var(--mono); font-size: 9px; text-transform: uppercase; font-weight: 700; }
.text-green { color: #22c55e; }
.text-red { color: #ef4444; }
.text-blue { color: #00e5ff; }
.text-gray { color: var(--text3); }

.spinner-small { width: 14px; height: 14px; border: 1.5px solid var(--border2); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
