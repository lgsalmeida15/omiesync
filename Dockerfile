# =============================================================
# Backend — omie-sync-api
# Multi-stage: build com Go 1.23, runtime com Alpine mínimo
# =============================================================

FROM golang:1.24-alpine AS builder

RUN apk --no-cache add ca-certificates git curl

# Instala golang-migrate
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o omie-sync-api ./cmd/api/main.go

# -------------------------------------------------------------

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/omie-sync-api .
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/db/migrations ./db/migrations

EXPOSE 8080

CMD ["./omie-sync-api"]
