<template>
  <div class="drawer-tab-content">
    <div v-if="!job" class="empty-state">
      <p>Nenhum job em execução ou selecionado.</p>
    </div>
    <template v-else>
      <div class="job-header">
        <div class="job-info">
          <span class="job-title">Job: {{ job.tipo.toUpperCase() }}</span>
          <span class="job-id td-mono">{{ job.id.split('-')[0] }}...</span>
        </div>
        <div class="job-status">
          <span :class="['pill', statusCls]">{{ job.status.toUpperCase() }}</span>
          <span v-if="job.iniciado_at" class="job-time td-mono">⏱ {{ duration }}</span>
        </div>
      </div>

      <div v-if="job.erro" class="error-banner">
        <div class="error-title">ERRO NO JOB</div>
        <div class="error-msg">{{ job.erro }}</div>
      </div>

      <div class="progress-list">
        <div v-for="p in progress" :key="p.executor" class="progress-item">
          <div class="progress-main">
            <div class="executor-name">{{ p.executor }}</div>
            <div class="executor-status">
              <span :class="['pill-small', moduleStatusCls(p.status)]">{{ p.status }}</span>
            </div>
          </div>
          
          <div class="progress-details">
            <div class="progress-bar-wrap">
              <div v-if="p.status === 'rodando' && p.total_paginas" class="progress-container">
                <div class="progress-bar" :style="{ width: (p.pagina_atual / p.total_paginas * 100) + '%' }"></div>
              </div>
              <div class="progress-stats td-mono">
                <span v-if="p.pagina_atual">{{ p.pagina_atual }} / {{ p.total_paginas || '?' }} pág.</span>
                <span v-else>—</span>
                <span>{{ p.registros_proc }} <small v-if="p.registros_total">/ {{ p.registros_total }}</small> reg.</span>
              </div>
            </div>

            <div class="progress-actions">
              <button 
                class="btn-inspect" 
                title="Ver Fila de Páginas" 
                @click="togglePages(p.executor)"
                :disabled="!job"
              >
                📄
              </button>
              <button 
                class="btn-inspect" 
                title="Inspecionar Payload" 
                @click="$emit('inspecionarPayload', p)"
                :disabled="!p.iniciado_at"
              >
                { }
              </button>
            </div>
          </div>

          <!-- Fila de Páginas (Sub-jobs) -->
          <div v-if="expandedExecutor === p.executor" class="page-queue-wrap">
            <div v-if="loadingPages" class="p-4 text-center"><div class="spinner-small"></div></div>
            <div v-else-if="pages.length === 0" class="p-4 text-center text-gray-500 text-xs">Nenhuma página encontrada.</div>
            <table v-else class="page-table">
              <thead>
                <tr>
                  <th>PÁG</th>
                  <th>STATUS</th>
                  <th>TENT.</th>
                  <th>REG.</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="pg in pages.filter(x => x.modulo === p.executor)" :key="pg.id">
                  <td>{{ pg.pagina }}</td>
                  <td>
                    <span :class="['status-dot', pg.status]"></span>
                    {{ pg.status }}
                  </td>
                  <td>{{ pg.tentativas }} / {{ pg.max_tentativas }}</td>
                  <td>{{ pg.registros_gravados }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          
          <div v-if="p.erro" class="executor-error td-mono">
            {{ formatErro(p.erro) }}
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import api from "@/api/client"

interface SyncJob {
  id: string
  empresa_id?: string
  tipo: string
  status: string
  erro: string
  iniciado_at: string | null
  concluido_at: string | null
}

interface PageRow {
  id: string
  modulo: string
  pagina: number
  total_paginas: number
  status: string
  tentativas: number
  max_tentativas: number
  registros_gravados: number
  erro?: string
}

interface SyncJobProgress {
  executor: string
  status: string
  pagina_atual: number
  total_paginas: number
  registros_proc: number
  registros_total: number
  erro: string | null
  iniciado_at: string | null
  concluido_at: string | null
  updated_at: string
  ultimo_payload?: unknown
  ultimo_response?: unknown
  erro_payload?: unknown
  erro_response?: string
}

const props = defineProps<{
  job: SyncJob | null
  progress: SyncJobProgress[]
}>()

defineEmits<{
  (e: 'inspecionarPayload', item: SyncJobProgress): void
}>()

const now = ref(Date.now())
const expandedExecutor = ref<string | null>(null)
const pages = ref<PageRow[]>([])
const loadingPages = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

async function togglePages(executor: string) {
  if (expandedExecutor.value === executor) {
    expandedExecutor.value = null
    return
  }
  
  expandedExecutor.value = executor
  await fetchPages()
}

async function fetchPages() {
  if (!props.job || !expandedExecutor.value) return
  
  loadingPages.value = true
  try {
    const empresaId = props.job.empresa_id || ""
    const r = await api.get(`/sync/${empresaId}/pages?job_id=${props.job.id}`)
    pages.value = r.data.data ?? []
  } catch (err) {
    console.error("Erro ao buscar páginas:", err)
  } finally {
    loadingPages.value = false
  }
}

// Se o job mudar, fecha a expansão
watch(() => props.job?.id, () => {
  expandedExecutor.value = null
  pages.value = []
})

onMounted(() => {
  timer = setInterval(() => { now.value = Date.now() }, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const duration = computed(() => {
  if (!props.job?.iniciado_at) return '-'
  const start = new Date(props.job.iniciado_at).getTime()
  const end = props.job.concluido_at ? new Date(props.job.concluido_at).getTime() : now.value
  const s = Math.round((end - start) / 1000)
  return s > 60 ? `${Math.floor(s / 60)}m ${s % 60}s` : `${s}s`
})

const statusCls = computed(() => {
  switch (props.job?.status) {
    case 'rodando': return 'pill-blue'
    case 'pendente': return 'pill-gray'
    case 'erro': return 'pill-red'
    case 'concluido': return 'pill-green'
    default: return 'pill-gray'
  }
})

function moduleStatusCls(s: string) {
  switch (s) {
    case 'concluido': return 'text-green'
    case 'erro': return 'text-red'
    case 'rodando': return 'text-blue'
    case 'pulado': return 'text-gray'
    default: return 'text-gray'
  }
}

function formatErro(erro: string | null | undefined): string {
  if (!erro) return ''
  return erro.replace(/^\[DLQ\]\s*/, '')
}
</script>

<style scoped>
.drawer-tab-content { padding: 24px; }
.empty-state { padding: 48px 24px; text-align: center; color: var(--text3); font-size: 12px; }

.job-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid var(--border); }
.job-info { display: flex; flex-direction: column; gap: 4px; }
.job-title { font-family: var(--mono); font-size: 13px; font-weight: 700; color: var(--text); }
.job-id { font-size: 10px; color: var(--text3); }
.job-status { display: flex; flex-direction: column; align-items: flex-end; gap: 6px; }
.job-time { font-size: 11px; color: var(--text2); }

.error-banner { background: rgba(239,68,68,0.08); border: 1px solid rgba(239,68,68,0.2); border-radius: 8px; padding: 12px 16px; margin-bottom: 24px; }
.error-title { font-family: var(--mono); font-size: 9px; font-weight: 700; color: var(--red); margin-bottom: 4px; }
.error-msg { font-size: 12px; color: var(--text); line-height: 1.4; }

.progress-list { display: flex; flex-direction: column; gap: 16px; }
.progress-item { background: rgba(255,255,255,0.02); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: var(--space-md); margin-bottom: var(--space-sm); }

.progress-main { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.executor-name { font-family: var(--mono); font-size: 12px; font-weight: 600; color: var(--accent); }

.progress-details { display: flex; align-items: center; gap: 16px; }
.progress-bar-wrap { flex: 1; display: flex; flex-direction: column; gap: 6px; }

.progress-container { width: 100%; height: 6px; background: var(--surface-tertiary); border-radius: 999px; overflow: hidden; }
.progress-bar { height: 100%; background: var(--brand-primary); border-radius: 999px; transition: width 0.3s ease; }
.progress-stats { display: flex; justify-content: space-between; font-size: 10px; color: var(--text3); }

.btn-inspect { background: var(--bg3); border: 1px solid var(--border2); color: var(--text2); border-radius: 4px; width: 28px; height: 28px; display: flex; align-items: center; justify-content: center; cursor: pointer; transition: var(--trans); font-size: 10px; }
.btn-inspect:hover:not(:disabled) { border-color: var(--accent); color: var(--accent); }
.btn-inspect:disabled { opacity: 0.3; cursor: not-allowed; }

.executor-error { margin-top: 12px; padding-top: 8px; border-top: 1px solid rgba(255,255,255,0.05); font-size: 10px; color: var(--red); }

.td-mono { font-family: var(--mono); }
.pill { display: inline-flex; padding: 2px 9px; border-radius: 20px; font-family: var(--mono); font-size: 10px; font-weight: 600; }
.pill-green { background: rgba(34,197,94,0.12); color: #22c55e; }
.pill-red { background: rgba(239,68,68,0.12); color: #ef4444; }
.pill-blue { background: rgba(0,229,255,0.1); color: #00e5ff; }
.pill-gray { background: rgba(255,255,255,0.06); color: var(--text3); }

.pill-small { font-family: var(--mono); font-size: 9px; text-transform: uppercase; font-weight: 700; }
.text-green { color: #22c55e; }
.text-red { color: #ef4444; }
.text-blue { color: #00e5ff; }
.text-gray { color: var(--text3); }

/* Fila de Páginas */
.page-queue-wrap {
  margin-top: 12px;
  padding: 12px;
  background: rgba(0,0,0,0.2);
  border-radius: 6px;
  border: 1px solid rgba(255,255,255,0.05);
}

.page-table {
  width: 100%;
  font-size: 10px;
  border-collapse: collapse;
}

.page-table th {
  text-align: left;
  color: var(--text3);
  font-family: var(--mono);
  padding: 4px 8px;
  border-bottom: 1px solid rgba(255,255,255,0.1);
}

.page-table td {
  padding: 6px 8px;
  color: var(--text2);
  border-bottom: 1px solid rgba(255,255,255,0.03);
}

.status-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin-right: 4px;
}

.status-dot.pendente { background: #94a3b8; }
.status-dot.rodando { background: #00e5ff; animation: pulse 1.5s infinite; }
.status-dot.concluido { background: #22c55e; }
.status-dot.erro { background: #ef4444; }
.status-dot.cancelado { background: #64748b; }

@keyframes pulse {
  0% { opacity: 1; }
  50% { opacity: 0.4; }
  100% { opacity: 1; }
}

.spinner-small {
  width: 12px;
  height: 12px;
  border: 2px solid rgba(255,255,255,0.1);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin: 0 auto;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
