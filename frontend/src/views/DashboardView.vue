<template>
  <div class="dash-root">

    <!-- Filtros teleportados para dentro da topbar -->
    <Teleport to="#topbar-filters">
      <div class="tf-inner" @click.stop>
        <!-- Ano -->
        <div class="fi">
          <span class="fi-label">ANO</span>
          <select class="fi-select" v-model="filtros.ano" @change="carregar">
            <option v-for="a in anosDisponiveis" :key="a" :value="a">{{ a }}</option>
          </select>
        </div>

        <!-- Contas correntes -->
        <div class="fi" v-if="filtrosDisponiveis.contas_correntes.length">
          <span class="fi-label">CONTAS</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('contas')">
              {{ filtros.contas_correntes.length ? `${filtros.contas_correntes.length} sel.` : 'Todas' }}<span class="chv">▾</span>
            </button>
            <div class="fi-dropdown" v-if="dropdown === 'contas'" @click.stop>
              <label v-for="cc in filtrosDisponiveis.contas_correntes" :key="cc.codigo" class="chk-item">
                <input type="checkbox" :value="cc.codigo" v-model="filtros.contas_correntes" @change="carregar" />
                {{ cc.descricao }}
              </label>
            </div>
          </div>
        </div>

        <!-- Departamentos -->
        <div class="fi" v-if="filtrosDisponiveis.departamentos.length">
          <span class="fi-label">DEPARTAMENTO</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('dept')">
              {{ filtros.departamentos.length ? `${filtros.departamentos.length} sel.` : 'Todos' }}<span class="chv">▾</span>
            </button>
            <div class="fi-dropdown fi-dropdown--wide" v-if="dropdown === 'dept'" @click.stop>
              <label v-for="d in filtrosDisponiveis.departamentos" :key="d" class="chk-item">
                <input type="checkbox" :value="d" v-model="filtros.departamentos" @change="carregar" />
                {{ d }}
              </label>
            </div>
          </div>
        </div>

        <!-- Categorias -->
        <div class="fi" v-if="filtrosDisponiveis.categorias.length">
          <span class="fi-label">CATEGORIA</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('cat')">
              {{ filtros.categorias.length ? `${filtros.categorias.length} sel.` : 'Todas' }}<span class="chv">▾</span>
            </button>
            <div class="fi-dropdown fi-dropdown--wide" v-if="dropdown === 'cat'" @click.stop>
              <label v-for="c in filtrosDisponiveis.categorias" :key="c" class="chk-item">
                <input type="checkbox" :value="c" v-model="filtros.categorias" @change="carregar" />
                {{ c }}
              </label>
            </div>
          </div>
        </div>

        <!-- Cliente/Fornecedor -->
        <div class="fi fi--grow">
          <span class="fi-label">CLIENTE / FORNECEDOR</span>
          <input class="fi-input" v-model="filtros.cliente" placeholder="Buscar..." @input="debouncedCarregar" />
        </div>

        <!-- Empresas -->
        <div class="fi" v-if="filtrosDisponiveis.empresas.length > 1">
          <span class="fi-label">EMPRESAS</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('emp')">
              {{ filtros.empresas.length ? `${filtros.empresas.length} sel.` : 'Todas' }}<span class="chv">▾</span>
            </button>
            <div class="fi-dropdown fi-dropdown--wide fi-dropdown--right" v-if="dropdown === 'emp'" @click.stop>
              <label v-for="e in filtrosDisponiveis.empresas" :key="e.id" class="chk-item">
                <input type="checkbox" :value="e.id" v-model="filtros.empresas" @change="carregar" />
                {{ e.nome }}
              </label>
            </div>
          </div>
        </div>

        <!-- Limpar -->
        <button class="fi-clear" @click="limparFiltros" title="Limpar filtros">✕</button>
      </div>
    </Teleport>

    <!-- Loading / erro -->
    <div v-if="carregando && !dados" class="state-msg">
      <AppSpinner /> Carregando dashboard...
    </div>
    <div v-else-if="erro" class="state-msg state-msg--erro">{{ erro }}</div>

    <!-- Conteúdo -->
    <div v-else-if="dados" class="dash-content">

      <!-- Cards KPI -->
      <div class="section-title">INDICADORES</div>
      <div class="cards-row">
        <div class="kpi-card kpi-receita">
          <div class="kpi-header">
            <span class="kpi-label">RECEITA TOTAL</span>
            <span class="kpi-icon kpi-icon--green">↑</span>
          </div>
          <div class="kpi-value kpi-value--green">{{ fmt(dados.cards.receita_total) }}</div>
          <div class="kpi-sub">Ano {{ filtros.ano }}</div>
        </div>
        <div class="kpi-card kpi-despesa">
          <div class="kpi-header">
            <span class="kpi-label">DESPESA TOTAL</span>
            <span class="kpi-icon kpi-icon--red">↓</span>
          </div>
          <div class="kpi-value kpi-value--red">{{ fmt(dados.cards.despesa_total) }}</div>
          <div class="kpi-sub">Ano {{ filtros.ano }}</div>
        </div>
        <div class="kpi-card kpi-resultado">
          <div class="kpi-header">
            <span class="kpi-label">RESULTADO</span>
            <span class="kpi-icon kpi-icon--accent">◈</span>
          </div>
          <div class="kpi-value" :class="dados.cards.resultado >= 0 ? 'kpi-value--green' : 'kpi-value--red'">
            {{ fmt(dados.cards.resultado) }}
          </div>
          <div class="kpi-sub">Receita − Despesa + Saldo CC</div>
        </div>
        <div class="kpi-card kpi-saldo">
          <div class="kpi-header">
            <span class="kpi-label">SALDO CONTAS</span>
            <span class="kpi-icon kpi-icon--yellow">⬡</span>
          </div>
          <div class="kpi-value kpi-value--yellow">{{ fmt(dados.cards.saldo_contas_correntes) }}</div>
          <div class="kpi-sub">Saldo inicial cadastrado</div>
        </div>
      </div>

      <!-- Gráficos -->
      <div class="charts-row">
        <div class="chart-card">
          <div class="chart-header">
            <div>
              <div class="chart-title">Receita vs Despesa</div>
              <div class="chart-sub">Evolução mensal — {{ filtros.ano }}</div>
            </div>
            <div class="chart-legend">
              <span class="leg-dot" style="background:var(--green)"></span>Receita
              <span class="leg-dot" style="background:var(--red)"></span>Despesa
              <span class="leg-dot" style="background:var(--accent)"></span>Resultado
            </div>
          </div>
          <div class="chart-wrap"><canvas ref="canvasRecDesp"></canvas></div>
        </div>

        <div class="chart-card">
          <div class="chart-header">
            <div>
              <div class="chart-title">Resultado Acumulado</div>
              <div class="chart-sub">Progressão mensal — {{ filtros.ano }}</div>
            </div>
            <div class="chart-legend">
              <span class="leg-dot" style="background:var(--accent)"></span>Acumulado
              <span class="leg-dot" style="background:var(--green)"></span>Positivo
              <span class="leg-dot" style="background:var(--red)"></span>Negativo
            </div>
          </div>
          <div class="chart-wrap"><canvas ref="canvasAcum"></canvas></div>
        </div>
      </div>

    </div>

    <div v-else class="state-msg">Nenhum dado disponível.</div>

  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { Chart, registerables } from 'chart.js'
