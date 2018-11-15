#!/usr/bin/env sh
set -x

ORACLE_HOST=${ORACLE_HOST:=localhost}
ORACLE_USER=${ORACLE_USER:=system}
ORACLE_DATABASE=${ORACLE_DATABASE:=xe}
ORACLE_DUMP_FILE=${ORACLE_DUMP_FILE:=oracle.dmp}
# Source sensitive environment variables from .env
if [ -f .env ]
then
    # shellcheck source=.env
    . ./.env
fi
sqlplus ${ORACLE_USER}/${ORACLE_PASSWORD}@${ORACLE_HOST}/${ORACLE_DATABASE} @\""${ORACLE_DUMP_FILE}\""

