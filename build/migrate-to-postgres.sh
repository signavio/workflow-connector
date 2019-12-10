#!/usr/bin/env sh
# Source sensitive environment variables from .env if file exists
# shellcheck source=.env
if [ -f .env ]
then
    . ./.env
fi
DATABASE_URL=${DATABASE_URL:=postgresql://test:test@localhost:5432/test}
cat << __EOF__ | psql -a -d "${DATABASE_URL}"
DROP USER IF EXISTS ${POSTGRES_USER};
CREATE USER ${POSTGRES_USER} WITH PASSWORD '${POSTGRES_PASSWORD}';
DROP DATABASE IF EXISTS ${POSTGRES_DATABASE};
CREATE DATABASE ${POSTGRES_DATABASE};
ALTER DATABASE ${POSTGRES_DATABASE} OWNER TO ${POSTGRES_USER};
GRANT ALL PRIVILEGES ON DATABASE ${POSTGRES_DATABASE} TO ${POSTGRES_USER};
\connect ${POSTGRES_DATABASE};
SET timezone='CET';
SET ROLE ${POSTGRES_USER};
DROP TABLE IF EXISTS zero_rows;
DROP TABLE IF EXISTS one_rows;
DROP TABLE IF EXISTS "funny column names";
DROP TABLE IF EXISTS ingredient_recipe;
DROP TABLE IF EXISTS inventory;
DROP TABLE IF EXISTS ingredients;
DROP TABLE IF EXISTS recipes;
DROP TABLE IF EXISTS equipment;
BEGIN;

CREATE TABLE IF NOT EXISTS zero_rows (
  id serial,
  name text,
  primary key (id)
);
CREATE TABLE IF NOT EXISTS one_rows (
  id serial,
  name text,
  primary key (id)
);
INSERT INTO one_rows (name)
  VALUES
  ('TESTNAME');
CREATE TABLE IF NOT EXISTS "funny column names" (
  id serial,
  jack_bob text,
  "cup smith" text,
  "bent.ski;" text,
  "utf8 string ڣ" text,
  primary key (id)
);
INSERT INTO "funny column names" ("cup smith", "bent.ski;", "utf8 string ڣ")
  VALUES
  ('foo', 'bar', 'baz');
CREATE TABLE IF NOT EXISTS equipment (
  id serial,
  name text,
  acquisition_cost decimal(10,5),
  purchase_date timestamp(3) null,
  primary key (id)
);

CREATE TABLE IF NOT EXISTS equipment (
  id serial,
  name text,
  acquisition_cost decimal(10,5),
  purchase_date timestamp(3) null,
  primary key (id)
);
INSERT INTO equipment (name, acquisition_cost, purchase_date)
  VALUES
  ('Bialetti Moka Express 6 cup', 25.95, '2017-12-11 12:00:00.123'),
  ('Sanremo Café Racer', 8477.85,'2017-12-12 12:00:00.123456'),
  ('Buntfink SteelKettle', 39.95,'2017-12-12 12:00:00'),
  ('Copper Coffee Pot Cezve', 49.95,'2017-12-12 12:00:00');

CREATE TABLE IF NOT EXISTS ingredients (
  id serial,
  name text,
  description text,
  primary key (id)
);
INSERT INTO ingredients (name,description)
  VALUES
  ('V60 paper filter', 'The best paper filter on the market'),
  ('Caffé Borbone Beans - Miscela Blu', 'Excellent beans for espresso'),
  ('Caffé Borbone Beans - Miscela Oro', 'Well balanced beans'),
  ('Filtered Water', 'Contains the perfect water hardness for espresso');

CREATE TABLE IF NOT EXISTS inventory (
  ingredient_id INT NOT NULL,
  quantity real,
  unit_of_measure text,
  foreign key (ingredient_id) references ingredients(id),
  primary key (ingredient_id)
);
INSERT INTO inventory (ingredient_id, quantity, unit_of_measure)
  VALUES
  (1, 100, 'Each'),
  (2, 10000, 'Gram'),
  (3, 5000, 'Gram'),
  (4, 100, 'Liter');

CREATE TABLE IF NOT EXISTS recipes (
  id serial,
  equipment_id integer,
  name text,
  instructions text,
  creation_date date,
  last_accessed timestamp(3) without time zone,
  last_modified timestamp with time zone,
  foreign key (equipment_id) references equipment(id),
  primary key (id)
);
INSERT INTO recipes (name, instructions, equipment_id, creation_date, last_accessed, last_modified)
  VALUES
  ('Espresso single shot','do this', 2, '2017-12-13', '2017-01-13 00:00:01', '2017-12-14T01:00:00.123'),
  ('Ibrik (turkish) coffee', 'do that', 4, '2017-12-13', '2017-01-13 00:00:02', '2017-12-14T01:00:00.123456'),
  ('Filter coffee', 'do bar', 3, '2017-12-13', '2017-01-13 12:00:00', '2017-12-14T01:00:00');

CREATE TABLE IF NOT EXISTS ingredient_recipe (
  id serial,
  ingredient_id INT NOT NULL,
  recipe_id INT NOT NULL,
  quantity real,
  unit_of_measure text,
  foreign key (ingredient_id) references ingredients(id),
  foreign key (recipe_id) references recipes(id),
  primary key (ingredient_id, recipe_id)
);
INSERT INTO ingredient_recipe (id, ingredient_id, recipe_id, quantity, unit_of_measure)
  VALUES
  (1, 2, 3, '30', 'Gram'),
  (2, 1, 3, '1', 'Each'),
  (3, 4, 3, '0.5', 'Liter'),
  (4, 3, 2, '20', 'Gram'),
  (5, 4, 2, '0.15', 'Liter');

COMMIT;
__EOF__

