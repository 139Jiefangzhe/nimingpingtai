#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REMOTE_HOST="${REMOTE_HOST:-root@47.94.135.253}"
REMOTE_PORT="${REMOTE_PORT:-22}"
REMOTE_DIR="${REMOTE_DIR:-/opt/niming-community-predeploy}"
PROJECT_NAME="${PROJECT_NAME:-niming-community-predeploy}"
COMPOSE_FILE="${COMPOSE_FILE:-deploy/docker-compose.community.predeploy.yml}"
ENV_TEMPLATE="${ENV_TEMPLATE:-deploy/community.predeploy.env.example}"
ANSWER_IMAGE="${ANSWER_IMAGE:-niming-answer-app}"
VAULT_IMAGE="${VAULT_IMAGE:-niming-vault-service}"
GO_PROXY="${GO_PROXY:-https://goproxy.cn,direct}"
GO_SUMDB="${GO_SUMDB:-off}"
NPM_REGISTRY="${NPM_REGISTRY:-https://registry.npmmirror.com}"
ALPINE_REPO="${ALPINE_REPO:-https://mirrors.tuna.tsinghua.edu.cn/alpine}"

random_alnum() {
  local length="$1"
  local value=""
  while ((${#value} < length)); do
    set +o pipefail
    value+=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c "${length}")
    set -o pipefail
  done
  printf '%s' "${value:0:length}"
}

git_dirty_suffix() {
  if git -C "${ROOT_DIR}" diff --quiet --ignore-submodules -- && git -C "${ROOT_DIR}" diff --cached --quiet --ignore-submodules --; then
    return
  fi
  printf '%s' "-dirty"
}

DEPLOY_TAG="${DEPLOY_TAG:-predeploy-$(date +%Y%m%d)-$(git -C "${ROOT_DIR}" rev-parse --short HEAD)$(git_dirty_suffix)}"
ENV_FILE_INPUT="${1:-${ENV_FILE:-}}"

log() {
  printf '[deploy] %s\n' "$*"
}

die() {
  printf '[deploy] %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [[ "${GENERATED_ENV_FILE:-0}" == "1" && -n "${WORK_ENV_FILE:-}" && -f "${WORK_ENV_FILE}" ]]; then
    rm -f "${WORK_ENV_FILE}"
  fi
  if [[ -n "${REMOTE_ENV_FILE:-}" && -f "${REMOTE_ENV_FILE}" ]]; then
    rm -f "${REMOTE_ENV_FILE}"
  fi
}

trap cleanup EXIT

require_tool() {
  command -v "$1" >/dev/null 2>&1 || die "missing required tool: $1"
}

ssh_remote() {
  ssh -p "${REMOTE_PORT}" "${REMOTE_HOST}" "$@"
}

compose_remote() {
  local subcommand="$1"
  ssh_remote "cd '${REMOTE_DIR}' && docker compose -p '${PROJECT_NAME}' --env-file .env -f '${REMOTE_DIR}/$(basename "${COMPOSE_FILE}")' ${subcommand}"
}

wait_for_health() {
  local container_name="$1"
  local attempts="${2:-90}"
  local delay="${3:-2}"
  local i
  for ((i=1; i<=attempts; i++)); do
    local status
    status="$(ssh_remote "docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' '${container_name}' 2>/dev/null || true")"
    if [[ "${status}" == "healthy" ]]; then
      return 0
    fi
    sleep "${delay}"
  done
  return 1
}

prepare_env_file() {
  if [[ -n "${ENV_FILE_INPUT}" ]]; then
    [[ -f "${ENV_FILE_INPUT}" ]] || die "env file not found: ${ENV_FILE_INPUT}"
    GENERATED_ENV_FILE=0
    WORK_ENV_FILE="${ENV_FILE_INPUT}"
    log "using env file: ${WORK_ENV_FILE}"
    return
  fi

  GENERATED_ENV_FILE=1
  WORK_ENV_FILE="$(mktemp)"
  local main_db_password
  local vault_db_password
  local admin_password
  local wecom_callback_token
  local wecom_callback_aes_key
  local vault_shared_token
  local vault_secret

  main_db_password="$(random_alnum 24)"
  vault_db_password="$(random_alnum 24)"
  admin_password="$(random_alnum 20)"
  wecom_callback_token="$(random_alnum 24)"
  wecom_callback_aes_key="$(random_alnum 43)"
  vault_shared_token="$(random_alnum 32)"
  vault_secret="$(random_alnum 32)"

  cat >"${WORK_ENV_FILE}" <<EOF
DEPLOY_TAG=${DEPLOY_TAG}
AUTO_INSTALL=true
ANSWER_BIND_PORT=9080
DB_TYPE=postgres
DB_HOST=postgres-main:5432
DB_NAME=answer
DB_USERNAME=answer
DB_PASSWORD=${main_db_password}
DB_FILE=
POSTGRES_MAIN_DB=answer
POSTGRES_MAIN_USER=answer
POSTGRES_MAIN_PASSWORD=${main_db_password}
POSTGRES_VAULT_DB=vault
POSTGRES_VAULT_USER=vault
POSTGRES_VAULT_PASSWORD=${vault_db_password}
LANGUAGE=zh-CN
SITE_NAME=匿名社区预演
SITE_URL=http://127.0.0.1:9080
CONTACT_EMAIL=admin@example.local
ADMIN_NAME=admin
ADMIN_PASSWORD=${admin_password}
ADMIN_EMAIL=admin@example.local
EXTERNAL_CONTENT_DISPLAY=always_display
SITE_ADDR=0.0.0.0:80
LOG_LEVEL=INFO
APP_BASE_URL=http://127.0.0.1:9080
WECOM_CORP_ID=replace-me
WECOM_AGENT_ID=replace-me
WECOM_APP_SECRET=replace-me
WECOM_CALLBACK_TOKEN=${wecom_callback_token}
WECOM_CALLBACK_AES_KEY=${wecom_callback_aes_key}
WECOM_DEFAULT_RETURN_TO=/community
VAULT_BASE_URL=http://vault-service:8091
VAULT_SHARED_TOKEN=${vault_shared_token}
VAULT_SECRET=${vault_secret}
EOF
  log "generated temporary env file with placeholder WeCom values: ${WORK_ENV_FILE}"
}

prepare_remote_env_file() {
  REMOTE_ENV_FILE="$(mktemp)"
  cp "${WORK_ENV_FILE}" "${REMOTE_ENV_FILE}"

  if grep -q '^DEPLOY_TAG=' "${REMOTE_ENV_FILE}"; then
    sed -i "s/^DEPLOY_TAG=.*/DEPLOY_TAG=${DEPLOY_TAG}/" "${REMOTE_ENV_FILE}"
  else
    printf '\nDEPLOY_TAG=%s\n' "${DEPLOY_TAG}" >>"${REMOTE_ENV_FILE}"
  fi
}

env_value() {
  local file="$1"
  local key="$2"
  awk -F= -v key="${key}" '$1 == key {print substr($0, index($0, "=") + 1); found=1} END {if (!found) exit 1}' "${file}"
}

validate_env_file() {
  local db_password
  local postgres_main_password
  local db_username
  local postgres_main_user
  local db_name
  local postgres_main_db

  db_password="$(env_value "${REMOTE_ENV_FILE}" "DB_PASSWORD")"
  postgres_main_password="$(env_value "${REMOTE_ENV_FILE}" "POSTGRES_MAIN_PASSWORD")"
  db_username="$(env_value "${REMOTE_ENV_FILE}" "DB_USERNAME")"
  postgres_main_user="$(env_value "${REMOTE_ENV_FILE}" "POSTGRES_MAIN_USER")"
  db_name="$(env_value "${REMOTE_ENV_FILE}" "DB_NAME")"
  postgres_main_db="$(env_value "${REMOTE_ENV_FILE}" "POSTGRES_MAIN_DB")"

  [[ "${db_password}" == "${postgres_main_password}" ]] || die "DB_PASSWORD must match POSTGRES_MAIN_PASSWORD"
  [[ "${db_username}" == "${postgres_main_user}" ]] || die "DB_USERNAME must match POSTGRES_MAIN_USER"
  [[ "${db_name}" == "${postgres_main_db}" ]] || die "DB_NAME must match POSTGRES_MAIN_DB"
}

disable_remote_auto_install() {
  ssh_remote "sed -i 's/^AUTO_INSTALL=.*/AUTO_INSTALL=/' '${REMOTE_DIR}/.env'"
  compose_remote "up -d answer-app >/dev/null"
}

main() {
  require_tool docker
  require_tool git
  require_tool scp
  require_tool ssh

  [[ -f "${ROOT_DIR}/${COMPOSE_FILE}" ]] || die "compose file not found: ${COMPOSE_FILE}"
  [[ -f "${ROOT_DIR}/${ENV_TEMPLATE}" ]] || die "env template not found: ${ENV_TEMPLATE}"

  prepare_env_file
  prepare_remote_env_file
  validate_env_file

  log "validating compose rendering locally"
  docker compose --env-file "${REMOTE_ENV_FILE}" -f "${ROOT_DIR}/${COMPOSE_FILE}" config >/dev/null

  log "building ${ANSWER_IMAGE}:${DEPLOY_TAG}"
  docker build \
    --build-arg "ALPINE_REPO=${ALPINE_REPO}" \
    --build-arg "GOPROXY=${GO_PROXY}" \
    --build-arg "GOSUMDB=${GO_SUMDB}" \
    --build-arg "NPM_REGISTRY=${NPM_REGISTRY}" \
    -t "${ANSWER_IMAGE}:${DEPLOY_TAG}" \
    -f "${ROOT_DIR}/Dockerfile" \
    "${ROOT_DIR}"

  log "building ${VAULT_IMAGE}:${DEPLOY_TAG}"
  docker build \
    --build-arg "ALPINE_REPO=${ALPINE_REPO}" \
    --build-arg "GOPROXY=${GO_PROXY}" \
    --build-arg "GOSUMDB=${GO_SUMDB}" \
    -t "${VAULT_IMAGE}:${DEPLOY_TAG}" \
    -f "${ROOT_DIR}/Dockerfile.vault" \
    "${ROOT_DIR}"

  log "creating remote directory ${REMOTE_DIR}"
  ssh_remote "mkdir -p '${REMOTE_DIR}'"

  log "copying compose and env files"
  scp -P "${REMOTE_PORT}" "${ROOT_DIR}/${COMPOSE_FILE}" "${REMOTE_HOST}:${REMOTE_DIR}/$(basename "${COMPOSE_FILE}")" >/dev/null
  scp -P "${REMOTE_PORT}" "${REMOTE_ENV_FILE}" "${REMOTE_HOST}:${REMOTE_DIR}/.env" >/dev/null

  log "loading images on remote host"
  docker save "${ANSWER_IMAGE}:${DEPLOY_TAG}" "${VAULT_IMAGE}:${DEPLOY_TAG}" | ssh -p "${REMOTE_PORT}" "${REMOTE_HOST}" "docker load"

  log "validating remote compose configuration"
  compose_remote "config >/dev/null"

  log "starting isolated predeploy stack"
  compose_remote "up -d"

  log "waiting for vault-service health"
  wait_for_health "${PROJECT_NAME}-vault-service-1" 90 2 || die "vault-service did not become healthy"

  log "waiting for answer-app health"
  wait_for_health "${PROJECT_NAME}-answer-app-1" 120 2 || die "answer-app did not become healthy"

  log "disabling AUTO_INSTALL in remote env for later restarts"
  disable_remote_auto_install
  wait_for_health "${PROJECT_NAME}-answer-app-1" 60 2 || die "answer-app did not recover after AUTO_INSTALL was cleared"

  log "running remote verification"
  ssh_remote "docker exec '${PROJECT_NAME}-answer-app-1' curl -fsS http://127.0.0.1:80/healthz >/dev/null"
  ssh_remote "docker exec '${PROJECT_NAME}-vault-service-1' curl -fsS http://127.0.0.1:8091/healthz >/dev/null"
  ssh_remote "docker exec '${PROJECT_NAME}-answer-app-1' curl -fsS http://127.0.0.1:80/community >/dev/null"

  local wecom_status
  wecom_status="$(ssh_remote "docker exec '${PROJECT_NAME}-answer-app-1' sh -lc 'curl -sS -o /dev/null -w \"%{http_code}\" http://127.0.0.1:80/answer/api/v1/wecom/auth/start'")"
  case "${wecom_status}" in
    200|302)
      ;;
    *)
      die "unexpected WeCom auth start status: ${wecom_status}"
      ;;
  esac

  log "remote stack status"
  compose_remote "ps"
  log "predeploy completed. Access through SSH tunnel: ssh -L 9080:127.0.0.1:9080 ${REMOTE_HOST}"
}

main "$@"
