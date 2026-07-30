#!/bin/bash
set -eu

export COMPOSE_FILE=ftw/docker-compose.yml

show_usage_and_exit() {
  >&2 cat <<'EOF'
Please execute run.sh in ftw directory.
example: (cd ftw; ./run.sh)
EOF
  exit 2
}

stop_docker_compose() {
  docker compose down -v
}

main() {
  # change to parent directory of run.sh
  cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

  if [ -z "${CRS_VERSION:-}" ]; then
    CRS_VERSION=$(grep -F github.com/corazawaf/coraza-coreruleset/v4 go.mod)
    export CRS_VERSION="${CRS_VERSION##* }"
  fi

  echo "Running FTW tests with CRS version: ${CRS_VERSION}"

  if [ "${SKIP_BUILD:-0}" -ne 1 ]; then
    docker compose build --pull --no-cache --build-arg "CRS_VERSION=${CRS_VERSION}"
  fi

  trap stop_docker_compose EXIT
  docker compose run --rm ftw 2>&1 | tee build/ftw-run.log
}

main "$@"
