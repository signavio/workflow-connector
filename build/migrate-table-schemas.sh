#!/bin/sh

cat << __EOF__ >> ~/migration.psql
BEGIN;

CREATE TABLE IF NOT EXISTS class (
id integer not null primary key,
section_code char(1) not null,
course_code char(4) not null,
organization_code varchar(8) not null);

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
WD=$(pwd)/build
psql ${DATABASE_URL} -f migration.psql
psql ${DATABASE_URL} -c "\copy equipment FROM '${WD}/equipment_data.csv' DELIMITER ',' CSV"
psql ${DATABASE_URL} -c "\copy maintenance FROM '${WD}/maintenance_data.csv' DELIMITER ',' CSV"
psql ${DATABASE_URL} -c "\copy warranty FROM '${WD}/warranty_data.csv' DELIMITER ',' CSV"
