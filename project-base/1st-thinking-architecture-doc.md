# **Adaptive English Learning Platform — System Architecture & Technical Specification**

# **1\. Executive Summary & Core Objectives**

This document details the software architecture, database design, API specification, and AI routing strategy for an adaptive, gamified English learning Progressive Web Application (PWA). The platform is engineered to drive high daily user retention (at least 30 minutes/day) through AI-personalized roadmaps, a virtual pet/plant state engine, and native integration with Google Calendar, Google Tasks, and Web Push Notifications.

# **2\. System Architecture & Technology Stack**

## **2.1 Technology Stack Matrix**

* **Frontend Application:** Nuxt 3 (Vue 3, Pinia state management, Tailwind CSS, @vite-pwa/nuxt PWA module, Service Workers).  
* **Backend Microservice:** Go (Golang) compiled binary running Gin/Fiber framework with integrated background cron worker.  
* **Primary Relational Database:** PostgreSQL (user profiles, auth tokens, roadmaps, exercise tracking, pet state).  
* **In-Memory Cache & Message Queue:** Redis (session context, rate limiting, temporary quiz state, Web Push delay queues).  
* **AI Provider Suite:** Multi-LLM setup incorporating Google Gemini (Flash & Pro) alongside OpenAI (GPT-4o-mini) and DeepSeek (DeepSeek-Chat).  
* **Infrastructure & Hosting:** Railway (PaaS) containerized deployment.

## **2.2 High-Level Architecture Diagram**

\+-------------------------------------------------------------------------------+

|                               Nuxt 3 PWA Client                               |

|---|

\+---------------------------------------+---------------------------------------+

                                        |

                                  REST / WebSocket

                                        |

\+---------------------------------------v---------------------------------------+

|                                Go API Server                                  |

|         (Gin/Fiber Engine, OAuth Handler, Scheduler, Cron Worker)             |

\+-----+-------------------+-------------------+--------------------+------------+

      |                   |                   |                    |

\+-----v-------+     \+-----v-------+     \+-----v--------+     \+-----+------+

| PostgreSQL  |     |    Redis    |     | Google APIs  |     | AI Agent   |

| (Database)  |     | (Cache/     |     | (Calendar &  |     | Provider   |

|             |     |  Queues)    |     |  Tasks)      |     | Router     |

\+-------------+     \+-------------+     \+--------------+     \+-----+------+

                                                                   |

                                                      \+------------+------------+

                                                      |                         |

                                              \+-------v-------+         \+-------v-------+

                                              | Google Gemini |         |  OpenAI /     |

                                              |     API       |         | DeepSeek API  |

                                              \+---------------+         \+---------------+

# **3\. Database Schema & Entity-Relationship Diagram (ERD)**

## **3.1 Entity-Relationship Diagram**

 \[ users \]                                  \[ pet\_states \]

 \---------                                  \--------------

 PK  id: UUID                               PK  id: UUID

     email: VARCHAR(255)                    FK  user\_id: UUID (1:1)

     google\_id: VARCHAR(255)                    plant\_name: VARCHAR(100)

     google\_refresh\_token: TEXT                 health\_points: INT (0-100)

     cefr\_current: ENUM('A1'..'C2')             stage: ENUM('seed'..'fruitful')

     target\_goal: VARCHAR(255)                  current\_streak: INT

     notification\_time: TIME                    last\_practiced\_at: TIMESTAMPTZ

     timezone: VARCHAR(50)                  

     created\_at: TIMESTAMPTZ                \[ push\_subscriptions \]

                                            \----------------------

        |                                   PK  id: UUID

        | 1:N                               FK  user\_id: UUID (1:N)

        \+-----------------------+               endpoint: TEXT

        |                       |               p256dh: TEXT

        v                       v               auth: TEXT

 \[ roadmaps \]             \[ daily\_progress \]

 \------------             \------------------

 PK  id: UUID             PK  id: UUID

 FK  user\_id: UUID        FK  user\_id: UUID

     roadmap\_json: JSONB      date: DATE

     is\_active: BOOLEAN       minutes\_spent: INT

     created\_at: TIMESTAMPTZ  is\_target\_met: BOOLEAN

                              (UNIQUE: user\_id, date)

        |

        | 1:N

        v

 \[ exercises \]

 \-------------

 PK  id: UUID

 FK  roadmap\_id: UUID

     day\_number: INT

     task\_type: ENUM('vocabulary','reading','practice')

     content\_json: JSONB

     is\_completed: BOOLEAN

