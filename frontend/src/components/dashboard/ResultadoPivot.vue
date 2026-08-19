<template>
  <div class="pv">
    <div v-if="carregando" class="pv-state"><AppSpinner /> Carregando resultado...</div>
    <div v-else-if="erro" class="pv-state pv-state--erro">{{ erro }}</div>
    <div v-else-if="!dados || !dados.linhas.length" class="pv-state">Nenhum dado no período.</div>

    <template v-else>
      <div class="pv-toolbar">
        <button class="pv-btn" @click="expandirAte(1)">Recolher tudo</button>
        <button class="pv-btn" @click="expandirAte(2)">Categoria superior</button>
        <button class="pv-btn" @click="expandirAte(3)">Categoria final</button>
        <button class="pv-btn" @click="expandirAte(4)">Expandir tudo</button>
        <span v-if="dados.mes_corte <= 12" class="pv-legenda">
          <span class="pv-chip-prev" /> a partir de {{ nomeMes[dados.mes_corte - 1] }} são valores previstos
        </span>
      </div>

      <div class="pv-scroll">
        <table class="pv-table">
          <thead>
            <tr>
              <th class="pv-th-dim">DESCRIÇÃO</th>
              <th v-for="(m, i) in nomeMes" :key="m"
                  :class="['pv-th-mes', { 'pv-previsto': i + 1 >= dados.mes_corte }]">{{ m }}</th>
              <th class="pv-th-total">TOTAL</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="n in linhasVisiveis" :key="n.id"
                :class="['pv-tr', `pv-nivel-${n.nivel}`, { 'pv-tr-folha': n.nivel === 4 }]">
              <td class="pv-td-dim" :style="{ paddingLeft: `${12 + (n.nivel - 1) * 18}px` }">
                <button v-if="n.temFilhos" class="pv-toggle" @click="alternar(n.id)">
                  {{ expandidos.has(n.id) ? '−' : '+' }}
                </button>
                <span v-else class="pv-toggle pv-toggle--vazio" />
                <span class="pv-rotulo">{{ n.rotulo }}</span>
              </td>
              <td v-for="(v, i) in n.meses" :key="i"
                  :class="['pv-td-num', { 'pv-previsto': i + 1 >= dados.mes_corte, 'pv-zero': v === 0, 'pv-neg': v < 0 }]">
                {{ v === 0 ? '—' : fmt(v) }}
              </td>
              <td class="pv-td-total" :class="{ 'pv-neg': n.total < 0 }">{{ fmt(n.total) }}</td>
            </tr>
          </tbody>
          <tfoot>
            <tr class="pv-tfoot">
              <td class="pv-td-dim">
                RESULTADO
                <span class="pv-nota" title="Receita menos despesa do período. Não inclui o saldo das contas correntes, por isso difere do card RESULTADO da Visão Geral.">?</span>
              </td>
              <td v-for="(v, i) in dados.resultado_mes" :key="i"
                  :class="['pv-td-num', { 'pv-previsto': i + 1 >= dados.mes_corte, 'pv-neg': v < 0 }]">
                {{ v === 0 ? '—' : fmt(v) }}
              </td>
              <td class="pv-td-total" :class="{ 'pv-neg': dados.resultado_total < 0 }">{{ fmt(dados.resultado_total) }}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { fetchPivot, type PivotData, type PivotLinha } from '@/api/pivot'
import type { DashboardParams } from '@/api/dashboard'
import AppSpinner from '@/components/ui/AppSpinner.vue'
import { fmtNumero } from '@/utils/formato'

const props = defineProps<{ grupoId: string; filtros: DashboardParams }>()

const nomeMes = ['Jan','Fev','Mar','Abr','Mai','Jun','Jul','Ago','Set','Out','Nov','Dez']

const dados      = ref<PivotData | null>(null)
const carregando = ref(false)
const erro       = ref('')
const expandidos = ref<Set<string>>(new Set())

