-- Rollback: remove test data
DELETE FROM test_table WHERE id IN (1, 2);
