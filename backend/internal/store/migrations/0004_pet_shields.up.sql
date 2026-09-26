-- Migration 0004 — the streak shield (pet streak shield plan).
-- shields: how many shields the pet holds, 0..2. saveTargetMetSQL awards one on
--   every 7th consecutive met day (LEAST(2, …)); penaliseMissSQL spends one in
--   place of the -30 / streak reset when a judged day was missed.
-- last_shield_used_on: the local YYYY-MM-DD a shield was last spent for; NULL
--   until the first spend. The client shows the spend for seven days.

ALTER TABLE pet_states
    ADD COLUMN shields INT NOT NULL DEFAULT 0 CHECK (shields BETWEEN 0 AND 2),
    ADD COLUMN last_shield_used_on DATE;
