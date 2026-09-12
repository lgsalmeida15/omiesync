/**
 * Modo foco: um bloco da tela ocupa a página inteira e os demais somem.
 *
 * A regra vive aqui, e não espalhada pelos componentes, porque `deveMostrar`
 * parece trivial demais para errar — e é justamente por isso que erraria: escrita
 * à mão em cinco lugares, basta um deles inverter o caso "sem foco" para a aba
 * abrir em branco, sem erro nenhum no console.
 */

/** Identificador de um bloco focável. Strings livres, únicas por tela. */
export type IdFoco = string

/**
 * Próximo estado ao clicar no botão de um bloco.
 *
 * Reclicar o bloco já expandido fecha; clicar em outro TROCA, em vez de empilhar
 * — dois blocos expandidos ao mesmo tempo não significam nada, e sair de um para
 * cair noutro seria confuso.
 */
export function proximoFoco(atual: IdFoco | null, id: IdFoco): IdFoco | null {
  return atual === id ? null : id
}

/**
 * Este bloco deve ser renderizado?
 *
 * Sem foco, TUDO aparece — este é o caso que não pode errar. Com foco, só o
 * bloco em foco.
 */
export function deveMostrar(foco: IdFoco | null, id: IdFoco): boolean {
  return foco === null || foco === id
}
