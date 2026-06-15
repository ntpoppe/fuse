#!/bin/sh
set -e

mysql_host="${MYSQL_HOST:-mysql}"
mysql_port="${MYSQL_PORT:-3306}"

echo "waiting for mysql at ${mysql_host}:${mysql_port}..."
while ! nc -z "$mysql_host" "$mysql_port" 2>/dev/null; do
	sleep 1
done
echo "mysql is ready"

exec fuse \
	-demo \
	-host 0.0.0.0 \
	-port 5000 \
	-state-db /data/fuse.db \
	-demo-sqlite-path /data/shop.db \
	-demo-mysql-dsn "${FUSE_DEMO_MYSQL_DSN:-demo:demo@tcp(mysql:3306)/fuse_test}"
