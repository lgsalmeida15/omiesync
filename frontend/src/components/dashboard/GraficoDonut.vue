<template>
  <section class="gr-card">
    <div class="gr-head">
      <div>
        <div class="gr-title">{{ titulo }}</div>
        <div class="gr-sub">{{ subtitulo }}</div>
      </div>
      <div class="gr-acoes">
        <button v-if="itens.length" class="gr-acao" @click="alternarTodos">
          {{ rotuloAcao }}
        </button>
        <!-- Só aparece quando o pai pede: as abas de contas montam este mesmo
             componente sem focoId e seguem sem o botão. -->
        <BotaoFoco v-if="focoId" :id="focoId" />
      </div>
    </div>

    <p v-if="!itens.length" class="gr-vazio">Nada a exibir no período.</p>

    <div v-else class="gr-corpo">
      <div class="gr-canvas-wrap">
        <canvas ref="canvasEl" />
        <div class="gr-centro">
          <div class="gr-centro-rot">TOTAL</div>
          <div class="gr-centro-val">{{ totalTexto }}</div>
        </div>
      </div>

      <div class="gr-legenda">
        <button
          v-for="(it, i) in itens" :key="it.rotulo"
          :class="['leg-item', {
            'leg-item--off': modoOcultar && !marcados.has(it.rotulo),
            'leg-item--apagado': !modoOcultar && haSelecao && !selecao.has(it.rotulo),
            'leg-item--ativo': !modoOcultar && selecao.has(it.rotulo),
          }]"
          @click="alternar(it.rotulo)"
        >
          <span class="leg-cor" :style="{ background: corDe(i) }" />
          <span class="leg-rot" :title="it.rotulo">{{ it.rotulo }}</span>
          <span class="leg-val">{{ fmtMoeda(it.valor) }}</span>
          <span class="leg-pct">{{ pct(it.valor) }}</span>
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { Chart } from 'chart.js'
import { fmtMoeda, fmtCompacto } from '@/utils/formato'
import { corDe, corDeAlpha } from '@/utils/paleta'
import { totalMarcados, type Agregado } from '@/utils/agregacao'
import { alternarRotulo } from '@/utils/fluxocruzado'
import BotaoFoco from '@/components/ui/BotaoFoco.vue'

const props = withDefaults(defineProps<{
  titulo: string
  subtitulo?: string
  itens: Agregado[]
  /**
   * 'ocultar'    — o clique esconde o item do gráfico e do seu total (abas de contas).
   * 'selecionar' — o clique recorta a tela inteira (aba Fluxo de Caixa).
   *
   * São semânticas opostas no mesmo gesto, e o visual também é o inverso: uma
   * risca o excluído, a outra realça o escolhido. Optado por conviver com as
   * duas em vez de trocar a das abas de contas, que ninguém pediu para mexer.
   */
  modo?: 'ocultar' | 'selecionar'
  /** Rótulos selecionados no modo 'selecionar'. Vazio = nenhum recorte. */
  selecionados?: string[]
  /** Id do bloco no modo foco. Sem ele, o botão de expandir não existe. */
  focoId?: string
}>(), { subtitulo: '', modo: 'ocultar', selecionados: () => [] })

const emit = defineEmits<{ (e: 'update:selecionados', v: string[]): void }>()

const modoOcultar = computed(() => props.modo === 'ocultar')
const selecao   = computed(() => new Set(props.selecionados))
const haSelecao = computed(() => selecao.value.size > 0)

const canvasEl = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null

// Todos marcados por padrão. Guardar os marcados (e não os ocultos) manteria a
// seleção do mês anterior ao trocar de período, escondendo categoria nova sem aviso.
const ocultos = ref(new Set<string>())
const marcados = computed(() => {
  const s = new Set(props.itens.map(i => i.rotulo))
  for (const o of ocultos.value) s.delete(o)
  return s
})
const todosMarcados = computed(() => marcados.value.size === props.itens.length)

/**
 * No modo 'selecionar' o total acompanha o recorte, e sem recorte é o mês
 * inteiro — assim o número no centro sempre corresponde ao que a tela está
 * mostrando abaixo.
 */
const total = computed(() =>
  modoOcultar.value
    ? totalMarcados(props.itens, marcados.value)
    : haSelecao.value
      ? totalMarcados(props.itens, selecao.value)
      : props.itens.reduce((s, i) => s + i.valor, 0)
)

const rotuloAcao = computed(() => {
  if (modoOcultar.value) return todosMarcados.value ? 'Desmarcar todas' : 'Marcar todas'
  return haSelecao.value ? 'Limpar seleção' : 'Todas as categorias'
})

/**
 * fmtCompacto devolve string vazia para zero — correto em rotulo de grafico,
 * onde "R$ 0" sobre barra sem altura so suja. No centro do donut o vazio parece
 * falha de renderizacao, entao aqui zero e escrito.
 */
const totalTexto = computed(() => total.value === 0 ? 'R$ 0' : fmtCompacto(total.value))

/**
 * Base dos percentuais.
 *
 * No modo 'selecionar' NÃO pode ser `total`, que segue o recorte: com uma
 * categoria escolhida ela apareceria como 100% e as demais somariam mais de 100%
 * entre si, porque cada uma estaria sendo dividida pelo valor da escolhida. A
 * fatia de cada categoria no mês não muda por causa do recorte.
 */
const baseDoPct = computed(() =>
  modoOcultar.value ? total.value : props.itens.reduce((s, i) => s + i.valor, 0)
)

function pct(valor: number): string {
  return baseDoPct.value === 0 ? '—' : `${((valor / baseDoPct.value) * 100).toFixed(1)}%`
}

