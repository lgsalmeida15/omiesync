-- Prompt do assistente editável pela tela de administração.
--
-- Vazio significa "usa o texto versionado no código" (internal/ia/prompt.go).
-- É o que preserva o comportamento atual sem precisar semear nada aqui, e o que
-- dá o caminho de volta: limpar o campo restaura o padrão.
--
-- O seed da 000033 insere só o id, então a coluna com DEFAULT basta.

ALTER TABLE _etl.ia_config
  ADD COLUMN IF NOT EXISTS system_prompt TEXT NOT NULL DEFAULT '';
