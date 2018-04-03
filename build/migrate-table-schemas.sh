#!/bin/sh

cat << __EOF__ >> ~/migration.psql
BEGIN;

CREATE TABLE IF NOT EXISTS class (
id integer not null primary key,
section_code char(1) not null,
course_code char(4) not null,
organization_code varchar(8) not null);

CREATE TABLE IF NOT EXISTS class_teacher (
id integer not null primary key,
class_id integer not null,
teacher_id integer not null);

CREATE TABLE IF NOT EXISTS teacher (
id integer not null primary key,
code char(6) not null);

CREATE TABLE IF NOT EXISTS organization (
id integer not null primary key,
code varchar(8) not null);

CREATE TABLE IF NOT EXISTS section (
id integer not null primary key,
code char(1) not null);

CREATE TABLE IF NOT EXISTS course (
id integer not null primary key,
code char(4) not null);

COMMIT;
__EOF__
DIR=$(pwd)/build
psql ${DATABASE_URL} -f migration.psql
for table in $(ls "${DIR}" | grep '\.csv$' | cut -d'-' -f1 ); do
    psql ${DATABASE_URL} -c "\copy ${table} FROM '${DIR}/${table}_data.csv' DELIMITER ',' CSV"
done
