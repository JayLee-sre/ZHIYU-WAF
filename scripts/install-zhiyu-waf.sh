#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="zhiyu-waf"
INSTALL_DIR="/opt/zhiyu-waf"
SERVICE_NAME="zhiyu-waf"
BACKEND_ADDR="127.0.0.1:80"
WAF_PORT="8080"
DASHBOARD_PORT="9090"
JWT_SECRET=""
BUILD_BACKEND="auto"
BUILD_FRONTEND="auto"
OPEN_FIREWALL="false"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'USAGE'
ZHIYU-WAF V2 安装/更新脚本

用法：
  sudo bash scripts/install-zhiyu-waf.sh [选项]

选项：
  --install-dir /opt/zhiyu-waf         安装目录
  --backend 127.0.0.1:3000             真实业务回源地址
  --waf-port 8080                      WAF 反向代理监听端口
  --dashboard-port 9090                管理控制台端口
  --jwt-secret VALUE                   控制台 JWT 密钥；未设置时自动生成
  --open-firewall                      尝试放行 WAF 与 Dashboard 端口
  --build-backend true|false|auto      是否编译后端，默认 auto
  --build-frontend true|false|auto     是否编译前端，默认 auto

V2 不执行 iptables/ip6tables 端口接管。请让 Nginx、负载均衡器或服务网关
显式将业务流量转发至 WAF 监听端口。内核级恶意 IP 封禁使用独立 nftables 表，
服务需具备 CAP_NET_ADMIN；没有该权限时应用层防护仍会继续工作。
USAGE
}

log() { printf '\033[1;34m[ZHIYU-WAF V2]\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m[ZHIYU-WAF V2] ERROR:\033[0m %s\n' "$*" >&2; exit 1; }
need_root() { [[ "${EUID}" -eq 0 ]] || die "请使用 root 执行，例如 sudo bash scripts/install-zhiyu-waf.sh"; }
command_exists() { command -v "$1" >/dev/null 2>&1; }
rand_secret() { if command_exists openssl; then openssl rand -hex 32; else tr -dc 'A-Za-z0-9' </dev/urandom | head -c 64; fi; }

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --install-dir) INSTALL_DIR="$2"; shift 2 ;;
      --backend) BACKEND_ADDR="$2"; shift 2 ;;
      --waf-port) WAF_PORT="$2"; shift 2 ;;
      --dashboard-port) DASHBOARD_PORT="$2"; shift 2 ;;
      --jwt-secret) JWT_SECRET="$2"; shift 2 ;;
      --open-firewall) OPEN_FIREWALL="true"; shift ;;
      --build-backend) BUILD_BACKEND="$2"; shift 2 ;;
      --build-frontend) BUILD_FRONTEND="$2"; shift 2 ;;
      -h|--help) usage; exit 0 ;;
      *) die "未知参数：$1" ;;
    esac
  done
}

validate_inputs() {
  [[ "${BACKEND_ADDR}" == *:* ]] || die "--backend 必须是 host:port，例如 127.0.0.1:3000"
  [[ "${WAF_PORT}" =~ ^[0-9]+$ ]] || die "--waf-port 必须是数字"
  [[ "${DASHBOARD_PORT}" =~ ^[0-9]+$ ]] || die "--dashboard-port 必须是数字"
  [[ -n "${JWT_SECRET}" ]] || JWT_SECRET="$(rand_secret)"
}

build_backend_if_needed() {
  local target="${ROOT_DIR}/bin/${APP_NAME}" should_build="${BUILD_BACKEND}"
  if [[ "${should_build}" == "auto" ]]; then [[ -x "${target}" ]] && should_build="false" || should_build="true"; fi
  if [[ "${should_build}" == "true" ]]; then
    command_exists go || die "未找到 go，无法编译后端"
    command_exists gcc || die "未找到 gcc；SQLite 构建需要 C 编译器"
    log "编译后端…"
    (cd "${ROOT_DIR}" && CGO_ENABLED=1 go build -o "${target}" ./cmd/zhiyu-waf)
  fi
  [[ -x "${target}" ]] || die "未找到可执行文件：${target}"
}

build_frontend_if_needed() {
  local dist="${ROOT_DIR}/web/dist/index.html" should_build="${BUILD_FRONTEND}"
  if [[ "${should_build}" == "auto" ]]; then [[ -f "${dist}" ]] && should_build="false" || should_build="true"; fi
  if [[ "${should_build}" == "true" ]]; then
    command_exists npm || die "未找到 npm，无法编译前端"
    log "编译前端…"
    (cd "${ROOT_DIR}/web" && npm ci && npm run build:raw)
  fi
  [[ -f "${dist}" ]] || die "未找到前端构建目录：${ROOT_DIR}/web/dist"
  rm -rf "${ROOT_DIR}/internal/dashboard/dist"
  cp -R "${ROOT_DIR}/web/dist" "${ROOT_DIR}/internal/dashboard/dist"
}

