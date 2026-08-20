<template>
  <section class="gr-card">
    <div class="gr-head">
      <div>
        <div class="gr-title">{{ titulo }}</div>
        <div class="gr-sub">{{ subtitulo }}</div>
      </div>
      <button v-if="itens.length" class="gr-acao" @click="alternarTodos">
        {{ todosMarcados ? 'Desmarcar todas' : 'Marcar todas' }}
      </button>
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
          :class="['leg-item', { 'leg-item--off': !marcados.has(it.rotulo) }]"
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
import { corDe } from '@/utils/paleta'
import { totalMarcados, type Agregado } from '@/utils/agregacao'

const props = withDefaults(defineProps<{
  titulo: string
  subtitulo?: string
  itens: Agregado[]
}>(), { subtitulo: '' })

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
const total = computed(() => totalMarcados(props.itens, marcados.value))

/**
 * fmtCompacto devolve string vazia para zero — correto em rotulo de grafico,
 * onde "R$ 0" sobre barra sem altura so suja. No centro do donut o vazio parece
 * falha de renderizacao, entao aqui zero e escrito.
 */
const totalTexto = computed(() => total.value === 0 ? 'R$ 0' : fmtCompacto(total.value))

function pct(valor: number): string {
  return total.value === 0 ? '—' : `${((valor / total.value) * 100).toFixed(1)}%`
}

function alternar(rotulo: string) {
  const s = new Set(ocultos.value)
  s.has(rotulo) ? s.delete(rotulo) : s.add(rotulo)
  ocultos.value = s
}

function alternarTodos() {
  ocultos.value = todosMarcados.value ? new Set(props.itens.map(i => i.rotulo)) : new Set()
}

function desenhar() {
  if (!canvasEl.value) return
  chart?.destroy()

  const visiveis = props.itens.filter(i => marcados.value.has(i.rotulo))
  chart = new Chart(canvasEl.value, {
    type: 'doughnut',
    data: {
      labels: visiveis.map(i => i.rotulo),
      datasets: [{
        data: visiveis.map(i => i.valor),
        backgroundColor: visiveis.map(i => corDe(props.itens.findIndex(x => x.rotulo === i.rotulo))),
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

watch(() => [props.itens, marcados.value], () => nextTick(desenhar), { deep: true })
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
.leg-cor { width: 11px; height: 11px; border-radius: 3px; }
.leg-rot {
  font-size: var(--fs-sm); color: var(--text-muted);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.leg-val { font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text); white-space: nowrap; }
.leg-pct { font-family: var(--font-display); font-size: var(--fs-xs); color: var(--text-dim); text-align: right; }
</style>