import ChartDataLabels from 'chartjs-plugin-datalabels'
import { useAuthStore } from '@/stores/auth'
import { fetchDashboard, type DashboardData, type FiltrosDisponiveis } from '@/api/dashboard'
import AppSpinner from '@/components/ui/AppSpinner.vue'

Chart.register(...registerables, ChartDataLabels)

const auth = useAuthStore()

// ── Estado ─────────────────────────────────────────────────────────────────
const dados      = ref<DashboardData | null>(null)
const carregando = ref(false)
const erro       = ref('')
const dropdown   = ref<string | null>(null)

const anoAtual = new Date().getFullYear()
const anosDisponiveis = computed(() => {
  const anos = []
  for (let a = anoAtual; a >= anoAtual - 5; a--) anos.push(a)
  return anos
})

const filtros = reactive({
  ano:              anoAtual,
  contas_correntes: [] as string[],
  departamentos:    [] as string[],
  categorias:       [] as string[],
  empresas:         [] as string[],
  cliente:          '',
})

const filtrosDisponiveis = reactive<FiltrosDisponiveis>({
  contas_correntes: [],
  departamentos:    [],
  categorias:       [],
  empresas:         [],
})

// ── Gráficos ───────────────────────────────────────────────────────────────
const canvasRecDesp = ref<HTMLCanvasElement | null>(null)
const canvasAcum    = ref<HTMLCanvasElement | null>(null)
let chartRecDesp: Chart | null = null
let chartAcum:    Chart | null = null

