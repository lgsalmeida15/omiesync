import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type Tema = 'dark' | 'light'

function temaSalvo(): Tema {
  return localStorage.getItem('theme') === 'light' ? 'light' : 'dark'
}

export const useUiStore = defineStore('ui', () => {
  const theme         = ref<Tema>(temaSalvo())
  /*
   * Limpeza única de 'sidebar_pinned'. A chave existiu enquanto o clique no
   * logo fixava a sidebar aberta — o que virou defeito: quem clicasse uma vez
   * ficava com a barra aberta para sempre, inclusive após recarregar. O
   * gatilho saiu, mas quem já clicou tem a chave gravada; sem remover, esses
   * navegadores carregariam estado morto indefinidamente.
   */
  localStorage.removeItem('sidebar_pinned')

  /**
   * Barra de filtros recolhida por padrão: fechada o cabeçalho fixo devolve
   * ~55px de altura ao conteúdo. A preferência persiste para quem trabalha
   * com os filtros sempre à vista.
   */
  const filtrosAbertos = ref(localStorage.getItem('filtros_abertos') === 'true')

  /**
   * Exibição de centavos. Persistida porque é preferência de leitura, não estado
   * de navegação: há empresas em que o centavo é irrelevante, e quem desliga não
   * deve precisar desligar de novo a cada acesso.
   *
   * Atua só onde existem centavos hoje — o pivô do Resultado e os cards de KPI.
   * Ver a nota em utils/formato.ts.
   */
  const mostrarCentavos = ref(localStorage.getItem('mostrar_centavos') !== 'false')

  function toggleCentavos() {
    mostrarCentavos.value = !mostrarCentavos.value
    localStorage.setItem('mostrar_centavos', String(mostrarCentavos.value))
  }

  /**
   * Modo foco da tabela: some a faixa de abas e a barra de botoes do pivo, para
   * a tabela ocupar o maximo da pagina. Barra superior, lateral e o botao de
   * filtros permanecem.
   *
   * Nao persiste: e um modo de trabalho, nao preferencia. Voltar dias depois a
   * uma tela sem abas, sem lembrar como entrou, seria uma armadilha — e a saida
   * so existe dentro do proprio modo.
   */
  const focoTabela = ref(false)

  function toggleFoco() {
    focoTabela.value = !focoTabela.value
  }

  function toggleFiltros() {
    filtrosAbertos.value = !filtrosAbertos.value
    localStorage.setItem('filtros_abertos', String(filtrosAbertos.value))
  }

  /**
   * O atributo no <html> e o localStorage acompanham o ref por watcher, não por
   * atribuição dentro do toggle. Assim qualquer caminho que mude `theme` — e não
   * só o botão — mantém DOM e armazenamento em sincronia.
   *
   * `immediate` cobre o caso de o valor salvo divergir do atributo que o script
   * anti-FOUC do index.html aplicou.
   */
  watch(theme, t => {
    document.documentElement.setAttribute('data-theme', t)
    localStorage.setItem('theme', t)
  }, { immediate: true })

  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  return { theme, filtrosAbertos, mostrarCentavos, focoTabela,
           toggleTheme, toggleFiltros, toggleCentavos, toggleFoco }
})
