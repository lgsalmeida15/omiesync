<template>
  <div class="fc">
    <div v-if="carregando" class="fc-state">
      <AppSpinner /> Carregando {{ rotuloCarregando }}...
    </div>
    <div v-else-if="erro" class="fc-state fc-state--erro">{{ erro }}</div>

    <template v-else-if="dados">
      <div class="fc-grid">
        <!-- Calendário -->
        <section class="fc-card fc-card--cal">
          <div class="fc-card-head">
            <div>
              <div class="fc-card-title">{{ tituloCalendario }}</div>
              <div class="fc-card-sub">{{ nomeMes[dados.mes - 1] }} de {{ dados.ano }}</div>
            </div>
            <button v-if="diaSelecionado" class="fc-btn" @click="diaSelecionado = null">
              Ver mês inteiro
            </button>
          </div>

          <div class="cal-semana">
            <span v-for="d in ['D','S','T','Q','Q','S','S']" :key="d">{{ d }}</span>
          </div>
          <div class="cal-grid">
            <div v-for="v in vaziosAntes" :key="`v${v}`" class="cal-vazio" />
            <button
              v-for="d in diasDoMes" :key="d.dia"
              :class="['cal-dia', {
                'cal-dia--sel': diaSelecionado === d.dia,
                'cal-dia--hoje': d.dia === diaDeHoje,
                'cal-dia--vazio': !d.temLancamento,
              }]"
              :disabled="!d.temLancamento"
              @click="diaSelecionado = diaSelecionado === d.dia ? null : d.dia"
            >
              <span class="cal-num">{{ d.dia }}</span>
              <span v-if="d.entradas" class="cal-val cal-val--in">{{ fmtCompacto(d.entradas) }}</span>
              <span v-if="d.saidas" class="cal-val cal-val--out">{{ fmtCompacto(-d.saidas) }}</span>
              <span v-if="d.temPrevisto" class="cal-prev" title="Contém valores previstos" />
            </button>
          </div>
        </section>

        <!-- Lateral: resumo + próximos vencimentos -->
        <div class="fc-lateral">
          <section class="fc-card">
            <div class="fc-card-title">{{ diaSelecionado ? `Resumo — dia ${diaSelecionado}` : 'Resumo do mês' }}</div>
            <div class="fc-card-sub">{{ diaSelecionado ? 'Seleção atual' : 'Acumulado do período' }}</div>

            <template v-if="tipo !== 'despesa'">
              <div class="res-linha">
                <span class="res-rot">Recebido</span>
                <span class="res-val res-val--in">{{ fmtMoeda(resumo.recebido) }}</span>
              </div>
              <div class="res-linha">
                <span class="res-rot">A receber<span class="res-prev">previsto</span></span>
                <span class="res-val res-val--in">{{ fmtMoeda(resumo.a_receber) }}</span>
              </div>
            </template>
            <template v-if="tipo !== 'receita'">
              <div class="res-linha">
                <span class="res-rot">Pago</span>
                <span class="res-val res-val--out">{{ fmtMoeda(resumo.pago) }}</span>
              </div>
              <div class="res-linha">
                <span class="res-rot">A pagar<span class="res-prev">previsto</span></span>
                <span class="res-val res-val--out">{{ fmtMoeda(resumo.a_pagar) }}</span>
              </div>
            </template>
            <div class="res-linha res-linha--total">
              <span class="res-rot">{{ umLadoSo ? 'Total' : 'Resultado' }}</span>
              <span class="res-val" :class="resumo.resultado < 0 ? 'res-val--out' : 'res-val--in'">
                {{ fmtMoeda(resumo.resultado) }}
              </span>
            </div>
          </section>

          <section class="fc-card fc-card--venc">
            <div class="fc-card-title">Próximos vencimentos</div>
            <div class="fc-card-sub">A partir de hoje, em qualquer mês</div>
            <p v-if="!dados.proximos_vencimentos.length" class="fc-vazio">Nada previsto adiante.</p>
            <div v-else class="venc-lista">
              <div v-for="(t, i) in dados.proximos_vencimentos" :key="i" class="venc-item">
                <div class="venc-data">{{ t.data.slice(0, 5) }}</div>
                <div class="venc-desc">
                  <span class="venc-nome">{{ t.descricao }}</span>
                  <span class="venc-cat">{{ t.categoria }}</span>
                </div>
                <div class="venc-val" :class="t.tipo === 'receita' ? 'res-val--in' : 'res-val--out'">
                  {{ fmtMoeda(t.valor) }}
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>

      <!-- Listagem: realizado e pendente, a coluna Status distingue -->
      <section class="fc-card">
        <div class="fc-card-head">
          <div>
            <div class="fc-card-title">TRANSAÇÕES</div>
            <div class="fc-card-sub">
              {{ diaSelecionado ? `Dia ${diaSelecionado}` : 'Mês inteiro' }} — efetuadas e pendentes
            </div>
          </div>
          <div class="fc-filtros">
            <input v-model="busca" class="fc-input" placeholder="Buscar descrição ou categoria..." />
            <select v-if="!umLadoSo" v-model="filtroTipo" class="fc-select">
              <option value="">Todos os tipos</option>
              <option value="receita">Recebimentos</option>
              <option value="despesa">Pagamentos</option>
            </select>
            <select v-model="filtroSituacao" class="fc-select">
              <option value="">Todas as situações</option>
              <option value="efetuada">Efetuadas</option>
              <option value="pendente">Pendentes</option>
            </select>
          </div>
        </div>

        <p v-if="!listagem.length" class="fc-vazio">Nenhuma transação no período.</p>
        <div v-else class="fc-scroll">
          <table class="fc-table">
            <thead>
              <tr>
                <th>DATA</th><th>DESCRIÇÃO</th><th>TIPO</th><th>CATEGORIA</th>
                <th class="ta-r">VALOR</th><th>STATUS</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(t, i) in listagem" :key="i">
                <td class="mono">{{ t.data }}</td>
                <td>{{ t.descricao }}</td>
                <td>
                  <span :class="['pill', t.tipo === 'receita' ? 'pill--in' : 'pill--out']">
                    {{ t.tipo === 'receita' ? 'Recebimento' : 'Pagamento' }}
                  </span>
                </td>
                <td>{{ t.categoria }}</td>
                <td class="ta-r mono" :class="t.tipo === 'receita' ? 'res-val--in' : 'res-val--out'">
                  {{ fmtMoeda(t.valor) }}
                </td>
                <td>
                  <span :class="['pill', pillStatus(t)]">{{ t.status }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { fetchFluxoCaixa, type FluxoCaixaData, type FluxoResumo, type FluxoTransacao } from '@/api/fluxocaixa'
import type { DashboardParams } from '@/api/dashboard'
import AppSpinner from '@/components/ui/AppSpinner.vue'
import { fmtMoeda, fmtCompacto } from '@/utils/formato'
import { filtrarTransacoes, classeStatus } from '@/utils/fluxo'

const props = withDefaults(defineProps<{
  grupoId: string
  filtros: DashboardParams
  mes: number
  /** Restringe a visão a um lado: abas Contas a Receber e Contas a Pagar. */
  tipo?: 'todos' | 'receita' | 'despesa'
}>(), { tipo: 'todos' })

const umLadoSo   = computed(() => props.tipo !== 'todos')
const soReceitas = computed(() => props.tipo === 'receita')

const tituloCalendario = computed(() => ({
  todos:   'CALENDÁRIO FINANCEIRO',
  receita: 'CALENDÁRIO DE RECEBIMENTOS',
  despesa: 'CALENDÁRIO DE PAGAMENTOS',
}[props.tipo]))

const rotuloCarregando = computed(() => ({
  todos:   'fluxo de caixa',
  receita: 'recebimentos',
  despesa: 'pagamentos',
}[props.tipo]))

const nomeMes = ['Janeiro','Fevereiro','Março','Abril','Maio','Junho',
                 'Julho','Agosto','Setembro','Outubro','Novembro','Dezembro']

const bruto          = ref<FluxoCaixaData | null>(null)
const carregando     = ref(false)
const erro           = ref('')
const diaSelecionado = ref<number | null>(null)
const busca          = ref('')
const filtroTipo     = ref('')
const filtroSituacao = ref('')

/**
 * Na visão de recebimentos o recorte é aplicado uma vez, aqui: transações,
 * resumo e próximos vencimentos passam a enxergar só receitas. Filtrar em cada
 * consumidor separadamente deixaria algum deles para trás — o resumo já vem
 * totalizado do servidor e precisa ser recalculado junto.
 */
const dados = computed<FluxoCaixaData | null>(() => {
  if (!bruto.value) return null
  if (!umLadoSo.value) return bruto.value

  const doLado = (t: FluxoTransacao) => t.tipo === props.tipo
  const transacoes = bruto.value.transacoes.filter(doLado)
  const resumo: FluxoResumo = {
    recebido: 0, a_receber: 0, pago: 0, a_pagar: 0, resultado: 0,
  }
  for (const t of transacoes) {
    if (soReceitas.value) t.realizado ? (resumo.recebido += t.valor) : (resumo.a_receber += t.valor)
    else                  t.realizado ? (resumo.pago += t.valor)     : (resumo.a_pagar += t.valor)
  }
  // Aqui o total é a soma do lado, não uma diferença: tudo na tela é do mesmo
  // sinal, então subtrair não teria contra o quê.
  resumo.resultado = soReceitas.value
    ? resumo.recebido + resumo.a_receber
    : resumo.pago + resumo.a_pagar

  return {
    ...bruto.value,
    transacoes,
    resumo,
    proximos_vencimentos: bruto.value.proximos_vencimentos.filter(doLado),
  }
})

// Emite as transações já recortadas para quem compõe a tela (os gráficos da aba
// Contas a Receber), evitando uma segunda chamada ao mesmo endpoint.
const emit = defineEmits<{ (e: 'dados', d: FluxoCaixaData | null): void }>()
watch(dados, d => emit('dados', d), { immediate: true })

// ── Calendário ─────────────────────────────────────────────────────────────
const diasNoMes = computed(() =>
  dados.value ? new Date(dados.value.ano, dados.value.mes, 0).getDate() : 0
)

/** Quantas células vazias antes do dia 1, para alinhar na coluna do dia da semana. */
const vaziosAntes = computed(() =>
  dados.value ? new Date(dados.value.ano, dados.value.mes - 1, 1).getDay() : 0
)

const diaDeHoje = computed(() => {
  const h = new Date()
  if (!dados.value) return -1
  return h.getFullYear() === dados.value.ano && h.getMonth() + 1 === dados.value.mes
    ? h.getDate() : -1
})

const diasDoMes = computed(() => {
  const base = Array.from({ length: diasNoMes.value }, (_, i) => ({
    dia: i + 1, entradas: 0, saidas: 0, temPrevisto: false, temLancamento: false,
  }))
  for (const t of dados.value?.transacoes ?? []) {
    const d = base[t.dia - 1]
    if (!d) continue
    if (t.tipo === 'receita') d.entradas += t.valor
    else d.saidas += t.valor
    if (!t.realizado) d.temPrevisto = true
    d.temLancamento = true
  }
  return base
})

// ── Resumo: recalculado no cliente quando há dia selecionado ────────────────
// Sem dia selecionado usa o total que veio do servidor, para não divergir por
// arredondamento do que o banco reportou.
const resumo = computed<FluxoResumo>(() => {
  if (!dados.value) return { recebido: 0, a_receber: 0, pago: 0, a_pagar: 0, resultado: 0 }
  if (diaSelecionado.value === null) return dados.value.resumo

  const r: FluxoResumo = { recebido: 0, a_receber: 0, pago: 0, a_pagar: 0, resultado: 0 }
  for (const t of dados.value.transacoes) {
    if (t.dia !== diaSelecionado.value) continue
    if (t.tipo === 'receita') t.realizado ? (r.recebido += t.valor) : (r.a_receber += t.valor)
    else                      t.realizado ? (r.pago += t.valor)     : (r.a_pagar += t.valor)
  }
  r.resultado = (r.recebido + r.a_receber) - (r.pago + r.a_pagar)
  return r
})

// ── Listagem: efetuadas e pendentes; a coluna Status distingue ─────────────
const listagem = computed(() =>
  filtrarTransacoes(dados.value?.transacoes ?? [], {
    dia: diaSelecionado.value,
    tipo: filtroTipo.value,
    situacao: filtroSituacao.value,
    busca: busca.value,
  })
)

const pillStatus = classeStatus

// ── Carregamento ───────────────────────────────────────────────────────────
async function carregar() {
  if (!props.grupoId) return
  carregando.value = true
  erro.value = ''
  try {
    bruto.value = await fetchFluxoCaixa(props.grupoId, { ...props.filtros, mes: props.mes })
    diaSelecionado.value = null
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { message?: string } } })?.response?.data?.message
    erro.value = msg || 'Erro ao carregar fluxo de caixa'
  } finally {
    carregando.value = false
  }
}

