#!/bin/zsh
set -euo pipefail

CHANNEL="${1:-wecom}"
HOSTNAME="$(hostname)"
NOW="$(date '+%Y-%m-%d %H:%M:%S')"
CHECK_CMD="cd /Users/will/Projects/Go/kabao/backend && GOCACHE=/tmp/kabao-go-build-cache go run ./cmd/check_scheduler_health --stale-minutes=3"
TEXT="Kabao 调度器健康检查异常
主机: ${HOSTNAME}
时间: ${NOW}
检查命令:
${CHECK_CMD}"

send_wecom() {
  : "${WECOM_WEBHOOK_URL:?WECOM_WEBHOOK_URL is required}"
  curl -sS -X POST "${WECOM_WEBHOOK_URL}" \
    -H 'Content-Type: application/json' \
    -d "{
      \"msgtype\": \"text\",
      \"text\": {
        \"content\": \"${TEXT}\"
      }
    }"
}

send_feishu() {
  : "${FEISHU_WEBHOOK_URL:?FEISHU_WEBHOOK_URL is required}"
  curl -sS -X POST "${FEISHU_WEBHOOK_URL}" \
    -H 'Content-Type: application/json' \
    -d "{
      \"msg_type\": \"text\",
      \"content\": {
        \"text\": \"${TEXT}\"
      }
    }"
}

case "${CHANNEL}" in
  wecom)
    send_wecom
    ;;
  feishu)
    send_feishu
    ;;
  *)
    echo "unknown channel: ${CHANNEL}" >&2
    echo "usage: $0 [wecom|feishu]" >&2
    exit 1
    ;;
esac
