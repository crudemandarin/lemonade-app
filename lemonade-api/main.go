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

	authCfg := auth.Config{
		Mode:              auth.Mode(secrets.AuthMode),
		FirebaseProjectID: secrets.FirebaseProjectID,
		AppEnv:            secrets.AppEnv,
		OnCloudRun:        secrets.OnCloudRun,
	}
	if err := authCfg.Validate(); err != nil {
		log.Fatalf("auth config: %v", err)
	}
	var opts []api.Option
	if authCfg.Mode == auth.ModeDev {
		log.Print("WARNING: AUTH_MODE=dev, the X-Username header is accepted. Never use this in production.")
		opts = append(opts, api.WithDevAuth())
	}
	if authCfg.FirebaseProjectID != "" {
		verifier, err := auth.NewFirebase(context.Background(), authCfg.FirebaseProjectID)
		if err != nil {
			log.Fatalf("init firebase auth: %v", err)
		}
		opts = append(opts, api.WithVerifier(verifier))
	}

	db := &libraries.Database{}
	if err := db.Init(secrets); err != nil {
		log.Fatalf("init database: %v", err)
	}

	gameStore := store.NewPostgres(db.DB)
	if err := gameStore.Migrate(); err != nil {
		log.Fatalf("migrate game tables: %v", err)
	}

	router := gin.Default()
	api.RegisterHealth(router)
	api.NewGame(gameStore, domain.DefaultConfig(), nil, opts...).Register(router)

	if err := router.Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
