#!/bin/bash
# Custom entrypoint for the Postgres service container. Wraps the standard Docker
# entrypoint to inject the cron.database_name parameter, so the pg_cron extension
# targets the correct database.

set -e

POSTGRES_DB=${POSTGRES_DB:-postgres}

exec /usr/local/bin/docker-entrypoint.sh "$@" -c "cron.database_name=${POSTGRES_DB}"
