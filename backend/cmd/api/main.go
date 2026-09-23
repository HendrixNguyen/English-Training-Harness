// Command api is the single Go process described in spec §2.1. This slice
// serves only GET /healthz; later slices mount their own route groups here.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/airouter"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/config"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/google"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/health"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/pet"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

func main() {
	// Cancellable so background work started with it (the pet hourly cron)
	// stops on SIGINT/SIGTERM instead of leaking past process shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pg, err := store.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pg.Close()

	rdb, err := store.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	applied, err := store.Migrate(ctx, pg.Migrator(), store.MigrationsFS)
	if err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if len(applied) > 0 {
		log.Printf("migrations applied: %v", applied)
	}

	aiRouter := airouter.NewRouter(airouter.ConfigFromEnv(os.Getenv))
	if providers := aiRouter.Providers(); len(providers) == 0 {
		log.Printf("airouter: no provider API keys set; AI-backed routes will answer 503")
	} else {
		log.Printf("airouter: providers %v", providers)
	}

	r := gin.Default()
	r.GET("/healthz", health.Handler(pg, rdb))

	tokens := auth.NewTokenIssuer(cfg.JWTSecret, time.Now)
	sessions := auth.NewRedisSessionStore(rdb)
	authSvc := auth.NewService(
		auth.NewGoogleClient(cfg.GoogleClientID, cfg.GoogleClientSecret),
		auth.NewPgUserRepo(pg.Pool),
		sessions,
		tokens,
	)

	studyCounter := quests.NewRedisCounter(rdb)
	petSvc := pet.NewService(
		pet.NewPgRepo(pg.Pool),
		pet.NewRedisChallengeStore(rdb),
		studyCounter, // pet reads the daily counter only through this interface
		time.Now,
	)

	questRepo := quests.NewPgRepo(pg.Pool) // satisfies both QuestRepo and ProgressRepo
	questSvc := quests.NewService(
		studyCounter,
		questRepo,
		questRepo,
		pet.NewQuestHook(petSvc),
		time.Now,
	)

	// Spec §8 hourly cron, in-process (§2.1). Sweeps at every :00 UTC.
	go pet.RunHourly(ctx, petSvc)

	v1 := r.Group("/api/v1")
	v1.POST("/auth/google", auth.Handler(authSvc))

	guarded := v1.Group("", auth.Require(tokens, sessions))
	guarded.GET("/quests/daily", quests.DailyHandler(questSvc))
	guarded.POST("/quests/progress", quests.ProgressHandler(questSvc))
	guarded.GET("/pet/status", pet.StatusHandler(petSvc))
	guarded.POST("/pet/revive", pet.ReviveHandler(petSvc))

	googleSvc := google.NewService(
		google.NewPgRefreshTokenSource(pg.Pool), // plaintext today; the §7 encryption fix replaces only this
		google.NewOAuthClient(cfg.GoogleClientID, cfg.GoogleClientSecret),
		google.NewHTTPCalendarClient(),
		google.NewHTTPTasksClient(),
		google.NewPgRepo(pg.Pool),
		time.Now,
	)
	guarded.POST("/integrations/google/sync", google.SyncHandler(googleSvc))

	log.Printf("listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
