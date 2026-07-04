param(
  [string]$ApiBaseUrl = "http://localhost:8081"
)

$ErrorActionPreference = "Stop"

function Invoke-Json($Method, $Path, $Body = $null, $Token = $null) {
  $headers = @{ "Content-Type" = "application/json" }
  if ($Token) { $headers.Authorization = "Bearer $Token" }
  $params = @{
    Method = $Method
    Uri = "$ApiBaseUrl$Path"
    Headers = $headers
  }
  if ($Body) { $params.Body = ($Body | ConvertTo-Json -Depth 8) }
  Invoke-RestMethod @params
}

Write-Host "Checking health..."
Invoke-Json GET "/health" | Out-Null
Invoke-Json GET "/ready" | Out-Null
Invoke-Json GET "/live" | Out-Null

$suffix = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$employer = Invoke-Json POST "/api/v1/employers/register" @{
  company_name = "Smoke Factory $suffix"
  contact_name = "Smoke Manager"
  email = "smoke.$suffix@example.com"
  password = "SmokePass12345!"
  city = "Pune"
  state = "Maharashtra"
}
$token = $employer.token

$job = Invoke-Json POST "/api/v1/employer/jobs" @{
  title = "Smoke Operator $suffix"
  role = "Machine Operator"
  description = "Smoke test job"
  skill_category = "Manufacturing"
  location_city = "Pune"
  location_state = "Maharashtra"
  shift_schedule = "Day shift"
  openings = 1
  required_verification_tier = "Low"
  is_active = $true
} $token

$worker = Invoke-Json POST "/api/v1/workers" @{
  phone_number = "+9199$($suffix.ToString().Substring($suffix.ToString().Length - 8))"
  full_name = "Smoke Worker"
  language_preference = "en"
  target_role = "Machine Operator"
  preferred_zone = "Pune"
}

$application = Invoke-Json POST "/api/v1/applications" @{
  user_id = $worker.worker.id
  job_id = $job.job.id
  source = "smoke_test"
}

Invoke-Json GET "/api/v1/employer/applications" $null $token | Out-Null

$start = (Get-Date).ToUniversalTime().AddDays(1).Date.AddHours(10).ToString("o")
$end = (Get-Date).ToUniversalTime().AddDays(1).Date.AddHours(10).AddMinutes(30).ToString("o")
Invoke-Json POST "/api/v1/employer/applications/$($application.application.id)/interview/direct" @{
  starts_at = $start
  ends_at = $end
  timezone = "Asia/Kolkata"
  factory_location = "Smoke Plant"
  google_maps_url = "https://maps.google.com/?q=Smoke+Plant"
} $token | Out-Null

Write-Host "Smoke test passed."
