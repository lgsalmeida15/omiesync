# Script para iniciar a API em modo desenvolvimento
# As variáveis são carregadas automaticamente do arquivo .env pelo código Go,
# mas este script garante que o ambiente esteja configurado corretamente.

$env:APP_ENV="development"

Write-Host "Iniciando Omie Sync API..." -ForegroundColor Cyan
go run cmd/api/main.go
