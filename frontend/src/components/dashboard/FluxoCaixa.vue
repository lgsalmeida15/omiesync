<template>
  <div class="fc">
    <div v-if="carregando" class="fc-state"><AppSpinner /> Carregando fluxo de caixa...</div>
    <div v-else-if="erro" class="fc-state fc-state--erro">{{ erro }}</div>

    <template v-else-if="dados">
      <div class="fc-grid">
        <!-- Calendário -->
        <section class="fc-card">
          <div class="fc-card-head">
            <div>
              <div class="fc-card-title">CALENDÁRIO FINANCEIRO</div>
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
            <div class="fc-card-title">{{ diaSelecionado ? `RESUMO — DIA ${diaSelecionado}` : 'RESUMO DO MÊS' }}</div>
            <div class="fc-card-sub">{{ diaSelecionado ? 'Seleção atual' : 'Acumulado do período' }}</div>

            <div class="res-linha">
              <span class="res-rot">Recebido</span>
              <span class="res-val res-val--in">{{ fmtMoeda(resumo.recebido) }}</span>
            </div>
            <div class="res-linha">
              <span class="res-rot">A receber<span class="res-prev">previsto</span></span>
              <span class="res-val res-val--in">{{ fmtMoeda(resumo.a_receber) }}</span>
            </div>
            <div class="res-linha">
              <span class="res-rot">Pago</span>
              <span class="res-val res-val--out">{{ fmtMoeda(resumo.pago) }}</span>
            </div>
            <div class="res-linha">
              <span class="res-rot">A pagar<span class="res-prev">previsto</span></span>
              <span class="res-val res-val--out">{{ fmtMoeda(resumo.a_pagar) }}</span>
            </div>
            <div class="res-linha res-linha--total">
              <span class="res-rot">Resultado</span>
              <span class="res-val" :class="resumo.resultado < 0 ? 'res-val--out' : 'res-val--in'">
                {{ fmtMoeda(resumo.resultado) }}
              </span>
            </div>
          </section>

          <section class="fc-card">
            <div class="fc-card-title">PRÓXIMOS VENCIMENTOS</div>
            <div class="fc-card-sub">A partir de hoje, em qualquer mês</div>
            <p v-if="!dados.proximos_vencimentos.length" class="fc-vazio">Nada previsto adiante.</p>
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
          </section>
        </div>
      </div>

      <!-- Listagem: só realizado, conforme combinado -->
      <section class="fc-card">
        <div class="fc-card-head">
          <div>
            <div class="fc-card-title">TRANSAÇÕES EFETUADAS</div>
            <div class="fc-card-sub">
              {{ diaSelecionado ? `Dia ${diaSelecionado}` : 'Mês inteiro' }} — pagas e recebidas
            </div>
          </div>
          <div class="fc-filtros">
            <input v-model="busca" class="fc-input" placeholder="Buscar descrição ou categoria..." />
            <select v-model="filtroTipo" class="fc-select">
              <option value="">Todos os tipos</option>
              <option value="receita">Recebimentos</option>
              <option value="despesa">Pagamentos</option>
            </select>
          </div>
        </div>

        <p v-if="!listagem.length" class="fc-vazio">Nenhuma transação efetuada no período.</p>
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
                <td><span class="pill pill--ok">{{ t.status }}</span></td>
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
import { fetchFluxoCaixa, type FluxoCaixaData, type FluxoResumo } from '@/api/fluxocaixa'
import type { DashboardParams } from '@/api/dashboard'
import AppSpinner from '@/components/ui/AppSpinner.vue'
import { fmtMoeda, fmtCompacto } from '@/utils/formato'

const props = defineProps<{ grupoId: string; filtros: DashboardParams; mes: number }>()

const nomeMes = ['Janeiro','Fevereiro','Março','Abril','Maio','Junho',
                 'Julho','Agosto','Setembro','Outubro','Novembro','Dezembro']

const dados          = ref<FluxoCaixaData | null>(null)
const carregando     = ref(false)
const erro           = ref('')
const diaSelecionado = ref<number | null>(null)
const busca          = ref('')
const filtroTipo     = ref('')

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

// ── Listagem: apenas realizado ─────────────────────────────────────────────
const listagem = computed(() => {
  const q = busca.value.trim().toLowerCase()
  return (dados.value?.transacoes ?? []).filter(t =>
    t.realizado &&
    (diaSelecionado.value === null || t.dia === diaSelecionado.value) &&
    (!filtroTipo.value || t.tipo === filtroTipo.value) &&
    (!q || t.descricao.toLowerCase().includes(q) || t.categoria.toLowerCase().includes(q))
  )
})

// ── Carregamento ───────────────────────────────────────────────────────────
async function carregar() {
  if (!props.grupoId) return
  carregando.value = true
  erro.value = ''
  try {
    dados.value = await fetchFluxoCaixa(props.grupoId, { ...props.filtros, mes: props.mes })
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
  font-family: var(--mono); font-size: 12px; color: var(--text3);
}
.fc-state--erro { color: var(--red); }

.fc-grid { display: grid; grid-template-columns: 1fr 320px; gap: 16px; align-items: start; }
@media (max-width: 1100px) { .fc-grid { grid-template-columns: 1fr; } }

.fc-lateral { display: flex; flex-direction: column; gap: 16px; }

.fc-card {
  background: var(--card); border: 1px solid var(--border);
  border-radius: 12px; padding: 16px;
}
.fc-card-head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 12px; flex-wrap: wrap; margin-bottom: 12px;
}
.fc-card-title {
  font-family: var(--mono); font-size: 10px; letter-spacing: 1.5px;
  color: var(--text3); font-weight: 600;
}
.fc-card-sub { font-size: 12px; color: var(--text2); margin-top: 2px; }
.fc-vazio { font-family: var(--mono); font-size: 11px; color: var(--text3); padding: 16px 0; }

