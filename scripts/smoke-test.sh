#!/usr/bin/env bash
set -euo pipefail

API_BASE_URL="${1:-http://localhost:8081}"
suffix="$(date +%s)"
phone_suffix="${suffix: -8}"

json_field() {
  python3 -c 'import json,sys; data=json.load(sys.stdin); cur=data; 
for part in sys.argv[1].split("."):
    cur=cur[part]
print(cur)' "$1"
}

request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local token="${4:-}"
  local admin_token="${5:-}"
  local args=(-fsS -X "$method" "$API_BASE_URL$path" -H "Content-Type: application/json")
  if [[ -n "$token" ]]; then args+=(-H "Authorization: Bearer $token"); fi
  if [[ -n "$admin_token" ]]; then args+=(-H "X-Admin-Token: $admin_token"); fi
  if [[ -n "$body" ]]; then args+=(-d "$body"); fi
  curl "${args[@]}"
}

echo "Checking probes..."
request GET /health >/dev/null
request GET /ready >/dev/null
request GET /live >/dev/null

echo "Registering employer..."
employer_json="$(request POST /api/v1/employers/register "{\"company_name\":\"Smoke Factory $suffix\",\"contact_name\":\"Smoke Manager\",\"email\":\"smoke.$suffix@example.com\",\"password\":\"SmokePass12345!\",\"city\":\"Pune\",\"state\":\"Maharashtra\"}")"
token="$(printf '%s' "$employer_json" | json_field token)"

echo "Creating job..."
job_json="$(request POST /api/v1/employer/jobs "{\"title\":\"Smoke Operator $suffix\",\"role\":\"Machine Operator\",\"description\":\"Smoke test job\",\"skill_category\":\"Manufacturing\",\"location_city\":\"Pune\",\"location_state\":\"Maharashtra\",\"shift_schedule\":\"Day shift\",\"openings\":1,\"required_verification_tier\":\"Low\",\"is_active\":true}" "$token")"
job_id="$(printf '%s' "$job_json" | json_field job.id)"

echo "Creating worker..."
worker_json="$(request POST /api/v1/workers "{\"phone_number\":\"+9199$phone_suffix\",\"full_name\":\"Smoke Worker\",\"language_preference\":\"en\",\"target_role\":\"Machine Operator\",\"preferred_zone\":\"Pune\"}")"
worker_id="$(printf '%s' "$worker_json" | json_field worker.id)"

echo "Applying to job..."
application_json="$(request POST /api/v1/applications "{\"user_id\":\"$worker_id\",\"job_id\":\"$job_id\",\"source\":\"smoke_test\"}")"
application_id="$(printf '%s' "$application_json" | json_field application.id)"

request GET /api/v1/employer/applications "" "$token" >/dev/null

start="$(python3 -c 'from datetime import datetime, timezone, timedelta; print((datetime.now(timezone.utc)+timedelta(days=1)).replace(hour=10, minute=0, second=0, microsecond=0).isoformat().replace("+00:00","Z"))')"
end="$(python3 -c 'from datetime import datetime, timezone, timedelta; print((datetime.now(timezone.utc)+timedelta(days=1)).replace(hour=10, minute=30, second=0, microsecond=0).isoformat().replace("+00:00","Z"))')"

echo "Scheduling interview..."
request POST "/api/v1/employer/applications/$application_id/interview/direct" "{\"starts_at\":\"$start\",\"ends_at\":\"$end\",\"timezone\":\"Asia/Kolkata\",\"factory_location\":\"Smoke Plant\",\"google_maps_url\":\"https://maps.google.com/?q=Smoke+Plant\"}" "$token" >/dev/null

echo "Smoke test passed."
