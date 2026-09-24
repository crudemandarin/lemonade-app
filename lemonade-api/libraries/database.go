package libraries

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	*gorm.DB
}

// Init connects to Postgres using the credentials in secrets.
func (db *Database) Init(secrets *Secrets) error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/Denver",
		secrets.DBHost,
		secrets.DBUsername,
		secrets.DBPassword,
		secrets.DBName,
		secrets.DBPort,
	)

	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	db.DB = conn
	return nil
}