.fc-btn, .fc-input, .fc-select {
  background: var(--bg3); color: var(--text2);
  border: 1px solid var(--border2); border-radius: 7px;
  padding: 5px 10px; font-family: var(--mono); font-size: 10px; outline: none;
}
.fc-btn { cursor: pointer; transition: var(--trans); }
.fc-btn:hover { border-color: var(--accent); color: var(--accent); }
.fc-filtros { display: flex; gap: 6px; flex-wrap: wrap; }
.fc-input { min-width: 200px; }

/* ── Calendário ── */
.cal-semana, .cal-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 4px; }
.cal-semana {
  margin-bottom: 4px; font-family: var(--mono); font-size: 9px;
  color: var(--text3); text-align: center;
}
.cal-vazio { aspect-ratio: 1 / 0.85; }
.cal-dia {
  position: relative; aspect-ratio: 1 / 0.85;
  display: flex; flex-direction: column; align-items: flex-start; gap: 1px;
  padding: 4px 5px; overflow: hidden;
  background: var(--bg3); border: 1px solid var(--border2); border-radius: 7px;
  cursor: pointer; transition: var(--trans); text-align: left;
}
.cal-dia:hover:not(:disabled) { border-color: var(--accent); }
.cal-dia--vazio { opacity: 0.35; cursor: default; }
.cal-dia--sel { border-color: var(--accent); background: rgba(0,229,255,0.08); }
.cal-dia--hoje .cal-num { color: var(--accent); font-weight: 700; }
.cal-num { font-family: var(--mono); font-size: 10px; color: var(--text2); }
.cal-val {
  font-family: var(--mono); font-size: 9px; line-height: 1.25;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%;
}
.cal-val--in  { color: var(--green); }
.cal-val--out { color: var(--red); }
/* Marca discreta de que o dia contém provisão, não só realizado. */
.cal-prev {
  position: absolute; top: 4px; right: 4px;
  width: 5px; height: 5px; border-radius: 50%;
  background: rgba(245,158,11,0.85);
}

/* ── Resumo ── */
.res-linha {
  display: flex; align-items: center; justify-content: space-between;
  padding: 7px 0; border-bottom: 1px solid var(--border);
}
.res-linha--total { border-bottom: none; border-top: 1px solid var(--border2); margin-top: 4px; padding-top: 10px; }
.res-rot { font-size: 12px; color: var(--text2); display: flex; align-items: center; gap: 6px; }
.res-prev {
  font-family: var(--mono); font-size: 8px; letter-spacing: 0.5px;
  padding: 1px 5px; border-radius: 10px;
  background: rgba(245,158,11,0.14); color: #f59e0b;
}
.res-val { font-family: var(--mono); font-size: 12px; font-weight: 600; }
.res-val--in  { color: var(--green); }
.res-val--out { color: var(--red); }
.res-linha--total .res-val { font-size: 14px; }

/* ── Próximos vencimentos ── */
.venc-item {
  display: grid; grid-template-columns: 42px 1fr auto; gap: 8px; align-items: center;
  padding: 7px 0; border-bottom: 1px solid var(--border);
}
.venc-item:last-child { border-bottom: none; }
.venc-data { font-family: var(--mono); font-size: 10px; color: var(--text3); }
.venc-desc { min-width: 0; display: flex; flex-direction: column; }
.venc-nome {
  font-size: 12px; color: var(--text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.venc-cat { font-family: var(--mono); font-size: 9px; color: var(--text3); }
.venc-val { font-family: var(--mono); font-size: 11px; font-weight: 600; white-space: nowrap; }

/* ── Tabela ── */
.fc-scroll { overflow-x: auto; }
.fc-table { width: 100%; border-collapse: collapse; min-width: 720px; }
.fc-table th {
  font-family: var(--mono); font-size: 9px; letter-spacing: 1px; color: var(--text3);
  text-align: left; padding: 9px 12px; border-bottom: 1px solid var(--border2); white-space: nowrap;
}
.fc-table td {
  padding: 8px 12px; font-size: 12px; color: var(--text);
  border-bottom: 1px solid var(--border);
}
.fc-table tr:last-child td { border-bottom: none; }
.fc-table tr:hover td { background: rgba(255,255,255,0.025); }
.ta-r { text-align: right; }
.mono { font-family: var(--mono); font-size: 11px; }

.pill {
  display: inline-flex; padding: 2px 8px; border-radius: 20px;
  font-family: var(--mono); font-size: 9px; font-weight: 600; white-space: nowrap;
}
.pill--in  { background: rgba(34,197,94,0.12); color: var(--green); }
.pill--out { background: rgba(239,68,68,0.12); color: var(--red); }
.pill--ok  { background: rgba(255,255,255,0.06); color: var(--text2); }
</style>
