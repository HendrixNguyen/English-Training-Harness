-- Reverse of 0001_init.up.sql. Children before parents, tables before types.

DROP TABLE IF EXISTS exercises;
DROP TABLE IF EXISTS roadmaps;
DROP TABLE IF EXISTS daily_progress;
DROP TABLE IF EXISTS pet_states;
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS task_category;
DROP TYPE IF EXISTS pet_stage;
DROP TYPE IF EXISTS cefr_level;
