#!/usr/bin/env sh

source ./.env
PGSQL_ROOT_HOST=${PGSQL_ROOT_HOST:=localhost}
PGSQL_ROOT_PASSWORD=${PGSQL_ROOT_PASSWORD:=root}
PGSQL_TEST_USER=${PGSQL_TEST_USER:=test}
PGSQL_TEST_PASSWORD=${PGSQL_TEST_PASSWORD:=test}
PGSQL_TEST_DATABASE=${PGSQL_TEST_DATABASE:=signavio_test}
cat << __EOF__ | psql -U postgres
DROP USER IF EXISTS ${PGSQL_TEST_USER};
CREATE USER ${PGSQL_TEST_USER} WITH PASSWORD '${PGSQL_TEST_PASSWORD}';
DROP DATABASE IF EXISTS ${PGSQL_TEST_DATABASE};
CREATE DATABASE ${PGSQL_TEST_DATABASE};
GRANT ALL PRIVILEGES ON DATABASE ${PGSQL_TEST_DATABASE} TO ${PGSQL_TEST_USER};

\connect ${PGSQL_TEST_DATABASE};
BEGIN;

CREATE TABLE IF NOT EXISTS equipment (
  id serial,
  name text,
  acquisition_cost decimal(10,5),
  purchase_date timestamp,
  primary key (id)
);
INSERT INTO equipment (name, acquisition_cost, purchase_date)
  VALUES
  ('Bialetti Moka Express 6 cup', 25.95, '2017-12-12 12:00:00'),
  ('Sanremo Café Racer', 8477.85,'2017-12-12 12:00:00'),
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
  equipment integer,
  name text,
  instructions text,
  foreign key (equipment) references equipment(id),
  primary key (id)
);
INSERT INTO recipes (name, instructions, equipment)
  VALUES
  ('Espresso single shot','do this', 2),
  ('Ibrik (turkish) coffee', 'do that', 4),
  ('Filter coffee', 'do bar', 2);

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

