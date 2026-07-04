param(
  [string]$ApiBaseUrl = "http://localhost:8081",
  [string]$AdminToken = "local-admin-token"
)

$ErrorActionPreference = "Stop"

function Invoke-Json($Method, $Path, $Body = $null, $Token = $null, $AdminTokenValue = $null) {
  $headers = @{ "Content-Type" = "application/json" }
  if ($Token) { $headers.Authorization = "Bearer $Token" }
  if ($AdminTokenValue) { $headers["X-Admin-Token"] = $AdminTokenValue }
  $params = @{
    Method = $Method
    Uri = "$ApiBaseUrl$Path"
    Headers = $headers
  }
  if ($Body) { $params.Body = ($Body | ConvertTo-Json -Depth 8) }
  Invoke-RestMethod @params
}

$suffix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$phoneSuffix = $suffix.ToString().Substring($suffix.ToString().Length - 8)

Write-Host "1. Checking API readiness"
Invoke-Json GET "/ready" | Out-Null

Write-Host "2. Registering demo employer"
$employer = Invoke-Json POST "/api/v1/employers/register" @{
  company_name = "Demo Factory $suffix"
  contact_name = "Demo Manager"
  email = "demo.$suffix@example.com"
  password = "DemoPass12345!"
  city = "Pune"
  state = "Maharashtra"
}
$token = $employer.token

Write-Host "3. Creating demo job"
$job = Invoke-Json POST "/api/v1/employer/jobs" @{
  title = "Assembly Operator $suffix"
  role = "Assembly Operator"
  description = "Demo-ready factory role"
  skill_category = "Manufacturing"
  location_city = "Pune"
  location_state = "Maharashtra"
  shift_schedule = "Day shift"
  openings = 3
  required_verification_tier = "Low"
  is_active = $true
} $token

Write-Host "4. Creating worker"
$worker = Invoke-Json POST "/api/v1/workers" @{
  phone_number = "+9188$phoneSuffix"
  full_name = "Demo Worker"
  language_preference = "en"
  target_role = "Assembly Operator"
  preferred_zone = "Pune"
}

Write-Host "5. Skipping identity verification for demo worker"
Invoke-Json POST "/api/v1/workers/$($worker.worker.id)/identity/skip" @{
  reason = "demo flow"
} | Out-Null

Write-Host "6. Applying worker to job"
$application = Invoke-Json POST "/api/v1/applications" @{
  user_id = $worker.worker.id
  job_id = $job.job.id
  source = "demo_script"
}

Write-Host "7. Viewing ATS"
Invoke-Json GET "/api/v1/employer/applications" $null $token | Out-Null

Write-Host "8. Scheduling direct interview"
$start = (Get-Date).ToUniversalTime().AddDays(1).Date.AddHours(10).ToString("o")
$end = (Get-Date).ToUniversalTime().AddDays(1).Date.AddHours(10).AddMinutes(30).ToString("o")
Invoke-Json POST "/api/v1/employer/applications/$($application.application.id)/interview/direct" @{
  starts_at = $start
  ends_at = $end
  timezone = "Asia/Kolkata"
  factory_location = "Demo Factory Gate 1"
  google_maps_url = "https://maps.google.com/?q=Demo+Factory"
} $token | Out-Null

Write-Host "9. Processing pending notifications"
Invoke-Json POST "/api/v1/dev/notifications/process-once" @{ limit = 10 } | Out-Null

Write-Host "10. Checking admin summary and processing mock payouts"
Invoke-Json GET "/api/v1/admin/summary" $null $null $AdminToken | Out-Null
Invoke-Json POST "/api/v1/admin/referrals/process-payouts" @{ limit = 20 } $null $AdminToken | Out-Null

Write-Host ""
Write-Host "Demo flow complete"
Write-Host "Employer email: $($employer.employer.email)"
Write-Host "Employer password: DemoPass12345!"
Write-Host "Worker phone: $($worker.worker.phone_number)"
Write-Host "Job ID: $($job.job.id)"
Write-Host "Application ID: $($application.application.id)"
