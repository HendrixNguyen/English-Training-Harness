# **Adaptive English Learning Platform — Backend Technical Specification**

# **1\. Executive Summary & Core Objectives**

This document serves as the primary technical specification for the Go-based Backend Microservice Architecture. It defines the database schema, Redis caching topology, REST API contracts, multi-LLM AI routing engine, background cron workers, security token lifecycle, and virtual plant growth mechanics for the Adaptive English Learning Platform.  
\---

# **2\. Architecture & Technology Stack**

*  Go (Golang) compiled binary running Gin/Fiber framework with integrated background cron worker.  
*  PostgreSQL (User profiles, OAuth tokens, roadmaps, exercise tracking, pet states).  
*  Redis (Session context, rate limiting, temporary quiz state, Web Push delay queues).  
*  Multi-LLM setup incorporating Google Gemini (Flash & Pro), OpenAI (GPT-4o-mini), and DeepSeek (DeepSeek-Chat).  
*  Railway (PaaS) containerized deployment.

```
+-------------------------------------------------------------------------------+
|                               Go API Server                                   |
|         (Gin/Fiber Engine, OAuth Handler, Scheduler, Cron Worker)             |
+-----+-------------------+-------------------+--------------------+------------+
      |                   |                   |                    |
+-----v-------+     +-----v-------+     +-----v--------+     +-----+------+
| PostgreSQL  |     |    Redis    |     | Google APIs  |     | AI Agent   |
| (Database)  |     | (Cache/     |     | (Calendar &  |     | Provider   |
|             |     |  Queues)    |     |  Tasks)      |     | Router     |
+-------------+     +-------------+     +--------------+     +-----+------+
                                                                   |
                                                      +------------+------------+
                                                      |                         |
                                              +-------v-------+         +-------v-------+
                                              | Google Gemini |         |  OpenAI /     |
                                              |     API       |         | DeepSeek API  |
                                              +---------------+         +---------------+
```

\---

# **3\. Database Schema & Entity-Relationship Diagram (ERD)**

## **3.1 Entity-Relationship Diagram**

```
 [ users ]                                  [ pet_states ]
 ---------                                  --------------
 PK  id: UUID                               PK  id: UUID
     email: VARCHAR(255)                    FK  user_id: UUID (1:1)
     google_id: VARCHAR(255)                    plant_name: VARCHAR(100)
     google_refresh_token: TEXT                 health_points: INT (0-100)
     cefr_current: ENUM('A1'..'C2')             stage: ENUM('seed'..'fruitful')
     target_goal: VARCHAR(255)                  current_streak: INT
     notification_time: TIME                    last_practiced_at: TIMESTAMPTZ
     timezone: VARCHAR(50)                  
     created_at: TIMESTAMPTZ                [ push_subscriptions ]
                                            ----------------------
        |                                   PK  id: UUID
        | 1:N                               FK  user_id: UUID (1:N)
        +-----------------------+               endpoint: TEXT
        |                       |               p256dh: TEXT
        v                       v               auth: TEXT
 [ roadmaps ]             [ daily_progress ]
 ------------             ------------------
 PK  id: UUID             PK  id: UUID
 FK  user_id: UUID        FK  user_id: UUID
     roadmap_json: JSONB      date: DATE
     is_active: BOOLEAN       minutes_spent: INT
     created_at: TIMESTAMPTZ  is_target_met: BOOLEAN
                              (UNIQUE: user_id, date)
        |
        | 1:N
        v
 [ exercises ]
 -------------
 PK  id: UUID
 FK  roadmap_id: UUID
     day_number: INT
     task_type: ENUM('vocabulary','reading','practice')
     content_json: JSONB
     is_completed: BOOLEAN
```

## **3.2 SQL DDL Migration Script**

```sql
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

-- Added by migration 0002 (google slice): ids for idempotent Calendar/Tasks re-sync.
CREATE TABLE google_sync (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    calendar_event_id TEXT,
    tasklist_id TEXT,
    roadmap_id UUID REFERENCES roadmaps(id) ON DELETE SET NULL,
    tasks_created_count INT NOT NULL DEFAULT 0,
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Added by migration 0003 (pet day-judgement): the pet's own once-per-day verdict markers.
ALTER TABLE pet_states
    ADD COLUMN last_target_met_date DATE,
    ADD COLUMN judged_through DATE;

-- Added by migration 0004 (streak shield): one shield per 7th consecutive met day, max 2; spent in place of a miss penalty.
ALTER TABLE pet_states
    ADD COLUMN shields INT NOT NULL DEFAULT 0 CHECK (shields BETWEEN 0 AND 2),
    ADD COLUMN last_shield_used_on DATE;
```

\---

# **4\. Redis Key Topology & Data Retention Strategy**

