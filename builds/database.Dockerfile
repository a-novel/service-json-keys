# PostgreSQL image with pg_cron pre-installed at build time.
# Database migrations are not included; run the migrations image separately after deployment.
FROM docker.io/library/postgres:18.3

ARG DEBIAN_FRONTEND=noninteractive

# ======================================================================================================================
# Prepare extension scripts.
# ======================================================================================================================
# Custom entrypoint used to run PostgreSQL with extensions.
COPY ./builds/database.entrypoint.sh /usr/local/bin/database.entrypoint.sh
RUN chmod +x /usr/local/bin/database.entrypoint.sh

# SQL script run on first container start to install PostgreSQL extensions.
COPY ./builds/database.sql /docker-entrypoint-initdb.d/init.sql

# ======================================================================================================================
# Install pg_cron.
# https://github.com/citusdata/pg_cron
# ======================================================================================================================
# Needed to build pg_cron.
RUN apt-get update && apt-get -y install git build-essential postgresql-server-dev-18

# Install from source for the latest version; the Debian apt package lags behind upstream releases.
RUN git clone https://github.com/citusdata/pg_cron.git
RUN cd pg_cron && \
    git fetch --tags && \
    # Use the latest tagged version, not the latest commit. \
    git checkout $(git describe --tags "$(git rev-list --tags --max-count=1)") && \
    make && make install

RUN cd / && \
    rm -rf /pg_cron && \
    # Remove build packages \
    apt-get remove -y git build-essential postgresql-server-dev-18 && \
    apt-get autoremove --purge -y && \
    apt-get clean && \
    apt-get purge

# Make sure the library is loaded by default.
RUN echo "shared_preload_libraries='pg_cron'" >> /usr/share/postgresql/postgresql.conf.sample
RUN chown postgres:postgres /usr/share/postgresql/postgresql.conf.sample && \
    chmod 664 /usr/share/postgresql/postgresql.conf.sample

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
# Default PostgreSQL port.
EXPOSE 5432

# Postgres does not provide a healthcheck by default.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD pg_isready || exit 1

# Use the custom entrypoint instead of the native one.
ENTRYPOINT ["/usr/local/bin/database.entrypoint.sh"]

# Restore the original command from the base image.
CMD ["postgres"]