watch(() => [props.grupoId, props.filtros, props.mes], carregar, { deep: true, immediate: true })
</script>

<style scoped>
.fc { display: flex; flex-direction: column; gap: 16px; }

.fc-state {
  padding: 48px; text-align: center; gap: 10px;
  display: flex; align-items: center; justify-content: center;
  font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim);
}
.fc-state--erro { color: var(--danger); }

/* O calendário para de crescer em 760px para manter as células compactas; a
   sobra da largura vai para a coluna lateral, senão viraria vazio à direita. */
.fc-grid {
  display: grid; grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sp-4); align-items: stretch; margin-bottom: var(--sp-4);
}
@media (max-width: 1100px) { .fc-grid { grid-template-columns: 1fr; } }

/* O card de vencimentos absorve a altura que sobra, então a coluna lateral
   termina na mesma linha do calendário em qualquer viewport. */
.fc-lateral { display: flex; flex-direction: column; gap: var(--sp-4); min-height: 0; }
.fc-card--venc {
  flex: 1 1 auto; min-height: 210px;
  display: flex; flex-direction: column;
}

.fc-card {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--r); padding: var(--sp-5);
}
/* O card do calendario e coluna flex para que .cal-grid possa usar flex:1 e
   preencher a altura, em vez de a altura sair do aspect-ratio da celula. */
