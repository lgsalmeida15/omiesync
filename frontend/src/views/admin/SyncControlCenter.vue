<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import api from '@/api/client'

interface JobAtivoAdmin {
  id: string
  empresa_id: string
  empresa_nome: string
  grupo_nome: string
  tipo: string
  status: string
  iniciado_at: string
  ultimo_heartbeat_at: string | null
  is_zumbi: boolean
}

interface SyncOverview {
  [status: string]: number
}

interface DLQPage {
  id: string
  job_id: string
  empresa_nome: string
  grupo_nome: string
  modulo: string
  pagina: number
  total_paginas: number
  tentativas: number
  max_tentativas: number
  erro: string | null
}

const auth = useAuthStore()
const overview = ref<SyncOverview>({})
const jobsAtivos = ref<JobAtivoAdmin[]>([])
const dlqPages = ref<DLQPage[]>([])
const loading = ref(false)
const recoveryLoading = ref(false)
const cancelingJobId = ref<string | null>(null)
const retryingPageId = ref<string | null>(null)
let pollInterval: number | null = null

const zumbiCount = computed(() => {
  return jobsAtivos.value.filter(j => j.is_zumbi).length
})

async function fetchOverview() {
  try {
    const r = await api.get('/admin/sync/overview')
    overview.value = r.data.data
  } catch (err: any) {
    console.error('Erro ao buscar overview:', err)
  }
}

async function fetchJobsAtivos() {
  try {
    const r = await api.get('/admin/sync/jobs/ativos')
    jobsAtivos.value = r.data.data
  } catch (err: any) {
    console.error('Erro ao buscar jobs ativos:', err)
  }
}

async function fetchDLQ() {
  try {
    const r = await api.get('/admin/sync/dlq')
    dlqPages.value = r.data.data
  } catch (err: any) {
    console.error('Erro ao buscar DLQ:', err)
  }
}

async function runRecovery() {
  if (!confirm('Executar startup recovery manualmente? Isso marcará todos os jobs presos como erro.')) return
  
  recoveryLoading.value = true
  try {
    await api.post('/admin/sync/startup-recovery')
    await Promise.all([fetchOverview(), fetchJobsAtivos()])
  } catch (err: any) {
    alert('Erro ao executar recovery: ' + (err.response?.data?.message || err.message))
  } finally {
    recoveryLoading.value = false
  }
}

async function cancelarJob(job: JobAtivoAdmin) {
  if (!confirm(`Cancelar o job de ${job.empresa_nome}?`)) return
  
  cancelingJobId.value = job.id
  try {
    await api.post(`/admin/sync/jobs/${job.id}/cancelar`)
    await Promise.all([fetchOverview(), fetchJobsAtivos()])
  } catch (err: any) {
    alert('Erro ao cancelar job: ' + (err.response?.data?.message || err.message))
  } finally {
    cancelingJobId.value = null
  }
}

async function retryPage(page: DLQPage) {
  retryingPageId.value = page.id
  try {
    await api.post(`/admin/sync/pages/${page.id}/retry`)
    await fetchDLQ()
  } catch (err: any) {
    alert('Erro ao agendar retry: ' + (err.response?.data?.message || err.message))
  } finally {
    retryingPageId.value = null
  }
}

function formatHeartbeat(ts: string | null): string {
  if (!ts) return '—'
  const diff = Math.floor((Date.now() - new Date(ts).getTime()) / 1000 / 60)
  if (diff < 1) return 'agora'
  return `${diff} min atrás`
}

function formatDateTime(ts: string): string {
  return new Date(ts).toLocaleString()
}

function formatErro(erro: string | null): string {
  if (!erro) return ''
  return erro.replace(/^\[DLQ\]\s*/, '')
}

// ── Manutenção operacional ─────────────────────────────────────────────────
interface ConsultaAtiva {
  pid: number
  estado: string
  esperando_por: string | null
  duracao_seg: number
  query: string
  aplicacao: string
}

interface RefreshResultado {
  view: string
  concorrente: boolean
  duracao_seg: number
  erro?: string
}

const consultas        = ref<ConsultaAtiva[]>([])
const consultasLoading = ref(false)
const cancelandoPid    = ref<number | null>(null)
const grupos           = ref<{ id: string; nome: string }[]>([])
const grupoRefresh     = ref('')
const refreshLoading   = ref(false)
const refreshResultado = ref<RefreshResultado[] | null>(null)

