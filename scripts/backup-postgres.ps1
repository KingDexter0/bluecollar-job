param(
  [string]$DatabaseUrl = $env:DATABASE_URL,
  [string]$OutputDir = ".\backups"
)

$ErrorActionPreference = "Stop"

if (-not $DatabaseUrl) {
  throw "DATABASE_URL is required. Pass -DatabaseUrl or set the DATABASE_URL environment variable."
}

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$backupPath = Join-Path $OutputDir "bluecollarjob-$timestamp.dump"

pg_dump --format=custom --no-owner --no-acl --file $backupPath $DatabaseUrl

Write-Host "PostgreSQL backup created: $backupPath"
