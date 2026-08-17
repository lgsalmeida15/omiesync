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
          <div class="gr-centro-val">{{ fmtCompacto(total) }}</div>
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
  background: var(--card); border: 1px solid var(--border);
  border-radius: 12px; padding: 16px;
}
.gr-head {
  display: flex; align-items: flex-start; justify-content: space-between;
  gap: 12px; margin-bottom: 12px;
}
.gr-title {
  font-family: var(--mono); font-size: 10px; letter-spacing: 1.5px;
  color: var(--text3); font-weight: 600;
}
.gr-sub { font-size: 12px; color: var(--text2); margin-top: 2px; }
.gr-vazio { font-family: var(--mono); font-size: 11px; color: var(--text3); padding: 24px 0; }
.gr-acao {
  background: var(--bg3); color: var(--text2);
  border: 1px solid var(--border2); border-radius: 7px;
  padding: 4px 9px; font-family: var(--mono); font-size: 9px;
  cursor: pointer; transition: var(--trans); white-space: nowrap;
}
.gr-acao:hover { border-color: var(--accent); color: var(--accent); }

.gr-corpo { display: grid; grid-template-columns: 190px minmax(0, 1fr); gap: 14px; align-items: center; }
@media (max-width: 640px) { .gr-corpo { grid-template-columns: 1fr; } }

.gr-canvas-wrap { position: relative; height: 190px; }
.gr-centro {
  position: absolute; inset: 0; display: flex; flex-direction: column;
  align-items: center; justify-content: center; pointer-events: none;
}
.gr-centro-rot { font-family: var(--mono); font-size: 8px; letter-spacing: 1.5px; color: var(--text3); }
.gr-centro-val { font-family: var(--mono); font-size: 17px; font-weight: 700; color: var(--text); }

.gr-legenda { display: flex; flex-direction: column; gap: 1px; max-height: 190px; overflow-y: auto; }
.leg-item {
  display: grid; grid-template-columns: 10px minmax(0, 1fr) auto 44px;
  gap: 7px; align-items: center;
  background: none; border: none; padding: 4px 6px; border-radius: 6px;
  cursor: pointer; text-align: left; transition: var(--trans);
}
.leg-item:hover { background: var(--bg3); }
.leg-item--off { opacity: 0.4; }
.leg-item--off .leg-rot { text-decoration: line-through; }
.leg-cor { width: 10px; height: 10px; border-radius: 3px; }
.leg-rot {
  font-size: 11px; color: var(--text2);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.leg-val { font-family: var(--mono); font-size: 10px; color: var(--text); white-space: nowrap; }
.leg-pct { font-family: var(--mono); font-size: 9px; color: var(--text3); text-align: right; }
</style>
