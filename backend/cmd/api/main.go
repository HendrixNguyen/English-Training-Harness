// Command api is the single Go process described in spec §2.1. This slice
// serves only GET /healthz; later slices mount their own route groups here.
package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/auth"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/config"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/health"
	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/store"
)

func main() {
	ctx := context.Background()

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

	r := gin.Default()
	r.GET("/healthz", health.Handler(pg, rdb))

	authSvc := auth.NewService(
		auth.NewGoogleClient(cfg.GoogleClientID, cfg.GoogleClientSecret),
		auth.NewPgUserRepo(pg.Pool),
		auth.NewRedisSessionStore(rdb),
		auth.NewTokenIssuer(cfg.JWTSecret, time.Now),
	)

	v1 := r.Group("/api/v1")
	v1.POST("/auth/google", auth.Handler(authSvc))

	// Later slices mount their routes on this group:
	//   guarded := v1.Group("", auth.Require(tokens, sessions))

	log.Printf("listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
