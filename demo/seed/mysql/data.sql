USE fuse_test;

INSERT INTO orders (user_id, total, status) VALUES
  (1,  99.50, 'shipped'),
  (1,  12.00, 'pending'),
  (2,  45.00, 'shipped'),
  (99,  1.00, 'orphan');
