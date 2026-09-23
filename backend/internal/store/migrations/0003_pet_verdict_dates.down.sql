-- Reverse of 0003_pet_verdict_dates.up.sql.

ALTER TABLE pet_states
    DROP COLUMN IF EXISTS last_target_met_date,
    DROP COLUMN IF EXISTS judged_through;
