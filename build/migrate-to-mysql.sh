#!/usr/bin/env sh

MYSQL_HOST=${MYSQL_HOST:=localhost}
MYSQL_ROOT_HOST=${MYSQL_ROOT_HOST:="%"}
MYSQL_USER=${MYSQL_USER:=signavio}
MYSQL_DATABASE=${MYSQL_DATABASE:=signavio_test}
# Source sensitive environment variables from .env
if [ -f .env ]
then
    # shellcheck source=.env
    . ./.env
fi
if [ -n "${MYSQL_PASSWORD}" ]
then
    MYSQL_CMD='mysql -v -u '"${MYSQL_USER}"' -h '"${MYSQL_HOST}"' -p'"${MYSQL_PASSWORD}"''
else
    MYSQL_CMD="mysql -v -u '"${MYSQL_USER}"' -h ${MYSQL_HOST}"
fi
cat << __EOF__ | ${MYSQL_CMD}
DROP DATABASE IF EXISTS ${MYSQL_DATABASE};
CREATE DATABASE ${MYSQL_DATABASE};

USE ${MYSQL_DATABASE}
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
  ("Bialetti Moka Express 6 cup", 25.95, STR_TO_DATE("2017-12-11T12:00:00Z", "%Y-%m-%dT%TZ")),
  ("Sanremo Café Racer", 8477.85, STR_TO_DATE("2017-12-12T12:00:00Z", "%Y-%m-%dT%TZ")),
  ("Buntfink SteelKettle", 39.95, STR_TO_DATE("2017-12-12T12:00:00Z", "%Y-%m-%dT%TZ")),
  ("Copper Coffee Pot Cezve", 49.95, STR_TO_DATE("2017-12-12T12:00:00Z", "%Y-%m-%dT%TZ"));

CREATE TABLE IF NOT EXISTS ingredients (
  id INT NOT NULL AUTO_INCREMENT,
  name text,
  description text,
  primary key (id)
);
INSERT INTO ingredients (name,description)
  VALUES
  ("V60 paper filter", "The best paper filter on the market"),
  ("Caffé Borbone Beans - Miscela Blu", "Excellent beans for espresso"),
  ("Caffé Borbone Beans - Miscela Oro", "Well balanced beans"),
  ("Filtered Water", "Contains the perfect water hardness for espresso");

CREATE TABLE IF NOT EXISTS inventory (
  ingredient_id INT NOT NULL,
  quantity real,
  unit_of_measure text,
  foreign key (ingredient_id) references ingredients(id),
  primary key (ingredient_id)
);
INSERT INTO inventory (ingredient_id, quantity, unit_of_measure)
  VALUES
  (1, 100, "Each"),
  (2, 10000, "Gram"),
  (3, 5000, "Gram"),
  (4, 100, "Liter");

CREATE TABLE IF NOT EXISTS recipes (
  id INT NOT NULL AUTO_INCREMENT,
  equipment_id integer,
  name text,
  instructions text,
  foreign key (equipment_id) references equipment(id),
  primary key (id)
);
INSERT INTO recipes (name, instructions, equipment_id)
  VALUES
  ("Espresso single shot","do this", 2),
  ("Ibrik (turkish) coffee", "do that", 4),
  ("Filter coffee", "do bar", 3);

CREATE TABLE IF NOT EXISTS ingredient_recipe (
  id INT NOT NULL AUTO_INCREMENT,
  ingredient_id INT NOT NULL,
  recipe_id INT NOT NULL,
  quantity real,
  unit_of_measure text,
  index (id),
  foreign key (ingredient_id) references ingredients(id),
  foreign key (recipe_id) references recipes(id),
  primary key (ingredient_id, recipe_id)
);
INSERT INTO ingredient_recipe (id, ingredient_id, recipe_id, quantity, unit_of_measure)
  VALUES
  (1, 2, 3, "30", "Gram"),
  (2, 1, 3, "1", "Each"),
  (3, 4, 3, "0.5", "Liter"),
  (4, 3, 2, "20", "Gram"),
  (5, 4, 2, "0.15", "Liter");

COMMIT;
__EOF__

