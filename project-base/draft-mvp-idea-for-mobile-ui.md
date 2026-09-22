# Fluency Quest — Product Requirement Document (PRD) & Product Brief

**Product Name:** Fluency Quest (Mobile RPG Language Learning Experience)  
**Version:** 1.0.0 — Production Blueprint  
**Target Platform:** Mobile (iOS / Android), Dark Theme Native Hybrid (PWA / React Native / Flutter)  
**Design System Reference:** Dark Retro-SciFi Cyberpunk / RPG Arcade (`Fluency Quest` — Space Grotesk / Monospace HUD)  
**Document Author:** Stitch AI & Product Design Team  
**Status:** Ready for Engineering & Production Implementation  

---

## 1. Executive Summary & Vision

### 1.1 The Core Problem
Traditional language learning platforms (e.g., Duolingo, Babbel) suffer from severe user drop-off at the intermediate plateau (CEFR B1–B2). Users encounter tedious repetitive drills, disconnected flashcards, and lack the high-stakes situational adrenaline required to achieve spontaneous real-world fluency and speech cadence.

### 1.2 The Solution: Fluency Quest
**Fluency Quest** turns language acquisition into an immersive retro-cyberpunk RPG adventure. Instead of "lessons", users embark on **Runic Expeditions**, forge **Gate Keys** through syntax mastery, confront real-time **Boss Raids** (e.g., *Dragon of Airport Customs*, *Bureaucracy Overlords*) evaluated via AI speech cadence and tone telemetry, and review grammatical mistakes in an interactive **Grimoire Lab**.

---

## 2. Target Audience & Personas

- **Primary Persona: "The Ambitious Professional / Polyglot Gamer" (Age 18–35)**
  - Motivated by gaming progression mechanics (XP, Loot, Trophies, Leaderboards, S-Rank completions).
  - Needs high-stakes speaking practice for job interviews, immigration/visa checkpoints, travel, and international negotiations.
  - Wants tangible CEFR level milestones rather than arbitrary streak counters.
- **Secondary Persona: "The Reluctant Intermediate" (CEFR B1)**
  - Has textbook knowledge of grammar but freezes when speaking with native speakers.
  - Benefits from AI-judged spoken cadence, tone accuracy, and rapid rebuttal drills.

---

## 3. Product Architecture & User Journey Map

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Onboarding & Placement                                  │
│    • Welcome & Google Auth Screen                           │
│    • Target Realm & Destiny Survey                          │
│    • Gatekeeper Placement Trial (Diagnostic)                │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. World Synthesis & Navigation                             │
│    • Procedural AI Cartographer Roadmap Synthesis           │
│    • Dynamic Adventure World Map (Stage Nodes & Portals)    │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. Core Gameplay Loop                                       │
│    • The Runic Gate & Key Trial (Chamber Lessons)           │
│    • Boss Raid Battle (AI Voice Counter-Arguments)          │
│    • Victory & Loot Spoils (XP, Relics, Title Drops)        │
│    • Grimoire of Errant Runes (Mistake Sanctuary Review)    │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. Meta-Progression & Social                                │
│    • Hunter Profile & Gear Vault (Relics, CEFR Radar)       │
│    • Hall of Runic Feats (Achievements Codex & Showcase)    │
│    • Colosseum of Orators (1v1 Arena - Upcoming Teaser)     │
└─────────────────────────────────────────────────────────────┘
```

---

## 4. Key Functional Modules & Screen Breakdown

### Module A: Authentication & Diagnostic Onboarding
1. **Welcome & Google Auth Screen** (`SCREEN_10`)
   - High-impact animated title with neon runic accents.
   - One-tap frictionless **Google Authentication** (Guest mode explicitly excluded for persistent cloud progression).
   - Terms of Service & Privacy Protocol badges.
2. **Target Realm & Destiny Survey** (`SCREEN_13`)
   - Branching goal selection: *Career & Global Negotiation*, *Immigration & Citizenship*, *Academic Mastery*, *Spontaneous Travel*.
   - Daily commitment selection (Casual: 10m, Dedicated: 20m, Mythic: 45m).
3. **Gatekeeper's Placement Trial** (`SCREEN_14`)
   - Interactive 3-minute diagnostic dialogue to estimate initial CEFR competency (A1 to C1).
   - Real-time cadence waveform and confidence scoring.
4. **AI Cartographer Synthesizing Roadmap** (`SCREEN_12`)
   - Procedural generation animation calculating initial skill matrix, generating custom nodes and boss checkpoints.

### Module B: World Map & Adventure Exploration
5. **Adventure Roadmap & Boss Stages** (`SCREEN_15`)
   - Interactive node-based map with interconnected stages, locked realms, and dungeon gates.
   - Top HUD telemetry: Level status, Hearts (HP), Mana XP, Active Streaks.
   - Boss Raid Portals requiring forged Gate Keys to enter.

### Module C: Learning Chambers & AI Boss Raids
6. **The Runic Gate & Key Trial** (`SCREEN_16`)
   - Core gamified lesson mechanics: solving situational grammar, idiom choices, and vocabulary puzzles.
   - Answering correctly forges **Key Shards (e.g. 2/3 Keys)** needed to unlock chamber doors.
   - Heart penalty system for erroneous syntax.
7. **Boss Raid: Dragon of Airport Customs** (`SCREEN_11`)
   - Full simulated spoken dialogue encounter with animated boss health bar and dialogue log.
   - Real-time AI voice evaluation analyzing:
     - **Cadence Score** (Fluid rhythm vs. hesitation latency).
     - **Tone & Pitch Accuracy**.
     - **Rebuttal Parry** (Deflecting aggressive immigration/customs questions).
8. **Victory & Loot Rewards** (`SCREEN_8`)
   - Post-raid victory celebration banner with rank classification (S-Tier, A-Tier).
   - Milestone CEFR progression bar (`B1 → B2 Cleared`).
   - Mythic Relic spoils (e.g. *Diplomat's Golden Seal*, *Key Shard of Schengen*).
   - Claimable Hunter titles (e.g. *"Customs Negotiator"*).

### Module D: Retention, Mistake Purification & Codex
9. **Grimoire of Errant Runes (Mistakes Vault)** (`SCREEN_9`)
   - Dedicated mistake sanctuary where every spoken or syntax blunder is archived.
   - "Rune Recasting" mechanic: Re-answering mistakes under timed pressure purifies them, restores spent Hearts, and awards Mana XP.
10. **Hunter Profile & Gear Vault** (`SCREEN_7`)
    - Visual **CEFR Fluency Skill Radar**: Spoken Cadence (94/100), Real-world Improv (91/100), Vocabulary Mastery (88/100), Listening Reflex (86/100), Grammar Armor (82/100).
    - 4-Slot Relic Arsenal: Equipping passive buffs (e.g., +15s speech time buffer, +10% pitch tolerance).
    - Active Guild Hunt Bounties.
11. **Hall of Runic Feats (Trophy Codex)** (`SCREEN_3` / `SCREEN_4`)
    - Overall completion metric (e.g., 24/60 Feats Unlocked, 40%).
    - Showcased Pinned Badges (Modular deck for displaying pride badges).
    - Interactive "CLAIM" mechanics awarding MP and exclusive hunter titles.
    - Encrypted / Classified hidden achievements teasing late-game milestones.
12. **Colosseum of Orators (Arena Teaser)** (`SCREEN_6`)
    - Real-time synchronous 1v1 PvP verbal combat teaser.
    - Pre-registration callout with exclusive *"Gladiator Vanguard"* rewards.

---

## 5. Visual Identity & Design System Specifications

| Token Dimension | Design System 1 (`Fluency Quest`) Token Standard |
| :--- | :--- |
| **Theme / Mode** | Dark High-Contrast Arcade / Cyber-SciFi |
| **Typography** | Primary: `Space Grotesk` (Headings, Buttons, Badges) / Secondary: `JetBrains Mono` / Tabular Monospace |
| **Background Surfaces** | `#0c0e17` (Lowest container), `#11131c` (Primary Surface), `#191b25` (Container Low), `#373943` (Borders/Dividers) |
| **Primary Accent** | `#6366f1` / Indigo-Violet (High-energy mana & primary CTAs) |
| **Secondary Accent** | `#06b6d4` / Cyan / Electric Neon (Telemetry, cadence waves, tech badges) |
| **Warning / Streak** | `#f59e0b` / Gold / Amber (Streak flame, S-Tier ranks, Mythic loot) |
| **Destructive / Boss** | `#ef4444` / Crimson (Enemy damage, depleted HP, locked sectors) |
| **Elevation & Border** | Hard 2px tactical borders (`border-outline-variant`), tactile offset drop-shadows (`shadow-[0_3px_0_0_#0c0e17]`) |
| **Navigation Shell** | Docked 5-tab bottom bar: `[Map, Battle, Lab, Arena, Hunter]` with tactile spring micro-interactions |

