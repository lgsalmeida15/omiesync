<template>
  <div class="dash-root">

    <!-- Filtros teleportados para dentro da topbar.
         `defer` é obrigatório: #topbar-filters é renderizado por AppTopbar, um irmão
         na mesma árvore. O Vue monta os filhos antes de inserir o elemento pai no
         documento, então sem `defer` o querySelector do target retorna null. -->
    <Teleport defer to="#topbar-filters">
      <div class="tf-inner" @click.stop>
        <!-- Ano -->
        <div class="fi">
          <span class="fi-label">ANO</span>
          <select class="fi-select" v-model="filtros.ano" @change="onAnoChange">
            <option v-for="a in anosDisponiveis" :key="a" :value="a">{{ a }}</option>
          </select>
        </div>

        <!-- Mês: só no Fluxo de Caixa; as outras abas são anuais -->
        <div class="fi" v-if="aba === 'fluxo'">
          <span class="fi-label">MÊS</span>
          <select class="fi-select" v-model.number="mesSelecionado">
            <option v-for="m in mesesDoAno" :key="m.v" :value="m.v">{{ m.n }}</option>
          </select>
        </div>

        <!-- Empresas: só aparece quando há mais de uma no grupo -->
        <div class="fi" v-if="filtrosDisponiveis.empresas.length > 1">
          <span class="fi-label">EMPRESA</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('emp', $event)">
              {{ rotuloLista('empresas', filtrosDisponiveis.empresas.length) }}<span class="chv">▾</span>
            </button>
          </div>
        </div>

        <!-- Contas correntes: apenas das empresas selecionadas -->
        <div class="fi" v-if="filtrosDisponiveis.contas_correntes.length">
          <span class="fi-label">CONTAS</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('contas', $event)">
              {{ rotuloLista('contas_correntes', filtrosDisponiveis.contas_correntes.length) }}<span class="chv">▾</span>
            </button>
          </div>
        </div>

        <!-- Departamentos -->
        <div class="fi" v-if="filtrosDisponiveis.departamentos.length">
          <span class="fi-label">DEPARTAMENTO</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('dept', $event)">
              {{ rotuloLista('departamentos', filtrosDisponiveis.departamentos.length) }}<span class="chv">▾</span>
            </button>
          </div>
        </div>

        <!-- Categorias -->
        <div class="fi" v-if="filtrosDisponiveis.categorias.length">
          <span class="fi-label">CATEGORIA</span>
          <div class="fi-multi">
            <button class="fi-select fi-trigger" @click.stop="toggleDropdown('cat', $event)">
              {{ rotuloCategorias }}<span class="chv">▾</span>
            </button>
          </div>
        </div>

        <!-- Cliente/Fornecedor com autocomplete -->
        <div class="fi fi--grow">
          <span class="fi-label">CLIENTE / FORNECEDOR</span>
          <input
            class="fi-input"
            v-model="filtros.cliente"
            placeholder="Buscar..."
            @input="onClienteInput"
            @focus="onClienteFocus($event)"
            @blur="onClienteBlur"
            autocomplete="off"
          />
        </div>

        <!-- Limpar -->
        <button class="fi-clear" @click="limparFiltros" title="Limpar filtros">✕</button>
      </div>
    </Teleport>

    <!-- Dropdowns fixos — renderizados no body via Teleport para escapar de overflow:hidden -->
    <Teleport to="body">
      <!-- Empresas -->
      <div v-if="dropdown === 'emp'" class="fd-fixed" :style="fdStyle" @click.stop>
        <div class="fd-acoes">
          <button class="fd-acao" @click="selecionarTodos('empresas')">Marcar todas</button>
        </div>
        <label v-for="e in filtrosDisponiveis.empresas" :key="e.id" class="chk-item">
          <input type="checkbox" :checked="marcado('empresas', e.id)"
                 @change="alternar('empresas', e.id, filtrosDisponiveis.empresas.map(x => x.id))" />
          {{ e.nome }}
        </label>
      </div>
      <!-- Contas correntes -->
      <div v-if="dropdown === 'contas'" class="fd-fixed" :style="fdStyle" @click.stop>
        <div class="fd-acoes">
          <button class="fd-acao" @click="selecionarTodos('contas_correntes')">Marcar todas</button>
        </div>
        <label v-for="cc in filtrosDisponiveis.contas_correntes" :key="cc.codigo" class="chk-item">
          <input type="checkbox" :checked="marcado('contas_correntes', cc.codigo)"
                 @change="alternar('contas_correntes', cc.codigo, filtrosDisponiveis.contas_correntes.map(x => x.codigo))" />
          {{ cc.descricao }}
        </label>
      </div>
      <!-- Departamentos -->
      <div v-if="dropdown === 'dept'" class="fd-fixed" :style="fdStyle" @click.stop>
        <div class="fd-acoes">
          <button class="fd-acao" @click="selecionarTodos('departamentos')">Marcar todos</button>
        </div>
        <label v-for="d in filtrosDisponiveis.departamentos" :key="d" class="chk-item">
          <input type="checkbox" :checked="marcado('departamentos', d)"
                 @change="alternar('departamentos', d, filtrosDisponiveis.departamentos)" />
          {{ d }}
        </label>
      </div>
      <!-- Categorias: lista de exclusão, com "Transferência" oculta por padrão -->
      <div v-if="dropdown === 'cat'" class="fd-fixed" :style="fdStyle" @click.stop>
        <div class="fd-acoes">
          <button class="fd-acao" @click="todasCategorias">Marcar todas</button>
          <button class="fd-acao" @click="padraoCategorias">Padrão</button>
        </div>
        <label v-for="c in filtrosDisponiveis.categorias" :key="c" class="chk-item">
          <input type="checkbox" :checked="categoriaMarcada(c)" @change="alternarCategoria(c)" />
          {{ c }}
        </label>
      </div>
      <!-- Autocomplete cliente -->
      <div v-if="dropdown === 'cli' && clienteSugestoes.length" class="fd-fixed" :style="fdStyle" @click.stop>
        <div
          v-for="s in clienteSugestoes"
          :key="s"
          class="chk-item chk-item--click"
          @mousedown.prevent="selecionarCliente(s)"
        >
          {{ s }}
        </div>
      </div>
    </Teleport>

    <!-- Abas -->
    <div class="dash-tabs">
      <button v-for="t in abas" :key="t.id"
              :class="['dash-tab', { active: aba === t.id }]"
              @click="aba = t.id">{{ t.label }}</button>
    </div>

    <!-- Loading / erro -->
    <div v-if="carregando && !dados" class="state-msg">
      <AppSpinner /> Carregando dashboard...
    </div>
    <div v-else-if="erro" class="state-msg state-msg--erro">{{ erro }}</div>

    <!-- Aba Resultado: componente carregado sob demanda para não pesar no bundle
         inicial do dashboard, que já é o maior da aplicação. -->
    <ResultadoPivot v-else-if="aba === 'resultado'" :grupo-id="grupoIDAtivo" :filtros="filtrosAtivos" />
    <FluxoCaixa v-else-if="aba === 'fluxo'" :grupo-id="grupoIDAtivo" :filtros="filtrosAtivos" :mes="mesSelecionado" />

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
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch, nextTick, defineAsyncComponent } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Chart, registerables } from 'chart.js'
import ChartDataLabels from 'chartjs-plugin-datalabels'
import { useAuthStore } from '@/stores/auth'
import { fetchDashboard, fetchFiltros, type DashboardData, type FiltrosDisponiveis } from '@/api/dashboard'
import AppSpinner from '@/components/ui/AppSpinner.vue'
import { fmtMoeda, fmtCompacto } from '@/utils/formato'

