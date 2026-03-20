#!/usr/bin/env bash
# local-node.sh — Start a local TRON test node using tronbox/tre (TRE)
#
# TRE provides a local TRON network with pre-funded accounts for development.
# gRPC is available at localhost:9090.
#
# Usage:
#   ./scripts/local-node.sh          # start (foreground)
#   ./scripts/local-node.sh start    # start (background)
#   ./scripts/local-node.sh stop     # stop and remove container
#   ./scripts/local-node.sh restart  # stop + start
#   ./scripts/local-node.sh logs     # tail container logs
#   ./scripts/local-node.sh status   # check if running

set -euo pipefail

CONTAINER_NAME="tron-local"
IMAGE="tronbox/tre"
HTTP_PORT=9090
GRPC_PORT=50051
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$(dirname "$SCRIPT_DIR")/accounts-data"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

is_running() {
    docker inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null | grep -q true
}

exists() {
    docker inspect "$CONTAINER_NAME" >/dev/null 2>&1
}

cmd_start() {
    if is_running; then
        echo -e "${GREEN}$CONTAINER_NAME is already running${NC}"
        echo "gRPC: localhost:$GRPC_PORT"
        return 0
    fi

    if exists; then
        echo "Starting existing container..."
        docker start "$CONTAINER_NAME"
    else
        mkdir -p "$DATA_DIR"
        echo "Pulling $IMAGE..."
        docker pull "$IMAGE" 2>/dev/null || true

        local mode="-d"
        if [[ "${1:-}" == "foreground" ]]; then
            mode="-it"
        fi

        echo "Starting $CONTAINER_NAME..."
        docker run $mode \
            -p "$HTTP_PORT:$HTTP_PORT" \
            -p "$GRPC_PORT:$GRPC_PORT" \
            --name "$CONTAINER_NAME" \
            -v "$DATA_DIR:/config" \
            "$IMAGE"
    fi

    echo -e "${GREEN}Local TRON node started${NC}"
    echo "gRPC: localhost:$GRPC_PORT"
    echo ""
    echo "Add to your .env:"
    echo "  TRONCTL_NODE=localhost:$GRPC_PORT"
    echo "  TRONCTL_HTTP=localhost:$HTTP_PORT"
}

cmd_stop() {
    if exists; then
        echo "Stopping $CONTAINER_NAME..."
        docker stop "$CONTAINER_NAME" 2>/dev/null || true
        docker rm "$CONTAINER_NAME" 2>/dev/null || true
        echo -e "${GREEN}Stopped${NC}"
    else
        echo "Container $CONTAINER_NAME not found"
    fi
}

cmd_logs() {
    if exists; then
        docker logs -f "$CONTAINER_NAME"
    else
        echo -e "${RED}Container $CONTAINER_NAME not found${NC}"
        exit 1
    fi
}

cmd_status() {
    if is_running; then
        echo -e "${GREEN}$CONTAINER_NAME is running${NC}"
        echo "gRPC: localhost:$GRPC_PORT"
        docker ps --filter "name=$CONTAINER_NAME" --format "table {{.Status}}\t{{.Ports}}"
    elif exists; then
        echo -e "${RED}$CONTAINER_NAME exists but is stopped${NC}"
    else
        echo -e "${RED}$CONTAINER_NAME not found${NC}"
    fi
}

case "${1:-}" in
    start)
        cmd_start
        ;;
    stop)
        cmd_stop
        ;;
    restart)
        cmd_stop
        cmd_start
        ;;
    logs)
        cmd_logs
        ;;
    status)
        cmd_status
        ;;
    "")
        cmd_start foreground
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|logs|status}"
        echo "  (no args) — start in foreground"
        exit 1
        ;;
esac