const fmt = (v: number) =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 }).format(v)

function fmtK(v: number) {
  const abs  = Math.abs(v)
  const sign = v < 0 ? '-' : ''
  if (abs >= 1_000_000) return `${sign}R$${(abs / 1_000_000).toFixed(1)}M`
  if (abs >= 1_000)     return `${sign}R$${(abs / 1_000).toFixed(0)}K`
  if (abs === 0)        return ''
  return fmt(v)
}

function chartColors() {
  const dark = document.documentElement.getAttribute('data-theme') !== 'light'
  return {
    grid:          dark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.06)',
    tick:          dark ? '#4a6070' : '#8fa3b4',
    label:         dark ? '#7a90a8' : '#4a6070',
    bg:            dark ? '#080c12' : '#fff',
    tooltip:       dark ? 'rgba(13,21,32,0.97)' : 'rgba(255,255,255,0.97)',
    tooltipBorder: dark ? 'rgba(255,255,255,0.12)' : 'rgba(0,0,0,0.12)',
    tooltipText:   dark ? '#e2eaf4' : '#0f1c2e',
  }
}

function buildChartRecDesp() {
  if (!canvasRecDesp.value || !dados.value) return
  chartRecDesp?.destroy()
  const c  = chartColors()
  const ms = dados.value.grafico_mensal

  chartRecDesp = new Chart(canvasRecDesp.value, {
    type: 'bar',
    data: {
      labels: ms.map(m => m.mes_nome),
      datasets: [
        {
          label: 'Receita', type: 'bar',
          data: ms.map(m => m.receita),
          backgroundColor: 'rgba(34,197,94,0.25)',
          borderColor: '#22c55e', borderWidth: 1.5, borderRadius: 4, order: 2,
        },
        {
          label: 'Despesa', type: 'bar',
          data: ms.map(m => m.despesa),
          backgroundColor: 'rgba(239,68,68,0.2)',
          borderColor: '#ef4444', borderWidth: 1.5, borderRadius: 4, order: 2,
        },
        {
          label: 'Resultado', type: 'line',
          data: ms.map(m => m.resultado_mes),
          borderColor: '#00e5ff', borderWidth: 2.5,
          backgroundColor: 'rgba(0,229,255,0.06)', fill: true, tension: 0.4,
          pointRadius: 4, pointBackgroundColor: '#00e5ff',
          pointBorderColor: c.bg, pointBorderWidth: 2, order: 1,
        },
      ],
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: c.tooltip, borderColor: c.tooltipBorder, borderWidth: 1,
          titleColor: c.tick, bodyColor: c.tooltipText, padding: 12,
          callbacks: { label: ctx => ` ${ctx.dataset.label}: ${fmtK(ctx.parsed.y)}` },
        },
        datalabels: {
          display: (ctx) => ctx.parsed.y !== 0,
          font: { family: 'var(--mono)', size: 9, weight: 'bold' },
          formatter: (v: number) => fmtK(v),
          // Barras: label acima; Linha: label acima do ponto
          anchor: (ctx) => ctx.dataset.type === 'line' ? 'end' : 'end',
          align:  (ctx) => ctx.dataset.type === 'line' ? 'top' : 'top',
          color:  (ctx) => {
            if (ctx.dataset.label === 'Receita')   return '#22c55e'
            if (ctx.dataset.label === 'Despesa')   return '#ef4444'
            if (ctx.dataset.label === 'Resultado') return '#00e5ff'
            return c.label
          },
          offset: 2,
        },
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: { color: c.tick, font: { family: 'var(--mono)', size: 10 } },
          border: { display: false },
        },
        y: {
          grid: { color: c.grid },
          ticks: { color: c.tick, font: { family: 'var(--mono)', size: 10 }, callback: v => fmtK(Number(v)) },
          border: { display: false },
        },
      },
      layout: { padding: { top: 24 } },
    },
  })
}