function alternar(rotulo: string) {
  if (!modoOcultar.value) {
    emit('update:selecionados', [...alternarRotulo(selecao.value, rotulo)])
    return
  }
  const s = new Set(ocultos.value)
  s.has(rotulo) ? s.delete(rotulo) : s.add(rotulo)
  ocultos.value = s
}

function alternarTodos() {
  if (!modoOcultar.value) {
    emit('update:selecionados', [])
    return
  }
  ocultos.value = todosMarcados.value ? new Set(props.itens.map(i => i.rotulo)) : new Set()
}

function desenhar() {
  if (!canvasEl.value) return
  chart?.destroy()

  // No modo 'selecionar' nenhuma fatia sai do donut: ele é o ponto de partida do
  // próximo clique, e remover as outras deixaria o usuário sem como comparar ou
  // voltar. O recorte aparece como esmaecimento.
  const visiveis = modoOcultar.value
    ? props.itens.filter(i => marcados.value.has(i.rotulo))
    : props.itens
  chart = new Chart(canvasEl.value, {
    type: 'doughnut',
    data: {
      labels: visiveis.map(i => i.rotulo),
      datasets: [{
        data: visiveis.map(i => i.valor),
        backgroundColor: visiveis.map(i => {
          const idx = props.itens.findIndex(x => x.rotulo === i.rotulo)
          const esmaecida = !modoOcultar.value && haSelecao.value && !selecao.value.has(i.rotulo)
          return esmaecida ? corDeAlpha(idx, 0.18) : corDe(idx)
        }),
        borderColor: 'transparent',
        borderWidth: 0,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: '68%',
      plugins: {
        legend: { display: false },   // a legenda lateral é interativa e substitui esta
        datalabels: { display: false },
        tooltip: {
          callbacks: { label: c => ` ${c.label}: ${fmtMoeda(Number(c.raw))}` },
        },
      },
    },
  })
}

watch(() => [props.itens, marcados.value, props.selecionados], () => nextTick(desenhar), { deep: true })
onMounted(() => nextTick(desenhar))
onBeforeUnmount(() => chart?.destroy())
</script>

<style scoped>
.gr-card {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--r); padding: var(--sp-5);
}
.gr-head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: var(--sp-4); margin-bottom: var(--sp-4); flex-wrap: wrap;
}
.gr-title {
  font-family: var(--font-display); font-size: var(--fs-md); font-weight: 600;
  color: var(--text);
}
.gr-sub { font-size: var(--fs-xs); color: var(--text-dim); margin-top: 2px; }
.gr-vazio { font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim); padding: 24px 0; }
.gr-acao {
  background: var(--surface-2); color: var(--text-muted);
  border: 1px solid var(--border-strong); border-radius: 7px;
  padding: 4px 9px; font-family: var(--font-display); font-size: var(--fs-xs);
  cursor: pointer; transition: var(--transition); white-space: nowrap;
}
.gr-acao:hover { border-color: var(--primary); color: var(--primary); }
/* Agrupa os botões do cabeçalho. Sem o wrapper, o segundo botão viraria um
   terceiro filho do .gr-head, que é space-between, e ficaria solto no meio. */
.gr-acoes { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }

.gr-corpo {
  display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: var(--sp-5); align-items: center;
}
@media (max-width: 640px) { .gr-corpo { grid-template-columns: 1fr; } }

.gr-canvas-wrap {
  position: relative; width: 100%;
  aspect-ratio: 1 / 1; max-height: 300px; margin: 0 auto;
}
/* O canvas sai do fluxo: no fluxo, a altura que o Chart.js atribui a ele
   realimenta a altura do contêiner e vence o aspect-ratio — o donut saía
   245x300 em vez de quadrado. Absoluto, quem manda na caixa é o aspect-ratio. */
.gr-canvas-wrap canvas { position: absolute; inset: 0; }
.gr-centro {
  position: absolute; inset: 0; display: flex; flex-direction: column;
  align-items: center; justify-content: center; pointer-events: none;
}
.gr-centro-rot { font-family: var(--font-display); font-size: var(--fs-xs); letter-spacing: 1.5px; color: var(--text-dim); }
.gr-centro-val {
  font-family: var(--font-display); font-feature-settings: "tnum" 1;
  font-size: var(--fs-xl); font-weight: 700; letter-spacing: -.02em; color: var(--text);
}

.gr-legenda { display: flex; flex-direction: column; gap: 2px; max-height: 300px; overflow-y: auto; }
.leg-item {
  display: grid; grid-template-columns: 11px minmax(0, 1fr) auto 52px;
  gap: 9px; align-items: center;
  background: none; border: none; padding: 4px 6px; border-radius: 6px;
  cursor: pointer; text-align: left; transition: var(--transition);
}
.leg-item:hover { background: var(--surface-2); }
.leg-item--off { opacity: 0.4; }
.leg-item--off .leg-rot { text-decoration: line-through; }
/* Modo 'selecionar': realça o escolhido em vez de riscar o excluído. Sem
   line-through de propósito — ali o item não foi removido da conta, só não é o
   foco do recorte. */
.leg-item--apagado { opacity: 0.45; }
.leg-item--ativo { background: var(--primary-weak); }
.leg-item--ativo .leg-rot { color: var(--text); font-weight: 600; }
.leg-cor { width: 11px; height: 11px; border-radius: 3px; }
.leg-rot {
  font-size: var(--fs-sm); color: var(--text-muted);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.leg-val { font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text); white-space: nowrap; }
.leg-pct { font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim); text-align: right; }
</style>
