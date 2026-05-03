#!/bin/sh
set -e

echo "🔄 Running database migrations..."

# Накатываем миграции (БД уже готова благодаря depends_on + healthcheck)
./migrate -cmd=up
echo "✅ Migrations applied"

# Запускаем основной сервис
echo "🚀 Starting service..."
exec ./server "$@"