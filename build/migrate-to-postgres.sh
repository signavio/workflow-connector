#!/usr/bin/env sh

# Source sensitive environment variables from .env if file exists
# shellcheck source=.env
if [ -f .env ]
then
    . ./.env
fi
POSTGRES_ROOT_HOST=${POSTGRES_ROOT_HOST:=localhost}
POSTGRES_ROOT_PASSWORD=${POSTGRES_ROOT_PASSWORD:=root}
POSTGRES_USER=${POSTGRES_USER:=signavio}
POSTGRES_ROOT_USER=${POSTGRES_ROOT_USER:=postgres}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:=test}
POSTGRES_DATABASE=${POSTGRES_DATABASE:=signavio_test}
cat << __EOF__ | psql -a -U ${POSTGRES_ROOT_USER} -h ${POSTGRES_ROOT_HOST}
DROP USER IF EXISTS ${POSTGRES_USER};
CREATE USER ${POSTGRES_USER} WITH PASSWORD '${POSTGRES_PASSWORD}';
DROP DATABASE IF EXISTS ${POSTGRES_DATABASE};
CREATE DATABASE ${POSTGRES_DATABASE};
ALTER DATABASE ${POSTGRES_DATABASE} OWNER TO ${POSTGRES_USER};
GRANT ALL PRIVILEGES ON DATABASE ${POSTGRES_DATABASE} TO ${POSTGRES_USER};
\connect ${POSTGRES_DATABASE};
SET ROLE ${POSTGRES_USER};
BEGIN;

CREATE TABLE IF NOT EXISTS equipment (
  id serial,
  name text,
  acquisition_cost decimal(10,5),
  purchase_date timestamp(3),
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
  creation_date timestamp(3) with time zone,
  last_accessed date,
  last_modified timestamp without time zone,
  foreign key (equipment_id) references equipment(id),
  primary key (id)
);
INSERT INTO recipes (name, instructions, equipment_id, creation_date, last_accessed, last_modified)
  VALUES
  ('Espresso single shot','do this', 2, '2017-12-13T23:00:00.123', '2017-01-13', '2017-12-14T00:00:00.123'),
  ('Ibrik (turkish) coffee', 'do that', 4, '2017-12-13T23:00:00.123', '2017-01-13', '2017-12-14T00:00:00.123456'),
  ('Filter coffee', 'do bar', 3, '2017-12-13T23:00:00.123', '2017-01-13', '2017-12-14T00:00:00');

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

