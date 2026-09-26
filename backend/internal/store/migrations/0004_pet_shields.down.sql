-- Reverse of 0004_pet_shields.up.sql.

ALTER TABLE pet_states
    DROP COLUMN IF EXISTS shields,
    DROP COLUMN IF EXISTS last_shield_used_on;