## **3.2 SQL DDL Migration Script**

CREATE TYPE cefr\_level AS ENUM ('A1', 'A2', 'B1', 'B2', 'C1', 'C2');

CREATE TYPE pet\_stage AS ENUM ('seed', 'sprout', 'sapling', 'flowering', 'fruitful', 'wilted');

CREATE TYPE task\_category AS ENUM ('vocabulary', 'reading', 'practice');

CREATE TABLE users (

    id UUID PRIMARY KEY DEFAULT gen\_random\_uuid(),

    email VARCHAR(255) UNIQUE NOT NULL,

    full\_name VARCHAR(255),

    google\_id VARCHAR(255) UNIQUE NOT NULL,

    google\_refresh\_token TEXT,

    cefr\_current cefr\_level DEFAULT 'A1',

    target\_goal VARCHAR(255) NOT NULL,

    notification\_time TIME DEFAULT '20:00:00',

    timezone VARCHAR(50) DEFAULT 'UTC',

    created\_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT\_TIMESTAMP

);

CREATE TABLE push\_subscriptions (

    id UUID PRIMARY KEY DEFAULT gen\_random\_uuid(),

    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,

    endpoint TEXT NOT NULL,

    p256dh TEXT NOT NULL,

    auth TEXT NOT NULL,

    created\_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT\_TIMESTAMP

);

CREATE TABLE pet\_states (

    id UUID PRIMARY KEY DEFAULT gen\_random\_uuid(),

    user\_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    plant\_name VARCHAR(100) DEFAULT 'My Green Buddy',

    health\_points INT DEFAULT 100 CHECK (health\_points BETWEEN 0 AND 100),

    stage pet\_stage DEFAULT 'sprout',

    current\_streak INT DEFAULT 0,

    last\_practiced\_at TIMESTAMP WITH TIME ZONE,

    updated\_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT\_TIMESTAMP

);

CREATE TABLE daily\_progress (

    id UUID PRIMARY KEY DEFAULT gen\_random\_uuid(),

    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,

    date DATE NOT NULL DEFAULT CURRENT\_DATE,

    minutes\_spent INT DEFAULT 0,

    is\_target\_met BOOLEAN DEFAULT FALSE,

    UNIQUE(user\_id, date)

);

CREATE TABLE roadmaps (

    id UUID PRIMARY KEY DEFAULT gen\_random\_uuid(),

    user\_id UUID REFERENCES users(id) ON DELETE CASCADE,

    roadmap\_json JSONB NOT NULL,

    is\_active BOOLEAN DEFAULT TRUE,

    created\_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT\_TIMESTAMP

);

CREATE TABLE exercises (

    id UUID PRIMARY KEY DEFAULT gen\_random\_uuid(),

    roadmap\_id UUID REFERENCES roadmaps(id) ON DELETE CASCADE,

    day\_number INT NOT NULL,

    task\_type task\_category NOT NULL,

    content\_json JSONB NOT NULL,

    is\_completed BOOLEAN DEFAULT FALSE

);

# **4\. Redis Key Topology & Data Retention Strategy**

| Key Schema Pattern | Type | Expiration (TTL) | Description |
| :---- | :---- | :---- | :---- |
| `sess:{user_id}:token` | **String** | 24 Hours | Active JWT session context and user metadata. |
| `quiz:placement:{user_id}` | **Hash** | 2 Hours | Transient storage for active placement test answers before grading. |
| `daily:accumulated:{user_id}:{YYYY-MM-DD}` | **String (Int)** | 48 Hours | Counter tracking active study duration in seconds for the target day. |
| `queue:webpush:delay` | **Sorted Set** | Persistent | Redis ZSET storing UNIX timestamps as scores to trigger scheduled Web Push reminders. |
| `ratelimit:ai:{user_id}` | **String (Int)** | 1 Minute | Slotted rate limiter preventing AI model generation abuse (max 5 req/min). |

# **5\. Architectural Sequence Flows**

## **5.1 Onboarding, Assessment & Initial One-Way Sync**

