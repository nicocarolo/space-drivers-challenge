# start with base image
FROM mysql:5.6

# import data into container
# All scripts in docker-entrypoint-initdb.d/ are automatically executed during container startup
#COPY ./database/*.sql /docker-entrypoint-initdb.d/
ADD ./database/migration.sql /docker-entrypoint-initdb.d/
