FROM postgres:16

ARG POSTGRES_DB

RUN apt-get update && \
    apt-get -y install postgresql-16-cron && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Укажите вашу БД для pg_cron
RUN echo "shared_preload_libraries = 'pg_cron'" >> /usr/share/postgresql/postgresql.conf.sample
RUN echo "cron.database_name = '${POSTGRES_DB}'" >> /usr/share/postgresql/postgresql.conf.sample
RUN echo "cron.log_run = 'false'" >> /usr/share/postgresql/postgresql.conf.sample
