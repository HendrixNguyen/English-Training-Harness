package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/onboarding"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// integrationContentProvider drives the real router without external AI.
type integrationContentProvider struct{ calls int }

func (p *integrationContentProvider) GenerateContent(_ context.Context, _, _ string) (string, error) {
	p.calls++
	return string(airouter.SampleContent("reading")), nil
}

// jsonEqual compares two JSON documents structurally: Postgres's jsonb round
// trip (a space after a colon, key order) is not the same bytes back, even
// though it is the same JSON.
func jsonEqual(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		t.Fatalf("unmarshal %s: %v", a, err)
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}
	return reflect.DeepEqual(va, vb)
}

// contentTargetRoadmap gives every module a distinct focus and every day a
// distinct title (integrationRoadmap in integration_test.go reuses one focus
// for all four modules, which cannot prove ContentTarget reads the *right*
// module/day).
func contentTargetRoadmap(cefrLevel string) airouter.Roadmap {
	r := airouter.Roadmap{Title: "Content target fixture", CEFRLevel: cefrLevel}
	day := 0
	for m := 1; m <= airouter.Modules; m++ {
		mod := airouter.Module{Week: m, Title: fmt.Sprintf("Week %d", m), Focus: fmt.Sprintf("Module %d focus", m)}
		for d := 1; d <= airouter.DaysPerModule; d++ {
			day++
			dayFixture := airouter.Day{Title: fmt.Sprintf("Day %d title", day)}
			for _, tt := range airouter.TaskTypes {
				dayFixture.Tasks = append(dayFixture.Tasks, airouter.Task{Type: tt, Title: "Day task: " + tt, DurationMinutes: 10, Content: json.RawMessage(`{}`)})
			}
			mod.Days = append(mod.Days, dayFixture)
		}
		r.Modules = append(r.Modules, mod)
	}
	return r
}

