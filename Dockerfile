# =============================================================
# Backend — omie-sync-api
# Multi-stage: build com Go 1.23, runtime com Alpine mínimo
# =============================================================

FROM golang:1.23-alpine AS builder

RUN apk --no-cache add ca-certificates git

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

EXPOSE 8080

CMD ["./omie-sync-api"]
