#!/usr/bin/env bash

set -euo pipefail

start_mysql() {
  local name="$1"
  local host_port="$2"
  local db="$3"

  if docker ps -a --format '{{.Names}}' | grep -qx "${name}"; then
    if ! docker start "${name}" >/dev/null 2>&1; then
      docker rm -f "${name}" >/dev/null 2>&1 || true
      docker run -d \
        --name "${name}" \
        -e MYSQL_ROOT_PASSWORD=root \
        -e MYSQL_DATABASE="${db}" \
        -e MYSQL_USER=be_ads \
        -e MYSQL_PASSWORD=be_ads \
        -p "${host_port}:3306" \
        mysql:8.4 >/dev/null
    fi
  else
    docker run -d \
      --name "${name}" \
      -e MYSQL_ROOT_PASSWORD=root \
      -e MYSQL_DATABASE="${db}" \
      -e MYSQL_USER=be_ads \
      -e MYSQL_PASSWORD=be_ads \
      -p "${host_port}:3306" \
      mysql:8.4 >/dev/null
  fi

  until docker exec "${name}" mysqladmin ping -h127.0.0.1 -ube_ads -pbe_ads --silent >/dev/null 2>&1; do
    sleep 1
  done

  if [[ "${name}" == "be-ads-raw-mysql" ]]; then
    docker exec "${name}" mysql -uroot -proot -e "
      CREATE USER IF NOT EXISTS 'debezium'@'%' IDENTIFIED BY 'debezium';
      GRANT SELECT, RELOAD, SHOW DATABASES, REPLICATION SLAVE, REPLICATION CLIENT, LOCK TABLES ON *.* TO 'debezium'@'%';
      FLUSH PRIVILEGES;
    " >/dev/null
  fi
}

start_clickhouse() {
  local name="be-ads-clickhouse"
  if docker ps -a --format '{{.Names}}' | grep -qx "${name}"; then
    if ! docker start "${name}" >/dev/null 2>&1; then
      docker rm -f "${name}" >/dev/null 2>&1 || true
      docker run -d \
        --name "${name}" \
        -e CLICKHOUSE_DB=be_ads \
        -e CLICKHOUSE_USER=be_ads \
        -e CLICKHOUSE_PASSWORD=be_ads \
        -p 8123:8123 \
        -p 9000:9000 \
        clickhouse/clickhouse-server:24.3 >/dev/null
    fi
  else
    docker run -d \
      --name "${name}" \
      -e CLICKHOUSE_DB=be_ads \
      -e CLICKHOUSE_USER=be_ads \
      -e CLICKHOUSE_PASSWORD=be_ads \
      -p 8123:8123 \
      -p 9000:9000 \
      clickhouse/clickhouse-server:24.3 >/dev/null
  fi

  until docker exec "${name}" clickhouse-client --user be_ads --password be_ads --query "SELECT 1" >/dev/null 2>&1; do
    sleep 1
  done
}

start_nats() {
  local name="be-ads-nats"
  if docker ps -a --format '{{.Names}}' | grep -qx "${name}"; then
    if ! docker start "${name}" >/dev/null 2>&1; then
      docker rm -f "${name}" >/dev/null 2>&1 || true
      docker run -d \
        --name "${name}" \
        -p 4222:4222 \
        -p 8222:8222 \
        nats:2.11 --js --sd=/data --http_port=8222 >/dev/null
    fi
  else
    docker run -d \
      --name "${name}" \
      -p 4222:4222 \
      -p 8222:8222 \
      nats:2.11 --js --sd=/data --http_port=8222 >/dev/null
  fi

  until curl -fsS http://127.0.0.1:8222/healthz >/dev/null 2>&1; do
    sleep 1
  done
}

start_mysql be-ads-raw-mysql 3307 be_ads_raw
start_mysql be-ads-serving-mysql 3308 be_ads_serving
start_clickhouse
start_nats

echo "raw mysql:     127.0.0.1:3307 db=be_ads_raw"
echo "serving mysql: 127.0.0.1:3308 db=be_ads_serving"
echo "clickhouse:    127.0.0.1:9000 db=be_ads"
echo "nats:          nats://127.0.0.1:4222"
