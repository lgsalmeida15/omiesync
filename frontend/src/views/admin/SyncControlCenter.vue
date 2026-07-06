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

onMounted(() => {
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
      <h1 class="text-2xl font-bold text-gray-800">Sync Control Center</h1>
      <div class="flex gap-2">
        <button 
          @click="runRecovery" 
          :disabled="recoveryLoading"
          class="px-4 py-2 bg-orange-600 text-white rounded hover:bg-orange-700 disabled:opacity-50 flex items-center gap-2"
        >
          <span v-if="recoveryLoading" class="animate-spin">↻</span>
          Recovery Manual
        </button>
        <button 
          @click="() => { fetchOverview(); fetchJobsAtivos(); fetchDLQ(); }" 
          class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          Atualizar
        </button>
      </div>
    </div>

    <!-- Cards de Resumo -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
      <div class="bg-white p-4 rounded-lg shadow border-l-4 border-blue-500">
        <div class="text-sm text-gray-500 font-medium uppercase">Rodando</div>
        <div class="text-2xl font-bold text-gray-800">{{ overview.rodando || 0 }}</div>
      </div>
      <div class="bg-white p-4 rounded-lg shadow border-l-4 border-yellow-500">
        <div class="text-sm text-gray-500 font-medium uppercase">Pendente</div>
        <div class="text-2xl font-bold text-gray-800">{{ overview.pendente || 0 }}</div>
      </div>
      <div class="bg-white p-4 rounded-lg shadow border-l-4 border-red-500">
        <div class="text-sm text-gray-500 font-medium uppercase">Erros Ativos</div>
        <div class="text-2xl font-bold text-gray-800">{{ overview.erro || 0 }}</div>
      </div>
      <div class="bg-white p-4 rounded-lg shadow border-l-4 border-orange-500">
        <div class="text-sm text-gray-500 font-medium uppercase">Zumbis</div>
        <div class="text-2xl font-bold text-gray-800">{{ zumbiCount }}</div>
      </div>
    </div>

    <!-- Jobs Ativos -->
    <div class="bg-white rounded-lg shadow mb-8">
      <div class="p-4 border-b">
        <h2 class="text-lg font-semibold text-gray-700">Jobs Ativos</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-left">
          <thead class="bg-gray-50 text-gray-600 text-sm uppercase font-medium">
            <tr>
              <th class="px-6 py-3">Empresa / Grupo</th>
              <th class="px-6 py-3">Tipo</th>
              <th class="px-6 py-3">Início</th>
              <th class="px-6 py-3">Heartbeat</th>
              <th class="px-6 py-3">Status</th>
              <th class="px-6 py-3 text-right">Ação</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 text-sm">
            <tr v-if="jobsAtivos.length === 0">
              <td colspan="6" class="px-6 py-8 text-center text-gray-500">Nenhum job ativo no momento</td>
            </tr>
            <tr v-for="job in jobsAtivos" :key="job.id" class="hover:bg-gray-50">
              <td class="px-6 py-4">
                <div class="font-medium text-gray-800">{{ job.empresa_nome }}</div>
                <div class="text-xs text-gray-500">{{ job.grupo_nome }}</div>
              </td>
              <td class="px-6 py-4">
                <span class="px-2 py-1 bg-gray-100 rounded text-xs text-gray-600 font-medium uppercase">
                  {{ job.tipo }}
                </span>
              </td>
              <td class="px-6 py-4 text-gray-600">
                {{ formatDateTime(job.iniciado_at) }}
              </td>
              <td class="px-6 py-4">
                <span :class="job.is_zumbi ? 'text-red-600 font-bold' : 'text-gray-600'">
                  {{ formatHeartbeat(job.ultimo_heartbeat_at) }}
                </span>
              </td>
              <td class="px-6 py-4">
                <div class="flex items-center gap-2">
                  <span v-if="job.is_zumbi" class="px-2 py-0.5 bg-red-100 text-red-700 text-[10px] font-bold rounded uppercase">
                    ZUMBI
                  </span>
                  <span v-else class="flex h-2 w-2 rounded-full bg-blue-500 animate-pulse"></span>
                  <span class="capitalize text-gray-700">{{ job.status }}</span>
                </div>
              </td>
              <td class="px-6 py-4 text-right">
                <button 
                  @click="cancelarJob(job)"
                  :disabled="cancelingJobId === job.id"
                  class="text-red-600 hover:text-red-800 font-medium disabled:opacity-50"
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
    <div class="bg-white rounded-lg shadow">
      <div class="p-4 border-b">
        <h2 class="text-lg font-semibold text-gray-700">Dead Letter Queue</h2>
      </div>
      <div v-if="dlqPages.length === 0" class="p-8 text-center text-gray-500">
        Nenhum item na DLQ
      </div>
      <div v-else class="overflow-x-auto">
        <table class="w-full text-left">
          <thead class="bg-gray-50 text-gray-600 text-sm uppercase font-medium">
            <tr>
              <th class="px-6 py-3">Empresa / Grupo</th>
              <th class="px-6 py-3">Módulo</th>
              <th class="px-6 py-3">Página</th>
              <th class="px-6 py-3">Tentativas</th>
              <th class="px-6 py-3">Erro</th>
              <th class="px-6 py-3 text-right">Ação</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 text-sm">
            <tr v-for="page in dlqPages" :key="page.id" class="hover:bg-gray-50">
              <td class="px-6 py-4">
                <div class="font-medium text-gray-800">{{ page.empresa_nome }}</div>
                <div class="text-xs text-gray-500">{{ page.grupo_nome }}</div>
              </td>
              <td class="px-6 py-4">
                <span class="px-2 py-1 bg-gray-100 rounded text-xs text-gray-600 font-medium uppercase">
                  {{ page.modulo }}
                </span>
              </td>
              <td class="px-6 py-4 text-gray-600">
                Pg {{ page.pagina }} / {{ page.total_paginas }}
              </td>
              <td class="px-6 py-4 text-gray-600">
                {{ page.tentativas }} / {{ page.max_tentativas }}
              </td>
              <td class="px-6 py-4">
                <div class="text-red-600 max-w-xs truncate" :title="page.erro || ''">
                  {{ formatErro(page.erro) || 'Erro desconhecido' }}
                </div>
              </td>
              <td class="px-6 py-4 text-right">
                <button 
                  @click="retryPage(page)"
                  :disabled="retryingPageId === page.id"
                  class="text-blue-600 hover:text-blue-800 font-medium disabled:opacity-50"
                >
                  {{ retryingPageId === page.id ? 'Agendando...' : 'Retry' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
