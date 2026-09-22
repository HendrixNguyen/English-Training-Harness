# **Adaptive English Learning Platform — Frontend Technical Specification**

# **1\. Executive Summary & Core Objectives**

This document serves as the primary technical specification for the Nuxt 3 Progressive Web Application (PWA) Frontend. It details the frontend architecture, Pinia state management, offline Service Worker caching strategies, UI data mapping, UI/UX design system, and full prototype wireframes for the Adaptive English Learning Platform.

---

# **2\. Frontend Technology Stack**

* Framework: Nuxt 3 (Vue 3, Composition API, TypeScript).  
* State Management: Pinia stores (useAuthStore, useQuestStore, usePetStore).  
* Styling & UI Kit: Tailwind CSS with custom Gamified Minimalist theme design tokens.  
* PWA Engine: @vite-pwa/nuxt module with custom Service Workers.  
* Offline Storage: IndexedDB / LocalStorage for offline queueing via StaleWhileRevalidate.  
* 

```
                     +---------------------------------------+
                     |         Nuxt 3 PWA Client             |
                     +-------------------+-------------------+
                                         |
               +-------------------------+-------------------------+
               |                         |                         |
               v                         v                         v
       +---------------+         +---------------+         +---------------+
       | useAuthStore  |         | useQuestStore |         |  usePetStore  |
       +---------------+         +---------------+         +---------------+
       | JWT Token     |         | Daily Quests  |         | Plant Health  |
       | User Profile  |         | Active Timer  |         | Stage SVG     |
       | Local Config  |         | Task Progress |         | Streak Count  |
       +---------------+         +---------------+         +---------------+
```

---

# **3\. Service Worker Caching Strategies**

* Static Assets & Exercises: StaleWhileRevalidate (Serves cached exercise cards immediately for zero-latency offline study).  
* User Progress & Pet Status: NetworkFirst (Ensures synchronized state across devices when connected to the internet).

---

# **4\. Pinia State Management Architecture**

* useAuthStore: Manages JWT tokens, user profile metadata (full\_name, cefr\_current), Google OAuth authentication state, and notification settings.  
* useQuestStore: Manages the active 10-minute/30-minute learning timers, active quest payload, task completions, and offline progress queueing.  
* usePetStore: Controls Virtual Plant SVG rendering, stage transitions (Seed \-\> Sprout \-\> Sapling \-\> Flowering \-\> Fruitful / Wilted), health bar animations, and streak counters.

---

# **5\. API Data to UI Mapping Matrix**

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

| UI View / Component | API Endpoint | Data Rendered |
| :---- | :---- | :---- |
| **Onboarding / Assessment** | `POST /api/v1/onboarding/assessment` | Adaptive placement test cards, goal selection (IELTS / Fluency), notification time picker. |
| **Virtual Plant Widget** | `GET /api/v1/pet/status` | Dynamic SVG plant stage (Seed \-\> Sprout \-\> Sapling \-\> Flowering \-\> Fruitful / Wilted), Health Bar %, Streak Count. |
| **Daily Progress Tracker** | `GET /api/v1/quests/daily` | Real-time minute accumulator (*20 / 30 mins completed*), daily goal completion progress bar. |
| **Today's Quest List** | `GET /api/v1/quests/daily` | 3 Exercise cards (10m Vocabulary, 10m Reading, 10m Practice) with completed/pending indicators. |
| **Curriculum Roadmap Tree** | `GET /api/v1/quests/daily` | 4-Week (28-Day) interactive timeline map showing completed, active, and locked milestones. |
| **Settings & Integration Modal** | `POST /api/v1/settings/notificationsPOST /api/v1/integrations/google/sync` | Google Calendar & Tasks sync status toggle, Web Push notification permissions, daily reminder time selector. |

---

# **6\. UI/UX Philosophy & Design System**

## **6.1 Core Visual Identity**