Client (Nuxt PWA)       Go API Backend       AI Agent Router      Google API Gateway

      |                       |                     |                     |

      |-- 1\. OAuth Callback \-\>|                     |                     |

      |   (Auth Code)         |                     |                     |

      |                       |-- 2\. Exchange Token \---------------------\>|

      |                       |\<-- Store Refresh Token \-------------------|

      |\<-- 3\. Return JWT \-----|                     |                     |

      |                       |                     |                     |

      |-- 4\. Submit Test \----\>|                     |                     |

      |                       |-- 5\. Route Task \---\>|                     |

      |                       |   (Grade & Gen JSON)|                     |

      |                       |\<-- Return Roadmap \--|                     |

      |                       |                     |                     |

      |                       |-- 6\. Push 30-min Recurring Event \--------\>|

      |                       |-- 7\. Push Daily Checklist Tasks \---------\>|

      |\<-- 8\. Init Dashboard \-|                     |                     |

## **5.2 Daily Practice Loop & Pet State Engine**

Client (Nuxt PWA)       Go API Backend          PostgreSQL DB            Redis Cache

      |                       |                      |                        |

      |-- 1\. Complete Task \--\>|                      |                        |

      |   (Duration: 10m)     |                      |                        |

      |                       |-- 2\. INCRBY 600sec \-------------------------\>|

      |                       |\<-- Return Total Seconds (e.g. 1800s) \---------|

      |                       |                      |                        |

      |                       |-- 3\. Mark Progress \-\>|                        |

      |                       |   (TargetMet \= True) |                        |

      |                       |                      |                        |

      |                       |-- 4\. Execute Pet Health State Change \-------\>|

      |                       |   (Health \+= 20%, Streak++)                   |

      |\<-- 5\. Render Growth \--|                      |                        |

      |    Animation & Status |                      |                        |

# **6\. AI Agent Router & Dynamic Roadmap Prompt Engine**

## **6.1 JSON System Prompt**

You are an elite AI Language Curriculum Architect. Your job is to create a structured, highly personalized learning roadmap for an English learner based on their current CEFR level, target goal, and daily study time commitment.

CRITICAL CONSTRAINTS:

1\. Output ONLY valid JSON matching the requested schema. No markdown backticks, no code blocks, no conversational preamble.

2\. Structure the output into 4 distinct Modules (Weeks).

3\. Each Module must contain 7 Daily Quests (Total 28 Days).

4\. Each Daily Quest MUST be calculated to take approximately 30 minutes to complete, split into 3 distinct tasks (10 mins each): Vocabulary/Grammar, Reading/Listening, and Practice/Interactive.

5\. Difficulty must scale progressively across the modules.

## **6.2 Complete Go Router Implementation**

o  
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
TaskPlacementTest TaskType \= "placement\_test"  
TaskRoadmapGen    TaskType \= "roadmap\_generation"  
TaskExerciseGen   TaskType \= "exercise\_generation"  
TaskEssayGrading  TaskType \= "essay\_grading"  
)

type ProviderType string

const (  
ProviderGemini   ProviderType \= "gemini"  
ProviderOpenAI   ProviderType \= "openai"  
ProviderDeepSeek ProviderType \= "deepseek"  
)

type LLMProvider interface {  
GenerateContent(ctx context.Context, systemPrompt string, userPrompt string) (string, error)  
}

type Router struct {  
providers  map\[ProviderType\]LLMProvider  
strategies map\[TaskType\]ProviderType  
}

func NewRouter() (\*Router, error) {  
providers := make(map\[ProviderType\]LLMProvider)if geminiKey := os.Getenv("GEMINI\_API\_KEY"); geminiKey \!= "" {

	providers\[ProviderGemini\] \= NewGeminiProvider(geminiKey)

}

if openaiKey := os.Getenv("OPENAI\_API\_KEY"); openaiKey \!= "" {

	providers\[ProviderOpenAI\] \= NewOpenAICompatibleProvider(os.Getenv("OPENAI\_BASE\_URL"), openaiKey, "gpt-4o-mini")

}

if deepseekKey := os.Getenv("DEEPSEEK\_API\_KEY"); deepseekKey \!= "" {

	providers\[ProviderDeepSeek\] \= NewOpenAICompatibleProvider(os.Getenv("DEEPSEEK\_BASE\_URL"), deepseekKey, "deepseek-chat")

}

if len(providers) \== 0 {

	return nil, fmt.Errorf("airouter: no API keys configured")

}

strategies := map\[TaskType\]ProviderType{

	TaskRoadmapGen:    ProviderGemini,

	TaskExerciseGen:   ProviderDeepSeek,

	TaskPlacementTest: ProviderGemini,

	TaskEssayGrading:  ProviderOpenAI,

}

return \&Router{providers: providers, strategies: strategies}, nil

}

