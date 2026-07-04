param(
  [Parameter(Mandatory = $true)]
  [string]$BackupPath,
  [string]$DatabaseUrl = $env:DATABASE_URL
)

$ErrorActionPreference = "Stop"

if (-not $DatabaseUrl) {
  throw "DATABASE_URL is required. Pass -DatabaseUrl or set the DATABASE_URL environment variable."
}
if (-not (Test-Path $BackupPath)) {
  throw "Backup file not found: $BackupPath"
}

pg_restore --clean --if-exists --no-owner --no-acl --dbname $DatabaseUrl $BackupPath

Write-Host "PostgreSQL restore completed from: $BackupPath"
