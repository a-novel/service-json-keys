# This is a custom postgres image that comes with pre-loaded extensions. It allows us to customize
# our instance at build time.
#
# Note: this image does not run the migrations from the main image, make sure to call the appropriate
# patch for this.
FROM docker.io/library/postgres:18.2

# ======================================================================================================================
# Prepare extension scripts.
# ======================================================================================================================
# Custom entrypoint used to run postgres with extensions.
COPY ./builds/database.entrypoint.sh /usr/local/bin/database.entrypoint.sh
RUN chmod +x /usr/local/bin/database.entrypoint.sh

# Initial migration of the image, used to setup extensions within postgres.
COPY ./builds/database.sql /docker-entrypoint-initdb.d/init.sql

# ======================================================================================================================
# Install pg_cron.
# https://github.com/citusdata/pg_cron
# ======================================================================================================================
# Needed to build pg_cron.
RUN apt-get update && apt-get -y install git build-essential postgresql-server-dev-18

# We install from source to ensure we have the latest version (no trust for Debian apt packages).
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
# Default postgres port.
EXPOSE 5432

# Postgres does not provide a healthcheck by default.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD pg_isready || exit 1

# Use our entrypoint instead of the native one.
ENTRYPOINT ["/usr/local/bin/database.entrypoint.sh"]

# Restore the original command from the base image.
CMD ["postgres"]