.fc-card--cal { display: flex; flex-direction: column; }
.fc-card-head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: var(--sp-4); flex-wrap: wrap; margin-bottom: var(--sp-4);
}
.fc-card-title {
  font-family: var(--font-display); font-size: var(--fs-md); font-weight: 600;
  color: var(--text);
}
.fc-card-sub { font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }
.fc-vazio { font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim); padding: 16px 0; }

.fc-btn, .fc-input, .fc-select {
  background: var(--surface-2); color: var(--text-muted);
  border: 1px solid var(--border-strong); border-radius: 7px;
  padding: 5px 10px; font-family: var(--font-display); font-size: var(--fs-xs); outline: none;
}
.fc-btn { cursor: pointer; transition: var(--transition); }
.fc-btn:hover { border-color: var(--primary); color: var(--primary); }
.fc-filtros { display: flex; gap: 6px; flex-wrap: wrap; }
.fc-input { min-width: 200px; }

/* ── Calendário ── */
.cal-semana, .cal-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 6px; }
.cal-semana {
  margin-bottom: 6px; font-family: var(--font-display); font-size: var(--fs-xs);
  font-weight: 600; letter-spacing: .04em;
  color: var(--text-dim); text-align: center;
}
.cal-grid { flex: 1; grid-auto-rows: minmax(56px, 1fr); }
.cal-vazio { min-height: 56px; }
.cal-dia {
  position: relative; min-height: 56px;
  display: flex; flex-direction: column; align-items: flex-start; gap: 0;
  padding: 6px 8px; overflow: hidden;
  /* Borda transparente e nao --border-strong: com a celula colapsada em 64px de
     largura a grade fechada pesava demais. A cor aparece no hover e na selecao. */
  background: var(--surface-2); border: 1px solid transparent; border-radius: var(--r-sm);
  cursor: pointer; transition: border-color var(--transition), background var(--transition);
  text-align: left;
}
.cal-dia:hover:not(:disabled) { border-color: var(--primary); }
.cal-dia--vazio { opacity: 0.35; cursor: default; }
.cal-dia--sel { border-color: var(--primary); background: var(--primary-weak); }
.cal-dia--hoje .cal-num { color: var(--primary); font-weight: 700; }
.cal-num { font-family: var(--font-display); font-size: var(--fs-sm); font-weight: 600; color: var(--text-muted); line-height: 1.3; }
.cal-val {
  font-family: var(--font-display); font-size: var(--fs-xs); line-height: 1.2;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%;
}
.cal-val--in  { color: var(--success); }
.cal-val--out { color: var(--danger); }
/* Marca discreta de que o dia contém pendência, não só realizado. */
.cal-prev {
  position: absolute; top: 5px; right: 5px;
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--warning);
}

