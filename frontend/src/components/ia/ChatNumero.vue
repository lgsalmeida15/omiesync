<template>
  <!--
    Card de indicador: um número só, em destaque.

    Existe porque a resposta mais comum do financeiro é um valor único — "quanto
    faturamos em setembro". Desenhar uma barra sozinha para mostrar isso ocupa
    200px de altura para comunicar menos que o número escrito grande.

    Não usa Chart.js de propósito: não há nada para desenhar, e carregar um
    canvas para exibir um texto seria peso sem retorno.
  -->
  <div class="cn-card">
    <div v-if="spec.titulo" class="cn-titulo">{{ spec.titulo }}</div>
    <div class="cn-valor">{{ valorFormatado }}</div>
    <div v-if="rotulo" class="cn-rotulo">{{ rotulo }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatarValor, type VisaoSpec } from '@/utils/visaospec'

const props = defineProps<{ spec: VisaoSpec }>()

const valor = computed(() => props.spec.series?.[0]?.valores?.[0] ?? 0)
const valorFormatado = computed(() => formatarValor(valor.value, props.spec.formato))

/*
 * O rótulo some quando repete o título.
 *
 * O modelo costuma escrever "Receita de setembro" nos dois campos, e a mesma
 * frase duas vezes num card de três linhas lê como defeito.
 */
const rotulo = computed(() => {
  const r = props.spec.rotulos?.[0] ?? ''
  return r.trim().toLowerCase() === (props.spec.titulo ?? '').trim().toLowerCase() ? '' : r
})
</script>

<style scoped>
.cn-card {
  margin-top: 10px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  padding: 14px 16px;
}

.cn-titulo {
  font-size: var(--fs-xs);
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: .02em;
  margin-bottom: 6px;
}

.cn-valor {
  font-size: var(--fs-xl);
  font-weight: 700;
  line-height: 1.15;
  color: var(--text);
  /* O valor é para ser comparado com o da tela do dashboard; tabular-nums já
     vem do body, e quebrar no meio de um número atrapalharia a conferência. */
  white-space: nowrap;
  overflow-x: auto;
}

.cn-rotulo {
  margin-top: 4px;
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
</style>
