param(
  [string]$EnvFile = ".env.staging",
  [string]$ComposeFile = "docker-compose.prod.yml",
  [string]$SmokeBaseUrl = "http://localhost:8081"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $EnvFile)) {
  throw "Missing $EnvFile. Copy .env.staging.example and fill real staging values."
}

Write-Host "Pulling latest code..."
git pull --ff-only

Write-Host "Building staging containers..."
docker compose --env-file $EnvFile -f $ComposeFile build

Write-Host "Running migrations..."
docker compose --env-file $EnvFile -f $ComposeFile run --rm api /app/migrate up

Write-Host "Starting services..."
docker compose --env-file $EnvFile -f $ComposeFile up -d

Write-Host "Waiting for API readiness..."
for ($i = 0; $i -lt 30; $i++) {
  try {
    Invoke-RestMethod "$SmokeBaseUrl/ready" | Out-Null
    break
  } catch {
    Start-Sleep -Seconds 2
  }
}

Write-Host "Running smoke test..."
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 -ApiBaseUrl $SmokeBaseUrl

Write-Host "Staging deployment complete."