func (r \*Router) Route(ctx context.Context, task TaskType, systemPrompt string, userPrompt string) (string, error) {  
providerType, ok := r.strategies\[task\]  
if \!ok {  
providerType \= ProviderGemini  
}provider, exists := r.providers\[providerType\]

if \!exists {

	for altType, altProvider := range r.providers {

		fmt.Printf("airouter: fallback from %s to %s for task %s\\n", providerType, altType, task)

		return altProvider.GenerateContent(ctx, systemPrompt, userPrompt)

	}

	return "", fmt.Errorf("airouter: no provider available for task %s", task)

}

return provider.GenerateContent(ctx, systemPrompt, userPrompt)

}

// Concrete Gemini Provider  
type GeminiProvider struct {  
apiKey     string  
httpClient \*http.Client  
}

func NewGeminiProvider(apiKey string) \*GeminiProvider {  
return \&GeminiProvider{apiKey: apiKey, httpClient: \&http.Client{Timeout: 30 \* time.Second}}  
}

func (g \*GeminiProvider) GenerateContent(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {  
baseUrl := os.Getenv("GEMINI\_BASE\_URL")  
if baseUrl \== "" {  
baseUrl \= "gemini.api.internal"  
}  
url := fmt.Sprintf("%s?key=%s", baseUrl, g.apiKey)reqBody := map\[string\]interface{}{

	"system\_instruction": map\[string\]interface{}{"parts": \[\]map\[string\]string{{"text": systemPrompt}}},

	"contents": \[\]map\[string\]interface{}{

		{"role": "user", "parts": \[\]map\[string\]string{{"text": userPrompt}}},

	},

	"generationConfig": map\[string\]interface{}{

		"response\_mime\_type": "application/json",

		"temperature":        0.2,

	},

}

jsonBytes, err := json.Marshal(reqBody)

if err \!= nil {

	return "", fmt.Errorf("gemini marshal error: %w", err)

}

req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))

if err \!= nil {

	return "", fmt.Errorf("gemini build error: %w", err)

}

req.Header.Set("Content-Type", "application/json")

resp, err := g.httpClient.Do(req)

if err \!= nil {

	return "", fmt.Errorf("gemini http error: %w", err)

}

defer resp.Body.Close()

body, \_ := io.ReadAll(resp.Body)

if resp.StatusCode \!= http.StatusOK {

	return "", fmt.Errorf("gemini api error (%d): %s", resp.StatusCode, string(body))

}