/* ── Resumo ── */
.res-linha {
  display: flex; align-items: center; justify-content: space-between;
  padding: 7px 0; border-bottom: 1px solid var(--border);
}
.res-linha--total { border-bottom: none; border-top: 1px solid var(--border-strong); margin-top: 4px; padding-top: 10px; }
.res-rot { font-size: var(--fs-xs); color: var(--text-muted); display: flex; align-items: center; gap: 6px; }
.res-prev {
  font-family: var(--font-display); font-size: var(--fs-xs); letter-spacing: 0.5px;
  padding: 1px 5px; border-radius: 10px;
  background: var(--warning-weak); color: var(--warning);
}
.res-val { font-family: var(--font-display); font-size: var(--fs-xs); font-weight: 600; }
.res-val--in  { color: var(--success); }
.res-val--out { color: var(--danger); }
.res-linha--total .res-val { font-size: var(--fs-base); }

/* ── Próximos vencimentos ── */
/* A lista rola em vez de esticar o card com o volume de títulos. A rolagem fica
   na lista, não no card, para o título continuar visível. */
/* flex-basis 140px, não auto: é o basis que a grade usa para dimensionar a
   linha. Com basis auto a lista inteira entrava na conta e esticava o
   calendário junto. O grow faz a lista preencher a altura que sobrar. */