---

## 6. Technical Stack & Architecture Recommendations

- **Frontend Client**: React Native / Flutter or Next.js PWA wrapped via Capacitor.
- **Styling**: Tailwind CSS + CSS Custom Properties tied to design tokens.
- **Audio / Speech AI Engine**:
  - Web Audio API / Native Audio Toolkit for real-time waveform visualization.
  - Speech-to-Text & Pronunciation Assessment API (e.g., Azure Speech SDK or Whisper fine-tuned for phoneme & cadence scoring).
  - LLM Dialogue Arbiter for Boss responses and dynamic counter-arguments.
- **Backend / Real-time Engine**:
  - Node.js / Go microservices with WebSocket endpoints for real-time raid telemetry and upcoming 1v1 PvP Colosseum duels.
  - PostgreSQL for hunter records, inventory relics, and achievements.
  - Redis for live streak tracking, leaderboards, and rate-limiting.

---

## 7. Success Metrics & KPIs

1. **D7 & D30 Retention**: Target >45% D7 retention and >28% D30 retention via streak protection and Trophy Codex milestones.
2. **Speaking Completion Rate**: >75% of users completing spoken Boss Raid encounters without abandoning speech exercises.
3. **Mistake Resolution Index (MRI)**: >60% of mistakes in the Grimoire of Errant Runes successfully purified within 48 hours.
4. **Organic Social Sharing**: Trophy showcase badges and Boss Victory certificates shared to social channels.

---

## 8. Development Roadmap & Milestones

- **Phase 1 (MVP Launch - Weeks 1–8)**:
  - Auth, Placement Test, Dynamic Map, Core Gate & Key Lessons (Worlds 1–2).
  - Boss Raid 1 (*Dragon of Airport Customs*) with automated cadence scoring.
  - Hunter Profile & Grimoire of Errant Runes.
- **Phase 2 (Retention & Depth - Weeks 9–14)**:
  - Trophy Codex & Achievement Claim engine.
  - Equipable Relics with passive combat bonuses in the Gear Vault.
  - Worlds 3 & 4 (Business & Job Interview Raids).
- **Phase 3 (Multiplayer & Live Ops - Weeks 15–20)**:
  - Colosseum of Orators (1v1 live verbal clashes).
  - Weekly Guild Raids and season ladder tournaments.
