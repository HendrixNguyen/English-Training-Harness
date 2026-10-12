-- Historical rows remain NULL: no invented answer or duration history.
ALTER TABLE exercises ADD COLUMN quiz_result JSONB;
