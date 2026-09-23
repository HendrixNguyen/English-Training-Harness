-- Migration 0003 — the pet's own once-per-day verdict markers (pet day-judgement plan).
-- last_target_met_date: the local YYYY-MM-DD whose §8 success (+20, streak+1) was
--   applied last; Service.OnTargetMet writes it under a conditional UPDATE so a
--   second call for the same day is a no-op whatever quests' Redis counter says.
-- judged_through: the latest local YYYY-MM-DD that can no longer be penalised —
--   its miss was applied, it was spared, or a passed revival resolved it. The
--   hourly sweep and Revive write it, also conditionally.
-- Both are NULL for existing rows; the sweep initialises judged_through on first contact.

ALTER TABLE pet_states
    ADD COLUMN last_target_met_date DATE,
    ADD COLUMN judged_through DATE;
