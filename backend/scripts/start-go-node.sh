#!/bin/zsh

set -e

kill -9 $(lsof -ti tcp:2345) 2>/dev/null || true
kill -9 $(lsof -ti tcp:8080) 2>/dev/null || true
kill -9 $(lsof -ti tcp:3000) 2>/dev/null || true
rm -f /tmp/kabao-go-debug.log

(
  cd /Users/will/Projects/Go/kabao/backend
  nohup dlv debug . --headless --listen=127.0.0.1:2345 --api-version=2 --accept-multiclient --continue > /tmp/kabao-go-debug.log 2>&1 &
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

go_port=$(printf '%s\n' "$go_endpoint" | sed -E 's/.*:([0-9]+).*/\1/')
if [ -z "$go_port" ]; then
  echo "Go 服务端口解析失败"
  exit 1
fi

echo "Go 服务启动成功：127.0.0.1:${go_port}"
echo "  ➜  Local:   https://localhost:${go_port}"
echo "  ➜  Network: https://10.0.0.20:${go_port}"

export PATH=/Users/will/.nvm/versions/node/v16.20.2/bin:$PATH
cd /Users/will/Projects/Go/kabao/frontend
nohup npm run dev > /tmp/kabao-node-dev.log 2>&1 &

node_started=0
node_endpoint=""
for i in {1..10}; do
  if lsof -nP -iTCP:3000 -sTCP:LISTEN >/dev/null 2>&1; then
    node_endpoint=$(lsof -nP -iTCP:3000 -sTCP:LISTEN | awk 'NR==2 {print $9; exit}')
    node_started=1
    break
  fi
  sleep 1
done

if [ "$node_started" -ne 1 ]; then
  echo "Node 服务启动失败，日志如下："
  cat /tmp/kabao-node-dev.log
  exit 1
fi

node_port=$(printf '%s\n' "$node_endpoint" | sed -E 's/.*:([0-9]+).*/\1/')
if [ -z "$node_port" ]; then
  echo "Node 服务端口解析失败"
  exit 1
fi

echo "Node 服务启动成功：127.0.0.1:${node_port}"
echo "  ➜  Local:   https://localhost:${node_port}"
echo "  ➜  Network: https://10.0.0.20:${node_port}"
echo "所有服务已启动完成"
echo "Go 日志：/tmp/kabao-go-debug.log"
echo "Node 日志：/tmp/kabao-node-dev.log"

exit 0
