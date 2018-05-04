#!/bin/sh

cat << __EOF__ | sqlite3 test.db
BEGIN;

CREATE TABLE IF NOT EXISTS teacher (
  id integer primary key autoincrement,
  code char(6) unique not null
);
UPDATE SQLITE_SEQUENCE SET seq = 1000 WHERE name = 'teacher';
INSERT INTO 'teacher' ('code') VALUES
  ('T12903'),
  ('T12144'),
  ('T12933');

CREATE TABLE IF NOT EXISTS organization (
  id integer primary key autoincrement,
  code varchar(8) unique not null
);
UPDATE SQLITE_SEQUENCE SET seq = 2000 WHERE name = 'organization';
INSERT INTO 'organization' ('code') VALUES
  ('MTRLGY'),
  ('FINBUS'),
  ('ASTRNMY'),
  ('MATSTAT');

CREATE TABLE IF NOT EXISTS section (
  id integer primary key autoincrement,
  code char(1) unique not null
);
UPDATE SQLITE_SEQUENCE SET seq = 3000 WHERE name = 'section';
INSERT INTO 'section' ('code') VALUES
  ('A'),
  ('B'),
  ('C'),
  ('D');

CREATE TABLE IF NOT EXISTS course (
  id integer primary key autoincrement,
  code char(4) unique not null
);
UPDATE SQLITE_SEQUENCE SET seq = 4000 WHERE name = 'course';
INSERT INTO 'course' ('code') VALUES
  ('M200'),
  ('M100'),
  ('T110'),
  ('T200');

CREATE TABLE IF NOT EXISTS class (
  id integer primary key autoincrement,
  class_coverage numeric,
  class_startdate date,
  class_time datetime,
  class_duration numeric,
  section_code char(1) not null,
  course_code char(4) not null,
  organization_code varchar(8) not null,
  foreign key (section_code) references section(code),
  foreign key (course_code) references course(code),
  foreign key (organization_code) references organization(code)
);
UPDATE SQLITE_SEQUENCE SET seq = 5000 WHERE name = 'class';
INSERT INTO 'class' (
  'class_coverage',
  'class_startdate',
  'class_time',
  'class_duration',
  'section_code',
  'course_code',
  'organization_code'
)
VALUES
  ('2','2018-09-09','2018-09-09 09:00','1','A','M200','MTRLGY'),
  ('4','2018-09-09','2018-09-09 09:00','1.5','B','M200','MTRLGY'),
  ('4','2018-09-09','2018-09-09 10:00','1','A','M100','FINBUS'),
  ('4','2018-09-08','2018-09-09 11:00','2.25','A','T110','ASTRNMY');

CREATE TABLE IF NOT EXISTS class_teacher (
  id integer primary key autoincrement,
  class_id integer not null,
  teacher_id integer not null,
  foreign key (class_id) references class(id),
  foreign key (teacher_id) references teacher(id)
);
UPDATE SQLITE_SEQUENCE SET seq = 6000 WHERE name = 'class_teacher';

COMMIT;
__EOF__
