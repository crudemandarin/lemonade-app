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

	return nil
}
