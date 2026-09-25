package main

import (
	"context"
	"log"

	"lemonade-api/internal/api"
	"lemonade-api/internal/auth"
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

	// Google sign-in is optional: without a Firebase project only username play works.
	var opts []api.Option
	if secrets.FirebaseProjectID != "" {
		verifier, err := auth.NewFirebase(context.Background(), secrets.FirebaseProjectID)
		if err != nil {
			log.Fatalf("init firebase auth: %v", err)
		}
		opts = append(opts, api.WithVerifier(verifier))
	} else {
		log.Print("FIREBASE_PROJECT_ID is not set: Google sign-in is off, players can only use a username")
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
	api.NewGame(gameStore, domain.DefaultConfig(), nil, opts...).Register(router)

	if err := router.Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
