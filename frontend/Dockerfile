# =============================================================
# Frontend — omie-sync (Vue 3 + Vite)
# Multi-stage: build com Node 20, runtime com nginx Alpine
# =============================================================

FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci --prefer-offline

COPY . .

# VITE_API_URL=/api faz o frontend chamar rotas relativas,
# que o nginx redireciona para o backend via rede interna.
ENV VITE_API_URL=/api

RUN npm run build

# -------------------------------------------------------------

FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