function buildChartAcum() {
  if (!canvasAcum.value || !dados.value) return
  chartAcum?.destroy()
  const c  = chartColors()
  const ac = dados.value.grafico_resultado_acumulado

  chartAcum = new Chart(canvasAcum.value, {
    type: 'bar',
    data: {
      labels: ac.map(m => m.mes_nome),
      datasets: [
        {
          label: 'Resultado mês', type: 'bar',
          data: ac.map(m => m.resultado_mes),
          backgroundColor: ac.map(m => m.resultado_mes >= 0 ? 'rgba(34,197,94,0.7)' : 'rgba(239,68,68,0.7)'),
          borderColor:     ac.map(m => m.resultado_mes >= 0 ? '#22c55e' : '#ef4444'),
          borderWidth: 1.5, borderRadius: 4, order: 2,
        },
        {
          label: 'Acumulado', type: 'line',
          data: ac.map(m => m.acumulado),
          borderColor: '#00e5ff', borderWidth: 2.5,
          backgroundColor: 'rgba(0,229,255,0.06)', fill: true, tension: 0.4,
          pointRadius: 4, pointBackgroundColor: '#00e5ff',
          pointBorderColor: c.bg, pointBorderWidth: 2, order: 1,
        },
      ],
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: c.tooltip, borderColor: c.tooltipBorder, borderWidth: 1,
          titleColor: c.tick, bodyColor: c.tooltipText, padding: 12,
          callbacks: { label: ctx => ` ${ctx.dataset.label}: ${fmtK(ctx.parsed.y)}` },
        },
        datalabels: {
          display: (ctx) => ctx.parsed.y !== 0,
          font: { family: 'var(--mono)', size: 9, weight: 'bold' },
          formatter: (v: number) => fmtK(v),
          anchor: 'end',
          align:  (ctx) => ctx.dataset.label === 'Acumulado' ? 'top' : (ctx.parsed.y >= 0 ? 'top' : 'bottom'),
          color: (ctx) => {
            if (ctx.dataset.label === 'Acumulado') return '#00e5ff'
            return ctx.parsed.y >= 0 ? '#22c55e' : '#ef4444'
          },
          offset: 2,
        },
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: { color: c.tick, font: { family: 'var(--mono)', size: 10 } },
          border: { display: false },
        },
        y: {
          grid: { color: c.grid },
          ticks: { color: c.tick, font: { family: 'var(--mono)', size: 10 }, callback: v => fmtK(Number(v)) },
          border: { display: false },
        },
      },
      layout: { padding: { top: 24, bottom: 24 } },
    },
  })
}

// Reconstrói após dados atualizados (flush:post garante DOM pronto)
watch(dados, () => {
  setTimeout(() => {
    buildChartRecDesp()
    buildChartAcum()
  }, 50)
}, { flush: 'post' })

// Reconstrói ao trocar tema
watch(
  () => document.documentElement.getAttribute('data-theme'),
  () => { if (dados.value) { buildChartRecDesp(); buildChartAcum() } }
)