.venc-lista { flex: 1 1 150px; min-height: 0; overflow-y: auto; padding-right: 6px; }
.venc-item {
  display: grid; grid-template-columns: 38px 1fr auto; gap: 8px; align-items: center;
  padding: 5px 0; border-bottom: 1px solid var(--border);
}
.venc-item:last-child { border-bottom: none; }
.venc-data { font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim); }
.venc-desc { min-width: 0; display: flex; flex-direction: column; }
.venc-nome {
  font-size: var(--fs-xs); line-height: 1.3; color: var(--text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.venc-cat {
  font-family: var(--font-display); font-size: var(--fs-xs); line-height: 1.3; color: var(--text-dim);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.venc-val { font-family: var(--font-display); font-size: var(--fs-xs); font-weight: 600; white-space: nowrap; }

/* ── Tabela ── */
.fc-scroll { overflow-x: auto; }
.fc-table { width: 100%; border-collapse: collapse; min-width: 720px; }
.fc-table th {
  font-family: var(--font-display); font-size: var(--fs-xs); letter-spacing: 1px; color: var(--text-dim);
  text-align: left; padding: 9px 12px; border-bottom: 1px solid var(--border-strong); white-space: nowrap;
}
.fc-table td {
  padding: 8px 12px; font-size: var(--fs-xs); color: var(--text);
  border-bottom: 1px solid var(--border);
}
.fc-table tr:last-child td { border-bottom: none; }
.fc-table tr:hover td { background: var(--surface-2); }
.ta-r { text-align: right; }
.mono { font-family: var(--font-display); font-size: var(--fs-xs); }

.pill {
  display: inline-flex; padding: 2px 8px; border-radius: 20px;
  font-family: var(--font-display); font-size: var(--fs-xs); font-weight: 600; white-space: nowrap;
}
.pill--in  { background: var(--success-weak); color: var(--success); }
.pill--out  { background: var(--danger-weak); color: var(--danger); }
.pill--pend { background: var(--warning-weak); color: var(--warning); }
</style>