// ── Montagem da árvore ─────────────────────────────────────────────────────
// O backend entrega as folhas já pivotadas por mês. Os níveis superiores são
// somados aqui: não há ida ao servidor a cada expansão.
interface No {
  id: string
  nivel: 1 | 2 | 3 | 4
  rotulo: string
  meses: number[]
  total: number
  temFilhos: boolean
  paiId: string | null
}

const zeros = () => Array(12).fill(0)

const arvore = computed<No[]>(() => {
  if (!dados.value) return []

  // Acumula por nível preservando a ordem de chegada (o SQL já ordena).
  const acc = new Map<string, No>()
  const ordem: string[] = []

  const somar = (id: string, nivel: No['nivel'], rotulo: string, paiId: string | null, l: PivotLinha) => {
    let n = acc.get(id)
    if (!n) {
      n = { id, nivel, rotulo, meses: zeros(), total: 0, temFilhos: nivel < 4, paiId }
      acc.set(id, n)
      ordem.push(id)
    }
    for (let i = 0; i < 12; i++) n.meses[i] += l.meses[i]
    n.total += l.total
  }

  for (const l of dados.value.linhas) {
    const i1 = `1|${l.tipo}`
    const i2 = `${i1}|${l.categoria_superior}`
    const i3 = `${i2}|${l.categoria_final}`
    const i4 = `${i3}|${l.cliente}`
    somar(i1, 1, l.tipo.toUpperCase(), null, l)
    somar(i2, 2, l.categoria_superior, i1, l)
    somar(i3, 3, l.categoria_final,    i2, l)
    somar(i4, 4, l.cliente,            i3, l)
  }

  // Reordena em profundidade para que cada nó apareça sob o seu pai.
  const porPai = new Map<string | null, No[]>()
  for (const id of ordem) {
    const n = acc.get(id)!
    const lista = porPai.get(n.paiId) ?? []
    lista.push(n)
    porPai.set(n.paiId, lista)
  }

  // Receita antes de despesa. O SQL ordena por `tipo`, e alfabeticamente "despesa"
  // vem primeiro — invertido em relação à leitura de um resultado.
  const pesoTipo = (rotulo: string) => (rotulo.toUpperCase() === 'RECEITA' ? 0 : 1)
  const raiz = porPai.get(null)
  if (raiz) raiz.sort((a, b) => pesoTipo(a.rotulo) - pesoTipo(b.rotulo))
  const saida: No[] = []
  const descer = (paiId: string | null) => {
    for (const n of porPai.get(paiId) ?? []) {
      saida.push(n)
      descer(n.id)
    }
  }
  descer(null)
  return saida
})

// Um nó aparece se todos os seus ancestrais estiverem expandidos.
const linhasVisiveis = computed(() =>
  arvore.value.filter(n => {
    let pai = n.paiId
    while (pai) {
      if (!expandidos.value.has(pai)) return false
      pai = arvore.value.find(x => x.id === pai)?.paiId ?? null
    }
    return true
  })
)

function alternar(id: string) {
  const s = new Set(expandidos.value)
  s.has(id) ? s.delete(id) : s.add(id)
  expandidos.value = s
}

function expandirAte(nivel: number) {
  expandidos.value = new Set(
    arvore.value.filter(n => n.nivel < nivel && n.temFilhos).map(n => n.id)
  )
}

// Notação contábil: negativo entre parênteses. Ver src/utils/formato.ts.
const fmt = fmtNumero

// ── Carregamento ───────────────────────────────────────────────────────────
async function carregar() {
  if (!props.grupoId) return
  carregando.value = true
  erro.value = ''
  try {
    dados.value = await fetchPivot(props.grupoId, props.filtros)
    expandirAte(2) // abre o primeiro nível: receita/despesa visíveis, resto fechado
  } catch (e: any) {
    erro.value = e?.response?.data?.message ?? 'Erro ao carregar resultado'
  } finally {
    carregando.value = false
  }
}

watch(() => [props.grupoId, props.filtros], carregar, { deep: true, immediate: true })
</script>

<style scoped>
.pv { display: flex; flex-direction: column; gap: 12px; }