var parsedResp struct {

	Candidates \[\]struct {

		Content struct {

			Parts \[\]struct {

				Text string \`json:"text"\`

			} \`json:"parts"\`

		} \`json:"content"\`

	} \`json:"candidates"\`

}

if err := json.Unmarshal(body, \&parsedResp); err \!= nil {

	return "", fmt.Errorf("gemini unmarshal error: %w", err)

}

if len(parsedResp.Candidates) \== 0 || len(parsedResp.Candidates\[0\].Content.Parts) \== 0 {

	return "", fmt.Errorf("gemini returned empty response")

}

return parsedResp.Candidates\[0\].Content.Parts\[0\].Text, nil

}

// Concrete OpenAI Compatible Driver  
type OpenAICompatibleProvider struct {  
baseURL    string  
apiKey     string  
model      string  
httpClient \*http.Client  
}

func NewOpenAICompatibleProvider(baseURL, apiKey, model string) \*OpenAICompatibleProvider {  
return \&OpenAICompatibleProvider{  
baseURL:    baseURL,  
apiKey:     apiKey,  
model:      model,  
httpClient: \&http.Client{Timeout: 30 \* time.Second},  
}  
}

func (o \*OpenAICompatibleProvider) GenerateContent(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {  
url := fmt.Sprintf("%s/chat/completions", o.baseURL)reqBody := map\[string\]interface{}{

	"model": o.model,

	"messages": \[\]map\[string\]string{

		{"role": "system", "content": systemPrompt},

		{"role": "user", "content": userPrompt},

	},

	"response\_format": map\[string\]string{"type": "json\_object"},

	"temperature":     0.2,

}

jsonBytes, err := json.Marshal(reqBody)

if err \!= nil {

	return "", fmt.Errorf("openai-compat marshal error: %w", err)

}

req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))

if err \!= nil {

	return "", fmt.Errorf("openai-compat request error: %w", err)

}

req.Header.Set("Content-Type", "application/json")

req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))

resp, err := o.httpClient.Do(req)

if err \!= nil {

	return "", fmt.Errorf("openai-compat http error: %w", err)

}

defer resp.Body.Close()

body, \_ := io.ReadAll(resp.Body)

if resp.StatusCode \!= http.StatusOK {

	return "", fmt.Errorf("openai-compat api error (%d): %s", resp.StatusCode, string(body))

}

var parsedResp struct {

	Choices \[\]struct {

		Message struct {

			Content string \`json:"content"\`

		} \`json:"message"\`

	} \`json:"choices"\`

}

if err := json.Unmarshal(body, \&parsedResp); err \!= nil {

	return "", fmt.Errorf("openai-compat unmarshal error: %w", err)

}

if len(parsedResp.Choices) \== 0 {

	return "", fmt.Errorf("openai-compat returned empty choices")

}

return parsedResp.Choices\[0\].Message.Content, nil

}

**Addendum (2026-09-25, evaluator):** the `http.Client{Timeout: 30 * time.Second}` in the snippets above is superseded. Deadlines are per task and travel in the context (`airouter.TaskTimeout`: 180 s for `roadmap_generation`, 30 s otherwise); the drivers' client has no timeout. `DefaultGeminiModel` is `gemini-3.8-flash` (`gemini-2.5-flash` answers 404 for accounts created after 2026-09). Each provider call retries once on 429/502/503/504 and logs elapsed time, token usage and the upstream error. The onboarding assessment keeps the graded level in `quiz:placement:{user_id}` (§4) so a re-submit after a failed roadmap step is not graded again.

\#\# 7\. Core REST API Endpoint Specifications

\* \*\*POST /api/v1/auth/google\*\*: OAuth code token swap & JWT issuance.

\* \*\*POST /api/v1/onboarding/assessment\*\*: Submits placement quiz answers, grades level, triggers AI roadmap generation.

\* \*\*GET /api/v1/quests/daily\*\*: Fetches current day's 30-minute exercise suite (3x 10-min tasks).

\* \*\*POST /api/v1/quests/progress\*\*: Records completed task minutes, updates Redis counter and daily progress.

\* \*\*GET /api/v1/roadmap\*\*: Fetches the active 28-day roadmap outline (module and day titles, task titles and durations) joined with per-day completion, for the roadmap tree.

\* \*\*GET /api/v1/pet/status\*\*: Retrieves current plant stage, health percentage, and active streak.

\* \*\*POST /api/v1/pet/revive\*\*: Triggers a 15-minute revival challenge when plant health hits 0%.

\* \*\*POST /api/v1/settings/notifications\*\*: Registers VAPID push subscription and updates preferred practice time.

\* \*\*POST /api/v1/integrations/google/sync\*\*: Pushes 30-minute recurring study block to Google Calendar and task list to Google Tasks.

\#\# 8\. Deployment & Execution Checklist (Railway)

1\.  \*\*Database Setup:\*\* Provision PostgreSQL and Redis plugins on Railway. Run DDL migration script.

2\.  \*\*Environment Variables:\*\*

    \*   \`DATABASE\_URL\`: PostgreSQL connection string

    \*   \`REDIS\_URL\`: Redis connection URI

    \*   \`GOOGLE\_CLIENT\_ID\` & \`GOOGLE\_CLIENT\_SECRET\`: Google OAuth credentials

    \*   \`GEMINI\_API\_KEY\`, \`OPENAI\_API\_KEY\`, \`DEEPSEEK\_API\_KEY\`: AI service tokens

    \*   \`VAPID\_PUBLIC\_KEY\` & \`VAPID\_PRIVATE\_KEY\`: Web Push parameters

    \*   \`JWT\_SECRET\`: HS256 session-token secret, at least 32 bytes

    \*   \`ENCRYPTION\_SECRET\_KEY\`: 32-byte hex key for the refresh-token cipher

3\.  \*\*Build & Deploy:\*\*

    \*   Deploy Nuxt 3 PWA with \`@vite-pwa/nuxt\` configured for service worker caching.

    \*   Deploy compiled Go binary as lightweight Docker container on Railway.

\---

\*\*Signature & Approval\*\*

\<span type="placeholder" placeholder-type="person"\>\</span\>

Technical Lead

\<span type="placeholder" placeholder-type="date"\>\</span\>

Approval Date  
