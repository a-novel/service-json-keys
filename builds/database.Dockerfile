# PostgreSQL image with this service's extensions pre-installed at build time.
# Run the migrations image separately after deployment to create the schema.
FROM docker.io/library/postgres:18.6-trixie

ENV POSTGRES_INITDB_ARGS=--auth=scram-sha-256

# Keep backup tooling with the database; archiving and scheduling remain opt-in.
# Preserve the base image's PostgreSQL binaries when installing packages.
RUN sha256sum /usr/lib/postgresql/18/bin/postgres /usr/lib/postgresql/18/lib/uuid-ossp.so > /tmp/database.sha256 \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates=20250419 \
        pgbackrest=2.59.1-1.pgdg13+1 \
    && sha256sum --check /tmp/database.sha256 \
    && gosu postgres pgbackrest version \
    && gosu postgres test -s /etc/ssl/certs/ca-certificates.crt \
    && gosu postgres openssl crl2pkcs7 -nocrl -certfile /etc/ssl/certs/ca-certificates.crt -out /dev/null \
    && rm -rf /var/lib/apt/lists/* /tmp/database.sha256

# SQL script run on first container start to install PostgreSQL extensions.
COPY ./builds/database.sql /docker-entrypoint-initdb.d/init.sql

EXPOSE 5432

# The postgres image ships no healthcheck of its own.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD ["pg_isready"]
