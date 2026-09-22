package pet

import (
	"context"
	"log"
	"time"
)

// NextTopOfHour is the next :00 strictly after now (spec §8: the cron runs
// at :00 UTC every hour).
func NextTopOfHour(now time.Time) time.Time {
	return now.Truncate(time.Hour).Add(time.Hour)
}

// RunHourly is the in-process cron worker from spec §2.1. It blocks until ctx
// is cancelled, calling svc.Sweep at every :00 UTC. Sweep errors are logged
// and the loop continues — one bad row must not stop tomorrow's decay.
func RunHourly(ctx context.Context, svc *Service) {
	for {
		now := svc.now()
		timer := time.NewTimer(NextTopOfHour(now).Sub(now))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		at := svc.now()
		n, err := svc.Sweep(ctx, at)
		if err != nil {
			log.Printf("pet: sweep at %s: penalised %d, errors: %v", at.UTC().Format(time.RFC3339), n, err)
			continue
		}
		if n > 0 {
			log.Printf("pet: sweep at %s: penalised %d", at.UTC().Format(time.RFC3339), n)
		}
	}
}
