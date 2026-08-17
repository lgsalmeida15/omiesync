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

.bar-lista { display: flex; flex-direction: column; gap: 3px; }
.bar-item {
  display: grid; grid-template-columns: minmax(0, 118px) minmax(0, 1fr) auto;
  gap: 9px; align-items: center;
  background: none; border: none; padding: 3px 5px; border-radius: 6px;
  cursor: pointer; text-align: left; transition: var(--trans);
}
.bar-item:hover { background: var(--bg3); }
.bar-item--off { opacity: 0.4; }
.bar-item--off .bar-nome { text-decoration: line-through; }
.bar-nome {
  font-size: 11px; color: var(--text2);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.bar-trilho {
  height: 13px; background: var(--bg3); border-radius: 4px; overflow: hidden;
}
.bar-preenchimento {
  display: block; height: 100%; border-radius: 4px;
  transition: width 0.25s ease;
}
.bar-valor {
  font-family: var(--mono); font-size: 10px; font-weight: 600;
  color: var(--text); white-space: nowrap;
}
</style>
