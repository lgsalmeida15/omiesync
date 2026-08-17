<template>
  <div class="cpt">
    <!-- Gráficos lado a lado, acima do calendário -->
    <div class="cpt-graficos">
      <GraficoDonut
        :titulo="rotulos.donut"
        :subtitulo="periodo"
        :itens="categorias"
      />
      <GraficoBarrasH
        :titulo="rotulos.barras"
        :subtitulo="periodo"
        :itens="entidades"
      />
    </div>

    <!-- Calendário, resumo, vencimentos e tabela, restritos ao tipo -->
    <FluxoCaixa
      :grupo-id="grupoId"
      :filtros="filtros"
      :mes="mes"
      :tipo="tipo"
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
import { porCategoria, topEntidades } from '@/utils/agregacao'

const props = defineProps<{
  grupoId: string
  filtros: DashboardParams
  mes: number
  tipo: 'receita' | 'despesa'
}>()

const NOME_MES = ['Janeiro','Fevereiro','Março','Abril','Maio','Junho',
                  'Julho','Agosto','Setembro','Outubro','Novembro','Dezembro']

// A entidade muda de nome conforme o lado: cliente recebe, fornecedor cobra.
const rotulos = computed(() => props.tipo === 'receita'
  ? { donut: 'RECEBIMENTOS POR CATEGORIA', barras: 'TOP 10 CLIENTES' }
  : { donut: 'PAGAMENTOS POR CATEGORIA',   barras: 'TOP 10 FORNECEDORES' }
)

// Os dados vêm do FluxoCaixa por evento, já recortados pelo tipo. Buscar de
// novo aqui duplicaria a chamada ao mesmo endpoint e abriria espaço para as
// duas metades da tela divergirem enquanto uma responde antes da outra.
const dados = ref<FluxoCaixaData | null>(null)
function onDados(d: FluxoCaixaData | null) { dados.value = d }

const periodo = computed(() =>
  dados.value ? `${NOME_MES[dados.value.mes - 1]} de ${dados.value.ano}` : ''
)

// Realizado e previsto juntos: os gráficos mostram a composição do mês, não só
// o que já foi liquidado.
const categorias = computed(() => porCategoria(dados.value?.transacoes ?? []))
const entidades  = computed(() => topEntidades(dados.value?.transacoes ?? []))
</script>

<style scoped>
.cpt { display: flex; flex-direction: column; gap: 16px; }
.cpt-graficos { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
@media (max-width: 1100px) { .cpt-graficos { grid-template-columns: 1fr; } }
</style>
