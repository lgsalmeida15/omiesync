-- Área de trabalho temporária no HML. Recebe a cópia crua da origem e é
-- apagada ao final da transformação (ou pelos rollbacks).
DROP SCHEMA IF EXISTS demo_stage CASCADE;
CREATE SCHEMA demo_stage;
CREATE TABLE demo_stage.clientes               (LIKE grupo_demo.clientes);
CREATE TABLE demo_stage.categorias             (LIKE grupo_demo.categorias);
CREATE TABLE demo_stage.departamentos          (LIKE grupo_demo.departamentos);
CREATE TABLE demo_stage.contas_correntes       (LIKE grupo_demo.contas_correntes);
CREATE TABLE demo_stage.contas_pagar           (LIKE grupo_demo.contas_pagar);
CREATE TABLE demo_stage.contas_receber         (LIKE grupo_demo.contas_receber);
CREATE TABLE demo_stage.movimentos_financeiros (LIKE grupo_demo.movimentos_financeiros);
CREATE TABLE demo_stage.extrato                (LIKE grupo_demo.extrato);
