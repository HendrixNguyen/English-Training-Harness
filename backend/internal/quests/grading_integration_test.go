package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/onboarding"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
	"os"
	"sync"
	"testing"
	"time"
)

type replacingCounter struct {
	Counter
	replace func()
}

func (c replacingCounter) Add(ctx context.Context, u, d string, s int64) (int64, error) {
	n, e := c.Counter.Add(ctx, u, d, s)
	if e == nil {
		c.replace()
	}
	return n, e
}

func TestIntegrationQuizResultPersistenceAndCAS(t *testing.T) {
	dbURL, redisURL := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_REDIS_URL")
	if dbURL == "" || redisURL == "" {
		t.Skip("TEST_DATABASE_URL/TEST_REDIS_URL unset")
	}
	ctx := context.Background()
	pg, err := store.NewPostgres(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pg.Close)
	if _, err = store.Migrate(ctx, pg.Migrator(), store.MigrationsFS); err != nil {
		t.Fatal(err)
	}
	rdb, err := store.NewRedis(ctx, redisURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	seed := func(suffix string) string {
		gid := "quiz-result-integration-" + suffix
		_, err := pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id=$1`, gid)
		if err != nil {
			t.Fatal(err)
		}
		var id string
		err = pg.Pool.QueryRow(ctx, `INSERT INTO users(email,google_id,target_goal,timezone) VALUES($1,$2,'','UTC') RETURNING id`, gid+"@example.com", gid).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id=$1`, gid) })
		_, err = onboarding.NewPgRepo(pg.Pool).SaveAssessment(ctx, id, onboarding.Assessment{CEFRLevel: "B1", TargetGoal: "integration", Timezone: "UTC", NotificationTime: "20:00:00", Roadmap: integrationRoadmap()})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	user, other := seed("owner"), seed("other")
	now := time.Now().UTC()
	date := LocalDate(now, time.UTC)
	counter := NewRedisCounter(rdb)
	t.Cleanup(func() {
		for _, id := range []string{user, other} {
			rdb.Client.Del(ctx, store.DailyAccumulatedKey(id, now))
		}
	})
	repo := NewPgRepo(pg.Pool)
	svc := NewService(counter, repo, repo, NopPet{}, fixedClock(now))
	suite, err := svc.Daily(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := svc.Daily(ctx, other)
	if err != nil {
		t.Fatal(err)
	}
	reset := func(id string, body json.RawMessage) {
		_, err := pg.Pool.Exec(ctx, `UPDATE exercises SET content_json=$2::jsonb,is_completed=false,quiz_result=NULL WHERE id=$1`, id, body)
		if err != nil {
			t.Fatal(err)
		}
	}
	read := func(id string) Exercise {
		rm, err := repo.ActiveRoadmap(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		e, err := repo.ProgressExercise(ctx, user, rm.ID, id, 1)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	id := suite.Tasks[0].ID
	reset(id, gradingExercise().ContentJSON)
	e := read(id)
	_, rev, _, _ := ContentContract(e)
	t.Run("validation_before_any_write", func(t *testing.T) {
		for _, tc := range []struct {
			name, id, rev, answers string
			status                 int
		}{
			{"partial", id, *rev, `{"q1":"A"}`, 400}, {"bad_option", id, *rev, `{"q1":"Z","q2":"A","q3":"A"}`, 400}, {"extra", id, *rev, `{"q1":"A","q2":"A","q3":"A","extra":"A"}`, 400}, {"stale", id, "stale", `{"q1":"A","q2":"A","q3":"A"}`, 409}, {"foreign", foreign.Tasks[0].ID, *rev, `{}`, 404}, {"invalid_id", "not-a-uuid", *rev, `{}`, 404},
		} {
			t.Run(tc.name, func(t *testing.T) {
				body := fmt.Sprintf(`{"exercise_id":%q,"duration_seconds":17,"content_revision":%q,"user_answers":%s}`, tc.id, tc.rev, tc.answers)
				w := postJSON(newQuestRouter(svc, user), "/api/v1/quests/progress", body)
				if w.Code != tc.status {
					t.Fatal(w.Code, w.Body.String())
				}
				total, err := counter.Total(ctx, user, date)
				if err != nil || total != 0 {
					t.Fatal(total, err)
				}
				var n int
				if err := pg.Pool.QueryRow(ctx, `SELECT count(*) FROM daily_progress WHERE user_id=$1`, user).Scan(&n); err != nil || n != 0 {
					t.Fatal(n, err)
				}
				got := read(id)
				if got.IsCompleted || got.QuizResult != nil {
					t.Fatal(got)
				}
			})
		}
	})
	t.Run("grade_persist_reopen_and_preserve", func(t *testing.T) {
		out, err := svc.RecordProgress(ctx, user, id, 47, ProgressSubmission{Revision: rev, Answers: map[string]string{"q1": "A", "q2": "A", "q3": "A"}})
		if err != nil {
			t.Fatal(err)
		}
		var q QuizResult
		if err := json.Unmarshal(out.QuizResult, &q); err != nil {
			t.Fatal(err)
		}
		if q.Status != "graded" || q.ScorePercent == nil || *q.ScorePercent != 67 || q.DurationSeconds != 47 || q.Answers[1].CorrectOption == nil || *q.Answers[1].CorrectOption != "B" || q.Answers[1].Explanation == nil {
			t.Fatal(string(out.QuizResult))
		}
		fresh := NewPgRepo(pg.Pool)
		freshSvc := NewService(counter, fresh, fresh, NopPet{}, fixedClock(now))
		daily, err := freshSvc.Daily(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		if !daily.Tasks[0].IsCompleted || string(daily.Tasks[0].QuizResult) != string(out.QuizResult) {
			t.Fatal("daily lost result")
		}
		target, err := fresh.ContentTarget(ctx, user, id)
		if err != nil || string(target.Exercise.QuizResult) != string(out.QuizResult) {
			t.Fatal("content target lost result", err)
		}
		again, err := freshSvc.RecordProgress(ctx, user, id, 11, ProgressSubmission{Answers: map[string]string{"q1": "B"}})
		if err != nil || string(again.QuizResult) != string(out.QuizResult) {
			t.Fatal("overwrote", err)
		}
		var done bool
		var raw json.RawMessage
		if err := pg.Pool.QueryRow(ctx, `SELECT is_completed,quiz_result FROM exercises WHERE id=$1`, foreign.Tasks[0].ID).Scan(&done, &raw); err != nil || done || raw != nil {
			t.Fatal("foreign changed", err)
		}
	})
	t.Run("replacement_between_validation_and_completion", func(t *testing.T) {
		id := suite.Tasks[1].ID
		reset(id, gradingExercise().ContentJSON)
		e := read(id)
		_, r, _, _ := ContentContract(e)
		replacement := json.RawMessage(`{"passage":"A concurrent replacement without these questions."}`)
		c := replacingCounter{Counter: counter, replace: func() { reset(id, replacement) }}
		racing := NewService(c, repo, repo, NopPet{}, fixedClock(now))
		_, err := racing.RecordProgress(ctx, user, id, 13, ProgressSubmission{Revision: r, Answers: map[string]string{"q1": "A", "q2": "B", "q3": "A"}})
		if !errors.Is(err, ErrExerciseContentChanged) {
			t.Fatal(err)
		}
		got := read(id)
		if got.IsCompleted || got.QuizResult != nil {
			t.Fatal("captured grade attached to replacement", got)
		}
	})
	t.Run("concurrent_completions_keep_one_selection_set", func(t *testing.T) {
		id := suite.Tasks[2].ID
		reset(id, gradingExercise().ContentJSON)
		e := read(id)
		_, r, _, _ := ContentContract(e)
		rm, _ := repo.ActiveRoadmap(ctx, user)
		const n = 8
		results := make([]json.RawMessage, n)
		errs := make([]error, n)
		var wg sync.WaitGroup
		for i := range n {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				choice := "A"
				if i%2 == 0 {
					choice = "B"
				}
				raw, err := gradeSubmission(e, int64(20+i), ProgressSubmission{Revision: r, Answers: map[string]string{"q1": choice, "q2": "B", "q3": "A"}})
				if err != nil {
					errs[i] = err
					return
				}
				results[i], errs[i] = repo.MarkComplete(ctx, user, rm.ID, e, raw)
			}(i)
		}
		wg.Wait()
		for i := range n {
			if errs[i] != nil || string(results[i]) != string(results[0]) {
				t.Fatalf("writer %d err %v result %s winner %s", i, errs[i], results[i], results[0])
			}
		}
		got := read(id)
		if !got.IsCompleted || string(got.QuizResult) != string(results[0]) {
			t.Fatal("completion and result disagree")
		}
	})

	t.Run("revisionless_invalid_study_persists_ungraded_choices", func(t *testing.T) {
		id := suite.Tasks[2].ID
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(gradingExercise().ContentJSON, &envelope); err != nil {
			t.Fatal(err)
		}
		envelope["content_schema"] = json.RawMessage(`1`)
		body, _ := json.Marshal(envelope)
		reset(id, body)
		out, err := svc.RecordProgress(ctx, user, id, 29, ProgressSubmission{Answers: map[string]string{"q2": "B"}})
		if err != nil {
			t.Fatal(err)
		}
		var result QuizResult
		if err := json.Unmarshal(out.QuizResult, &result); err != nil {
			t.Fatal(err)
		}
		if result.Status != "ungraded" || result.QuestionCount != 3 || result.AnsweredCount != 1 || result.CorrectCount != nil || result.ScorePercent != nil || result.Answers[0].CorrectOption != nil || result.Answers[0].IsCorrect != nil || result.Answers[0].Explanation != nil {
			t.Fatal(string(out.QuizResult))
		}
		got := read(id)
		if !got.IsCompleted || string(got.QuizResult) != string(out.QuizResult) {
			t.Fatal("legacy selection did not persist")
		}
	})

	t.Run("owned_active_snapshot_CAS", func(t *testing.T) {
		id := suite.Tasks[2].ID
		reset(id, gradingExercise().ContentJSON)
		e := read(id)
		rm, _ := repo.ActiveRoadmap(ctx, user)
		raw, err := gradeSubmission(e, 19, ProgressSubmission{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.MarkComplete(ctx, other, rm.ID, e, raw); !errors.Is(err, ErrExerciseNotFound) {
			t.Fatal("foreign CAS", err)
		}
		_, err = pg.Pool.Exec(ctx, `UPDATE roadmaps SET is_active=false WHERE id=$1`, rm.ID)
		if err != nil {
			t.Fatal(err)
		}
		_, err = repo.MarkComplete(ctx, user, rm.ID, e, raw)
		if !errors.Is(err, ErrExerciseNotFound) {
			t.Fatal("inactive CAS", err)
		}
		_, err = pg.Pool.Exec(ctx, `UPDATE roadmaps SET is_active=true WHERE id=$1`, rm.ID)
		if err != nil {
			t.Fatal(err)
		}
		got := read(id)
		if got.IsCompleted || got.QuizResult != nil {
			t.Fatal(got)
		}
	})
}
