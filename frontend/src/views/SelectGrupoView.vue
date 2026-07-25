<template>
  <div class="select-grupo-page">
    <div class="card">
      <div class="logo">
        <div class="logo-icon">O</div>
        <div class="logo-text">Omie<span>Sync</span></div>
      </div>

      <h2 class="title">Selecionar Grupo</h2>
      <p class="subtitle">{{ isTroca ? 'Escolha o grupo para continuar' : 'Você pertence a múltiplos grupos. Escolha um para continuar.' }}</p>

      <div v-if="loading" class="loading">Carregando grupos...</div>

      <div v-else-if="grupos.length === 0" class="empty">
        Nenhum grupo disponível.
      </div>

      <div v-else class="grupos-list">
        <button
          v-for="g in grupos"
          :key="g.id"
          class="grupo-btn"
          :class="{ selected: selectedId === g.id, loading: selecting === g.id }"
          :disabled="!!selecting"
          @click="handleSelect(g.id)"
        >
          <div class="grupo-icon">{{ g.nome[0].toUpperCase() }}</div>
          <div class="grupo-info">
            <div class="grupo-nome">{{ g.nome }}</div>
            <div class="grupo-slug">{{ g.slug }}</div>
          </div>
          <div v-if="selecting === g.id" class="spinner" />
        </button>
      </div>

      <div v-if="error" class="error">{{ error }}</div>

      <button v-if="isTroca" class="cancel-btn" @click="cancelar">Cancelar</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore, type GrupoInfo } from '@/stores/auth'

const auth     = useAuthStore()
const router   = useRouter()
const route    = useRoute()

// isTroca = usuário já autenticado quer trocar de grupo
const isTroca  = computed(() => !!auth.accessToken)

const grupos   = ref<GrupoInfo[]>([])
const loading  = ref(false)
const selecting = ref('')
const selectedId = ref('')
const error    = ref('')

onMounted(async () => {
  if (isTroca.value) {
    // Busca grupos do usuário autenticado
    loading.value = true
    try {
      grupos.value = await auth.fetchGrupos()
    } catch {
      error.value = 'Erro ao carregar grupos.'
    } finally {
      loading.value = false
    }
  } else {
    // Usa grupos salvos do flow de login
    grupos.value = auth.pendingGrupos
    if (grupos.value.length === 0) {
      // Sem estado pendente — redireciona para login
      router.replace('/login')
    }
  }
})

async function handleSelect(grupoID: string) {
  if (selecting.value) return
  selecting.value = grupoID
  selectedId.value = grupoID
  error.value = ''

  try {
    if (isTroca.value) {
      await auth.trocaGrupo(grupoID)
    } else {
      await auth.selectGrupo(grupoID)
    }
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (e: unknown) {
    error.value = 'Erro ao selecionar grupo. Tente novamente.'
    selectedId.value = ''
  } finally {
    selecting.value = ''
  }
}

function cancelar() {
  router.back()
}
</script>

<style scoped>
.select-grupo-page {
  min-height: 100vh;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.card {
  width: 100%;
  max-width: 440px;
  background: var(--bg2);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 40px 32px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-icon {
  width: 36px; height: 36px; border-radius: 10px;
  background: linear-gradient(135deg, var(--accent), var(--accent3));
  display: flex; align-items: center; justify-content: center;
  font-size: 17px; font-weight: 800; color: #080c12;
}

.logo-text {
  font-size: 18px; font-weight: 800; color: var(--text);
}
.logo-text span { color: var(--accent); }

.title {
  font-size: 20px; font-weight: 700; color: var(--text);
  margin: 0;
}

.subtitle {
  font-size: 13px; color: var(--text2);
  margin: 0; line-height: 1.5;
}

.loading, .empty {
  color: var(--text3); font-size: 13px; text-align: center; padding: 16px 0;
}

.grupos-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.grupo-btn {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid var(--border);
  background: var(--bg3);
  cursor: pointer;
  color: var(--text);
  text-align: left;
  transition: var(--trans);
  position: relative;
}

.grupo-btn:hover:not(:disabled) {
  border-color: var(--accent);
  background: rgba(0,229,255,0.06);
}

.grupo-btn.selected {
  border-color: var(--accent);
  background: rgba(0,229,255,0.09);
}

.grupo-btn:disabled {
  opacity: 0.6;
  cursor: default;
}

.grupo-icon {
  width: 36px; height: 36px; border-radius: 8px;
  background: linear-gradient(135deg, var(--accent2), var(--accent3));
  display: flex; align-items: center; justify-content: center;
  font-size: 15px; font-weight: 800; color: #080c12;
  flex-shrink: 0;
}

.grupo-info { flex: 1; min-width: 0; }

.grupo-nome {
  font-size: 14px; font-weight: 600; color: var(--text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.grupo-slug {
  font-family: var(--mono); font-size: 10px; color: var(--text3); margin-top: 2px;
}

.spinner {
  width: 16px; height: 16px; border-radius: 50%;
  border: 2px solid var(--border2);
  border-top-color: var(--accent);
  animation: spin 0.7s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

.error {
  font-size: 13px; color: var(--red);
  background: rgba(239,68,68,0.08);
  border: 1px solid rgba(239,68,68,0.2);
  border-radius: 8px;
  padding: 10px 14px;
}

.cancel-btn {
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text2);
  padding: 10px;
  cursor: pointer;
  font-size: 13px;
  transition: var(--trans);
}
.cancel-btn:hover { background: var(--bg3); color: var(--text); }
</style>
