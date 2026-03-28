#!/bin/zsh

set -e

blue='\033[34m'
green='\033[32m'
reset='\033[0m'

zsh /Users/will/Projects/Go/kabao/backend/scripts/stop-go-node.sh
rm -f /tmp/kabao-go-debug.log /tmp/kabao-go-dlv.pid /tmp/kabao-go-bin.pid

(
  cd /Users/will/Projects/Go/kabao/backend
  nohup dlv debug . --headless --listen=127.0.0.1:2345 --api-version=2 --accept-multiclient --continue --output /tmp/__debug_bin > /tmp/kabao-go-debug.log 2>&1 &
  echo $! > /tmp/kabao-go-dlv.pid
)

started=0
go_endpoint=""
for i in {1..20}; do
  if lsof -nP -iTCP:8080 -sTCP:LISTEN >/dev/null 2>&1; then
    go_endpoint=$(lsof -nP -iTCP:8080 -sTCP:LISTEN | awk 'NR==2 {print $9; exit}')
    started=1
    break
  fi
  sleep 1
done

if [ "$started" -ne 1 ]; then
  echo "Go 服务启动失败，日志如下："
  cat /tmp/kabao-go-debug.log
  exit 1
fi

go_bin_pid=""
for i in {1..10}; do
  go_bin_pid=$(pgrep -P "$(cat /tmp/kabao-go-dlv.pid 2>/dev/null || true)" -f "__debug_bin" 2>/dev/null | head -n 1 || true)
  if [ -n "$go_bin_pid" ]; then
    echo "$go_bin_pid" > /tmp/kabao-go-bin.pid
    break
  fi
  sleep 1
done

go_port=$(printf '%s\n' "$go_endpoint" | sed -E 's/.*:([0-9]+).*/\1/')
if [ -z "$go_port" ]; then
  echo "Go 服务端口解析失败"
  exit 1
fi

echo "Go 服务启动成功：127.0.0.1:${go_port}"
printf "  ${green}➜${reset}  Local:   ${blue}https://localhost:%s${reset}\n" "$go_port"
printf "  ${green}➜${reset}  Network: ${blue}https://10.0.0.20:%s${reset}\n" "$go_port"
echo "Go 日志：/tmp/kabao-go-debug.log"

exit 0