// TestIntegrationContentTargetAndSaveContent is counted by CI's
// backend-integration job (see integration_test.go's own comment on the
// convention) and skips locally without TEST_DATABASE_URL.
func TestIntegrationContentTargetAndSaveContent(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL unset; run `make up` and export it to run integration tests")
	}
	ctx := context.Background()

	pg, err := store.NewPostgres(ctx, dbURL)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	t.Cleanup(pg.Close)
	if _, err := store.Migrate(ctx, pg.Migrator(), store.MigrationsFS); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	obRepo := onboarding.NewPgRepo(pg.Pool)
	repo := NewPgRepo(pg.Pool)

	seedUser := func(t *testing.T, gid, email string) string {
		t.Helper()
		_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid)
		var userID string
		if err := pg.Pool.QueryRow(ctx,
			`INSERT INTO users (email, google_id, target_goal, timezone) VALUES ($1,$2,$3,$4) RETURNING id`,
			email, gid, "pass an interview", "UTC").Scan(&userID); err != nil {
			t.Fatalf("inserting user %s: %v", gid, err)
		}
		t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id = $1`, gid) })
		return userID
	}

	exerciseID := func(t *testing.T, roadmapID string, day int, taskType string) string {
		t.Helper()
		var id string
		if err := pg.Pool.QueryRow(ctx,
			`SELECT id FROM exercises WHERE roadmap_id = $1 AND day_number = $2 AND task_type = $3::task_category`,
			roadmapID, day, taskType).Scan(&id); err != nil {
			t.Fatalf("finding day %d %s exercise: %v", day, taskType, err)
		}
		return id
	}

	const mainGid = "google-content-target-main"
	userID := seedUser(t, mainGid, "content-target@example.com")
	roadmapID, err := obRepo.SaveAssessment(ctx, userID, onboarding.Assessment{
		CEFRLevel: "B1", TargetGoal: "pass an interview", Timezone: "UTC", NotificationTime: "20:00:00",
		Roadmap: contentTargetRoadmap("B1"),
	})
	if err != nil {
		t.Fatalf("SaveAssessment: %v", err)
	}

	day9ID := exerciseID(t, roadmapID, 9, "vocabulary")

	t.Run("first GET generates one reading task and reopening reads persisted bytes", func(t *testing.T) {
		id := exerciseID(t, roadmapID, 1, "reading")
		target, err := repo.ContentTarget(ctx, userID, id)
		if err != nil {
			t.Fatal(err)
		}
		provider := &integrationContentProvider{}
		router := airouter.NewRouterWithProviders(map[airouter.ProviderType]airouter.LLMProvider{airouter.ProviderDeepSeek: provider})
		limiter := &fakeContentLimiter{}
		svc := NewContentService(repo, router, limiter, func() time.Time { return target.RoadmapCreatedAt.Add(time.Minute) })
		handler := newContentRouter(svc, userID)
		var firstBody string
		for i := 0; i < 2; i++ {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/quests/exercises/"+id, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("GET %d: %d %s", i, response.Code, response.Body)
			}
			var task Task
			if err := json.Unmarshal(response.Body.Bytes(), &task); err != nil {
				t.Fatal(err)
			}
			var meta exerciseMeta
			if err := json.Unmarshal(task.ContentJSON, &meta); err != nil {
				t.Fatal(err)
			}
			if meta.ContentSchema != 1 {
				t.Fatalf("content_schema = %d, want 1", meta.ContentSchema)
			}
			if err := airouter.ValidateContent("reading", "B1", meta.Content); err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				firstBody = response.Body.String()
			} else if response.Body.String() != firstBody {
				t.Error("reopening did not return the persisted task unchanged")
			}
		}
		if provider.calls != 1 || limiter.calls != 1 {
			t.Errorf("provider calls=%d limiter calls=%d, want 1 each", provider.calls, limiter.calls)
		}
		var typed int
		if err := pg.Pool.QueryRow(ctx, `SELECT count(*) FROM exercises WHERE roadmap_id = $1 AND content_json->>'content_schema' = '1'`, roadmapID).Scan(&typed); err != nil {
			t.Fatal(err)
		}
		if typed != 1 {
			t.Errorf("typed exercises = %d, want only the opened task", typed)
		}
	})

	t.Run("ContentTarget reads module 2's focus and day 9's title", func(t *testing.T) {
		target, err := repo.ContentTarget(ctx, userID, day9ID)
		if err != nil {
			t.Fatalf("ContentTarget: %v", err)
		}
		if target.ModuleFocus != "Module 2 focus" {
			t.Errorf("ModuleFocus = %q, want %q", target.ModuleFocus, "Module 2 focus")
		}
		if target.DayTitle != "Day 9 title" {
			t.Errorf("DayTitle = %q, want %q", target.DayTitle, "Day 9 title")
		}
		if target.Level != "B1" {
			t.Errorf("Level = %q, want B1", target.Level)
		}
		if target.Goal != "pass an interview" {
			t.Errorf("Goal = %q, want %q", target.Goal, "pass an interview")
		}
		if target.Timezone != "UTC" {
			t.Errorf("Timezone = %q, want UTC", target.Timezone)
		}
		if target.Exercise.DayNumber != 9 || target.Exercise.TaskType != "vocabulary" {
			t.Errorf("Exercise = %+v, want day 9 vocabulary", target.Exercise)
		}
	})

	t.Run("another user's exercise id is not found", func(t *testing.T) {
		otherID := seedUser(t, "google-content-target-other", "content-target-other@example.com")
		if _, err := repo.ContentTarget(ctx, otherID, day9ID); !errors.Is(err, ErrExerciseNotFound) {
			t.Errorf("err = %v, want ErrExerciseNotFound", err)
		}
	})

	t.Run("a non-UUID id is not found, not a Postgres error", func(t *testing.T) {
		if _, err := repo.ContentTarget(ctx, userID, "not-a-uuid"); !errors.Is(err, ErrExerciseNotFound) {
			t.Errorf("err = %v, want ErrExerciseNotFound", err)
		}
	})

	t.Run("an exercise on a deactivated roadmap is not found", func(t *testing.T) {
		deactivatedGid := "google-content-target-deactivated"
		deactivatedUserID := seedUser(t, deactivatedGid, "content-target-deactivated@example.com")
		firstRoadmapID, err := obRepo.SaveAssessment(ctx, deactivatedUserID, onboarding.Assessment{
			CEFRLevel: "B1", TargetGoal: "integration", Timezone: "UTC", NotificationTime: "20:00:00",
			Roadmap: contentTargetRoadmap("B1"),
		})
		if err != nil {
			t.Fatalf("SaveAssessment (first): %v", err)
		}
		firstExerciseID := exerciseID(t, firstRoadmapID, 1, "practice")

		// A second SaveAssessment deactivates the first roadmap (insertActiveRoadmap
		// deactivates any active roadmap for the user before inserting the new one).
		if _, err := obRepo.SaveAssessment(ctx, deactivatedUserID, onboarding.Assessment{
			CEFRLevel: "B1", TargetGoal: "integration", Timezone: "UTC", NotificationTime: "20:00:00",
			Roadmap: contentTargetRoadmap("B1"),
		}); err != nil {
			t.Fatalf("SaveAssessment (second): %v", err)
		}

		if _, err := repo.ContentTarget(ctx, deactivatedUserID, firstExerciseID); !errors.Is(err, ErrExerciseNotFound) {
			t.Errorf("err = %v, want ErrExerciseNotFound for an exercise on a deactivated roadmap", err)
		}
	})

	t.Run("SaveContent snapshot CAS writes once and leaves roadmap unchanged", func(t *testing.T) {
		practiceID := exerciseID(t, roadmapID, 10, "practice")

		var roadmapJSONBefore []byte
		if err := pg.Pool.QueryRow(ctx, `SELECT roadmap_json FROM roadmaps WHERE id = $1`, roadmapID).Scan(&roadmapJSONBefore); err != nil {
			t.Fatalf("reading roadmap_json before: %v", err)
		}

		contentA := json.RawMessage(`{"marker":"A"}`)
		before, err := repo.ContentTarget(ctx, userID, practiceID)
		if err != nil {
			t.Fatal(err)
		}
		_, won, err := repo.SaveContent(ctx, userID, practiceID, before.Exercise.ContentJSON, contentA, "B1")
		if !won {
			t.Fatal("first CAS lost")
		}
		saved, readErr := repo.ContentTarget(ctx, userID, practiceID)
		if readErr != nil {
			t.Fatal(readErr)
		}
		out := saved.Exercise.ContentJSON
		if err != nil {
			t.Fatalf("SaveContent(A): %v", err)
		}
		var rowA struct {
			Title           string          `json:"title"`
			DurationMinutes int             `json:"duration_minutes"`
			Content         json.RawMessage `json:"content"`
			ContentSchema   int             `json:"content_schema"`
		}
		if err := json.Unmarshal(out, &rowA); err != nil {
			t.Fatalf("unmarshal content_json after SaveContent(A): %v", err)
		}
		if rowA.ContentSchema != 1 {
			t.Errorf("content_schema = %d, want 1", rowA.ContentSchema)
		}
		if !jsonEqual(t, rowA.Content, contentA) {
			t.Errorf("content = %s, want %s", rowA.Content, contentA)
		}
		if rowA.Title != "Day task: practice" || rowA.DurationMinutes != 10 {
			t.Errorf("title/duration_minutes = %q/%d, want the seed's own", rowA.Title, rowA.DurationMinutes)
		}

		contentB := json.RawMessage(`{"marker":"B"}`)
		_, won, err = repo.SaveContent(ctx, userID, practiceID, before.Exercise.ContentJSON, contentB, "B1")
		if won {
			t.Fatal("stale second CAS won")
		}
		saved, readErr = repo.ContentTarget(ctx, userID, practiceID)
		if readErr != nil {
			t.Fatal(readErr)
		}
		out2 := saved.Exercise.ContentJSON
		if err != nil {
			t.Fatalf("SaveContent(B): %v", err)
		}
		var rowB struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(out2, &rowB); err != nil {
			t.Fatalf("unmarshal content_json after SaveContent(B): %v", err)
		}
		if !jsonEqual(t, rowB.Content, contentA) {
			t.Errorf("second SaveContent returned %s, want the first writer's %s", rowB.Content, contentA)
		}

		var roadmapJSONAfter []byte
		if err := pg.Pool.QueryRow(ctx, `SELECT roadmap_json FROM roadmaps WHERE id = $1`, roadmapID).Scan(&roadmapJSONAfter); err != nil {
			t.Fatalf("reading roadmap_json after: %v", err)
		}
		if string(roadmapJSONBefore) != string(roadmapJSONAfter) {
			t.Error("roadmaps.roadmap_json changed after SaveContent")
		}
	})

	t.Run("concurrent initial saves elect exactly one winner", func(t *testing.T) {
		id := exerciseID(t, roadmapID, 11, "practice")
		const writers = 8
		start := make(chan struct{})
		outputs := make([]bool, writers)
		before, err := repo.ContentTarget(ctx, userID, id)
		if err != nil {
			t.Fatal(err)
		}
		errs := make([]error, writers)
		var wg sync.WaitGroup
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				_, outputs[i], errs[i] = repo.SaveContent(ctx, userID, id, before.Exercise.ContentJSON, json.RawMessage(fmt.Sprintf(`{"writer":%d}`, i)), "B1")
			}(i)
		}
		close(start)
		wg.Wait()
		var stored json.RawMessage
		if err := pg.Pool.QueryRow(ctx, `SELECT content_json FROM exercises WHERE id = $1`, id).Scan(&stored); err != nil {
			t.Fatal(err)
		}
		winners := 0
		for i := 0; i < writers; i++ {
			if errs[i] != nil {
				t.Fatalf("writer %d: %v", i, errs[i])
			}
			if outputs[i] {
				winners++
				if !strings.Contains(string(stored), fmt.Sprintf(`"writer": %d`, i)) {
					t.Errorf("winner %d not stored: %s", i, stored)
				}
			}
		}
		if winners != 1 {
			t.Fatalf("winners=%d want1", winners)
		}
	})

	t.Run("old missing focus falls back to module title", func(t *testing.T) {
		if _, err := pg.Pool.Exec(ctx, `UPDATE roadmaps SET roadmap_json=jsonb_set(roadmap_json,'{modules,1,focus}','"   "') WHERE id=$1`, roadmapID); err != nil {
			t.Fatal(err)
		}
		target, err := repo.ContentTarget(ctx, userID, day9ID)
		if err != nil {
			t.Fatal(err)
		}
		if target.ModuleFocus != "Week 2" || target.DayTitle != "Day 9 title" {
			t.Fatalf("focus=%q day=%q", target.ModuleFocus, target.DayTitle)
		}
	})
	t.Run("CAS refuses foreign completed inactive and changed rows", func(t *testing.T) {
		id := exerciseID(t, roadmapID, 1, "practice")
		before, err := repo.ContentTarget(ctx, userID, id)
		if err != nil {
			t.Fatal(err)
		}
		other := seedUser(t, "google-content-cas-other", "cas-other@example.com")
		testNoWrite := func(owner string, snapshot json.RawMessage) {
			t.Helper()
			var beforeBytes, afterBytes json.RawMessage
			if err := pg.Pool.QueryRow(ctx, `SELECT content_json FROM exercises WHERE id=$1`, id).Scan(&beforeBytes); err != nil {
				t.Fatal(err)
			}
			_, won, err := repo.SaveContent(ctx, owner, id, snapshot, airouter.SampleContent("practice"), "B1")
			if err != nil || won {
				t.Fatalf("CAS won%t err%v", won, err)
			}
			if err := pg.Pool.QueryRow(ctx, `SELECT content_json FROM exercises WHERE id=$1`, id).Scan(&afterBytes); err != nil {
				t.Fatal(err)
			}
			if string(beforeBytes) != string(afterBytes) {
				t.Fatal("refused CAS mutated stored bytes")
			}
		}
		testNoWrite(other, before.Exercise.ContentJSON)
		if _, err := pg.Pool.Exec(ctx, `UPDATE exercises SET is_completed=true WHERE id=$1`, id); err != nil {
			t.Fatal(err)
		}
		testNoWrite(userID, before.Exercise.ContentJSON)
		if _, err := pg.Pool.Exec(ctx, `UPDATE exercises SET is_completed=false WHERE id=$1`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := pg.Pool.Exec(ctx, `UPDATE roadmaps SET is_active=false WHERE id=$1`, roadmapID); err != nil {
			t.Fatal(err)
		}
		testNoWrite(userID, before.Exercise.ContentJSON)
		if _, err := pg.Pool.Exec(ctx, `UPDATE roadmaps SET is_active=true WHERE id=$1`, roadmapID); err != nil {
			t.Fatal(err)
		}
		testNoWrite(userID, json.RawMessage(`{}`))
	})
	t.Run("invalid cache and failed repair preserve bytes and scope", func(t *testing.T) {
		other := seedUser(t, "google-content-repair-foreign", "repair-foreign@example.com")
		for _, mode := range []string{"ordinary", "malformed repair", "completed", "foreign", "inactive", "future"} {
			t.Run(mode, func(t *testing.T) {
				day := 1
				if mode == "future" {
					day = 2
				}
				id := exerciseID(t, roadmapID, day, "vocabulary")
				invalid := contentJSON("Invalid vocabulary", `{"words":[]}`, true)
				if _, err := pg.Pool.Exec(ctx, `UPDATE exercises SET content_json=$2::jsonb,is_completed=false WHERE id=$1`, id, invalid); err != nil {
					t.Fatal(err)
				}
				target, err := repo.ContentTarget(ctx, userID, id)
				if err != nil {
					t.Fatal(err)
				}
				var before json.RawMessage
				if err := pg.Pool.QueryRow(ctx, `SELECT content_json FROM exercises WHERE id=$1`, id).Scan(&before); err != nil {
					t.Fatal(err)
				}
				owner := userID
				path := "/api/v1/quests/exercises/" + id + "?repair=true"
				wantStatus := 409
				wantError := "exercise_content_invalid"
				wantAI := 0
				switch mode {
				case "ordinary":
					path = "/api/v1/quests/exercises/" + id
				case "malformed repair":
					wantStatus = 502
					wantError = "ai_bad_output"
					wantAI = 2
				case "completed":
					if _, err := pg.Pool.Exec(ctx, `UPDATE exercises SET is_completed=true WHERE id=$1`, id); err != nil {
						t.Fatal(err)
					}
				case "foreign":
					owner = other
					wantStatus = 404
					wantError = "exercise_not_found"
				case "inactive":
					wantStatus = 404
					wantError = "exercise_not_found"
					if _, err := pg.Pool.Exec(ctx, `UPDATE roadmaps SET is_active=false WHERE id=$1`, roadmapID); err != nil {
						t.Fatal(err)
					}
					defer func() {
						if _, err := pg.Pool.Exec(ctx, `UPDATE roadmaps SET is_active=true WHERE id=$1`, roadmapID); err != nil {
							t.Error(err)
						}
					}()
				case "future":
					wantStatus = 404
					wantError = "exercise_not_found"
				}
				ai := &fakeContentAI{answers: []string{"bad", "still bad", string(airouter.SampleContent("vocabulary"))}}
				lim := &fakeContentLimiter{}
				svc := NewContentService(repo, ai, lim, func() time.Time { return target.RoadmapCreatedAt.Add(time.Minute) })
				response := httptest.NewRecorder()
				newContentRouter(svc, owner).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
				if response.Code != wantStatus || !strings.Contains(response.Body.String(), `"error":"`+wantError+`"`) || ai.calls != wantAI {
					t.Fatalf("status%d body%s calls%d", response.Code, response.Body, ai.calls)
				}
				wantLimiter := 0
				if wantAI > 0 {
					wantLimiter = 1
				}
				if lim.calls != wantLimiter {
					t.Fatalf("limiter calls%d want%d", lim.calls, wantLimiter)
				}
				var after json.RawMessage
				if err := pg.Pool.QueryRow(ctx, `SELECT content_json FROM exercises WHERE id=$1`, id).Scan(&after); err != nil {
					t.Fatal(err)
				}
				if string(before) != string(after) {
					t.Fatal("failed/ineligible repair changed stored bytes")
				}
			})
		}
	})

	t.Run("concurrent explicit repairs reread valid winner once", func(t *testing.T) {
		id := exerciseID(t, roadmapID, 1, "reading")
		invalid := contentJSON("Repair reading", `{"passage":"tiny","questions":[]}`, true)
		if _, err := pg.Pool.Exec(ctx, `UPDATE exercises SET content_json=$2::jsonb WHERE id=$1`, id, invalid); err != nil {
			t.Fatal(err)
		}
		initial, err := repo.ContentTarget(ctx, userID, id)
		if err != nil {
			t.Fatal(err)
		}
		const writers = 8
		recording := &countingContentRepo{PgRepo: repo}
		provider := &barrierContentProvider{ready: make(chan struct{}), writers: writers}
		router := airouter.NewRouterWithProviders(map[airouter.ProviderType]airouter.LLMProvider{airouter.ProviderDeepSeek: provider})
		limiter := &countingContentLimiter{}
		svc := NewContentService(recording, router, limiter, func() time.Time { return initial.RoadmapCreatedAt.Add(time.Minute) })
		tasks := make([]Task, writers)
		errs := make([]error, writers)
		var wg sync.WaitGroup
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func(i int) { defer wg.Done(); tasks[i], errs[i] = svc.exercise(ctx, userID, id, true) }(i)
		}
		wg.Wait()
		winner, err := repo.ContentTarget(ctx, userID, id)
		if err != nil {
			t.Fatal(err)
		}
		expected := toTask(winner.Exercise)
		for i := 0; i < writers; i++ {
			if errs[i] != nil {
				t.Fatalf("repair%d: %v", i, errs[i])
			}
			if tasks[i].ContentRevision == nil || *tasks[i].ContentRevision != *expected.ContentRevision || !jsonEqual(t, tasks[i].ContentJSON, winner.Exercise.ContentJSON) {
				t.Fatalf("repair%d returned loser", i)
			}
		}
		if recording.wins.Load() != 1 || recording.reads.Load() != 2*writers-1 || recording.saves.Load() != writers || provider.calls.Load() != writers || limiter.calls.Load() != writers {
			t.Fatalf("wins%d reads%d saves%d provider%d limiter%d", recording.wins.Load(), recording.reads.Load(), recording.saves.Load(), provider.calls.Load(), limiter.calls.Load())
		}
		// Ordinary cached access after repair/profile change neither generates nor
		// revalidates against the changed learner level.
		if _, err := pg.Pool.Exec(ctx, `UPDATE users SET cefr_current='C2' WHERE id=$1`, userID); err != nil {
			t.Fatal(err)
		}
		task, err := svc.Exercise(ctx, userID, id)
		if err != nil || task.ContentRevision == nil || *task.ContentRevision != *expected.ContentRevision || provider.calls.Load() != writers {
			t.Fatalf("cached err=%v task=%+v calls%d", err, task, provider.calls.Load())
		}
	})

	t.Run("SaveContent for an unknown id loses CAS", func(t *testing.T) {
		if _, won, err := repo.SaveContent(ctx, userID, "00000000-0000-0000-0000-000000000000", json.RawMessage(`{}`), json.RawMessage(`{}`), "B1"); err != nil || won {
			t.Fatalf("unknown CAS=%t err=%v", won, err)
		}
	})
}

// These counters prove the real-service repair race has one persistence winner
// and precisely one additional owned read per loser, without a generation loop.
type countingContentRepo struct {
	*PgRepo
	reads, saves, wins atomic.Int32
}

func (r *countingContentRepo) ContentTarget(ctx context.Context, user, id string) (ContentTarget, error) {
	r.reads.Add(1)
	return r.PgRepo.ContentTarget(ctx, user, id)
}
func (r *countingContentRepo) SaveContent(ctx context.Context, user, id string, original, content json.RawMessage, level string) (json.RawMessage, bool, error) {
	r.saves.Add(1)
	saved, won, err := r.PgRepo.SaveContent(ctx, user, id, original, content, level)
	if won {
		r.wins.Add(1)
	}
	return saved, won, err
}

type countingContentLimiter struct{ calls atomic.Int32 }

func (l *countingContentLimiter) Allow(context.Context, string) error { l.calls.Add(1); return nil }

type barrierContentProvider struct {
	calls   atomic.Int32
	ready   chan struct{}
	writers int
}

func (p *barrierContentProvider) GenerateContent(ctx context.Context, _, _ string) (string, error) {
	i := p.calls.Add(1)
	if i == int32(p.writers) {
		close(p.ready)
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-p.ready:
	}
	var c airouter.ReadingContent
	_ = json.Unmarshal(airouter.SampleContent("reading"), &c)
	c.Glossary[0].Definition = fmt.Sprintf("The definition from writer %d.", i)
	body, _ := json.Marshal(c)
	return string(body), nil
}
