#!/bin/zsh
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BACKEND_DIR="${PROJECT_ROOT}/backend"
CHANNEL="wecom"
INSTALL_CRON=0
INSTALL_SUPERVISOR=0

usage() {
  cat <<EOF
Usage:
  $0 --all [--channel wecom|feishu]
  $0 --install-cron [--channel wecom|feishu]
  $0 --install-supervisor

Options:
  --all                 Install both cron and supervisor configs
  --install-cron        Install cron config to /etc/cron.d/kabao_scheduler_health
  --install-supervisor  Install supervisor config to /etc/supervisor/conf.d/kabao_services.conf
  --channel             Alert channel for cron failure hook, default: wecom
  --project-root        Override project root path
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --all)
      INSTALL_CRON=1
      INSTALL_SUPERVISOR=1
      shift
      ;;
    --install-cron)
      INSTALL_CRON=1
      shift
      ;;
    --install-supervisor)
      INSTALL_SUPERVISOR=1
      shift
      ;;
    --channel)
      CHANNEL="${2:-}"
      shift 2
      ;;
    --project-root)
      PROJECT_ROOT="${2:-}"
      BACKEND_DIR="${PROJECT_ROOT}/backend"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ "${INSTALL_CRON}" -eq 0 && "${INSTALL_SUPERVISOR}" -eq 0 ]]; then
  usage >&2
  exit 1
fi

if [[ "${CHANNEL}" != "wecom" && "${CHANNEL}" != "feishu" ]]; then
  echo "Unsupported channel: ${CHANNEL}" >&2
  exit 1
fi

mkdir -p /tmp/kabao-go-build-cache

render_cron() {
  cat <<EOF
SHELL=/bin/zsh
PATH=/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin

*/2 * * * * root cd ${BACKEND_DIR} && GOCACHE=/tmp/kabao-go-build-cache go run ./cmd/check_scheduler_health --stale-minutes=3 >> /tmp/kabao_scheduler_health.log 2>&1 || ${BACKEND_DIR}/scripts/alert_scheduler_health.sh ${CHANNEL} >> /tmp/kabao_scheduler_health_alert.log 2>&1
EOF
}

render_supervisor() {
  cat <<EOF
[program:kabao_merchant_service]
directory=${BACKEND_DIR}
command=/bin/zsh -lc 'GOCACHE=/tmp/kabao-go-build-cache KABAO_SERVICE_NAME=merchant_service go run ./cmd/merchant_service'
autostart=true
autorestart=true
startsecs=5
stopasgroup=true
killasgroup=true
stdout_logfile=/tmp/kabao_merchant_service.log
stderr_logfile=/tmp/kabao_merchant_service.err.log
environment=KABAO_ENV="production"

[program:kabao_user_service]
directory=${BACKEND_DIR}
command=/bin/zsh -lc 'GOCACHE=/tmp/kabao-go-build-cache KABAO_SERVICE_NAME=user_service go run ./cmd/user_service'
autostart=true
autorestart=true
startsecs=5
stopasgroup=true
killasgroup=true
stdout_logfile=/tmp/kabao_user_service.log
stderr_logfile=/tmp/kabao_user_service.err.log
environment=KABAO_ENV="production"

[program:kabao_admin_service]
directory=${BACKEND_DIR}
command=/bin/zsh -lc 'GOCACHE=/tmp/kabao-go-build-cache KABAO_SERVICE_NAME=admin_service go run ./cmd/admin_service'
autostart=true
autorestart=true
startsecs=5
stopasgroup=true
killasgroup=true
stdout_logfile=/tmp/kabao_admin_service.log
stderr_logfile=/tmp/kabao_admin_service.err.log
environment=KABAO_ENV="production"
EOF
}

if [[ "${INSTALL_CRON}" -eq 1 ]]; then
  render_cron | sudo tee /etc/cron.d/kabao_scheduler_health >/dev/null
  sudo chmod 644 /etc/cron.d/kabao_scheduler_health
  echo "Installed cron config: /etc/cron.d/kabao_scheduler_health"
fi

if [[ "${INSTALL_SUPERVISOR}" -eq 1 ]]; then
  render_supervisor | sudo tee /etc/supervisor/conf.d/kabao_services.conf >/dev/null
  sudo chmod 644 /etc/supervisor/conf.d/kabao_services.conf
  echo "Installed supervisor config: /etc/supervisor/conf.d/kabao_services.conf"
  echo "Next:"
  echo "  sudo supervisorctl reread"
  echo "  sudo supervisorctl update"
  echo "  sudo supervisorctl status"
fi
