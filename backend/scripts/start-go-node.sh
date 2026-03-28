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
for i in {1..20}; do
  if lsof -nP -iTCP:8080 -sTCP:LISTEN >/dev/null 2>&1; then
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

echo "Go 服务启动成功：127.0.0.1:8080"

export PATH=/Users/will/.nvm/versions/node/v16.20.2/bin:$PATH
cd /Users/will/Projects/Go/kabao/frontend
exec npm run dev
