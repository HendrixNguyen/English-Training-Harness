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
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/middleware"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/notify"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/onboarding"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/pet"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/quests"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

// petForOnboarding adapts *pet.Service to onboarding.Pet. onboarding defines
// its own PetState so it never imports pet (pet imports quests; quests' tests
// import onboarding — an import here would be a cycle).
type petForOnboarding struct{ svc *pet.Service }

func (p petForOnboarding) Ensure(ctx context.Context, userID string) (onboarding.PetState, error) {
	st, err := p.svc.Ensure(ctx, userID)
	if err != nil {
		return onboarding.PetState{}, err
	}
	return onboarding.PetState{PlantName: st.PlantName, HealthPoints: st.HealthPoints, Stage: st.Stage}, nil
}

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

	// Spec §8: the PWA is a separate Railway service on its own origin, so
	// every browser call is cross-origin. Global, so preflights for paths with
	// no OPTIONS route reach it (see middleware.CORS).
	origins, err := middleware.ParseOrigins(cfg.FrontendOrigin)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	r.Use(middleware.CORS(origins))
	log.Printf("cors: allowing %v", origins)

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

	// Spec §2.1 reminder worker, in-process, polling the §4 queue:webpush:delay
	// ZSET every 30 s. It starts only when both VAPID keys (spec §9) are set;
	// without them settings are stored but nothing is sent.
	var pushSender notify.Sender
	if cfg.VAPIDPublicKey != "" && cfg.VAPIDPrivateKey != "" {
		sender, err := notify.NewWebPushSender(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
		if err != nil {
			log.Fatalf("notify: %v", err)
		}
		pushSender = sender
	}
	notifySvc := notify.NewService(
		notify.NewPgRepo(pg.Pool),
		notify.NewRedisQueue(rdb),
		pushSender,
		studyCounter, // notify reads the daily counter only through this interface
		time.Now,
	)
	if pushSender != nil {
		go notify.RunWorker(ctx, notifySvc, notify.PollInterval)
	} else {
		log.Printf("notify: VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY unset; reminder settings are stored but no Web Push is sent")
	}

	v1 := r.Group("/api/v1")
	v1.Use(middleware.BodyLimit(middleware.MaxBodyBytes)) // before any route: a group's middleware is copied at registration
	v1.POST("/auth/google", auth.Handler(authSvc))

	guarded := v1.Group("", auth.Require(tokens, sessions))
	guarded.GET("/quests/daily", quests.DailyHandler(questSvc))
	guarded.POST("/quests/progress", quests.ProgressHandler(questSvc))
	guarded.GET("/pet/status", pet.StatusHandler(petSvc))
	guarded.POST("/pet/revive", pet.ReviveHandler(petSvc))
	guarded.POST("/settings/notifications", notify.SettingsHandler(notifySvc))

	googleSvc := google.NewService(
		google.NewPgRefreshTokenSource(pg.Pool), // plaintext today; the §7 encryption fix replaces only this
		google.NewOAuthClient(cfg.GoogleClientID, cfg.GoogleClientSecret),
		google.NewHTTPCalendarClient(),
		google.NewHTTPTasksClient(),
		google.NewPgRepo(pg.Pool),
		time.Now,
	)
	guarded.POST("/integrations/google/sync", google.SyncHandler(googleSvc))
	onboardingSvc := onboarding.NewService(
		onboarding.NewPgRepo(pg.Pool),
		onboarding.NewRedisQuizStore(rdb),
		airouter.NewRedisRateLimiter(rdb),
		aiRouter,
		petForOnboarding{svc: petSvc},
		time.Now,
	)
	guarded.GET("/onboarding/quiz", onboarding.QuizHandler())
	guarded.POST("/onboarding/assessment", onboarding.AssessmentHandler(onboardingSvc))

	log.Printf("listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
