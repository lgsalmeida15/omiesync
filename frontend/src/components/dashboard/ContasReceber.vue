<template>
  <div class="cr">
    <!-- Gráficos lado a lado, acima do calendário -->
    <div class="cr-graficos">
      <GraficoDonut
        titulo="RECEBIMENTOS POR CATEGORIA"
        :subtitulo="periodo"
        :itens="categorias"
      />
      <GraficoBarrasH
        titulo="TOP 10 CLIENTES"
        :subtitulo="periodo"
        :itens="clientes"
      />
    </div>

    <!-- Calendário, resumo, vencimentos e tabela, restritos a receitas -->
    <FluxoCaixa
      :grupo-id="grupoId"
      :filtros="filtros"
      :mes="mes"
      tipo="receita"
      @dados="onDados"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import FluxoCaixa from './FluxoCaixa.vue'
import GraficoDonut from './GraficoDonut.vue'
import GraficoBarrasH from './GraficoBarrasH.vue'
import type { DashboardParams } from '@/api/dashboard'
import type { FluxoCaixaData } from '@/api/fluxocaixa'
import { porCategoria, topClientes } from '@/utils/agregacao'

defineProps<{ grupoId: string; filtros: DashboardParams; mes: number }>()

const NOME_MES = ['Janeiro','Fevereiro','Março','Abril','Maio','Junho',
                  'Julho','Agosto','Setembro','Outubro','Novembro','Dezembro']

// Os dados vêm do FluxoCaixa por evento, já recortados em receitas. Buscar de
// novo aqui duplicaria a chamada ao mesmo endpoint e abriria espaço para as
// duas metades da tela divergirem enquanto uma responde antes da outra.
const dados = ref<FluxoCaixaData | null>(null)
function onDados(d: FluxoCaixaData | null) { dados.value = d }

const periodo = computed(() =>
  dados.value ? `${NOME_MES[dados.value.mes - 1]} de ${dados.value.ano}` : ''
)

// Recebido e a receber juntos: o gráfico mostra a composição do mês, não só o
// que já entrou.
const categorias = computed(() => porCategoria(dados.value?.transacoes ?? []))
const clientes   = computed(() => topClientes(dados.value?.transacoes ?? []))
</script>

<style scoped>
.cr { display: flex; flex-direction: column; gap: 16px; }
.cr-graficos { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
@media (max-width: 1100px) { .cr-graficos { grid-template-columns: 1fr; } }
</style>
