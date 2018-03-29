#!/bin/sh

cat << __EOF__ >> ~/migration.psql
CREATE TABLE IF NOT EXISTS equipment (" +
id integer not null primary key, " +
name text, " +
acquisition_cost real, " +
purchase_date timestamp); " +
INSERT INTO equipment(id, name, acquisition_cost, purchase_date) " +
VALUES " +
(1,'Stainless Steel Cooling Spiral',119.0,'2017-09-07 12:00:00'), " +
(2,'Fermentation Tank (50L)',250.0,'2014-09-07 11:00:00'), " +
(3,'Temperature Gauge',49.99,'2017-09-04 11:00:00'), " +
(4,'Masch Tun (50L)',199.99,'2016-09-04 11:00:00'), " +
(5,'Malt mill 550',1270,'2016-09-04 11:00:00'); " +

CREATE TABLE IF NOT EXISTS maintenance (" +
id integer not null primary key, " +
equipment_id integer not null, " +
maintenance_performed timestamp, " +
notes text, " +
next_maintenance timestamp); " +
INSERT INTO maintenance(id, equipment_id, maintenance_performed, notes, next_maintenance) " +
VALUES " +
(1,3,'2017-02-07 12:00:00','Nothing noteworthy 1','2018-12-01 12:00:00'), " +
(2,2,'2015-02-07 12:00:00','Nothing noteworthy 2','2016-11-01 12:00:00'), " +
(3,3,'2017-02-07 12:00:00','Nothing noteworthy 3','2018-11-01 12:00:00'), " +
(4,1,'2017-02-07 12:00:00','Nothing noteworthy 4','2018-11-01 12:00:00'), " +
(5,2,'2016-02-07 12:00:00','Nothing noteworthy 5','2017-11-01 12:00:00'), " +
(6,2,'2017-02-07 12:00:00','Nothing noteworthy 6','2018-11-01 12:00:00'); " +

CREATE TABLE IF NOT EXISTS warranty (" +
id integer not null primary key, " +
equipment_id integer not null, " +
start_date timestamp, " +
duration_in_years number, " +
conditions text); " +
INSERT INTO warranty(id, equipment_id, start_date, duration_in_years, conditions) " +
VALUES " +
(1,1,'2016-02-20 12:00:00',3,'warranty covers parts and labour'), " +
(2,2,'2016-10-02 12:00:00',2,'warranty only for parts'), " +
(3,3,'2017-02-19 12:00:00',3,'warranty covers parts and labour'), " +
(4,5,'2017-02-19 12:00:00',2,'warranty only for parts'); "

__EOF__
psql -f migration.psql
