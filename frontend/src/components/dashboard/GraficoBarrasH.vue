<template>
  <section class="gr-card">
    <div class="gr-head">
      <div>
        <div class="gr-title">{{ titulo }}</div>
        <div class="gr-sub">{{ subtitulo }}</div>
      </div>
      <button v-if="itens.length" class="gr-acao" @click="alternarTodos">
        {{ todosMarcados ? 'Desmarcar todos' : 'Marcar todos' }}
      </button>
    </div>

    <p v-if="!itens.length" class="gr-vazio">Nada a exibir no período.</p>

    <div v-else class="bar-lista">
      <button
        v-for="(it, i) in itens" :key="it.rotulo"
        :class="['bar-item', { 'bar-item--off': !marcados.has(it.rotulo) }]"
        @click="alternar(it.rotulo)"
      >
        <span class="bar-nome" :title="it.rotulo">{{ it.rotulo }}</span>
        <span class="bar-trilho">
          <span
            class="bar-preenchimento"
            :style="{ width: largura(it), background: corDe(i) }"
          />
        </span>
        <span class="bar-valor">{{ fmtMoeda(it.valor) }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { fmtMoeda } from '@/utils/formato'
import { corDe } from '@/utils/paleta'
import type { Agregado } from '@/utils/agregacao'

const props = withDefaults(defineProps<{
  titulo: string
  subtitulo?: string
  itens: Agregado[]
}>(), { subtitulo: '' })

// Barras em CSS e não em Chart.js: são poucas, precisam de nome à esquerda e
// valor à direita alinhados, e cada linha é clicável. Um canvas dificultaria
// as três coisas sem nenhum ganho.
const ocultos = ref(new Set<string>())
const marcados = computed(() => {
  const s = new Set(props.itens.map(i => i.rotulo))
  for (const o of ocultos.value) s.delete(o)
  return s
})
const todosMarcados = computed(() => marcados.value.size === props.itens.length)

// A escala usa só o que está visível: ao desmarcar o maior, os demais crescem
// e voltam a ser comparáveis entre si.
const maiorVisivel = computed(() =>
  Math.max(0, ...props.itens.filter(i => marcados.value.has(i.rotulo)).map(i => i.valor))
)

function largura(it: Agregado): string {
  if (!marcados.value.has(it.rotulo) || maiorVisivel.value <= 0) return '0%'
  return `${Math.max(2, (it.valor / maiorVisivel.value) * 100)}%`
}

function alternar(rotulo: string) {
  const s = new Set(ocultos.value)
  s.has(rotulo) ? s.delete(rotulo) : s.add(rotulo)
  ocultos.value = s
}

function alternarTodos() {
  ocultos.value = todosMarcados.value ? new Set(props.itens.map(i => i.rotulo)) : new Set()
}
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

.bar-lista { display: flex; flex-direction: column; gap: 3px; }
.bar-item {
  display: grid; grid-template-columns: minmax(0, 150px) minmax(0, 1fr) auto;
  gap: var(--sp-3); align-items: center;
  background: none; border: none; padding: 3px 5px; border-radius: 6px;
  cursor: pointer; text-align: left; transition: var(--transition);
}
.bar-item:hover { background: var(--surface-2); }
.bar-item--off { opacity: 0.4; }
.bar-item--off .bar-nome { text-decoration: line-through; }
.bar-nome {
  font-size: var(--fs-sm); color: var(--text-muted);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.bar-trilho {
  height: 14px; background: var(--surface-2); border-radius: 5px; overflow: hidden;
}
.bar-preenchimento {
  display: block; height: 100%; border-radius: 5px;
  transition: width 0.25s ease;
}
.bar-valor {
  font-family: var(--font-display); font-size: var(--fs-xs); font-weight: 600;
  color: var(--text); white-space: nowrap;
}
</style>