async function fetchConsultas() {
  consultasLoading.value = true
  try {
    const r = await api.get('/admin/manutencao/consultas')
    consultas.value = r.data.data ?? []
  } catch { consultas.value = [] }
  finally { consultasLoading.value = false }
}

async function cancelarConsulta(c: ConsultaAtiva) {
  if (!confirm(`Cancelar a consulta ${c.pid}?\n\nEla é interrompida imediatamente. Um REFRESH é transacional, então a view mantém o conteúdo anterior.`)) return
  cancelandoPid.value = c.pid
  try {
    await api.post(`/admin/manutencao/consultas/${c.pid}/cancelar`)
    await fetchConsultas()
  } catch (e: any) {
    alert(e?.response?.data?.message ?? 'Erro ao cancelar consulta')
  } finally { cancelandoPid.value = null }
}

async function fetchGrupos() {
  try {
    const r = await api.get('/admin/grupos?page=1&per_page=100')
    grupos.value = r.data.data ?? []
    if (!grupoRefresh.value && grupos.value.length) grupoRefresh.value = grupos.value[0].id
  } catch { /* sem grupos, o seletor fica vazio */ }
}

async function refreshViews() {
  if (!grupoRefresh.value) return
  const nome = grupos.value.find(g => g.id === grupoRefresh.value)?.nome ?? ''
  if (!confirm(`Atualizar as views gerenciais de "${nome}"?\n\nSe já estiverem populadas, roda em modo concorrente e o dashboard segue disponível. Caso contrário, fica indisponível até concluir.`)) return
  refreshLoading.value = true
  refreshResultado.value = null
  try {
    const r = await api.post(`/admin/manutencao/grupos/${grupoRefresh.value}/refresh-views`)
    refreshResultado.value = r.data.data ?? []
  } catch (e: any) {
    alert(e?.response?.data?.message ?? 'Erro ao atualizar views')
  } finally { refreshLoading.value = false }
}

function fmtDuracao(seg: number): string {
  if (seg < 60) return `${Math.round(seg)}s`
  const m = Math.floor(seg / 60)
  return m < 60 ? `${m}min ${Math.round(seg % 60)}s` : `${Math.floor(m / 60)}h ${m % 60}min`
}

