#!/usr/bin/env sh
set -x

ORACLE_HOST=${ORACLE_HOST:=localhost}
ORACLE_USER=${ORACLE_USER:=system}
ORACLE_DATABASE=${ORACLE_DATABASE:=xe}
ORACLE_DUMP_FILE=${ORACLE_DUMP_FILE:=oracle.dmp}
# Set the database timezone to something other than UTC for our tests
export ORA_SDTZ=01:00
export NLS_LANG=".AL32UTF8"
# Source sensitive environment variables from .env
if [ -f .env ]
then
    # shellcheck source=.env
    . ./.env
fi
cat << __EOF__ | sqlplus ${ORACLE_USER}/${ORACLE_PASSWORD}@${ORACLE_HOST}/${ORACLE_DATABASE}
BEGIN
EXECUTE IMMEDIATE 'DROP TABLE ingredient_recipe';
EXECUTE IMMEDIATE 'DROP TABLE inventory';
EXECUTE IMMEDIATE 'DROP TABLE ingredients';
EXECUTE IMMEDIATE 'DROP TABLE recipes';
EXECUTE IMMEDIATE 'DROP TABLE equipment';
EXCEPTION
WHEN OTHERS THEN
IF SQLCODE != -942 THEN
RAISE;
END IF;
END;
/

CREATE TABLE equipment (
  "id" integer generated by default as identity,
  "name" nvarchar2(256),
  "acquisition_cost" number(10,5),
  "purchase_date" timestamp(6) with time zone null,
  primary key ("id")
);
INSERT INTO equipment ("name", "acquisition_cost", "purchase_date")
  VALUES ('Bialetti Moka Express 6 cup',25.95,to_timestamp_tz('2017-12-11T12:00:00.123+00:00', 'YYYY-MM-DD"T"HH24:MI:SSXFFTZH:TZM'));
INSERT INTO equipment ("name", "acquisition_cost", "purchase_date")
  VALUES ('Sanremo Café Racer',8477.85,to_timestamp_tz('2017-12-12T12:00:00.123000+00:00', 'YYYY-MM-DD"T"HH24:MI:SSXFFTZH:TZM'));
INSERT INTO equipment ("name", "acquisition_cost", "purchase_date")
  VALUES ('Buntfink SteelKettle',39.95,to_timestamp_tz('2017-12-12T12:00:00.000+00:00', 'YYYY-MM-DD"T"HH24:MI:SSXFFTZH:TZM'));
INSERT INTO equipment ("name", "acquisition_cost", "purchase_date")
  VALUES ('Copper Coffee Pot Cezve',49.95,to_timestamp_tz('2017-12-12T12:00:00.000+00:00', 'YYYY-MM-DD"T"HH24:MI:SSXFFTZH:TZM'));

CREATE TABLE ingredients (
  "id" integer generated by default as identity,
  "name" nvarchar2(1024),
  "description" nvarchar2(1024),
  primary key ("id")
);
INSERT INTO ingredients ("name","description")
  VALUES ('V60 paper filter', 'The best paper filter on the market');
INSERT INTO ingredients ("name","description")
  VALUES ('Caffé Borbone Beans - Miscela Blu', 'Excellent beans for espresso');
INSERT INTO ingredients ("name","description")
  VALUES ('Caffé Borbone Beans - Miscela Oro', 'Well balanced beans');
INSERT INTO ingredients ("name","description")
  VALUES ('Filtered Water', 'Contains the perfect water hardness for espresso');

CREATE TABLE inventory (
  "ingredient_id" number,
  "quantity" number,
  "unit_of_measure" nvarchar2(256),
  foreign key ("ingredient_id") references ingredients("id"),
  primary key ("ingredient_id")
);
INSERT INTO inventory ("ingredient_id", "quantity", "unit_of_measure")
  VALUES (1, 100, 'Each');
INSERT INTO inventory ("ingredient_id", "quantity", "unit_of_measure")
  VALUES (2, 10000, 'Gram');
INSERT INTO inventory ("ingredient_id", "quantity", "unit_of_measure")
  VALUES (3, 5000, 'Gram');
INSERT INTO inventory ("ingredient_id", "quantity", "unit_of_measure")
  VALUES (4, 100, 'Liter');

CREATE TABLE recipes (
  "id" integer generated by default as identity,
  "equipment_id" number,
  "name" nvarchar2(1024),
  "instructions" nvarchar2(1024),
  "creation_date" timestamp(3),
  "last_accessed" date,
  "last_modified" timestamp(6) with time zone,
  foreign key ("equipment_id") references equipment("id"),
  primary key ("id")
);
INSERT INTO recipes ("name", "instructions", "equipment_id", "creation_date", "last_accessed", "last_modified")
  VALUES ('Espresso single shot','do this', 2, to_timestamp('2017-12-13T00:00:00.123', 'YYYY-MM-DD"T"HH24:MI:SSXFF3'), to_date('00:00:01', 'HH24:MI:SS'), to_timestamp_tz('2017-12-14T00:00:00.123+00:00', 'YYYY-MM-DD"T"HH24:MI:SSXFFTZH:TZM'));
INSERT INTO recipes ("name", "instructions", "equipment_id", "creation_date", "last_accessed", "last_modified")
  VALUES ('Ibrik (turkish) coffee', 'do that', 4, to_timestamp('2017-12-13T00:00:00.123', 'YYYY-MM-DD"T"HH24:MI:SSXFF3'), to_date('00:00:02', 'HH24:MI:SS'), to_timestamp_tz('2017-12-14T00:00:00.123456+00:00', 'YYYY-MM-DD"T"HH24:MI:SSXFFTZH:TZM'));
INSERT INTO recipes ("name", "instructions", "equipment_id", "creation_date", "last_accessed", "last_modified")
  VALUES ('Filter coffee', 'do bar', 3, to_timestamp('2017-12-13T00:00:00.123', 'YYYY-MM-DD"T"HH24:MI:SSXFF3'), to_date('12:00:00', 'HH24:MI:SS'), to_timestamp_tz('2017-12-14T00:00:00.000+00:00', 'YYYY-MM-DD"T"HH24:MI:SSXFFTZH:TZM'));

CREATE TABLE ingredient_recipe (
  "id" integer generated by default as identity,
  "ingredient_id" number not null,
  "recipe_id" number not null,
  "quantity" number,
  "unit_of_measure" nvarchar2(1024),
  foreign key ("ingredient_id") references ingredients("id"),
  foreign key ("recipe_id") references recipes("id"),
  primary key ("ingredient_id", "recipe_id")
);
INSERT INTO ingredient_recipe ("id", "ingredient_id", "recipe_id", "quantity", "unit_of_measure")
  VALUES (1, 2, 3, 30, 'Gram');
INSERT INTO ingredient_recipe ("id", "ingredient_id", "recipe_id", "quantity", "unit_of_measure")
  VALUES (2, 1, 3, 1, 'Each');
INSERT INTO ingredient_recipe ("id", "ingredient_id", "recipe_id", "quantity", "unit_of_measure")
  VALUES (3, 4, 3, 0.5, 'Liter');
INSERT INTO ingredient_recipe ("id", "ingredient_id", "recipe_id", "quantity", "unit_of_measure")
  VALUES (4, 3, 2, 20, 'Gram');
INSERT INTO ingredient_recipe ("id", "ingredient_id", "recipe_id", "quantity", "unit_of_measure")
  VALUES (5, 4, 2, 0.15, 'Liter');
__EOF__
