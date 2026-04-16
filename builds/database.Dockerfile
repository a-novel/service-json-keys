# PostgreSQL image with pg_cron pre-installed at build time.
# Database migrations are not included; run the migrations image separately after deployment.
FROM docker.io/library/postgres:18.3 AS builder

ARG DEBIAN_FRONTEND=noninteractive

# Install build tools, compile pg_cron from source. The final stage copies only the
# compiled extension files, leaving all build tooling in this discarded layer.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    build-essential \
    postgresql-server-dev-18 \
  && git clone https://github.com/citusdata/pg_cron.git \
  && cd pg_cron \
  && git fetch --tags \
  && git checkout "$(git describe --tags "$(git rev-list --tags --max-count=1)")" \
  && make \
  && make install

FROM docker.io/library/postgres:18.3

# Copy only the compiled extension artifacts; build tools stay in the builder stage.
# Update these paths when bumping the PostgreSQL major version.
COPY --from=builder /usr/lib/postgresql/18/lib/pg_cron.so /usr/lib/postgresql/18/lib/
COPY --from=builder /usr/share/postgresql/18/extension/pg_cron.control /usr/share/postgresql/18/extension/
COPY --from=builder /usr/share/postgresql/18/extension/pg_cron--*.sql /usr/share/postgresql/18/extension/

# Custom entrypoint used to run PostgreSQL with extensions.
COPY --chmod=755 ./builds/database.entrypoint.sh /usr/local/bin/database.entrypoint.sh

# SQL script run on first container start to install PostgreSQL extensions.
COPY ./builds/database.sql /docker-entrypoint-initdb.d/init.sql

# Make sure the library is loaded by default.
RUN echo "shared_preload_libraries='pg_cron'" >> /usr/share/postgresql/postgresql.conf.sample \
  && chown postgres:postgres /usr/share/postgresql/postgresql.conf.sample \
  && chmod 664 /usr/share/postgresql/postgresql.conf.sample

# Default PostgreSQL port.
EXPOSE 5432

# Postgres does not provide a healthcheck by default.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD pg_isready || exit 1

# Use the custom entrypoint instead of the native one.
ENTRYPOINT ["/usr/local/bin/database.entrypoint.sh"]

# Restore the original command from the base image.
CMD ["postgres"]