.pv-state {
  padding: 48px; text-align: center;
  font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim);
  display: flex; align-items: center; justify-content: center; gap: 10px;
}
.pv-state--erro { color: var(--danger); }

.pv-toolbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.pv-btn {
  background: var(--surface-2); color: var(--text-muted);
  border: 1px solid var(--border-strong); border-radius: 7px;
  padding: 5px 11px; font-family: var(--font-display); font-size: var(--fs-xs);
  letter-spacing: 0.5px; cursor: pointer; transition: var(--transition);
}
.pv-btn:hover { border-color: var(--primary); color: var(--primary); }

.pv-legenda {
  margin-left: auto; display: flex; align-items: center; gap: 6px;
  font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim);
}
.pv-chip-prev {
  width: 10px; height: 10px; border-radius: 3px;
  background: var(--warning-weak); border: 1px solid var(--warning);
}

/* A tabela rola dentro do próprio container — a página nunca rola na horizontal. */
.pv-scroll {
  overflow-x: auto;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
}
.pv-table { width: 100%; border-collapse: collapse; min-width: 900px; }

.pv-table th {
  position: sticky; top: 0; z-index: 2;
  background: var(--surface);
  font-family: var(--font-display); font-size: var(--fs-xs); letter-spacing: 1px;
  color: var(--text-dim); font-weight: 600;
  padding: 10px 12px; border-bottom: 1px solid var(--border-strong);
  white-space: nowrap;
}
.pv-th-dim   { text-align: left; min-width: 280px; }
.pv-th-mes   { text-align: right; }
.pv-th-total { text-align: right; border-left: 1px solid var(--border-strong); }

.pv-td-dim {
  padding: 7px 12px; font-size: var(--fs-xs); color: var(--text);
  white-space: nowrap; display: flex; align-items: center; gap: 6px;
}
.pv-td-num, .pv-td-total {
  padding: 7px 12px; text-align: right;
  font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-muted);
  white-space: nowrap;
}
.pv-td-total { border-left: 1px solid var(--border); font-weight: 600; color: var(--text); }
.pv-zero { color: var(--text-dim); opacity: 0.45; }
/* Negativo em vermelho, somado aos parênteses da notação contábil. !important
   porque .pv-td-total define cor própria e a especificidade empata. */
.pv-neg { color: var(--danger) !important; }
.pv-previsto { background: var(--warning-weak); }

.pv-tr { border-bottom: 1px solid var(--border); }
.pv-tr:hover td { background: var(--surface-2); }
.pv-nivel-1 .pv-td-dim { font-weight: 700; letter-spacing: 0.5px; }
.pv-nivel-1 td { background: var(--surface-2); }
.pv-nivel-2 .pv-td-dim { font-weight: 600; }
.pv-tr-folha .pv-td-dim { color: var(--text-muted); }

.pv-toggle {
  width: 16px; height: 16px; flex-shrink: 0;
  display: inline-flex; align-items: center; justify-content: center;
  background: var(--surface-2); border: 1px solid var(--border-strong); border-radius: 4px;
  color: var(--text-muted); font-family: var(--font-display); font-size: var(--fs-xs); line-height: 1;
  cursor: pointer; transition: var(--transition); padding: 0;
}
.pv-toggle:hover { border-color: var(--primary); color: var(--primary); }
.pv-toggle--vazio { background: none; border: none; cursor: default; }

.pv-rotulo { overflow: hidden; text-overflow: ellipsis; }

.pv-tfoot td {
  position: sticky; bottom: 0;
  background: var(--surface); border-top: 1px solid var(--border-strong);
  font-weight: 700; color: var(--text);
  font-family: var(--font-display); font-size: var(--fs-xs); padding: 10px 12px;
}
.pv-tfoot .pv-td-dim { font-size: var(--fs-xs); letter-spacing: 1px; }
.pv-nota {
  display: inline-flex; align-items: center; justify-content: center;
  width: 13px; height: 13px; margin-left: 6px;
  border: 1px solid var(--border-strong); border-radius: 50%;
  font-size: var(--fs-xs); color: var(--text-dim); cursor: help; font-weight: 400;
}
</style>
