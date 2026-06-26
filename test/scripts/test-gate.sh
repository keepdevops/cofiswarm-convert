#!/usr/bin/env bash
set -euo pipefail
BIN="$(cd "$(dirname "$0")/../.." && pwd)/bin/cofiswarm-convert"
PORT=18015

# Hermetic converter: a fake script emitting the NDJSON progress contract so the
# gate exercises the real spawn + parse + status path without Python/mlx_lm.
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"; kill "${PID:-0}" 2>/dev/null || true' EXIT
FAKE="${WORK}/fake_convert.sh"
cat > "$FAKE" <<'SH'
#!/bin/sh
echo '{"status":"running","step":"downloading_and_converting","pct":5}'
echo '{"status":"done","step":"done","pct":100,"output":"/tmp/demo"}'
SH
chmod +x "$FAKE"

export COFISWARM_CONVERT_PYTHON="/bin/sh"
export COFISWARM_CONVERT_SCRIPT="$FAKE"
export COFISWARM_CONVERT_OUTPUT_DIR="$WORK"

"$BIN" -listen ":$PORT" &
PID=$!
sleep 1

J=$(curl -s -X POST "http://127.0.0.1:$PORT/api/models/convert" \
  -H 'Content-Type: application/json' \
  -d '{"hf_repo":"test/model","output_name":"demo"}')
echo "$J" | grep -q job_id
ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['job_id'])" <<<"$J")

# Poll until the worker reaches a terminal state (real execution, not a stub).
STATUS=""
for _ in $(seq 1 50); do
  STATUS=$(curl -s "http://127.0.0.1:$PORT/api/models/convert/$ID" \
    | python3 -c "import json,sys; print(json.load(sys.stdin)['status'])")
  case "$STATUS" in done|error) break;; esac
  sleep 0.1
done
[ "$STATUS" = "done" ] || { echo "FAIL: job ended in status '$STATUS', want done"; exit 1; }
echo "ok: convert job runs end-to-end (status=$STATUS)"
