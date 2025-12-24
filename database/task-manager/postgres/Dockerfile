FROM postgres:16

RUN apt-get update && \
    apt-get -y install postgresql-16-cron && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Укажите вашу БД для pg_cron
RUN echo "shared_preload_libraries = 'pg_cron'" >> /usr/share/postgresql/postgresql.conf.sample
RUN echo "cron.database_name = 'task_manager'" >> /usr/share/postgresql/postgresql.conf.sample