onMounted(() => {
  fetchConsultas()
  fetchGrupos()
  fetchOverview()
  fetchJobsAtivos()
  fetchDLQ()
  pollInterval = window.setInterval(fetchJobsAtivos, 30000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<template>
  <div class="p-6">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-text-main">Sync Control Center</h1>
      <div class="flex gap-2">
        <button 
          @click="runRecovery" 
          :disabled="recoveryLoading"
          class="px-4 py-2 bg-warning text-oncolor rounded hover:bg-warning-fill disabled:opacity-50 flex items-center gap-2"
        >
          <span v-if="recoveryLoading" class="animate-spin">↻</span>
          Recovery Manual
        </button>
        <button 
          @click="() => { fetchOverview(); fetchJobsAtivos(); fetchDLQ(); }" 
          class="px-4 py-2 bg-primary text-oncolor rounded hover:bg-primary-hover"
        >
          Atualizar
        </button>
      </div>
    </div>

    <!-- Cards de Resumo -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
      <div class="bg-surface p-4 rounded-lg shadow border-l-4 border-primary">
        <div class="text-sm text-text-dim font-medium uppercase">Rodando</div>
        <div class="text-2xl font-bold text-text-main">{{ overview.rodando || 0 }}</div>
      </div>
      <div class="bg-surface p-4 rounded-lg shadow border-l-4 border-warning">
        <div class="text-sm text-text-dim font-medium uppercase">Pendente</div>
        <div class="text-2xl font-bold text-text-main">{{ overview.pendente || 0 }}</div>
      </div>
      <div class="bg-surface p-4 rounded-lg shadow border-l-4 border-danger">
        <div class="text-sm text-text-dim font-medium uppercase">Erros Ativos</div>
        <div class="text-2xl font-bold text-text-main">{{ overview.erro || 0 }}</div>
      </div>
      <div class="bg-surface p-4 rounded-lg shadow border-l-4 border-warning">
        <div class="text-sm text-text-dim font-medium uppercase">Zumbis</div>
        <div class="text-2xl font-bold text-text-main">{{ zumbiCount }}</div>
      </div>
    </div>

    <!-- Jobs Ativos -->
    <div class="bg-surface rounded-lg shadow mb-8">
      <div class="p-4 border-b">
        <h2 class="text-lg font-semibold text-text-main">Jobs Ativos</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-left">
          <thead class="bg-surface-2 text-text-muted text-sm uppercase font-medium">
            <tr>
              <th class="px-6 py-3">Empresa / Grupo</th>
              <th class="px-6 py-3">Tipo</th>
              <th class="px-6 py-3">Início</th>
              <th class="px-6 py-3">Heartbeat</th>
              <th class="px-6 py-3">Status</th>
              <th class="px-6 py-3 text-right">Ação</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-token text-sm">
            <tr v-if="jobsAtivos.length === 0">
              <td colspan="6" class="px-6 py-8 text-center text-text-dim">Nenhum job ativo no momento</td>
            </tr>
            <tr v-for="job in jobsAtivos" :key="job.id" class="hover:bg-surface-2">
              <td class="px-6 py-4">
                <div class="font-medium text-text-main">{{ job.empresa_nome }}</div>
                <div class="text-xs text-text-dim">{{ job.grupo_nome }}</div>
              </td>
              <td class="px-6 py-4">
                <span class="px-2 py-1 bg-surface-2 rounded text-xs text-text-muted font-medium uppercase">
                  {{ job.tipo }}
                </span>
              </td>
              <td class="px-6 py-4 text-text-muted">
                {{ formatDateTime(job.iniciado_at) }}
              </td>
              <td class="px-6 py-4">
                <span :class="job.is_zumbi ? 'text-danger font-bold' : 'text-text-muted'">
                  {{ formatHeartbeat(job.ultimo_heartbeat_at) }}
                </span>
              </td>
              <td class="px-6 py-4">
                <div class="flex items-center gap-2">
                  <span v-if="job.is_zumbi" class="px-2 py-0.5 bg-danger-weak text-danger text-xs font-bold rounded uppercase">
                    ZUMBI
                  </span>
                  <span v-else class="flex h-2 w-2 rounded-full bg-primary animate-pulse"></span>
                  <span class="capitalize text-text-main">{{ job.status }}</span>
                </div>
              </td>
              <td class="px-6 py-4 text-right">
                <button 
                  @click="cancelarJob(job)"
                  :disabled="cancelingJobId === job.id"
                  class="text-danger hover:text-danger-fill font-medium disabled:opacity-50"
                >
                  {{ cancelingJobId === job.id ? 'Cancelando...' : 'Cancelar' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Dead Letter Queue -->
    <div class="bg-surface rounded-lg shadow">
      <div class="p-4 border-b">
        <h2 class="text-lg font-semibold text-text-main">Dead Letter Queue</h2>
      </div>
      <div v-if="dlqPages.length === 0" class="p-8 text-center text-text-dim">
        Nenhum item na DLQ
      </div>
      <div v-else class="overflow-x-auto">
        <table class="w-full text-left">
          <thead class="bg-surface-2 text-text-muted text-sm uppercase font-medium">
            <tr>
              <th class="px-6 py-3">Empresa / Grupo</th>
              <th class="px-6 py-3">Módulo</th>
              <th class="px-6 py-3">Página</th>
              <th class="px-6 py-3">Tentativas</th>
              <th class="px-6 py-3">Erro</th>
              <th class="px-6 py-3 text-right">Ação</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-token text-sm">
            <tr v-for="page in dlqPages" :key="page.id" class="hover:bg-surface-2">
              <td class="px-6 py-4">
                <div class="font-medium text-text-main">{{ page.empresa_nome }}</div>
                <div class="text-xs text-text-dim">{{ page.grupo_nome }}</div>
              </td>
              <td class="px-6 py-4">
                <span class="px-2 py-1 bg-surface-2 rounded text-xs text-text-muted font-medium uppercase">
                  {{ page.modulo }}
                </span>
              </td>
              <td class="px-6 py-4 text-text-muted">
                Pg {{ page.pagina }} / {{ page.total_paginas }}
              </td>
              <td class="px-6 py-4 text-text-muted">
                {{ page.tentativas }} / {{ page.max_tentativas }}
              </td>
              <td class="px-6 py-4">
                <div class="text-danger max-w-xs truncate" :title="page.erro || ''">
                  {{ formatErro(page.erro) || 'Erro desconhecido' }}
                </div>
              </td>
              <td class="px-6 py-4 text-right">
                <button 
                  @click="retryPage(page)"
                  :disabled="retryingPageId === page.id"
                  class="text-primary hover:text-primary-hover font-medium disabled:opacity-50"
                >
                  {{ retryingPageId === page.id ? 'Agendando...' : 'Retry' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Manutenção operacional de banco -->
    <div class="bg-surface rounded-lg shadow mt-6">
      <div class="px-6 py-4 border-b flex items-center justify-between flex-wrap gap-3">
        <h2 class="text-lg font-semibold">Manutenção de banco</h2>
        <div class="flex items-center gap-2">
          <select v-model="grupoRefresh" class="border rounded px-2 py-1 text-sm">
            <option v-for="g in grupos" :key="g.id" :value="g.id">{{ g.nome }}</option>
          </select>
          <button
            @click="refreshViews"
            :disabled="refreshLoading || !grupoRefresh"
            class="bg-primary text-oncolor px-3 py-1.5 rounded text-sm font-medium hover:bg-primary-hover disabled:opacity-50"
          >
            {{ refreshLoading ? 'Atualizando...' : 'Atualizar views gerenciais' }}
          </button>
        </div>
      </div>

      <div v-if="refreshResultado" class="px-6 py-3 border-b bg-surface-2 text-sm space-y-1">
        <div v-for="r in refreshResultado" :key="r.view" class="flex items-center gap-2">
          <span class="font-mono text-xs">{{ r.view }}</span>
          <span v-if="r.erro" class="text-danger">falhou: {{ r.erro }}</span>
          <template v-else>
            <span class="text-text-muted">{{ fmtDuracao(r.duracao_seg) }}</span>
            <span v-if="r.concorrente" class="text-success text-xs">
              concorrente — dashboard permaneceu disponível
            </span>
            <span v-else class="text-warning text-xs">
              bloqueante — a view ainda não estava populada
            </span>
          </template>
        </div>
      </div>

      <div class="px-6 py-4">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm font-semibold text-text-main">
            Consultas em execução há mais de 5s
          </h3>
          <button @click="fetchConsultas" :disabled="consultasLoading"
                  class="text-sm text-primary hover:text-primary-hover disabled:opacity-50">
            {{ consultasLoading ? 'Carregando...' : 'Atualizar lista' }}
          </button>
        </div>

        <p v-if="!consultas.length" class="text-sm text-text-dim">
          Nenhuma consulta longa em andamento.
        </p>

        <table v-else class="w-full text-sm">
          <thead class="bg-surface-2 text-xs uppercase text-text-dim">
            <tr>
              <th class="px-3 py-2 text-left">PID</th>
              <th class="px-3 py-2 text-left">Duração</th>
              <th class="px-3 py-2 text-left">Estado</th>
              <th class="px-3 py-2 text-left">Consulta</th>
              <th class="px-3 py-2 text-right">Ação</th>
            </tr>
          </thead>
          <tbody class="divide-y">
            <tr v-for="c in consultas" :key="c.pid">
              <td class="px-3 py-2 font-mono text-xs">{{ c.pid }}</td>
              <td class="px-3 py-2" :class="c.duracao_seg > 60 ? 'text-danger font-medium' : ''">
                {{ fmtDuracao(c.duracao_seg) }}
              </td>
              <td class="px-3 py-2 text-xs">
                {{ c.esperando_por ? `esperando: ${c.esperando_por}` : c.estado }}
              </td>
              <td class="px-3 py-2 font-mono text-xs text-text-muted truncate max-w-md" :title="c.query">
                {{ c.query }}
              </td>
              <td class="px-3 py-2 text-right">
                <button @click="cancelarConsulta(c)" :disabled="cancelandoPid === c.pid"
                        class="text-danger hover:text-danger-fill font-medium disabled:opacity-50">
                  {{ cancelandoPid === c.pid ? 'Cancelando...' : 'Cancelar' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
