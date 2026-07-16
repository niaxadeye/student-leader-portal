# Dev environment launcher for Windows (PowerShell).
#   Usage:  ./dev.ps1            - infra + api + worker + frontend
#           ./dev.ps1 -NoFront   - backend only (api + worker)
#           ./dev.ps1 -InfraOnly - docker only (Postgres/Redis/MinIO)
#
# Each process (api / worker / vite) opens in its own PowerShell window.
# To stop dev, just close those windows.
param(
  [switch]$NoFront,
  [switch]$InfraOnly
)

$ErrorActionPreference = 'Stop'
$Root = $PSScriptRoot

# --- load .env into current process environment ----------------------------
$envFile = Join-Path $Root '.env'
if (-not (Test-Path $envFile)) {
  Write-Error ".env not found in $Root (copy it from .env.example)"
}
Get-Content $envFile | ForEach-Object {
  $line = $_.Trim()
  if ($line -eq '' -or $line.StartsWith('#')) { return }
  $idx = $line.IndexOf('=')
  if ($idx -lt 1) { return }
  $name = $line.Substring(0, $idx).Trim()
  $value = $line.Substring($idx + 1).Trim()
  [Environment]::SetEnvironmentVariable($name, $value, 'Process')
}

# --- 1. infrastructure (Postgres/Redis/MinIO) ------------------------------
Write-Host '==> docker compose up -d' -ForegroundColor Cyan
docker compose up -d
if ($LASTEXITCODE -ne 0) { Write-Error 'docker compose failed' }

if ($InfraOnly) {
  Write-Host '==> Infrastructure ready (InfraOnly).' -ForegroundColor Green
  return
}

# --- 2. migrations ---------------------------------------------------------
Write-Host '==> go run ./cmd/admin migrate' -ForegroundColor Cyan
Push-Location (Join-Path $Root 'backend')
go run ./cmd/admin migrate
if ($LASTEXITCODE -ne 0) { Pop-Location; Write-Error 'migrations failed' }
Pop-Location

# --- 3. api and worker in separate windows ---------------------------------
# .env is already in this process env -> child windows inherit it.
$backend = Join-Path $Root 'backend'
Write-Host "==> starting api (port $($env:HTTP_PORT))" -ForegroundColor Cyan
Start-Process powershell -ArgumentList @(
  '-NoExit', '-Command',
  "cd '$backend'; Write-Host 'API' -ForegroundColor Green; go run ./cmd/api"
)
Write-Host '==> starting worker' -ForegroundColor Cyan
Start-Process powershell -ArgumentList @(
  '-NoExit', '-Command',
  "cd '$backend'; Write-Host 'WORKER' -ForegroundColor Green; go run ./cmd/worker"
)

# --- 4. frontend (vite) ----------------------------------------------------
if (-not $NoFront) {
  $frontend = Join-Path $Root 'frontend'
  Write-Host '==> starting frontend (vite, port 5173)' -ForegroundColor Cyan
  Start-Process powershell -ArgumentList @(
    '-NoExit', '-Command',
    "cd '$frontend'; Write-Host 'FRONTEND' -ForegroundColor Green; npm run dev"
  )
}

Write-Host ''
Write-Host 'Done. Opened windows: api, worker' -ForegroundColor Green
if (-not $NoFront) { Write-Host '  + frontend' -ForegroundColor Green }
Write-Host "API:      http://127.0.0.1:$($env:HTTP_PORT)"
Write-Host "Health:   http://127.0.0.1:$($env:HTTP_PORT)/health/ready"
if (-not $NoFront) { Write-Host 'Frontend: http://127.0.0.1:5173' }
