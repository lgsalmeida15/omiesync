# omie-sync

Plataforma multi-tenant de sincronização de dados do ERP Omie.  
Backend em Go · Frontend em Vue 3 · PostgreSQL por tenant.

---

## Arquitetura

```
Frontend (Vue 3 + Vite)
  └── nginx proxy /api/ → Backend

Backend (Go — Chi, pgx, zerolog, JWT)
  └── PostgreSQL
        ├── schema _etl        → controle (grupos, empresas, usuários, jobs, audit)
        └── schema grupo_<X>   → dados Omie por tenant (clientes, contas, extrato…)
```

---

## Stack

| Camada     | Tecnologia                              |
|------------|-----------------------------------------|
| Backend    | Go 1.23, Chi v5, pgx/v5, zerolog, JWT  |
| Frontend   | Vue 3, Vite, Tailwind CSS, Pinia        |
| Banco      | PostgreSQL 16                           |
| Deploy     | Coolify + Docker (multi-stage)          |

---

## Ambientes

| Ambiente | Frontend                                  | Backend                                  |
|----------|-------------------------------------------|------------------------------------------|
| HML      | https://hml-front-omiesync.otmiz.tech     | https://hml-back-omiesync.otmiz.tech     |

---

## Rodando localmente

### Pré-requisitos

- Go 1.23+
- Node 20+
- PostgreSQL 16+

### Backend

```bash
cp .env.example .env
# edite .env com suas credenciais locais

go mod download
go run ./cmd/api/main.go
```

### Frontend

```bash
cd frontend
cp .env.example .env.local
# edite VITE_API_URL=http://localhost:8080

npm install
npm run dev
```

### Migrations

```bash
migrate -path db/migrations -database "$DATABASE_URL" up
```

---

## Variáveis de ambiente (backend)

| Variável               | Obrigatória | Descrição                                      |
|------------------------|-------------|------------------------------------------------|
| `DATABASE_URL`         | Sim         | Connection string PostgreSQL                   |
| `JWT_SECRET`           | Sim         | Chave HMAC-SHA256, mínimo 32 chars             |
| `PORT`                 | Não         | Porta HTTP (padrão: `8080`)                    |
| `APP_ENV`              | Não         | `development` ou `production`                  |
| `LOG_LEVEL`            | Não         | `debug`, `info`, `warn`, `error`               |
| `CORS_ORIGIN`          | Não         | URL do frontend (ex: `https://app.otmiz.tech`) |
| `WORKER_MAX_CONCURRENT`| Não         | Max jobs ETL simultâneos (padrão: `20`)        |

Veja `.env.example` para referência completa.

---

## Deploy (Coolify)

Três serviços no mesmo projeto Coolify, conectados por rede privada interna:

### 1. Postgres
- Imagem: `postgres:16`
- Volume persistente para `/var/lib/postgresql/data`
- Variáveis: `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`

### 2. Backend
- Dockerfile: `./Dockerfile` (raiz do repositório)
- Porta: `8080`
- Variáveis: conforme `.env.example`
- `DATABASE_URL` usa o nome do serviço interno: `postgres:5432`

### 3. Frontend
- Dockerfile: `./frontend/Dockerfile`
- Porta: `80`
- Não requer variáveis de ambiente (API URL baked como `/api/` no build)
- O nginx faz proxy de `/api/` → `http://backend:8080/`

### Ordem de deploy
1. Postgres (aguardar healthy)
2. Backend (rodar migrations antes de abrir tráfego)
3. Frontend

### Migrations no deploy
Execute após o backend subir:
```bash
docker exec <container-backend> sh -c \
  "migrate -path /app/db/migrations -database \"$DATABASE_URL\" up"
```

---

## Segurança

- Autenticação JWT (HS256, 15 min) + Refresh Token opaco (7 dias)
- Isolamento de tenant por schema PostgreSQL
- Audit log assíncrono em todas as rotas
- Rate limiting por IP: global 300 req/min, login 10 req/min, sync/forçar 5 req/min
- SQL Explorer: SELECT-only, timeout 30s, limite 1000 linhas, schema isolado

---

## Estrutura de diretórios

```
omie-sync/
├── cmd/api/main.go          # Entrypoint
├── internal/
│   ├── auth/                # JWT, login, refresh, middleware
│   ├── grupos/              # CRUD de grupos (tenants)
│   ├── empresas/            # CRUD de empresas + carência de exclusão
│   ├── sync/                # Controle de jobs ETL, SSE
│   ├── etl/                 # Executores por módulo Omie
│   ├── worker/              # Pool de workers com semáforo
│   ├── query/               # SQL Explorer (SELECT-only)
│   ├── audit/               # Middleware de auditoria
│   ├── webhooks/            # Dispatcher assíncrono
│   └── ...
├── db/
│   ├── migrations/          # Migrations numeradas (.up/.down)
│   └── queries/             # SQL para sqlc
├── frontend/
│   ├── src/
│   │   ├── views/           # Páginas Vue
│   │   ├── components/      # Componentes reutilizáveis
│   │   ├── stores/          # Pinia stores
│   │   └── api/             # Clients HTTP
│   ├── Dockerfile
│   └── nginx.conf
├── Dockerfile               # Backend
├── .env.example
└── README.md
```
