#!/bin/zsh

set -e

kill_pid() {
  local pid="$1"
  if [ -n "$pid" ]; then
    kill -9 "$pid" 2>/dev/null || true
  fi
}

kill_pids_by_port() {
  local port="$1"
  local pids
  pids=$(lsof -ti tcp:"$port" 2>/dev/null || true)
  for pid in $pids; do
    kill_pid "$pid"
  done
}

kill_pidfile() {
  local pidfile="$1"
  if [ -f "$pidfile" ]; then
    local pid
    pid=$(cat "$pidfile" 2>/dev/null || true)
    kill_pid "$pid"
    for child in $(pgrep -P "$pid" 2>/dev/null || true); do
      kill_pid "$child"
    done
    rm -f "$pidfile"
  fi
}

kill_pids_by_port 2345
kill_pids_by_port 8080
kill_pids_by_port 3000
kill_pidfile /tmp/kabao-go-dlv.pid
kill_pidfile /tmp/kabao-go-bin.pid
kill_pidfile /tmp/kabao-node-dev.pid

exit 0
