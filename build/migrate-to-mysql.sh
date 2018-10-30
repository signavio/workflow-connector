#!/usr/bin/env sh

source ./.env
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
  ("Bialetti Moka Express 6 cup", 25.95, "2017-12-12 12:00:00"),
  ("Sanremo Café Racer", 8477.85,"2017-12-12 12:00:00"),
  ("Buntfink SteelKettle", 39.95,"2017-12-12 12:00:00"),
  ("Copper Coffee Pot Cezve", 49.95,"2017-12-12 12:00:00");

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