backup_existing() {
  [[ -d "${INSTALL_DIR}" ]] || return
  local backup="${INSTALL_DIR}/backups/install.$(date +%Y%m%d%H%M%S)"
  log "备份当前安装到 ${backup}"
  mkdir -p "${backup}"
  [[ -f "${INSTALL_DIR}/bin/${APP_NAME}" ]] && cp -a "${INSTALL_DIR}/bin/${APP_NAME}" "${backup}/${APP_NAME}" || true
  [[ -f "${INSTALL_DIR}/configs/zhiyu-waf.yaml" ]] && cp -a "${INSTALL_DIR}/configs/zhiyu-waf.yaml" "${backup}/zhiyu-waf.yaml" || true
}

install_files() {
  mkdir -p "${INSTALL_DIR}/bin" "${INSTALL_DIR}/configs/rules" "${INSTALL_DIR}/data" "${INSTALL_DIR}/logs" "${INSTALL_DIR}/web"
  install -m 0755 "${ROOT_DIR}/bin/${APP_NAME}" "${INSTALL_DIR}/bin/${APP_NAME}"
  cp -a "${ROOT_DIR}/configs/rules/." "${INSTALL_DIR}/configs/rules/"
  rm -rf "${INSTALL_DIR}/web/dist"
  cp -a "${ROOT_DIR}/web/dist" "${INSTALL_DIR}/web/dist"
}

write_config() {
  cat >"${INSTALL_DIR}/configs/zhiyu-waf.yaml" <<EOF
proxy:
  listen_addr: ":${WAF_PORT}"
  backend_addr: "${BACKEND_ADDR}"
  trusted_proxies:
    - "127.0.0.1/32"
    - "::1/128"
  read_timeout: 30
  write_timeout: 30

dashboard:
  listen_addr: ":${DASHBOARD_PORT}"
  jwt_secret: "${JWT_SECRET}"
  cors_origins:
    - "http://127.0.0.1:${DASHBOARD_PORT}"

ai:
  enabled: false
  provider: "openai"
  fail_open: true
  providers:
    openai:
      api_key: ""
      model: ""
      base_url: ""

engine:
  rules_dir: "${INSTALL_DIR}/configs/rules"
  rate_limit:
    requests_per_minute: 60
    burst_size: 10

ssh:
  enabled: true
  max_fails: 5
  ban_minutes: 30

storage:
  path: "${INSTALL_DIR}/data/zhiyu-waf.db"
EOF
}

write_systemd() {
  cat >/etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=ZHIYU-WAF V2 local-first Web Application Firewall
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/bin/${APP_NAME} -config ${INSTALL_DIR}/configs/zhiyu-waf.yaml
Restart=always
RestartSec=5
# Enable this capability only when host nftables synchronization is desired.
AmbientCapabilities=CAP_NET_ADMIN
CapabilityBoundingSet=CAP_NET_ADMIN
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF
}

configure_firewall() {
  [[ "${OPEN_FIREWALL}" == "true" ]] || return
  if command_exists firewall-cmd && systemctl is-active firewalld >/dev/null 2>&1; then
    firewall-cmd --permanent --add-port="${WAF_PORT}/tcp" >/dev/null
    firewall-cmd --permanent --add-port="${DASHBOARD_PORT}/tcp" >/dev/null
    firewall-cmd --reload >/dev/null
  elif command_exists ufw && ufw status >/dev/null 2>&1; then
    ufw allow "${WAF_PORT}/tcp" >/dev/null
    ufw allow "${DASHBOARD_PORT}/tcp" >/dev/null
  else
    log "未检测到已启用的 firewalld/ufw，跳过端口放行"
  fi
}

start_service() {
  systemctl daemon-reload
  systemctl enable "${SERVICE_NAME}" >/dev/null
  systemctl restart "${SERVICE_NAME}"
  sleep 2
  systemctl is-active --quiet "${SERVICE_NAME}" || { systemctl status "${SERVICE_NAME}" --no-pager -l || true; die "服务启动失败"; }
}

verify_install() {
  curl -fsS "http://127.0.0.1:${DASHBOARD_PORT}/health" >/dev/null || die "控制台健康检查失败"
  log "安装完成"
  echo "控制台地址： http://服务器IP:${DASHBOARD_PORT}/"
  echo "WAF 入口：   由上游代理转发至服务器IP:${WAF_PORT}"
  echo "配置文件：   ${INSTALL_DIR}/configs/zhiyu-waf.yaml"
  echo "服务状态：   systemctl status ${SERVICE_NAME} --no-pager -l"
}

main() {
  parse_args "$@"; need_root; validate_inputs; build_frontend_if_needed; build_backend_if_needed
  backup_existing; install_files; write_config; write_systemd; configure_firewall; start_service; verify_install
}
main "$@"
