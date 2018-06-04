#!/bin/sh

MYSQL_ROOT_HOST=${MYSQL_ROOT_HOST:=localhost}
MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD:=root}
MYSQL_TEST_USER=${MYSQL_TEST_USER:=test}
MYSQL_TEST_PASSWORD=${MYSQL_TEST_PASSWORD:=test}
MYSQL_TEST_DATABASE=${MYSQL_TEST_DATABASE:=signavio_test}
cat << __EOF__ | mysql -u root -h "${MYSQL_ROOT_HOST}" -p"${MYSQL_ROOT_PASSWORD}"
DROP USER IF EXISTS '${MYSQL_TEST_USER}'@'${MYSQL_ROOT_HOST}';
CREATE USER '${MYSQL_TEST_USER}'@'${MYSQL_ROOT_HOST}' IDENTIFIED BY '${MYSQL_TEST_PASSWORD}';
DROP DATABASE IF EXISTS ${MYSQL_TEST_DATABASE};
CREATE DATABASE ${MYSQL_TEST_DATABASE};
GRANT ALL ON ${MYSQL_TEST_DATABASE}.* TO '${MYSQL_TEST_USER}'@'${MYSQL_ROOT_HOST}' WITH GRANT OPTION;
FLUSH PRIVILEGES;

USE ${MYSQL_TEST_DATABASE}
BEGIN;

CREATE TABLE IF NOT EXISTS equipment (
  id INT NOT NULL AUTO_INCREMENT,
  name text,
  acquisition_cost decimal(10,5),
  purchase_date datetime,
  primary key (id)
);
INSERT INTO equipment (name, acquisition_cost, purchase_date)
  VALUES
  ("Stainless Steel Mash Tun (50L)", 999.00, "2017-12-12 12:00:00"),
  ("HolzbierFaß (200L)", 512.23, "2017-12-12 12:00:00"),
  ("Refractometer", 129.00, "2017-12-12 12:00:00")
;

CREATE TABLE IF NOT EXISTS person (
  id INT NOT NULL AUTO_INCREMENT,
  preferred_name text,
  family_name text,
  email_address varchar(128) unique not null,
  primary key (id)
);
INSERT INTO person (preferred_name,family_name,email_address)
  VALUES
  ("Jane","Feather", "jane.feather@example.com"),
  ("Jack","Calm", "jack.calm@example.com"),
  ("Bob","White", "bob.white@example.com"),
  ("Joe","Frei", "joe.frei@example.com");

CREATE TABLE IF NOT EXISTS maintenance (
  id INT NOT NULL AUTO_INCREMENT,
  date_scheduled datetime,
  date_performed datetime,
  equipment_id integer,
  maintainer_id integer,
  comments text,
  foreign key (equipment_id) references equipment(id),
  foreign key (maintainer_id) references person(id),
  primary key (id)
);
INSERT INTO maintenance (date_scheduled,date_performed,comments,equipment_id,maintainer_id) VALUES
  ("2017-02-03 02:00:00", "2018-02-03 12:22:01", "It went well!", 1, 1),
  ("2017-02-03 02:00:00", "2018-02-03 12:22:01", "It went poorly!", 2, 1),
  ("2017-02-03 02:00:00", "2018-02-03 12:22:01", "It went okay!", 1, 2),
  ("2017-02-03 02:00:00", "2018-02-03 12:22:01", "It went great!", 3, 2);

CREATE TABLE IF NOT EXISTS warranty (
  id INT NOT NULL AUTO_INCREMENT,
  type text,
  duration_in_weeks integer,
  date_from datetime,
  primary key (id)
);
INSERT INTO warranty (type,duration_in_weeks,date_from) VALUES
  ("parts and labour", 104, "2017-10-02 05:00:00"),
  ("parts and labour", 156, "2017-10-02 05:00:00"),
  ("parts", 104, "2017-10-02 05:00:00"),
  ("parts and labour", 104, "2017-10-02 05:00:00");

CREATE TABLE IF NOT EXISTS maintenance_warranty (
  maintenance_id INT NOT NULL,
  warranty_id INT NOT NULL,
  foreign key (maintenance_id) references maintenance(id),
  foreign key (warranty_id) references warranty(id),
  primary key(maintenance_id, warranty_id)
);
INSERT INTO maintenance_warranty (maintenance_id, warranty_id) VALUES
  (1, 1),
  (2, 2),
  (3, 3),
  (1, 2);

COMMIT;
__EOF__

