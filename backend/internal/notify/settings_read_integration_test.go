package notify

import (
	"context"
	"errors"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
	"os"
	"testing"
	"time"
)

func TestIntegrationSettingsReadReloadOwnedPreferences(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	ctx := context.Background()
	pg, err := store.NewPostgres(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pg.Close)
	if _, err = store.Migrate(ctx, pg.Migrator(), store.MigrationsFS); err != nil {
		t.Fatal(err)
	}
	var owner, other string
	for i, p := range []*string{&owner, &other} {
		gid := []string{"settings-read-owner", "settings-read-other"}[i]
		_, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id=$1`, gid)
		if err = pg.Pool.QueryRow(ctx, `INSERT INTO users(email,google_id) VALUES($1,$2) RETURNING id`, gid+"@example.com", gid).Scan(p); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = pg.Pool.Exec(ctx, `DELETE FROM users WHERE google_id=$1`, gid) })
	}
	repo := NewPgRepo(pg.Pool)
	log := &callLog{}
	now := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	svc := NewService(repo, newFakeQueue(log), nil, nil, func() time.Time { return now })
	saved, err := svc.UpdateSettings(ctx, owner, SettingsRequest{NotificationTime: "07:30:00", Timezone: "America/New_York"})
	if err != nil {
		t.Fatal(err)
	}
	reloaded := NewService(NewPgRepo(pg.Pool), nil, nil, nil, func() time.Time { return now })
	out, err := reloaded.ReadSettings(ctx, owner)
	if err != nil || out.NotificationTime != saved.NotificationTime || out.NextReminderAt != saved.NextReminderAt || out.Timezone != "America/New_York" || out.PushConfigured {
		t.Fatalf("%+v %v", out, err)
	}
	otherOut, err := reloaded.ReadSettings(ctx, other)
	if err != nil || otherOut.NotificationTime != "20:00:00" || otherOut.Timezone != "UTC" {
		t.Fatalf("%+v %v", otherOut, err)
	}
	if _, err = reloaded.ReadSettings(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrUserNotFound) {
		t.Fatal(err)
	}
}
