# BookOnline — Documentação do Projeto

**Organização:** OTM Group · Paulínia/SP  
**Versão atual:** V1.0 (em construção)  
**Público:** Uso interno OTM Group — arquitetura preparada para expansão SaaS  
**Última atualização:** 2026-07-27

---

## Índice

1. [Visão Geral](#1-visão-geral)
2. [Arquitetura](#2-arquitetura)
3. [Módulos e Status](#3-módulos-e-status)
4. [Mapeamento Completo da API](#4-mapeamento-completo-da-api)
5. [Modelo de Dados](#5-modelo-de-dados)
6. [Frontend — Rotas e UX](#6-frontend--rotas-e-ux)
7. [Coleção Postman](#7-coleção-postman)
8. [Guia de Desenvolvimento](#8-guia-de-desenvolvimento)
9. [Roadmap](#9-roadmap)

---

## 1. Visão Geral

**BookOnline** é a plataforma centralizada da OTMIZ SERVIÇOS para gestão financeira multi-empresa integrada ao ERP Omie. Permite que equipes administrativas monitorem sincronização de dados, visualizem posição financeira consolidada e controlem o acesso de usuários por empresa e grupo.

### Proposta de valor

- Sincronização automatizada de dados Omie (contas a pagar/receber, extrato, movimentos, etc.)
- Controle granular de acesso por grupo de empresas e por usuário
- Painel de monitoramento de sync com histórico, progresso e reprocessamento
- SQL Explorer para consultas ad-hoc nos dados sincronizados

### Modelo de negócio

Cada **Grupo** representa um cliente (ou divisão interna). Cada grupo tem um schema PostgreSQL isolado. Um usuário pode pertencer a múltiplos grupos com roles diferentes em cada um.

---

## 2. Arquitetura

### Stack

| Camada | Tecnologia |
|---|---|
| Backend | Go 1.22+, Chi router, pgx/v5, sqlc, zerolog, JWT HS256 |
| Frontend | Vue 3, Vite, Pinia, Vue Router, TypeScript |
| Banco de dados | PostgreSQL (multi-schema, multi-tenant) |
| Deploy | Docker + Coolify (auto-deploy via GitHub push) |
| Autenticação | JWT Access Token (15min) + Refresh Token opaque UUID (7 dias) |

### Camadas do backend

```
HTTP Request
  └── Chi Router
        └── Middleware Chain
              ├── RequestID + RealIP + Recoverer
              ├── CORS
              ├── Rate Limit (300 req/min global por IP)
              └── Audit (loga TODAS as rotas — sem exceção)
                    └── Handler (decode → call service → encode)
                          └── Service (regras de negócio, orquestra)
                                ├── Repository (sqlc queries)
                                ├── WebhookDispatcher (async goroutine)
                                └── AuditRepository (grava log)
```

### Modelo multi-tenant

```
Schema _etl          → controle central
                        grupos, empresas, usuarios, usuario_grupos,
                        permissoes, refresh_tokens, audit_logs,
                        sync_jobs, sync_control, sync_job_pages,
                        sync_job_progress, webhooks, deletion_queue,
                        omie_endpoint_config, empresa_executor_config

Schema tenant        → criado ao criar um grupo (ex: grupo_alpha)
                        clientes, categorias, departamentos,
                        contas_correntes, contas_a_pagar,
                        contas_a_receber, extrato,
                        movimentos_financeiros, ordens_servico, projetos
```

### Sistema de roles

| Role | Escopo | Acesso |
|---|---|---|
| `admin_global` | Plataforma | Todos os grupos, todas as funções |
| `admin_grupo` | Grupo | Apenas o grupo do JWT, funções administrativas |
| `viewer` | Grupo | Leitura com permissão explícita por empresa |

> **Importante:** a role é armazenada por grupo em `_etl.usuario_grupos.role`. Um mesmo usuário pode ser `admin_grupo` em um grupo e `viewer` em outro. O campo `admin_global` em `_etl.usuarios.role` é um privilégio de plataforma que sempre prevalece.

---

## 3. Módulos e Status

| Módulo | Painel | Status | Observações |
|---|---|---|---|
| Login + Seleção de Grupo | Ambos | ✅ Completo | Fluxo multi-grupo com `pre_auth_token` |
| Trocar Grupo | Ambos | ✅ Completo | Disponível apenas quando usuário tem >1 grupo |
| Perfil | Ambos | ⚠️ Parcial | Página existe; troca de senha não implementada |
| Dashboard | Ambos | 🔲 Planejado | Fluxo de caixa, títulos atrasados, saldo em conta |
| Grupos (CRUD) | Admin Global | ✅ Completo | Provisiona schema tenant automaticamente |
| Sync Control Center | Admin Global | ✅ Completo | Overview global, jobs ativos, DLQ, recovery |
| Config Omie (endpoints) | Admin Global | ✅ Completo | Configura pageSize, retry, módulos por grupo |
| SQL Explorer | Admin Global + Grupo | ✅ Completo | SELECT-only, timeout 30s, limite 1000 linhas |
| Usuários Admin Global | Admin Global | 🔲 V1.x | Gestão global de usuários (cross-group) |
| Reset senha (admin) | Admin Global | 🔲 V1.x | Admin global redefine senha de admin_grupo |
| Webhooks (gestão) | Admin Global | 🔲 V1.x | Notificações de eventos por grupo |
| Empresas (CRUD) | Admin Grupo | ✅ Completo | Soft delete com carência 30 dias |
| Usuários + multi-grupo | Admin Grupo | ✅ Completo | Role isolada por grupo |
| Permissões granulares | Admin Grupo | ⚠️ Parcial | Backend OK; UI criada mas sem validação completa |
| Sync (painel) | Admin Grupo | ✅ Completo | Status, jobs, histórico, SSE, reprocessamento |
| Sync (agendamento UI) | Admin Grupo | ⚠️ Parcial | Backend OK; UI não reflete o agendamento real |
| Dados sincronizados | Admin Grupo | ✅ Completo | API de leitura dos 10 módulos Omie |

---

## 4. Mapeamento Completo da API

**Base URL:** configurada via `CORS_ORIGIN` / proxy frontend  
**Autenticação:** `Authorization: Bearer <access_token>` (exceto rotas públicas)  
**Formato de resposta padrão:**

```json
{ "success": true, "message": "OK", "data": { } }
```

---

### 4.1 Auth — `/auth`

| Método | Rota | Auth | Rate Limit | Descrição |
|---|---|---|---|---|
| `POST` | `/auth/login` | — | 10/min | Autentica. Se multi-grupo: retorna `needs_select=true` + `pre_auth_token` |
| `POST` | `/auth/select-grupo` | `pre_auth_token` (body) | 10/min | Seleciona grupo inicial; emite JWT final |
| `POST` | `/auth/logout` | — | — | Revoga refresh token |
| `POST` | `/auth/refresh` | — | 20/min | Renova access token via refresh token (rotação obrigatória) |
| `GET` | `/auth/me` | Bearer | — | Dados do usuário autenticado (role e grupo_id do JWT) |
| `GET` | `/auth/grupos` | Bearer | — | Lista grupos disponíveis para o usuário |
| `POST` | `/auth/troca-grupo` | Bearer | — | Troca grupo ativo; emite novo JWT com role do grupo destino |

**Login — request:**
```json
{ "email": "user@example.com", "password": "senha" }
```

**Login — response (um grupo):**
```json
{
  "success": true,
  "data": {
    "access_token": "...",
    "refresh_token": "...",
    "usuario": { "id": "...", "nome": "...", "email": "...", "role": "admin_grupo" }
  }
}
```

**Login — response (multi-grupo):**
```json
{
  "success": true,
  "data": {
    "needs_select": true,
    "pre_auth_token": "...",
    "grupos": [{ "id": "...", "nome": "...", "slug": "...", "schema_name": "..." }]
  }
}
```

---

### 4.2 Grupos — `/admin/grupos`

Requer: autenticado. `POST /` requer `admin_global`. Leitura/edição requer `admin_global` ou `admin_grupo` membro do grupo.

| Método | Rota | Role mínima | Descrição |
|---|---|---|---|
| `GET` | `/admin/grupos` | Autenticado | Lista grupos (admin_global vê todos; admin_grupo vê o seu) |
| `POST` | `/admin/grupos` | admin_global | Cria grupo + provisiona schema tenant |
| `GET` | `/admin/grupos/{grupoID}` | admin_global / admin_grupo | Retorna grupo por ID |
| `PUT` | `/admin/grupos/{grupoID}` | admin_global / admin_grupo | Atualiza grupo |
| `DELETE` | `/admin/grupos/{grupoID}` | admin_global / admin_grupo | Soft delete (exige empresas inativas) |

---

### 4.3 Empresas — `/admin/grupos/{grupoID}/empresas`

Requer: `admin_global` ou `admin_grupo` membro do grupo.

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/admin/grupos/{grupoID}/empresas` | Lista empresas do grupo |
| `POST` | `/admin/grupos/{grupoID}/empresas` | Cria empresa (vincula credenciais Omie) |
| `GET` | `/admin/grupos/{grupoID}/empresas/{empresaID}` | Retorna empresa por ID |
| `PUT` | `/admin/grupos/{grupoID}/empresas/{empresaID}` | Atualiza empresa |
| `DELETE` | `/admin/grupos/{grupoID}/empresas/{empresaID}` | Soft delete: `deleted_at` + fila de carência 30 dias |
| `POST` | `/admin/grupos/{grupoID}/empresas/{empresaID}/reativar` | Cancela exclusão e reativa empresa |

---

### 4.4 Usuários — `/admin/grupos/{grupoID}/usuarios`

Requer: `admin_global` ou `admin_grupo`.

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/admin/grupos/{grupoID}/usuarios` | Lista usuários do grupo (com role por grupo) |
| `POST` | `/admin/grupos/{grupoID}/usuarios` | Cria usuário ou vincula existente ao grupo |
| `GET` | `/admin/grupos/{grupoID}/usuarios/{usuarioID}` | Retorna usuário por ID |
| `PUT` | `/admin/grupos/{grupoID}/usuarios/{usuarioID}` | Atualiza dados e role no grupo |
| `PUT` | `/admin/grupos/{grupoID}/usuarios/{usuarioID}/password` | Atualiza senha do usuário |
| `DELETE` | `/admin/grupos/{grupoID}/usuarios/{usuarioID}` | Remove usuário do grupo |

---

### 4.5 Permissões — `/admin/permissoes`

Requer: `admin_global` ou `admin_grupo`. Permissão granular por usuário × empresa × recurso × ação.

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/admin/permissoes/grant` | Concede permissão (`usuario_id`, `empresa_id`, `recurso`, `acao`) |
| `POST` | `/admin/permissoes/revoke` | Revoga permissão |
| `GET` | `/admin/permissoes/usuario/{usuarioID}` | Lista permissões de um usuário |
| `GET` | `/admin/permissoes/empresa/{empresaID}` | Lista permissões de uma empresa |

**Recursos:** `dashboard`, `sync`, `admin`  
**Ações:** `ver`, `editar`, `forcar_sync`

---

### 4.6 Sync — `/sync`

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| `GET` | `/sync/{empresaID}/stream` | Bearer via `?token=` | SSE: eventos de sync em tempo real |
| `GET` | `/sync/{empresaID}/status` | Bearer | Status atual do sync da empresa |
| `GET` | `/sync/{empresaID}/pages` | Bearer | Lista páginas/módulos disponíveis para sync |
| `GET` | `/sync/{empresaID}/jobs` | Bearer | Histórico de jobs da empresa |
| `GET` | `/sync/{empresaID}/jobs/{jobID}/progress` | Bearer | Progresso detalhado de um job |
| `POST` | `/sync/{empresaID}/forcar` | admin | Força sync imediato (incremental ou full via body) |
| `PUT` | `/sync/{empresaID}/configurar` | admin | Configura agendamento e parâmetros de sync |
| `GET` | `/sync/{empresaID}/executors` | admin | Lista configs dos executors da empresa |
| `PUT` | `/sync/{empresaID}/executors/{executor}` | admin | Atualiza config de um executor específico |

> **SSE:** o `EventSource` do browser não envia headers; o token é passado via `?token=<access_token>`.

---

### 4.7 Sync Admin Global — `/admin/sync`

Requer: `admin_global`.

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/admin/sync/overview` | Visão geral de todos os jobs em andamento |
| `GET` | `/admin/sync/jobs/ativos` | Lista jobs ativos em todos os grupos |
| `GET` | `/admin/sync/dlq` | Dead Letter Queue — jobs com falha persistente |
| `POST` | `/admin/sync/pages/{pageID}/retry` | Reprocessa uma página com falha |
| `POST` | `/admin/sync/jobs/{jobID}/cancelar` | Cancela um job em andamento |
| `POST` | `/admin/sync/startup-recovery` | Recupera jobs interrompidos por restart do servidor |

---

### 4.8 Config Omie — `/admin/omie-config`

Requer: `admin_global`.

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/admin/omie-config` | Lista todas as configs de endpoints Omie |
| `GET` | `/admin/omie-config/{modulo}` | Retorna config de um módulo específico |
| `PUT` | `/admin/omie-config/{modulo}` | Atualiza config (pageSize, retry, etc.) |

---

### 4.9 SQL Explorer — `/admin/grupos/{grupoID}/query`

Requer: `admin_global` ou `admin_grupo`. Rate limit: 20 queries/min por usuário.

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/admin/grupos/{grupoID}/query` | Executa query SELECT no schema do grupo |

**Request:**
```json
{ "sql": "SELECT * FROM clientes WHERE nome ILIKE '%alpha%'" }
```

**Response:**
```json
{
  "success": true,
  "data": {
    "columns": ["id", "nome", "email"],
    "rows": [[1, "Alpha Corp", "alpha@example.com"]],
    "row_count": 1,
    "truncated": false
  }
}
```

**Restrições de segurança:**
- Apenas `SELECT` permitido (rejeita INSERT/UPDATE/DELETE/DROP/TRUNCATE/ALTER/CREATE/GRANT/REVOKE/EXECUTE/CALL/DO/COPY)
- Sem acesso ao schema `_etl`
- Timeout de 30 segundos (`SET LOCAL statement_timeout = '30s'`)
- Máximo 1000 linhas (LIMIT automático)
- Executa em `BEGIN READ ONLY`

---

### 4.10 Dados Sincronizados — `/dados`

Leitura dos dados Omie já sincronizados no schema tenant da empresa.

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/dados/{empresaID}/clientes` | Clientes cadastrados no Omie |
| `GET` | `/dados/{empresaID}/categorias` | Categorias financeiras |
| `GET` | `/dados/{empresaID}/departamentos` | Departamentos |
| `GET` | `/dados/{empresaID}/contas-correntes` | Contas correntes |
| `GET` | `/dados/{empresaID}/contas-pagar` | Contas a pagar |
| `GET` | `/dados/{empresaID}/contas-receber` | Contas a receber |
| `GET` | `/dados/{empresaID}/movimentos` | Movimentos financeiros |
| `GET` | `/dados/{empresaID}/extrato` | Extrato bancário |
| `GET` | `/dados/{empresaID}/ordens-servico` | Ordens de serviço |
| `GET` | `/dados/{empresaID}/projetos` | Projetos |

---

### 4.11 Health Check

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| `GET` | `/health` | — | Retorna `{"status":"ok"}` — usado pelo Coolify para liveness |

---

## 5. Modelo de Dados

### Schema `_etl` (controle central)

```
grupos
  id (uuid PK), nome, slug, schema_name, deleted_at, created_at, updated_at

empresas
  id (uuid PK), grupo_id (FK grupos), nome, cnpj,
  omie_app_key, omie_app_secret,
  status_sync (ativa|pausada|deletando|inativa),
  deleted_at, created_at, updated_at

usuarios
  id (uuid PK), nome, email, password (bcrypt),
  role (admin_global|admin_grupo|viewer),
  ativo, created_at, updated_at
  NOTE: role aqui é o privilégio de plataforma. Role por grupo fica em usuario_grupos.

usuario_grupos                          ← tabela de junção N:N
  usuario_id (FK), grupo_id (FK),
  role (admin_grupo|viewer),            ← role isolada por grupo
  created_at
  PK: (usuario_id, grupo_id)

permissoes
  id (uuid PK), usuario_id, empresa_id,
  recurso (dashboard|sync|admin),
  acao (ver|editar|forcar_sync),
  created_at

refresh_tokens
  id (uuid PK), usuario_id, token (uuid),
  expires_at, revoked, created_at

audit_logs
  id (uuid PK), request_id, user_id, user_email, role,
  method, path, query_params,
  status_code, request_body, response_body,
  ip_address, user_agent, duration_ms, created_at

sync_jobs
  id (uuid PK), empresa_id,
  tipo (incremental|full),
  status (pendente|rodando|concluido|falhou|cancelado),
  started_at, finished_at,
  heartbeat_at, created_at

sync_control
  id (uuid PK), empresa_id,
  schedule_incremental (cron), schedule_full (cron),
  ultima_sync_incremental, ultima_sync_full,
  proximo_incremental, proximo_full

sync_job_pages
  id (uuid PK), job_id (FK),
  executor, pagina, status,
  tentativas, erro, created_at, updated_at

sync_job_progress
  id (uuid PK), job_id (FK), empresa_id,
  executor, status, mensagem,
  total, processado, falhou,
  payload (jsonb), raw_payload (jsonb),
  created_at, updated_at

omie_endpoint_config
  id (uuid PK), modulo, endpoint_url,
  page_size, max_retries, timeout_segundos,
  ativo, created_at, updated_at

empresa_executor_config
  id (uuid PK), empresa_id, executor,
  ativo, full_sync_interval_horas,
  created_at, updated_at

webhooks
  id (uuid PK), grupo_id, url,
  eventos (text[]), ativo, created_at

deletion_queue
  id (uuid PK), empresa_id,
  execute_at (NOW() + 30 dias),
  executed_at, created_at
```

### Schema tenant (ex: `grupo_alpha`)

Todas as tabelas incluem `empresa_id` como coluna de isolamento por empresa dentro do grupo.

```
clientes, categorias, departamentos,
contas_correntes, contas_a_pagar, contas_a_receber,
extrato, movimentos_financeiros,
ordens_servico, projetos
```

---

## 6. Frontend — Rotas e UX

### Rotas Vue Router

| Path | View | Meta | Observações |
|---|---|---|---|
| `/login` | `LoginView.vue` | public | Redireciona para Dashboard se autenticado |
| `/select-grupo` | `SelectGrupoView.vue` | selectGrupo | Exibida apenas quando `needs_select=true` pós-login |
| `/` | `DashboardView.vue` | requiresAuth | Página inicial autenticada |
| `/grupos` | `GruposView.vue` | admin_global | CRUD de grupos |
| `/grupos/:grupoId/empresas` | `EmpresasView.vue` | admin_global | Empresas de um grupo específico |
| `/empresas` | `EmpresasView.vue` | admin_grupo | Empresas do grupo corrente |
| `/usuarios` | `UsuariosView.vue` | admin_grupo | Usuários do grupo corrente |
| `/permissoes` | `PermissoesView.vue` | admin_grupo | Gestão de permissões granulares |
| `/sync` | `SyncView.vue` | admin_grupo | Lista de empresas com status de sync |
| `/sync/:empresaId` | `SyncEmpresaView.vue` | admin_grupo | Detalhe de sync (SSE, jobs, config) |
| `/omie-config` | `OmieConfigView.vue` | admin_global | Configuração de endpoints Omie |
| `/admin/sync-control` | `SyncControlCenter.vue` | admin_global | Visão global de sync, DLQ, recovery |
| `/sql-explorer` | `SqlExplorerView.vue` | admin_global, admin_grupo | Editor SQL ad-hoc |
| `/perfil` | `PerfilView.vue` | requiresAuth | Perfil do usuário |
| `/403` | `ForbiddenView.vue` | — | Acesso negado |

### Guards de rota

```
1. Rota pública        → permitir; se autenticado redireciona para /
2. Rota selectGrupo    → permitir; se não autenticado e sem pre_auth redireciona para /login
3. Demais rotas        → exige auth; verifica roles; se sem permissão → /403
```

### Fluxo de autenticação

```
Login
  ├── Um grupo:   POST /auth/login → setTokens → fetchMe → refreshMeusGrupos → Dashboard
  └── Multi-grupo: POST /auth/login (needs_select) → /select-grupo
                      → POST /auth/select-grupo → setTokens → fetchMe → refreshMeusGrupos → Dashboard

Trocar grupo (sidebar "Trocar Grupo" — visível apenas se meusGrupos.length > 1)
  → POST /auth/troca-grupo → setTokens → fetchMe → refreshMeusGrupos → Dashboard
```

### Stores Pinia

**`auth.ts`**
- `user` — dados do usuário autenticado (role reflete o grupo corrente do JWT)
- `accessToken`, `refreshToken` — persistidos no localStorage
- `preAuthToken`, `pendingGrupos` — estado temporário de seleção de grupo pós-login
- `meusGrupos` — lista de grupos do usuário (persistida no localStorage; controla visibilidade de "Trocar Grupo")
- Computed: `isAuthenticated`, `needsGroupSelect`, `isAdminGlobal`, `isAdminGrupo`, `isViewer`, `isAdmin`

**`ui.ts`**
- `sidebarPinned` — estado de expansão/colapso do sidebar

### Sidebar — lógica de visibilidade

```
PRINCIPAL       → Dashboard (sempre)

ADMINISTRACAO   → admin_global: Grupos, Sync Control
                → admin_grupo: Empresas, Usuarios, Permissoes

SISTEMA         → admin_grupo: Sync
                → admin_global: Config Omie
                → admin_global + admin_grupo: SQL Explorer
                → meusGrupos.length > 1: Trocar Grupo
                → todos: Perfil
```

### SyncEmpresaView — arquitetura de detalhe

- SSE conectado no `onMounted`, desconectado no `onUnmounted`
- Tabs: **Progresso** | **Histórico** | **Config**
- Aba inicial via `?aba=` na query string
- `loadExecutorConfigs()` chamado no `onMounted` (evita executor configs vazio)
- Botões ▶ (incremental) e ↺ (full) no header
- Componentes reutilizados: `SyncDrawerProgresso`, `SyncDrawerHistorico`, `SyncDrawerConfig`

---

## 7. Coleção Postman

Arquivo: `BookOnline.postman_collection.json` (na raiz do repositório)

### Variáveis de ambiente

| Variável | Valor típico |
|---|---|
| `base_url` | `http://localhost:8080` |
| `access_token` | (preenchido automaticamente pelo script de login) |
| `refresh_token` | (preenchido automaticamente) |
| `pre_auth_token` | (preenchido automaticamente no fluxo multi-grupo) |
| `grupo_id` | UUID do grupo corrente |
| `empresa_id` | UUID da empresa corrente |

### Política de manutenção

Todo novo endpoint adicionado ao backend deve ter entrada correspondente na coleção. A coleção é a fonte de verdade para integração e testes manuais.

> **Nota:** A coleção ainda não foi gerada. Está prevista como próximo entregável de documentação.

---

## 8. Guia de Desenvolvimento

### Pré-requisitos

- Go 1.22+
- Node.js 20+
- PostgreSQL 15+
- `sqlc` CLI — `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- `migrate` CLI — `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

### Variáveis de ambiente

```env
DATABASE_URL=postgres://user:pass@localhost:5432/omie_sync?sslmode=disable
JWT_SECRET=minimo-32-caracteres-aleatorios-aqui
PORT=8080
APP_ENV=development
CORS_ORIGIN=http://localhost:5173,http://localhost:5174
```

### Setup local

```bash
# Backend
go mod download
go run ./cmd/api/main.go          # migrations rodam automaticamente no startup

# Frontend
cd frontend
npm install
npm run dev
```

### Fluxo de migrations

As migrations rodam **automaticamente** ao iniciar o servidor (via `db.RunMigrations()` em `main.go`). Nunca são fatais — o servidor sempre sobe independente do resultado.

**Baseline:** se `schema_migrations` estiver vazia mas `_etl.grupos` já existir, todas as migrations são registradas como aplicadas sem executar (evita re-execução em banco existente).

```bash
# Rodar manualmente (opcional)
migrate -path db/migrations -database $DATABASE_URL up
migrate -path db/migrations -database $DATABASE_URL down 1
```

### Fluxo de queries SQL (sqlc)

```bash
# 1. Editar o arquivo SQL em db/queries/
# 2. Gerar o código Go
sqlc generate

# NUNCA editar /sqlc/generated/ manualmente
```

### Checklist ao adicionar nova feature

```
[ ] Interface definida antes da implementação
[ ] sqlc generate rodou após editar queries
[ ] app_secret não aparece em nenhum log, response ou error message
[ ] Audit middleware cobre o endpoint (automático via routes.go)
[ ] Testes escritos e passando: go test ./...
[ ] go vet ./... sem warnings
[ ] Erros wrapped: fmt.Errorf("pacote.Func: %w", err)
[ ] context.Context é o primeiro parâmetro nas funções de banco
[ ] Novo endpoint adicionado ao Postman collection
[ ] PROJETO.md atualizado se houver mudança na API ou nos módulos
```

### Comandos de referência

```bash
go test ./...                        # todos os testes
go test ./internal/auth/... -v       # módulo específico
go build ./...                       # build completo
go vet ./...                         # análise estática
go run ./cmd/api/main.go             # servidor local
```

### Deploy (Coolify)

- Backend e frontend são serviços separados no Coolify
- Auto-deploy ativado: push para `master` → redeploy automático
- Migrations rodam no startup do container
- Health check: `GET /health` → `{"status":"ok"}`

---

## 9. Roadmap

### V1.0 — Em construção (critério de conclusão)

```
[x] Autenticação JWT + multi-grupo
[x] Gestão de grupos (CRUD + schema tenant)
[x] Gestão de empresas (CRUD + carência 30 dias)
[x] Gestão de usuários multi-grupo com role por grupo
[x] Sync Control (jobs, histórico, SSE, reprocessamento)
[x] Config Omie (endpoints, pageSize, retry)
[x] SQL Explorer (SELECT-only, timeout, limite)
[x] Audit Log em todas as rotas
[ ] Dashboard financeiro (Fluxo de Caixa, Títulos atrasados, Saldo em conta)
[ ] Perfil — troca de senha
[ ] Permissões granulares — validação completa no backend + UI funcional
[ ] Sync — agendamento UI refletindo o estado real
[ ] Postman collection completa e versionada no GitHub
```

### V1.x — Admin Global completo

```
[ ] Gestão de usuários Admin Global (visualizar/criar usuários cross-group)
[ ] Reset de senha por admin_global para admin_grupo
[ ] Gestão de webhooks (eventos por grupo: empresa.pausada, sync.falhou, etc.)
[ ] Modo Global para admin_global (operar sem contexto de grupo — T3 deferida)
```

### V2.0 — Expansão SaaS (planejado)

```
[ ] Onboarding self-service (criação de conta + grupo pelo próprio cliente)
[ ] Billing e planos (limite de empresas/usuários por plano)
[ ] Multi-instância (separação de ambiente por cliente externo)
[ ] Portal do cliente (acesso restrito ao próprio grupo sem painel admin)
```

---

*Este documento é a fonte de verdade do projeto BookOnline. Atualizar sempre que houver mudança de API, novo módulo ou alteração de status no roadmap.*