// ── Carregamento ───────────────────────────────────────────────────────────
async function carregar() {
  if (!auth.user) {
    try { await auth.fetchMe() } catch { return }
  }
  // admin_global pode não ter grupo_id no token — usa o primeiro grupo disponível
  const grupoID = auth.user?.grupo_id || auth.meusGrupos[0]?.id
  if (!grupoID) {
    erro.value = 'Nenhum grupo associado. Faça login novamente.'
    return
  }

  carregando.value = true
  erro.value = ''
  dropdown.value = null

  try {
    const res = await fetchDashboard(grupoID, {
      ano:              filtros.ano,
      empresas:         filtros.empresas.length         ? filtros.empresas         : undefined,
      contas_correntes: filtros.contas_correntes.length ? filtros.contas_correntes : undefined,
      departamentos:    filtros.departamentos.length    ? filtros.departamentos    : undefined,
      categorias:       filtros.categorias.length       ? filtros.categorias       : undefined,
      cliente:          filtros.cliente || undefined,
    })

    dados.value = res

    const semFiltros =
      !filtros.empresas.length && !filtros.contas_correntes.length &&
      !filtros.departamentos.length && !filtros.categorias.length && !filtros.cliente
    if (semFiltros) Object.assign(filtrosDisponiveis, res.filtros_disponiveis)

  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
    erro.value = msg || 'Erro ao carregar dashboard'
  } finally {
    carregando.value = false
  }
}

let debounceTimer: ReturnType<typeof setTimeout>
function debouncedCarregar() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(carregar, 400)
}

function limparFiltros() {
  filtros.contas_correntes = []
  filtros.departamentos    = []
  filtros.categorias       = []
  filtros.empresas         = []
  filtros.cliente          = ''
  filtros.ano              = anoAtual
  carregar()
}

function toggleDropdown(key: string) {
  dropdown.value = dropdown.value === key ? null : key
}

onMounted(async () => {
  if (!auth.user) {
    try { await auth.fetchMe() } catch { /* guard já trata */ }
  }
  carregar()
  document.addEventListener('click', () => { dropdown.value = null })
})

onBeforeUnmount(() => {
  chartRecDesp?.destroy()
  chartAcum?.destroy()
  clearTimeout(debounceTimer)
})
</script>

<style scoped>
.dash-root { display: flex; flex-direction: column; min-height: 100%; }

/* ── Filtros na topbar (via Teleport) ──────────────────────────────────── */
.tf-inner {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  flex-wrap: nowrap;
  overflow: hidden;
  padding: 0 8px;
  flex: 1;
}

.fi { display: flex; flex-direction: column; gap: 2px; flex-shrink: 0; }
.fi--grow { flex: 1; min-width: 120px; }

.fi-label {
  font-family: var(--mono);
  font-size: 8px;
  letter-spacing: 1.5px;
  color: var(--text3);
  text-transform: uppercase;
  white-space: nowrap;
}

.fi-select,
.fi-input {
  font-family: var(--mono);
  font-size: 10px;
  padding: 4px 8px;
  border-radius: 6px;
  border: 1px solid var(--border2);
  background: var(--bg3);
  color: var(--text2);
  outline: none;
  cursor: pointer;
  transition: border-color 0.2s;
  appearance: none;
  height: 28px;
  white-space: nowrap;
}
.fi-select:focus, .fi-input:focus { border-color: rgba(0,229,255,0.4); }
.fi-input::placeholder { color: var(--text3); }

.fi-multi { position: relative; }

.fi-trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-width: 80px;
  cursor: pointer;
  text-align: left;
}
.chv { font-size: 9px; color: var(--text3); margin-left: 4px; }

.fi-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  min-width: 200px;
  max-height: 220px;
  overflow-y: auto;
  background: var(--card);
  border: 1px solid var(--border2);
  border-radius: 8px;
  box-shadow: var(--shadow);
  z-index: 200;
  padding: 4px 0;
}
.fi-dropdown--wide { min-width: 260px; }
.fi-dropdown--right { left: auto; right: 0; }

