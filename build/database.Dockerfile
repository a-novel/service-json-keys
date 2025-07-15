FROM docker.io/library/postgres:17

# ======================================================================================================================
# Prepare extension scripts.
# ======================================================================================================================
# Custom entrypoint used to run extensions at runtime.
# We copy it to the same directory as the original entrypoint from the base image. Thus, our entrypoint can
# reference the original.
COPY ./build/database.entrypoint.sh /usr/local/bin/database.entrypoint.sh
RUN chmod +x /usr/local/bin/database.entrypoint.sh

COPY ./build/database.sql /docker-entrypoint-initdb.d/init.sql

RUN apt-get update && apt-get -y install git build-essential postgresql-server-dev-17

# ======================================================================================================================
# Install pg_cron.
# https://github.com/citusdata/pg_cron
# ======================================================================================================================
# We install from source to ensure we have the latest version (no trust for Debian apt packages).
RUN git clone https://github.com/citusdata/pg_cron.git
RUN cd pg_cron && \
    git fetch --tags && \
    # Use the latest tagged version, not the latest commit.
    git checkout $(git describe --tags "$(git rev-list --tags --max-count=1)") && \
    make && make install

RUN cd / && \
        rm -rf /pg_cron && \
        apt-get remove -y git build-essential postgresql-server-dev-17 && \
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
EXPOSE 5432

# Postgres does not provide a healthcheck by default.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD pg_isready || exit 1

ENTRYPOINT ["/usr/local/bin/database.entrypoint.sh"]

# Restore original command from the base image.
CMD ["postgres"]
