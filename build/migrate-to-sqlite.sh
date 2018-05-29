#!/bin/sh

cat << __EOF__ | sqlite3 test.db
BEGIN;

CREATE TABLE IF NOT EXISTS equipment (
  id integer primary key autoincrement,
  name text,
  acquisition_cost decimal(10,5),
  purchase_date date
);
INSERT INTO 'equipment' ('name','acquisition_cost','purchase_date')
  VALUES
  ('Stainless Steel Mash Tun (50L)', 999.00,'2017-12-12T12:00:00Z'),
  ('HolzbierFaß (200L)', 512.23,'2017-12-12T12:00:00Z'),
  ('Refractometer', 129.00,'2017-12-12T12:00:00Z')
;

CREATE TABLE IF NOT EXISTS person (
  id integer primary key autoincrement,
  preferred_name text,
  family_name text,
  email_address text unique not null
);
INSERT INTO 'person' ('preferred_name','family_name','email_address')
  VALUES
  ('Jane','Feather','jane.feather@example.com'),
  ('Jack','Calm','jack.calm@example.com'),
  ('Bob','White','bob.white@example.com'),
  ('Joe','Frei','joe.frei@example.com');

CREATE TABLE IF NOT EXISTS maintenance (
  id integer primary key autoincrement,
  date_scheduled datetime,
  date_performed datetime,
  equipment_id integer,
  maintainer_id integer,
  comments text,
  foreign key (equipment_id) references equipment(id),
  foreign key (maintainer_id) references person(id)
);
INSERT INTO 'maintenance' ('date_scheduled','date_performed','comments','equipment_id','maintainer_id') VALUES
  ('2017-02-03T02:00:00Z', '2018-02-03T12:22:01Z', 'It went well!', 1, 1),
  ('2017-02-03T02:00:00Z', '2018-02-03T12:22:01Z', 'It went poorly!', 2, 1),
  ('2017-02-03T02:00:00Z', '2018-02-03T12:22:01Z', 'It went okay!', 1, 2),
  ('2017-02-03T02:00:00Z', '2018-02-03T12:22:01Z', 'It went great!', 3, 2);

CREATE TABLE IF NOT EXISTS warranty (
  id integer primary key autoincrement,
  type text,
  duration_in_weeks integer,
  date_from datetime
);
INSERT INTO 'warranty' ('type','duration_in_weeks','date_from') VALUES
  ('parts and labour', 104, '2017-10-02T05:00:00Z'),
  ('parts and labour', 156, '2017-10-02T05:00:00Z'),
  ('parts', 104, '2017-10-02T05:00:00Z'),
  ('parts and labour', 104, '2017-10-02T05:00:00Z');

CREATE TABLE IF NOT EXISTS maintenance_warranty (
  maintenance_id integer,
  warranty_id integer,
  foreign key (maintenance_id) references maintenance(id),
  foreign key (warranty_id) references warranty(id),
  primary key(maintenance_id, warranty_id)
);
INSERT INTO 'maintenance_warranty' ('maintenance_id', 'warranty_id') VALUES
  (1, 1),
  (2, 2),
  (3, 3),
  (1, 2);

COMMIT;
__EOF__


