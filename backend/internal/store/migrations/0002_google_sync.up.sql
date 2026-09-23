-- Migration 0002 — google_sync: the Google Calendar event and Tasks list ids
-- that POST /api/v1/integrations/google/sync (backend spec §6.4) needs for an
-- idempotent re-sync. Spec §3.2 has no column for them; one row per user,
-- because the Calendar event exists even when the user has no roadmap.
-- roadmap_id records which roadmap the task list was built for, so a new
-- roadmap gets a fresh list while a same-roadmap re-sync creates nothing.

CREATE TABLE google_sync (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    calendar_event_id TEXT,
    tasklist_id TEXT,
    roadmap_id UUID REFERENCES roadmaps(id) ON DELETE SET NULL,
    tasks_created_count INT NOT NULL DEFAULT 0,
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
