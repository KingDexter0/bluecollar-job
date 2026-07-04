param(
  [string]$TargetRef = "HEAD~1",
  [string]$EnvFile = ".env.staging",
  [string]$ComposeFile = "docker-compose.prod.yml",
  [string]$HealthUrl = "http://localhost:8081/health"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $EnvFile)) {
  throw "Missing $EnvFile."
}

Write-Host "Rolling back to $TargetRef..."
git fetch --all --prune
git checkout $TargetRef

Write-Host "Rebuilding and restarting services..."
docker compose --env-file $EnvFile -f $ComposeFile build
docker compose --env-file $EnvFile -f $ComposeFile up -d

Write-Host "Checking health..."
Invoke-RestMethod $HealthUrl | Out-Null
Write-Host "Rollback complete."
