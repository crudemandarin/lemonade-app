package libraries

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Secrets struct {
	DBHost     string
	DBUsername string
	DBPassword string
	DBName     string
	DBPort     string

	// Auth. See internal/auth.Config for the rules that are enforced at startup.
	AuthMode          string // AUTH_MODE: "firebase" (default) or "dev"
	FirebaseProjectID string // FIREBASE_PROJECT_ID
	AppEnv            string // APP_ENV: "production" refuses AUTH_MODE=dev
	OnCloudRun        bool   // K_SERVICE is set by Cloud Run
}

// Init loads .env (if present) and populates s from the environment.
func (s *Secrets) Init() error {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: %w", err)
	}

	s.DBHost = os.Getenv("DB_HOST")
	s.DBUsername = os.Getenv("DB_USERNAME")
	s.DBPassword = os.Getenv("DB_PASSWORD")
	s.DBName = os.Getenv("DB_NAME")
	s.DBPort = os.Getenv("DB_PORT")

	s.AuthMode = os.Getenv("AUTH_MODE")
	s.FirebaseProjectID = os.Getenv("FIREBASE_PROJECT_ID")
	s.AppEnv = os.Getenv("APP_ENV")
	s.OnCloudRun = os.Getenv("K_SERVICE") != ""

	return nil
}
