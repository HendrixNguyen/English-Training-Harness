-- Migration 0001 — initial schema, transcribed verbatim from
-- project-base/1st-thinking-architecture-doc.md §3.2 (the spec escapes underscores; this does not).
-- gen_random_uuid() is core in PostgreSQL 13+; docker-compose.yml pins postgres:16.

CREATE TYPE cefr_level AS ENUM ('A1', 'A2', 'B1', 'B2', 'C1', 'C2');

CREATE TYPE pet_stage AS ENUM ('seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted');

CREATE TYPE task_category AS ENUM ('vocabulary', 'reading', 'practice');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255),
    google_id VARCHAR(255) UNIQUE NOT NULL,
    google_refresh_token TEXT,
    cefr_current cefr_level DEFAULT 'A1',
    target_goal VARCHAR(255) NOT NULL,
    notification_time TIME DEFAULT '20:00:00',
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE push_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pet_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plant_name VARCHAR(100) DEFAULT 'My Green Buddy',
    health_points INT DEFAULT 100 CHECK (health_points BETWEEN 0 AND 100),
    stage pet_stage DEFAULT 'sprout',
    current_streak INT DEFAULT 0,
    last_practiced_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE daily_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    minutes_spent INT DEFAULT 0,
    is_target_met BOOLEAN DEFAULT FALSE,
    UNIQUE(user_id, date)
);

CREATE TABLE roadmaps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    roadmap_json JSONB NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE exercises (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    roadmap_id UUID REFERENCES roadmaps(id) ON DELETE CASCADE,
    day_number INT NOT NULL,
    task_type task_category NOT NULL,
    content_json JSONB NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE
);
