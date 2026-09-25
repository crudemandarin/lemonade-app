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

	// Optional Google sign-in (securing an account). Empty turns it off: players can
	// still play by username alone.
	FirebaseProjectID string // FIREBASE_PROJECT_ID
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

	s.FirebaseProjectID = os.Getenv("FIREBASE_PROJECT_ID")

	return nil
}
