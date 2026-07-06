<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/api/client'

interface OmieConfig {
  id: string
  modulo: string
  endpoint_path: string
  action: string
  array_field: string
  page_size: number
  ativo: boolean
  ignorar_delta: boolean
  notas: string | null
  updated_at: string
  updated_by_email: string | null
}

const configs = ref<OmieConfig[]>([])
const loading = ref(false)
const saving = ref(false)
const selectedConfig = ref<OmieConfig | null>(null)
const showModal = ref(false)

async function loadConfigs() {
  loading.value = true
  try {
    const r = await api.get('/admin/omie-config')
    configs.value = r.data.data
  } catch {
    // silencioso
  } finally {
    loading.value = false
  }
}

function editConfig(config: OmieConfig) {
  selectedConfig.value = { ...config }
  showModal.value = true
}

async function saveConfig() {
  if (!selectedConfig.value) return
  
  saving.value = true
  try {
    const r = await api.put(`/admin/omie-config/${selectedConfig.value.modulo}`, {
      endpoint_path: selectedConfig.value.endpoint_path,
      action: selectedConfig.value.action,
      array_field: selectedConfig.value.array_field,
      page_size: selectedConfig.value.page_size,
      ativo: selectedConfig.value.ativo,
      notas: selectedConfig.value.notas
    })
    
    // Atualiza na lista
    const index = configs.value.findIndex(c => c.modulo === selectedConfig.value?.modulo)
    if (index !== -1) {
      configs.value[index] = r.data.data
    }
    
    showModal.value = false
    selectedConfig.value = null
  } catch (err: any) {
    alert(err.response?.data?.message || 'Erro ao salvar configuração')
  } finally {
    saving.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString('pt-BR')
}

onMounted(loadConfigs)
</script>

<template>
  <div class="p-6">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-bold text-gray-800">Configuração de Endpoints Omie</h1>
      <button 
        @click="loadConfigs" 
        class="px-4 py-2 bg-gray-100 text-gray-700 rounded hover:bg-gray-200 transition"
        :disabled="loading"
      >
        {{ loading ? 'Carregando...' : 'Atualizar' }}
      </button>
    </div>

    <div class="bg-white rounded-lg shadow overflow-hidden">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Módulo</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Endpoint</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Array Campo</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Ações</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="c in configs" :key="c.modulo" class="hover:bg-gray-50 transition">
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{{ c.modulo }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ c.endpoint_path }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ c.action }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ c.array_field }}</td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span 
                :class="[
                  'px-2 inline-flex text-xs leading-5 font-semibold rounded-full',
                  c.ativo ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                ]"
              >
                {{ c.ativo ? 'Ativo' : 'Inativo' }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
              <button @click="editConfig(c)" class="text-indigo-600 hover:text-indigo-900">Editar</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal de Edição -->
    <div v-if="showModal" class="fixed inset-0 z-10 overflow-y-auto" aria-labelledby="modal-title" role="dialog" aria-modal="true">
      <div class="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
        <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" aria-hidden="true" @click="showModal = false"></div>

        <span class="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">&#8203;</span>

        <div class="inline-block align-middle bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
          <div class="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
            <h3 class="text-lg leading-6 font-medium text-gray-900 mb-4" id="modal-title">
              Editar: {{ selectedConfig?.modulo }}
            </h3>
            
            <div class="space-y-4" v-if="selectedConfig">
              <div>
                <label class="block text-sm font-medium text-gray-700">Endpoint Path</label>
                <input v-model="selectedConfig.endpoint_path" type="text" class="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
              </div>
              
              <div>
                <label class="block text-sm font-medium text-gray-700">Action</label>
                <input v-model="selectedConfig.action" type="text" class="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
              </div>
              
              <div>
                <label class="block text-sm font-medium text-gray-700">Array Field</label>
                <input v-model="selectedConfig.array_field" type="text" class="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
              </div>

              <div class="flex space-x-4">
                <div class="flex-1">
                  <label class="block text-sm font-medium text-gray-700">Page Size</label>
                  <input v-model.number="selectedConfig.page_size" type="number" class="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm">
                </div>
                <div class="flex items-center mt-6">
                  <input v-model="selectedConfig.ativo" type="checkbox" class="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded">
                  <label class="ml-2 block text-sm text-gray-900">Ativo</label>
                </div>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-700">Notas</label>
                <textarea v-model="selectedConfig.notas" rows="3" class="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"></textarea>
              </div>

              <div class="text-xs text-gray-500" v-if="selectedConfig.updated_at">
                Última edição: {{ selectedConfig.updated_by_email || 'Sistema' }} — {{ formatDate(selectedConfig.updated_at) }}
              </div>
            </div>
          </div>
          <div class="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
            <button 
              @click="saveConfig" 
              type="button" 
              class="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-indigo-600 text-base font-medium text-white hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:ml-3 sm:w-auto sm:text-sm"
              :disabled="saving"
            >
              {{ saving ? 'Salvando...' : 'Salvar' }}
            </button>
            <button 
              @click="showModal = false" 
              type="button" 
              class="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
            >
              Cancelar
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