Chart.register(...registerables, ChartDataLabels)

const auth   = useAuthStore()
const router = useRouter()
const route  = useRoute()

const ResultadoPivot = defineAsyncComponent(
  () => import('@/components/dashboard/ResultadoPivot.vue')
)
const FluxoCaixa = defineAsyncComponent(
  () => import('@/components/dashboard/FluxoCaixa.vue')
)

const abas = [
  { id: 'geral',     label: 'VISÃO GERAL' },
  { id: 'resultado', label: 'RESULTADO'   },
  { id: 'fluxo',     label: 'FLUXO DE CAIXA' },
] as const
const aba = ref<'geral' | 'resultado' | 'fluxo'>('geral')

/**
 * Mês só existe no Fluxo de Caixa. Visão Geral e Resultado são anuais — filtrar
 * por mês ali esvaziaria onze das doze colunas, então o seletor some fora da aba.
 */
const mesSelecionado = ref(new Date().getMonth() + 1)
const mesesDoAno = [
  { v: 1,  n: 'Janeiro' },   { v: 2,  n: 'Fevereiro' }, { v: 3,  n: 'Março' },
  { v: 4,  n: 'Abril' },     { v: 5,  n: 'Maio' },      { v: 6,  n: 'Junho' },
  { v: 7,  n: 'Julho' },     { v: 8,  n: 'Agosto' },    { v: 9,  n: 'Setembro' },
  { v: 10, n: 'Outubro' },   { v: 11, n: 'Novembro' },  { v: 12, n: 'Dezembro' },
]

