package main

import (
	"context"
	"log"

	"lemonade-api/internal/api"
	"lemonade-api/internal/domain"
	"lemonade-api/internal/store"
	"lemonade-api/libraries"

	"github.com/gin-gonic/gin"
)

func main() {
	secrets := &libraries.Secrets{}
	if err := secrets.Init(); err != nil {
		log.Fatalf("init secrets: %v", err)
	}

	db := &libraries.Database{}
	if err := db.Init(secrets); err != nil {
		log.Fatalf("init database: %v", err)
	}

	gameStore := store.NewPostgres(db.DB)
	if err := gameStore.Migrate(); err != nil {
		log.Fatalf("migrate game tables: %v", err)
	}

	// One-off and idempotent: grant what stored runs already prove (decision TBD, Goals).
	if n, err := api.BackfillAchievements(context.Background(), gameStore); err != nil {
		log.Fatalf("backfill achievements: %v", err)
	} else if n > 0 {
		log.Printf("achievements backfill granted %d", n)
	}

	router := gin.Default()
	api.RegisterHealth(router)
	api.NewGame(gameStore, domain.DefaultConfig(), nil).Register(router)

	if err := router.Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
