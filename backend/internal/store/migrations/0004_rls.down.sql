-- 0004 down — disable RLS on the same eight tables. Grants are not restored:
-- they were Supabase's own defaults, not ours to recreate.
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE push_subscriptions DISABLE ROW LEVEL SECURITY;
ALTER TABLE pet_states DISABLE ROW LEVEL SECURITY;
ALTER TABLE daily_progress DISABLE ROW LEVEL SECURITY;
ALTER TABLE roadmaps DISABLE ROW LEVEL SECURITY;
ALTER TABLE exercises DISABLE ROW LEVEL SECURITY;
ALTER TABLE google_sync DISABLE ROW LEVEL SECURITY;
ALTER TABLE schema_migrations DISABLE ROW LEVEL SECURITY;