.chk-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  font-size: 11px;
  color: var(--text2);
  cursor: pointer;
  transition: background 0.15s;
}
.chk-item:hover { background: var(--bg3); }
.chk-item input { accent-color: var(--accent); cursor: pointer; flex-shrink: 0; }

.fi-clear {
  padding: 4px 8px;
  height: 28px;
  border-radius: 6px;
  border: 1px solid var(--border2);
  background: var(--bg3);
  color: var(--text3);
  cursor: pointer;
  font-size: 10px;
  align-self: flex-end;
  flex-shrink: 0;
  transition: var(--trans);
}
.fi-clear:hover { border-color: var(--red); color: var(--red); }

/* ── Estados ───────────────────────────────────────────────────────────── */
.state-msg {
  display: flex; align-items: center; gap: 10px;
  justify-content: center; padding: 60px 24px;
  font-family: var(--mono); font-size: 12px; color: var(--text3);
}
.state-msg--erro { color: var(--red); }

/* ── Conteúdo ──────────────────────────────────────────────────────────── */
.dash-content { display: flex; flex-direction: column; gap: 20px; }

.section-title {
  display: flex; align-items: center; gap: 8px;
  font-family: var(--mono); font-size: 10px;
  color: var(--text3); letter-spacing: 2px; text-transform: uppercase;
  margin-bottom: 4px;
}
.section-title::before {
  content: ''; width: 18px; height: 1.5px;
  background: var(--accent); opacity: 0.7;
}

/* ── Cards KPI ─────────────────────────────────────────────────────────── */
.cards-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
}
@media (max-width: 1100px) { .cards-row { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 560px)  { .cards-row { grid-template-columns: 1fr; } }

.kpi-card {
  background: var(--card); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 18px 20px;
  position: relative; overflow: hidden; transition: var(--trans);
}
.kpi-card::before {
  content: ''; position: absolute; top: 0; left: 0; right: 0; height: 2px;
}
.kpi-receita::before   { background: var(--green); }
.kpi-despesa::before   { background: var(--red); }
.kpi-resultado::before { background: var(--accent); }
.kpi-saldo::before     { background: var(--yellow); }
.kpi-card:hover { transform: translateY(-2px); box-shadow: var(--shadow); }

.kpi-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 12px; }
.kpi-label  { font-family: var(--mono); font-size: 10px; font-weight: 500; color: var(--text3); letter-spacing: 1.5px; text-transform: uppercase; }

.kpi-icon { width: 32px; height: 32px; border-radius: 8px; display: flex; align-items: center; justify-content: center; font-size: 15px; }
.kpi-icon--green  { background: rgba(34,197,94,0.1);  color: var(--green); }
.kpi-icon--red    { background: rgba(239,68,68,0.1);  color: var(--red); }
.kpi-icon--accent { background: rgba(0,229,255,0.1);  color: var(--accent); }
.kpi-icon--yellow { background: rgba(245,158,11,0.1); color: var(--yellow); }

.kpi-value { font-size: 22px; font-weight: 800; line-height: 1; margin-bottom: 6px; letter-spacing: -0.5px; }
.kpi-value--green  { color: var(--green); }
.kpi-value--red    { color: var(--red); }
.kpi-value--yellow { color: var(--yellow); }
.kpi-sub { font-family: var(--mono); font-size: 10px; color: var(--text3); }

/* ── Gráficos ──────────────────────────────────────────────────────────── */
.charts-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
}
@media (max-width: 1100px) { .charts-row { grid-template-columns: 1fr; } }

.chart-card {
  background: var(--card); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 20px 22px;
}
.chart-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  margin-bottom: 16px; flex-wrap: wrap; gap: 8px;
}
.chart-title { font-size: 14px; font-weight: 700; }
.chart-sub   { font-family: var(--mono); font-size: 10px; color: var(--text3); margin-top: 2px; }

.chart-legend {
  display: flex; align-items: center; gap: 10px;
  font-family: var(--mono); font-size: 10px; color: var(--text2); flex-wrap: wrap;
}
.leg-dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 3px; }

.chart-wrap { position: relative; height: 300px; }
</style>