| Key Schema Pattern | Type | Expiration (TTL) | Description |
| :---- | :---- | :---- | :---- |
| sess:{user\_id}:token | String | 24 Hours | Active JWT session context and user metadata. |
| quiz:placement:{user\_id} | Hash | 2 Hours | Transient storage for active placement test answers before grading. |
| daily:accumulated:{user\_id}:{YYYY-MM-DD} | String (Int) | 48 Hours | Counter tracking active study duration in seconds for the target day. |
| queue:webpush:delay | Sorted Set | Persistent | Redis ZSET storing UNIX timestamps as scores to trigger scheduled Web Push reminders. |
| ratelimit:ai:{user\_id} | String (Int) | 1 Minute | Slotted rate limiter preventing AI model generation abuse (max 5 req/min). |

\---

# **5\. AI Agent Router & Dynamic Roadmap Engine**

## **5.1 Go Router Implementation**

```go
package airouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type TaskType string

const (
	TaskPlacementTest TaskType = "placement_test"
	TaskRoadmapGen    TaskType = "roadmap_generation"
	TaskExerciseGen   TaskType = "exercise_generation"
	TaskEssayGrading  TaskType = "essay_grading"
)

type ProviderType string

const (
	ProviderGemini   ProviderType = "gemini"
	ProviderOpenAI   ProviderType = "openai"
	ProviderDeepSeek ProviderType = "deepseek"
)

type LLMProvider interface {
	GenerateContent(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}

type Router struct {
	providers  map[ProviderType]LLMProvider
	strategies map[TaskType]ProviderType
}

func NewRouter() (*Router, error) {
	providers := make(map[ProviderType]LLMProvider)
	if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
		providers[ProviderGemini] = NewGeminiProvider(geminiKey)
	}
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		providers[ProviderOpenAI] = NewOpenAICompatibleProvider(os.Getenv("OPENAI_BASE_URL"), openaiKey, "gpt-4o-mini")
	}
	if deepseekKey := os.Getenv("DEEPSEEK_API_KEY"); deepseekKey != "" {
		providers[ProviderDeepSeek] = NewOpenAICompatibleProvider(os.Getenv("DEEPSEEK_BASE_URL"), deepseekKey, "deepseek-chat")
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("airouter: no API keys configured")
	}

	strategies := map[TaskType]ProviderType{
		TaskRoadmapGen:    ProviderGemini,
		TaskExerciseGen:   ProviderDeepSeek,
		TaskPlacementTest: ProviderGemini,
		TaskEssayGrading:  ProviderOpenAI,
	}

	return &Router{providers: providers, strategies: strategies}, nil
}
```

\---

# **6\. REST API Endpoint Specifications & DTO Contracts**

## **6.1 Authentication & Onboarding Endpoints**

* POST /api/v1/auth/google  
  * Description: Swaps Google OAuth authorization code for tokens, encrypts refresh token with AES-256-GCM, and issues signed JWT.  
  * Request Headers: Content-Type: application/json  
  * Request Body:

```json
{"code": "4/0AeaYSHC...", "redirect_uri": "postmessage"}
```

  * Response (200 OK):

```json
{"access_token": "eyJhbGciOiJKV1QiLC...", "token_type": "Bearer", "expires_in": 86400, "user": {"id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", "email": "user@example.com", "full_name": "Nguyen Hendrix", "cefr_current": "B1"}}
```

* POST /api/v1/onboarding/assessment  
  * Description: Submitting placement quiz answers, invoking Gemini AI for CEFR grading, generating 28-day roadmap JSON, and creating initial pet state.  
  * Request Headers: Authorization: Bearer \<JWT\>, Content-Type: application/json  
  * Request Body:

```json
{"target_goal": "IELTS 7.0 Preparation", "notification_time": "20:00:00", "timezone": "Asia/Ho_Chi_Minh", "answers": [{ "question_id": "q1", "selected_option": "B" }, { "question_id": "q2", "selected_option": "A" }]}
```

  * Response (201 Created):

```json
{"status": "success", "assessed_level": "B1", "roadmap_id": "b11c22d3-44e5-66f7-88a9-00bbccddeeff", "pet_state": {"plant_name": "My Green Buddy", "health_points": 100, "stage": "sprout"}}
```

## **6.2 Quests & Progress Endpoints**

* GET /api/v1/quests/daily  
  * Description: Fetching current day's 30-minute exercise suite (3x 10-min tasks).  
  * Request Headers: Authorization: Bearer \<JWT\>  
  * Response (200 OK):

```json
{"date": "2026-09-22", "day_number": 1, "total_minutes_required": 30, "accumulated_seconds": 600, "is_target_met": false, "tasks": [{"id": "ex-01", "task_type": "vocabulary", "title": "10 Key Business Email Phrasings", "duration_minutes": 10, "is_completed": true, "content_json": { "words": [{ "term": "Inquire", "definition": "To ask for information" }] }}]}
```

* POST /api/v1/quests/progress  
  * Description: Incrementing Redis counter (INCRBY). When 1,800s (30 mins) is reached, sets is\_target\_met \= true, increases plant health (+20%), and increments streak.  
  * Request Headers: Authorization: Bearer \<JWT\>, Content-Type: application/json  
  * Request Body:

