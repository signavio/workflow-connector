#!/bin/sh

cat << __EOF__ >> ~/migration.psql
BEGIN;

CREATE TABLE IF NOT EXISTS teacher (
id serial not null primary key,
code char(6) unique not null);

CREATE TABLE IF NOT EXISTS organization (
id serial not null primary key,
code varchar(8) unique not null);

CREATE TABLE IF NOT EXISTS section (
id serial not null primary key,
code char(1) unique not null);

CREATE TABLE IF NOT EXISTS course (
id serial not null primary key,
code char(4) unique not null);

CREATE TABLE IF NOT EXISTS class (
id serial not null primary key,
class_coverage numeric,
class_startdate date,
class_time time,
class_duration numeric,
section_code char(1) not null references section(code),
course_code char(4) not null references course(code),
organization_code varchar(8) not null references organization(code));

CREATE TABLE IF NOT EXISTS class_teacher (
id serial not null primary key,
class_id integer not null references class(id),
teacher_id char(6) not null references teacher(code));


COMMIT;
__EOF__
DIR=$(pwd)/build
psql ${DATABASE_URL} -f migration.psql
for table in teacher organization section course class; do
    psql ${DATABASE_URL} -c "\copy ${table} $(sed -n 1p ${DIR}/${table}-data.csv) FROM '${DIR}/${table}-data.csv' DELIMITER ',' CSV HEADER"
done
