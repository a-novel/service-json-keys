# PostgreSQL image with this service's extensions pre-installed at build time.
# Run the migrations image separately after deployment to create the schema.
FROM docker.io/library/postgres:18.4

ARG DEBIAN_FRONTEND=noninteractive

# SQL script run on first container start to install PostgreSQL extensions.
COPY ./builds/database.sql /docker-entrypoint-initdb.d/init.sql

EXPOSE 5432

# The postgres image ships no healthcheck of its own.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD pg_isready || exit 1
