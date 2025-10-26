#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

trap 'kill 0' SIGINT SIGTERM EXIT

go run "$ROOT_DIR/login"/main.go &
LOGIN_PID=$!

go run "$ROOT_DIR/persist"/main.go &
PERSIST_PID=$!

go run "$ROOT_DIR/game"/main.go &
GAME_PID=$!

go run "$ROOT_DIR/gate"/main.go &
GATE_PID=$!

wait $LOGIN_PID $PERSIST_PID $GAME_PID $GATE_PID
