package notify

import (
	"context"
	"log"
	"time"
)

// RunWorker is the in-process reminder worker from spec §2.1 — the same shape
// as pet.RunHourly. It blocks until ctx is cancelled, running svc.Tick every
// `every` (PollInterval in production). Errors are logged and the loop
// continues: one bad subscription must not stop everyone else's reminder.
// Run exactly one worker per deployment: Tick's "re-slot then send" is safe
// against crashes, not against two processes popping the same member.
func RunWorker(ctx context.Context, svc *Service, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		at := svc.now()
		stats, err := svc.Tick(ctx, at)
		if err != nil {
			log.Printf("notify: tick at %s: %+v, errors: %v", at.UTC().Format(time.RFC3339), stats, err)
			continue
		}
		if stats.Due > 0 {
			log.Printf("notify: tick at %s: %+v", at.UTC().Format(time.RFC3339), stats)
		}
	}
}
