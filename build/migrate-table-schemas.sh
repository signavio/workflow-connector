#!/bin/sh

cat << __EOF__ >> ~/migration.psql
BEGIN;

CREATE TABLE IF NOT EXISTS equipment ( 
id integer not null primary key,  
name text,  
acquisition_cost real,  
purchase_date timestamp);  

CREATE TABLE IF NOT EXISTS maintenance ( 
id integer not null primary key,  
equipment_id integer not null,  
maintenance_performed timestamp,  
notes text,  
next_maintenance timestamp);  

CREATE TABLE IF NOT EXISTS warranty ( 
id integer not null primary key,  
equipment_id integer not null,  
start_date timestamp,  
duration_in_years numeric,  
conditions text);  

COMMIT;
__EOF__
psql ${DATABASE_URL} -f migration.psql
psql ${DATABASE_URL} -c "\copy equipment FROM './equipment_data.csv' DELIMITER ',' CSV"
psql ${DATABASE_URL} -c "\copy maintenance FROM './maintenance_data.csv' DELIMITER ',' CSV"
psql ${DATABASE_URL} -c "\copy warranty FROM './warranty_data.csv' DELIMITER ',' CSV"