// Grupo resolvido em carregar(); a aba Resultado consulta o mesmo grupo.
const grupoIDAtivo = ref('')

// Espelha os filtros da topbar no formato que a API espera, para que as duas abas
// mostrem sempre o mesmo recorte.
const filtrosAtivos = computed(() => ({
  ano:              filtros.ano,
  empresas:         filtros.empresas.length         ? filtros.empresas         : undefined,
  contas_correntes: filtros.contas_correntes.length ? filtros.contas_correntes : undefined,
  departamentos:    filtros.departamentos.length    ? filtros.departamentos    : undefined,
  categorias:       filtros.categorias.length       ? filtros.categorias       : undefined,
  cliente:          filtros.cliente || undefined,
  categorias_excluir: filtros.categorias_excluir.length ? filtros.categorias_excluir : undefined,
}))

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

/**
 * Categoria oculta por padrão. É lista de EXCLUSÃO: se fosse de inclusão, uma
 * categoria nova no Omie ficaria fora dos números em silêncio até alguém marcá-la.
 */
const CATEGORIA_OCULTA_PADRAO = 'Transferência'

const filtros = reactive({
  ano:               anoAtual,
  contas_correntes:  [] as string[],
  departamentos:     [] as string[],
  categorias:        [] as string[],
  empresas:          [] as string[],
  cliente:           '',
  categorias_excluir: [CATEGORIA_OCULTA_PADRAO] as string[],
})

/**
 * Ordem da cascata. Cada filtro é restringido apenas pelos anteriores, e mudar um
 * limpa os seguintes — senão sobra uma conta selecionada que não pertence mais à
 * empresa escolhida e o resultado vem vazio sem explicação.
 */
const ordemCascata = ['empresas', 'contas_correntes', 'departamentos', 'cliente'] as const

function limparAbaixo(alterado: string) {
  const i = ordemCascata.indexOf(alterado as typeof ordemCascata[number])
  if (i < 0) return
  for (const campo of ordemCascata.slice(i + 1)) {
    if (campo === 'cliente') filtros.cliente = ''
    else filtros[campo] = []
  }
}

type CampoLista = 'empresas' | 'contas_correntes' | 'departamentos'

/**
 * Seleção vazia significa "todas". As caixas então aparecem MARCADAS nesse estado —
 * é o que de fato está sendo exibido, e deixá-las desmarcadas mostrando tudo seria
 * enganoso.
 */
function marcado(campo: CampoLista, valor: string): boolean {
  return filtros[campo].length === 0 || filtros[campo].includes(valor)
}

function alternar(campo: CampoLista, valor: string, todos: string[]) {
  const atual = filtros[campo]
  if (atual.length === 0) {
    // Saindo do "todas": materializa a lista completa menos o desmarcado.
    filtros[campo] = todos.filter(v => v !== valor)
  } else if (atual.includes(valor)) {
    filtros[campo] = atual.filter(v => v !== valor)
  } else {
    const proximo = [...atual, valor]
    // Marcar tudo volta a "todas" (array vazio), para que item novo entre sozinho.
    filtros[campo] = proximo.length === todos.length ? [] : proximo
  }
  limparAbaixo(campo)
  carregar()
}

/**
 * Ano é o topo da cascata: um departamento ou categoria que existia em 2026 pode não
 * existir em 2025, e a seleção obsoleta produziria tela vazia sem explicação. Limpar
 * é preferível — o custo é ter de remarcar ao comparar o mesmo recorte entre anos.
 * A exclusão de categorias é preservada, porque é preferência de exibição.
 */
function onAnoChange() {
  filtros.empresas         = []
  filtros.contas_correntes = []
  filtros.departamentos    = []
  filtros.categorias       = []
  filtros.cliente          = ''
  carregar()
}

/** Marcar todos e limpar têm o mesmo efeito: array vazio = sem filtro = todas. */
function selecionarTodos(campo: CampoLista) {
  filtros[campo] = []
  limparAbaixo(campo)
  carregar()
}

// ── Categorias: lista de exclusão, não de inclusão ─────────────────────────
function categoriaMarcada(valor: string): boolean {
  return !filtros.categorias_excluir.includes(valor)
}

function alternarCategoria(valor: string) {
  filtros.categorias_excluir = filtros.categorias_excluir.includes(valor)
    ? filtros.categorias_excluir.filter(v => v !== valor)
    : [...filtros.categorias_excluir, valor]
  carregar()
}

