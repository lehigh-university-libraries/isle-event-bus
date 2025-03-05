#!/usr/bin/env bash

set -eou pipefail

SCRIPT_DIR=$(dirname "$(realpath "$0")")
cd "$SCRIPT_DIR/../"

docker build \
  -t libops/isle-event-bus:main .

cd "$SCRIPT_DIR"
docker build \
  -t fs:main .

docker compose up -d --quiet-pull > /dev/null 2>&1

docker exec ci-test-1 apk update > /dev/null 2>&1
docker exec ci-test-1 apk add bash curl file > /dev/null 2>&1
docker exec ci-test-1 /test.sh
echo $?

docker compose down > /dev/null 2>&1
