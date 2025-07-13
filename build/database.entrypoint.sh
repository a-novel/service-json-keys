#!/bin/bash
set -e

# Default value for POSTGRES_DB if not set
POSTGRES_DB=${POSTGRES_DB:-postgres}

# ======================================================================================================================
# Install pg_cron.
# https://github.com/citusdata/pg_cron
# ======================================================================================================================

# Append the cron.database_name configuration to postgresql.conf
echo "cron.database_name='${POSTGRES_DB:-postgres}'" >> /usr/share/postgresql/postgresql.conf.sample

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================

# Execute the original entrypoint script
exec /usr/local/bin/docker-entrypoint.sh "$@"