function todasCategorias() {
  filtros.categorias_excluir = []
  carregar()
}

function padraoCategorias() {
  filtros.categorias_excluir = [CATEGORIA_OCULTA_PADRAO]
  carregar()
}

/** Rótulo do gatilho: informa o estado sem obrigar a abrir o dropdown. */
function rotuloLista(campo: CampoLista, total: number): string {
  const n = filtros[campo].length
  return n === 0 ? 'Todas' : `${n} de ${total}`
}

const rotuloCategorias = computed(() => {
  const ex = filtros.categorias_excluir.length
  if (ex === 0) return 'Todas'
  if (ex === 1 && filtros.categorias_excluir[0] === CATEGORIA_OCULTA_PADRAO) {
    return 'Todas exc. Transf.'
  }
  return `${ex} oculta${ex > 1 ? 's' : ''}`
})

const filtrosDisponiveis = reactive<FiltrosDisponiveis>({
  contas_correntes: [],
  departamentos:    [],
  categorias:       [],
  clientes:         [],
  empresas:         [],
})

// Posição do dropdown fixo (calculada via getBoundingClientRect)
const fdPos = reactive({ top: 0, left: 0, right: 0, useRight: false })
const fdStyle = computed(() => fdPos.useRight
  ? { top: `${fdPos.top}px`, right: `${fdPos.right}px` }
  : { top: `${fdPos.top}px`, left: `${fdPos.left}px` }
)

// Autocomplete cliente
const clienteSugestoes = computed(() => {
  const q = filtros.cliente.trim().toLowerCase()
  if (!q || q.length < 2) return []
  return filtrosDisponiveis.clientes
    .filter(c => c.toLowerCase().includes(q))
    .slice(0, 20)
})

// ── Gráficos ───────────────────────────────────────────────────────────────
const canvasRecDesp = ref<HTMLCanvasElement | null>(null)
const canvasAcum    = ref<HTMLCanvasElement | null>(null)
let chartRecDesp: Chart | null = null
let chartAcum:    Chart | null = null

