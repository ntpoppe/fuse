CREATE DATABASE IF NOT EXISTS fuse_test;
USE fuse_test;

CREATE TABLE orders (
  id       INT PRIMARY KEY AUTO_INCREMENT,
  user_id  INT NOT NULL,
  total    DECIMAL(10, 2) NOT NULL,
  status   VARCHAR(20) NOT NULL
);

CREATE USER IF NOT EXISTS 'demo'@'%' IDENTIFIED BY 'demo';
GRANT SELECT ON fuse_test.orders TO 'demo'@'%';
FLUSH PRIVILEGES;
