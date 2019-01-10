#!/bin/sh

cat << __EOF__ | sqlite3 test.db
BEGIN;
DROP TABLE IF EXISTS equipment;
DROP TABLE IF EXISTS ingredients;
DROP TABLE IF EXISTS inventory;
DROP TABLE IF EXISTS recipes;
DROP TABLE IF EXISTS ingredient_recipe;

CREATE TABLE IF NOT EXISTS equipment (
  id integer primary key autoincrement,
  name text,
  acquisition_cost decimal(10,5),
  purchase_date date
);
INSERT INTO 'equipment' ('name','acquisition_cost','purchase_date')
  VALUES
  ('Bialetti Moka Express 6 cup', 25.95, '2017-12-11T12:00:00Z'),
  ('Sanremo Café Racer', 8477.85,'2017-12-12T12:00:00Z'),
  ('Buntfink SteelKettle', 39.95,'2017-12-12T12:00:00Z'),
  ('Copper Coffee Pot Cezve', 49.95,'2017-12-12T12:00:00Z');

CREATE TABLE IF NOT EXISTS ingredients (
  id integer primary key autoincrement,
  name text,
  description text
);
INSERT INTO 'ingredients' ('name', 'description')
  VALUES
  ('V60 paper filter', 'The best paper filter on the market'),
  ('Caffé Borbone Beans - Miscela Blu', 'Excellent beans for espresso'),
  ('Caffé Borbone Beans - Miscela Oro', 'Well balanced beans'),
  ('Filtered Water', 'Contains the perfect water hardness for espresso');

CREATE TABLE IF NOT EXISTS inventory (
  ingredient_id integer primary key,
  quantity real,
  unit_of_measure text,
  foreign key (ingredient_id) references ingredients(id)
);
INSERT INTO 'inventory' ('ingredient_id','quantity','unit_of_measure')
  VALUES
  (1, 100, 'Each'),
  (2, 10000, 'Gram'),
  (3, 5000, 'Gram'),
  (4, 100, 'Liter');

CREATE TABLE IF NOT EXISTS recipes (
  id integer primary key autoincrement,
  equipment_id integer,
  name text,
  instructions text,
  foreign key (equipment_id) references equipment(id)
);
INSERT INTO 'recipes' ('name', 'instructions', 'equipment_id')
  VALUES
  ('Espresso single shot','do this', 2),
  ('Ibrik (turkish) coffee', 'do that', 4),
  ('Filter coffee', 'do bar', 3);

CREATE TABLE IF NOT EXISTS ingredient_recipe (
  id integer not null,
  ingredient_id integer,
  recipe_id integer,
  quantity text,
  unit_of_measure text,
  foreign key (ingredient_id) references ingredients(id),
  foreign key (recipe_id) references recipes(id),
  primary key (ingredient_id, recipe_id)
);
INSERT INTO 'ingredient_recipe' ('id', 'ingredient_id', 'recipe_id', 'quantity', 'unit_of_measure')
  VALUES
  (1, 2, 3, '30', 'Gram'),
  (2, 1, 3, '1', 'Each'),
  (3, 4, 3, '0.5', 'Liter'),
  (4, 3, 2, '20', 'Gram'),
  (5, 4, 2, '0.15', 'Liter');

COMMIT;
__EOF__