* Design Style: Modern Gamified Minimalist (Clean, friendly, micro-animated).  
* Format: Mobile-First Responsive PWA (Optimized for touch/swipe gestures on mobile and auto-scaled for desktop viewports).  
* Color Palette:  
  * Emerald Green (\#10B981): Represents plant growth, success states, and completed lessons.  
  * Amber Gold (\#F59E0B): Represents streak counts, rewards, and milestone badges.  
  * Coral Red (\#EF4444): Used for health depletion warnings and wilted state alerts.  
  * Slate Gray (\#1E293B): Dark/Light mode high-contrast text and clean background containers.

---

# **7\. Prototype Wireframes & User Flow Layouts**

## **7.1 Wireframe Prototype 1: Onboarding & Goal Selection**

```
+---------------------------------------------------+
|               Chào mừng bạn! 🌱                   |
|                                                   |
|  [ Nút Đăng nhập nhanh bằng Google OAuth ]       |
|                                                   |
|  Mục tiêu học của bạn là gì?                     |
|  +-------------------+  +----------------------+  |
|  | 🎓 IELTS 7.0      |  | 💼 Business English  |  |
|  +-------------------+  +----------------------+  |
|                                                   |
|  Chọn giờ nhắc học hằng ngày: [ 20:00  ▼ ]        |
+---------------------------------------------------+
```

## **7.2 Wireframe Prototype 2: Main Dashboard & Virtual Plant Hub**

```
+---------------------------------------------------+
|  [Avatar] Nguyen Hendrix     🔥 Streak: 5 Ngày    |
+---------------------------------------------------+
|                                                   |
|                +-----------------+                |
|                |   (Cây ảo SVG)  |                |
|                |      🌱         |                |
|                +-----------------+                |
|            Máu Cây: [████████░░] 80%              |
|         💬 "Tưới cho tớ 10 phút học đi!"          |
|                                                   |
+---------------------------------------------------+
|  TIẾN ĐỘ HÔM NAY: 20 / 30 PHÚT                    |
|  [=========================>......] 66%            |
+---------------------------------------------------+
|  Nhiệm vụ hôm nay (Quests):                       |
|  [✓] 1. Từ vựng Email Công việc    (10m)  [Xong]  |
|  [▶] 2. Đọc hiểu Mẫu Thư Thương mại(10m)  [Học]   |
|  [ ] 3. Viết Phản hồi Khách hàng   (10m)  [Khóa]  |
+---------------------------------------------------+
```

## **7.3 Wireframe Prototype 3: Distraction-Free Learning Room**

```
+---------------------------------------------------+
|  < Quay lại                 ⏱️ Thời gian: 09:42   |
+---------------------------------------------------+
|  Câu 2 / 10                                       |
|                                                   |
|  "Choose the correct formal phrasing for requesting|
|   a price quotation:"                             |
|                                                   |
|  (A) Give me the cost details right now.          |
|  (B) Could you please provide a price quotation?  |
|  (C) Send me how much this thing costs.           |
|                                                   |
|  +---------------------------------------------+  |
|  |           [ Gửi đáp án / Tiếp tục ]         |  |
|  +---------------------------------------------+  |
+---------------------------------------------------+
```

## **7.4 Wireframe Prototype 4: Visual Curriculum Roadmap Tree (28 Days)**

```
+---------------------------------------------------+
|  Lộ trình học 28 ngày                             |
+---------------------------------------------------+
|                                                   |
|        [⭐ Ngày 1: Đã hoàn thành]                 |
|                   \                               |
|                    \                              |
|               [⭐ Ngày 2: Đã hoàn thành]          |
|                    /                              |
|                   /                               |
|        [🌱 Ngày 3: HÔM NAY - Đang học]            |
|                   \                               |
|                    \                              |
|               [🔒 Ngày 4: Chưa mở khóa]           |
|                                                   |
+---------------------------------------------------+
```

## **7.5 Wireframe Prototype 5: Plant Revival Mode (Health \= 0%)**

```
+---------------------------------------------------+
|  ⚠️ CÂY XANH ĐANG BỊ HÉO RŨ!                     |
+---------------------------------------------------+
|                                                   |
|                +-----------------+                |
|                | (Cây héo - 0%)  |                |
|                |      🥀         |                |
|                +-----------------+                |
|                                                   |
|  "Bạn đã bỏ học 2 ngày liên tiếp. Hãy hoàn thành |
|   Bài kiểm tra Cứu Cây 15 phút để hồi sinh!"      |
|                                                   |
|  +---------------------------------------------+  |
|  |     [ 🚨 Cứu Cây Ngay (Quiz 15 Phút) ]      |  |
|  +---------------------------------------------+  |
+---------------------------------------------------+
```

## 

## 

## 

## 

## 

* 