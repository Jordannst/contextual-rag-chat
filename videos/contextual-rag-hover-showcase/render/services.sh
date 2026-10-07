#!/usr/bin/env bash
# Start/stop the demo mock API (:5000) and the production Next.js app (:3000).
#   render/services.sh start|stop|restart-mock
set -euo pipefail
HERE="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(cd "$HERE/../.." && pwd)"
LOGS="${LOGS:-$HERE/qa/logs}"
mkdir -p "$LOGS"

port_pid() { (ss -ltnp 2>/dev/null | grep ":$1 " | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2) || true; }
stop_port() { local p; p="$(port_pid "$1")"; if [ -n "$p" ]; then kill "$p"; sleep 0.5; fi; }
start_mock() { (cd "$HERE" && nohup node mock/server.mjs > "$LOGS/mock.log" 2>&1 &); sleep 1; }
start_app() { (cd "$REPO" && nohup npx next start -p 3000 > "$LOGS/next.log" 2>&1 &); sleep 4; }

case "${1:-start}" in
  start) [ -n "$(port_pid 5000)" ] || start_mock; [ -n "$(port_pid 3000)" ] || start_app ;;
  stop) stop_port 5000; stop_port 3000 ;;
  restart-mock) stop_port 5000; start_mock ;;
esac
curl -s -o /dev/null -w "mock :5000 -> %{http_code}\n" http://localhost:5000/api/documents || true
curl -s -o /dev/null -w "app  :3000 -> %{http_code}\n" http://localhost:3000/ || true