// Formatação contábil: negativo entre parênteses. Ver src/utils/formato.ts.
const fmt  = fmtMoeda
const fmtK = fmtCompacto

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
          display: (ctx) => (ctx.parsed?.y ?? 0) !== 0,
          font: { family: 'var(--mono)', size: 9, weight: 'bold' },
          formatter: (v: number) => fmtK(v),
          anchor: 'end',
          align:  'top',
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
          display: (ctx) => (ctx.parsed?.y ?? 0) !== 0,
          font: { family: 'var(--mono)', size: 9, weight: 'bold' },
          formatter: (v: number) => fmtK(v),
          anchor: 'end',
          align:  (ctx) => ctx.dataset.label === 'Acumulado' ? 'top' : ((ctx.parsed?.y ?? 0) >= 0 ? 'top' : 'bottom'),
          color: (ctx) => {
            if (ctx.dataset.label === 'Acumulado') return '#00e5ff'
            return (ctx.parsed?.y ?? 0) >= 0 ? '#22c55e' : '#ef4444'
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

// Dispara imediatamente com o valor atual de auth.user (immediate: true).
// Se user já existe ao montar → chama carregar() de imediato.
// Se user ainda é null (edge case de timing) → aguarda até ser definido.
watch(() => auth.user, (user) => {
  if (user && !dados.value && !carregando.value) carregar()
}, { immediate: true })

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

// Reconstrói ao voltar da aba Resultado.
//
// As abas usam cadeia v-if, então ir para Resultado DESMONTA o .dash-content e
// destrói os <canvas>. Ao voltar, novos canvas são criados, mas as instâncias do
// Chart.js ainda apontam para os antigos — e o gráfico aparece vazio.
//
// Antes só o watch de `dados` reconstruía, e trocar de aba não altera os dados.
// Era por isso que recarregar a página ou limpar os filtros "consertava": ambos
// mudam `dados`.
watch(aba, novaAba => {
  if (novaAba !== 'geral' || !dados.value) return
  nextTick(() => {
    buildChartRecDesp()
    buildChartAcum()
  })
})

// ── Carregamento ───────────────────────────────────────────────────────────
async function carregar() {
  if (!auth.user) {
    try { await auth.fetchMe() } catch {
      erro.value = 'Sessão inválida. Faça login novamente.'
      return
    }
  }
  // Só admin_global cai no primeiro grupo da lista — ele não tem grupo próprio e a
  // escolha é arbitrária por natureza. Para quem tem vários grupos, adivinhar seria
  // exibir dados de um cliente que a pessoa não selecionou: manda escolher.
  let grupoID = auth.user?.grupo_id
  if (!grupoID) {
    if (auth.isAdminGlobal) {
      grupoID = auth.meusGrupos[0]?.id
    } else if (auth.meusGrupos.length > 1) {
      router.push({ name: 'SelectGrupo', query: { redirect: route.fullPath } })
      return
    } else {
      grupoID = auth.meusGrupos[0]?.id
    }
  }
  if (!grupoID) {
    erro.value = 'Nenhum grupo associado. Faça login novamente.'
    return
  }
  grupoIDAtivo.value = grupoID

  carregando.value = true
  erro.value = ''

  try {
    // Opções e dados em paralelo. As opções vêm do endpoint próprio, em cascata:
    // antes as cinco consultas DISTINCT rodavam junto com a agregação pesada a cada
    // mudança de filtro.
    const [res, opcoes] = await Promise.all([
      fetchDashboard(grupoID, filtrosAtivos.value),
      fetchFiltros(grupoID, filtrosAtivos.value),
    ])

    dados.value = res
    Object.assign(filtrosDisponiveis, opcoes)

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
  // Volta ao padrão, não a "tudo visível": Transferência segue oculta.
  filtros.categorias_excluir = [CATEGORIA_OCULTA_PADRAO]
  carregar()
}

function toggleDropdown(key: string, event: MouseEvent) {
  if (dropdown.value === key) { dropdown.value = null; return }
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const spaceRight = window.innerWidth - rect.right
  fdPos.top      = rect.bottom + 4
  fdPos.left     = rect.left
  fdPos.right    = spaceRight < 270 ? window.innerWidth - rect.right : 0
  fdPos.useRight = spaceRight < 270
  dropdown.value = key
}

function closeDropdown() { dropdown.value = null }

function onClienteInput() {
  debouncedCarregar()
  // Mostra autocomplete se tiver 2+ chars
  if (filtros.cliente.trim().length >= 2) {
    dropdown.value = 'cli'
  } else {
    dropdown.value = null
  }
}

function onClienteFocus(event: FocusEvent) {
  if (filtros.cliente.trim().length >= 2) {
    const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
    fdPos.top      = rect.bottom + 4
    fdPos.left     = rect.left
    fdPos.useRight = false
    dropdown.value = 'cli'
  }
}

function onClienteBlur() {
  // Delay para permitir o mousedown na sugestão disparar antes do blur fechar
  setTimeout(() => { if (dropdown.value === 'cli') dropdown.value = null }, 150)
}

function selecionarCliente(nome: string) {
  filtros.cliente = nome
  dropdown.value  = null
  carregar()
}

onMounted(() => {
  document.addEventListener('click', closeDropdown)
})

onBeforeUnmount(() => {
  chartRecDesp?.destroy()
  chartAcum?.destroy()
  clearTimeout(debounceTimer)
  document.removeEventListener('click', closeDropdown)
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
  overflow: hidden;   /* clip horizontal — dropdowns usam position:fixed e escapam disso */
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

/* Dropdown fixo — Teleportado para body, escapa de qualquer overflow:hidden */
.fd-fixed {
  position: fixed;
  min-width: 220px;
  max-width: 320px;
  max-height: 260px;
  overflow-y: auto;
  background: var(--card);
  border: 1px solid var(--border2);
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.4);
  z-index: 9999;
  padding: 4px 0;
}

.fd-acoes {
  display: flex; gap: 6px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--border2);
  position: sticky; top: 0;
  background: var(--card);
}
.fd-acao {
  flex: 1;
  background: var(--bg3); color: var(--text2);
  border: 1px solid var(--border2); border-radius: 6px;
  padding: 4px 8px;
  font-family: var(--mono); font-size: 9px; letter-spacing: 0.5px;
  cursor: pointer; transition: var(--trans); white-space: nowrap;
}
.fd-acao:hover { border-color: var(--accent); color: var(--accent); }

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
.chk-item--click { user-select: none; }
.chk-item--click:hover { background: var(--bg3); color: var(--accent); }

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

.dash-tabs {
  display: flex; gap: 4px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 18px;
}
.dash-tab {
  background: none; border: none;
  border-bottom: 2px solid transparent;
  padding: 9px 16px; margin-bottom: -1px;
  font-family: var(--mono); font-size: 10px; letter-spacing: 1.2px;
  color: var(--text3); cursor: pointer; transition: var(--trans);
}
.dash-tab:hover { color: var(--text2); }
.dash-tab.active { color: var(--accent); border-bottom-color: var(--accent); }

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