```json
{"exercise_id": "ex-02", "duration_seconds": 600, "user_answers": { "q1": "A" }}
```

  * Response (200 OK):

```json
{"daily_seconds_spent": 1200, "daily_minutes_spent": 20, "is_target_met": false, "pet_health": 100, "streak_count": 5}
```

## **6.3 Pet State & Revival Endpoints**

* GET /api/v1/pet/status  
  * Description: Retrieving plant stage, health %, and streak count.  
  * shields (0\-2) and last\_shield\_used\_on (YYYY\-MM\-DD or null) are additive (streak shield, 2026\-09\-25): one shield is earned on every 7th consecutive met day and one is spent, in place of the miss penalty, on a missed day.  
  * Request Headers: Authorization: Bearer \<JWT\>  
  * Response (200 OK):

```json
{"plant_name": "My Green Buddy", "health_points": 80, "stage": "sprout", "current_streak": 5, "last_practiced_at": "2026-09-21T20:15:00Z", "shields": 1, "last_shield_used_on": null}
```

* POST /api/v1/pet/revive  
  * Description: Triggering 15-minute revival challenge when plant health reaches 0% (wilted). Resets health to 50% upon passing.  
  * Request Headers: Authorization: Bearer \<JWT\>, Content-Type: application/json  
  * Request Body:

```json
{"answers": { "q1": "C", "q2": "B" }}
```

  * Response (200 OK):

```json
{"revival_passed": true, "pet_state": {"health_points": 50, "stage": "sprout", "current_streak": 0}}
```

## **6.4 Settings & Integration Endpoints**

* POST /api/v1/settings/notifications  
  * Description: Saving Web Push VAPID subscriptions and updating preferred practice time.  
  * Request Headers: Authorization: Bearer \<JWT\>, Content-Type: application/json  
  * Request Body:

```json
{"notification_time": "20:00:00", "push_subscription": {"endpoint": "push_subscription_endpoint_string", "p256dh": "BNc5T...", "auth": "aX8v..."}}
```

  * Response (200 OK):

```json
{"status": "updated", "notification_time": "20:00:00"}
```

* POST /api/v1/integrations/google/sync  
  * Description: Asynchronously pushing 30-minute recurring study blocks to Google Calendar and task checklists to Google Tasks.  
  * Request Headers: Authorization: Bearer \<JWT\>  
  * Response (200 OK):

```json
{"status": "synced", "calendar_event_id": "cal_evt_12345", "tasks_created_count": 3}
```

\---

# **7\. Security & Token Lifecycle Architecture**

*  users.google\_refresh\_token encrypted with AES-256-GCM via ENCRYPTION\_SECRET\_KEY (32-byte hex).  
*  JWT access token validated against Redis key (sess:{user\_id}:token) with a 24-hour TTL.

|             |     |  Queues)    |     |  Tasks)      |     | Router     |

\---

# **8\. Virtual Plant Math & Hourly Cron Worker**

*  Runs at :00 UTC every hour to detect users hitting midnight in their local timezone (timezone).  
* Inactivity Logic:  
  * If daily time \< 1,800 seconds: Health \= Max(0, Health \- 30).  
  * If Health \== 0: Transition plant stage to wilted.  
* Success Logic:  
  * If daily target met (\>= 30 mins): Health \= Min(100, Health \+ 20\) and Streak \= Streak \+ 1\.

\---

# **9\. Deployment & Infrastructure Checklist (Railway)**

1.  Provision PostgreSQL & Redis plugins on Railway and execute DDL migration scripts.  
2.  Inject DATABASE\_URL, REDIS\_URL, GOOGLE\_CLIENT\_ID, GOOGLE\_CLIENT\_SECRET, GEMINI\_API\_KEY, OPENAI\_API\_KEY, DEEPSEEK\_API\_KEY, ENCRYPTION\_SECRET\_KEY, JWT\_SECRET (at least 32 bytes), VAPID\_PUBLIC\_KEY/VAPID\_PRIVATE\_KEY, and FRONTEND\_ORIGIN (the PWA's origin, comma-separated if several).  
3.  Deploy compiled Go binary in a lightweight Docker container on Railway.
4.  Addendum (2026-09-25, see `deploy/README.md`): `ENCRYPTION_SECRET_KEY` (64 hex) and `JWT_SECRET` (≥ 32 bytes) are boot requirements; `FRONTEND_ORIGIN` must list the PWA's exact origin. Hosting today is the free split (API on Railway Free, Postgres on Supabase via the session-mode pooler on port 5432 with `sslmode=require` — Railway egress is IPv4-only — Redis on Upstash over `rediss://`, PWA on Cloudflare Pages); later the owner's Dokploy server runs `deploy/compose.yml`. The images are `backend/Dockerfile` and `frontend/Dockerfile`.

# 

## 

## 

## 

## 

## 

* 