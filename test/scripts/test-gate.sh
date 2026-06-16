#!/usr/bin/env bash
set -euo pipefail
BIN="$(cd "$(dirname "$0")/../.." && pwd)/bin/cofiswarm-convert"
PORT=18015
"$BIN" -listen ":$PORT" &
PID=$!
trap 'kill $PID 2>/dev/null' EXIT
sleep 1
J=$(curl -s -X POST "http://127.0.0.1:$PORT/api/models/convert" \
  -H 'Content-Type: application/json' \
  -d '{"hf_repo":"test/model","output_name":"demo"}')
echo "$J" | grep -q job_id
ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['job_id'])" <<<"$J")
curl -s "http://127.0.0.1:$PORT/api/models/convert/$ID" | grep -q running
echo "ok: convert job queue API"
