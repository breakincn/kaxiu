#!/bin/zsh

set -e

blue='\033[34m'
reset='\033[0m'

stopped_services=()
failed_services=()

service_label_for_port() {
  case "$1" in
    2345) echo "Go 调试服务" ;;
    8080) echo "Go 服务" ;;
    3000) echo "Node 服务" ;;
    *) echo "服务" ;;
  esac
}

record_stopped_service() {
  local label="$1"
  local port="$2"
  local entry="$label:$port"
  local seen
  for seen in "${stopped_services[@]}"; do
    if [ "$seen" = "$entry" ]; then
      return 0
    fi
  done
  stopped_services+=("$entry")
}

record_failed_service() {
  local label="$1"
  local port="$2"
  local entry="$label:$port"
  local seen
  for seen in "${failed_services[@]}"; do
    if [ "$seen" = "$entry" ]; then
      return 0
    fi
  done
  failed_services+=("$entry")
}

kill_pid() {
  local pid="$1"
  if [ -n "$pid" ]; then
    kill -9 "$pid" 2>/dev/null
  fi
}

kill_pids_by_port() {
  local port="$1"
  local pids
  local label
  label="$(service_label_for_port "$port")"
  pids=$(lsof -ti tcp:"$port" 2>/dev/null || true)
  for pid in $pids; do
    if kill_pid "$pid"; then
      record_stopped_service "$label" "$port"
    else
      record_failed_service "$label" "$port"
    fi
  done
}

for i in {1..3}; do
  kill_pids_by_port 2345
  kill_pids_by_port 8080
  kill_pids_by_port 3000
  if ! lsof -ti tcp:2345 >/dev/null 2>&1 && ! lsof -ti tcp:8080 >/dev/null 2>&1 && ! lsof -ti tcp:3000 >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

remaining_ports=()
for port in 2345 8080 3000; do
  if lsof -ti tcp:$port >/dev/null 2>&1; then
    remaining_ports+=("$port")
  fi
done

if [ "${#stopped_services[@]}" -gt 0 ]; then
  echo "已停止以下服务："
  for item in "${stopped_services[@]}"; do
    echo "  - $item"
  done
fi

if [ "${#failed_services[@]}" -gt 0 ]; then
  echo "以下服务未停止，当前环境对目标进程没有 kill 权限："
  for item in "${failed_services[@]}"; do
    echo "  - $item"
  done
fi

if [ "${#remaining_ports[@]}" -gt 0 ]; then
  echo "仍在监听的端口：${(j:,:)remaining_ports}"
  manual_kill_pids=()
  for port in "${remaining_ports[@]}"; do
    for pid in $(lsof -ti tcp:$port 2>/dev/null || true); do
      manual_kill_pids+=("$pid")
    done
  done
  if [ "${#manual_kill_pids[@]}" -gt 0 ]; then
    manual_kill_command=""
    for port in "${remaining_ports[@]}"; do
      if [ -n "$manual_kill_command" ]; then
        manual_kill_command="${manual_kill_command}; "
      fi
      manual_kill_command="${manual_kill_command}kill -9 \$(lsof -ti tcp:${port})"
    done
    echo "手工执行：${blue}${manual_kill_command}${reset}"
  fi
fi

if [ "${#stopped_services[@]}" -eq 0 ] && [ "${#failed_services[@]}" -eq 0 ]; then
  echo "未发现正在运行的 go + node 服务"
fi

exit 0
