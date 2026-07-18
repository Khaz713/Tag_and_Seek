#!/bin/bash

NETWORK_NAME="tag-and-seek-cluster"
COOKIE="9701713"
CONTAINERS=("rabbit1" "rabbit2" "rabbit3" "postgres")

POSTGRES_USER="postgres"
POSTGRES_PASSWORD="postgres"
POSTGRES_DB="tag-and-seek"
POSTGRES_VOLUME="tag-and-seek-data"

start_or_create() {
  local name=$1
  local extra_ports=$2

  if docker container inspect "$name" >/dev/null 2>&1; then
    echo "Container '$name' already exists. Starting it..."
    docker start "$name"
    return 1  # already existed
  else
    echo "Container '$name' not found. Creating it..."
    docker run -d \
      --name "$name" \
      --hostname "$name" \
      --network "$NETWORK_NAME" \
      -e RABBITMQ_ERLANG_COOKIE="$COOKIE" \
      $extra_ports \
      rabbitmq:management
    return 0  # freshly created
  fi
}

start_or_create_postgres() {
  if docker container inspect "postgres" >/dev/null 2>&1; then
    echo "Container 'postgres' already exists. Starting it..."
    docker start "postgres"
  else
    echo "Container 'postgres' not found. Creating it..."

    # ensure the named volume exists (docker run would create it anyway,
    # but this makes the intent explicit)
    docker volume inspect "$POSTGRES_VOLUME" >/dev/null 2>&1 || \
      docker volume create "$POSTGRES_VOLUME"

    docker run -d \
      --name "postgres" \
      --hostname "postgres" \
      --network "$NETWORK_NAME" \
      -e POSTGRES_USER="$POSTGRES_USER" \
      -e POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
      -e POSTGRES_DB="$POSTGRES_DB" \
      -v "$POSTGRES_VOLUME":/var/lib/postgresql/data \
      -p 5433:5432 \
      postgres:16
  fi
}

do_start() {
  docker network inspect "$NETWORK_NAME" >/dev/null 2>&1 || \
    docker network create "$NETWORK_NAME"

  start_or_create "rabbit1" "-p 5672:5672 -p 15672:15672"
  rabbit1_new=$?

  start_or_create "rabbit2" ""
  rabbit2_new=$?

  start_or_create "rabbit3" ""
  rabbit3_new=$?

  start_or_create_postgres

  if [ "$rabbit2_new" -eq 0 ] || [ "$rabbit3_new" -eq 0 ]; then
    echo "Waiting for rabbit1 to boot before clustering..."
    sleep 5
  fi

  if [ "$rabbit2_new" -eq 0 ]; then
    docker exec rabbit2 rabbitmqctl stop_app
    docker exec rabbit2 rabbitmqctl join_cluster rabbit@rabbit1
    docker exec rabbit2 rabbitmqctl start_app
  fi

  if [ "$rabbit3_new" -eq 0 ]; then
    docker exec rabbit3 rabbitmqctl stop_app
    docker exec rabbit3 rabbitmqctl join_cluster rabbit@rabbit1
    docker exec rabbit3 rabbitmqctl start_app
  fi
}

do_stop() {
  for name in "${CONTAINERS[@]}"; do
    if docker container inspect "$name" >/dev/null 2>&1; then
      echo "Stopping container '$name'..."
      docker stop "$name"
    else
      echo "Container '$name' not found, skipping."
    fi
  done
}

case "$1" in
  start)
    do_start
    ;;
  stop)
    do_stop
    ;;
  *)
    echo "Usage: $0 {start|stop}"
    exit 1
    ;;
esac