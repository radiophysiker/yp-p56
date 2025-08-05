package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DatabaseConnection struct {
	db *sqlx.DB
}

func NewPostgreSQLConnection(databaseURI string) (*DatabaseConnection, error) {
	if databaseURI == "" {
		return nil, fmt.Errorf("database URI is not set")
	}

	db, err := sqlx.Connect("postgres", databaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DatabaseConnection{
		db: db,
	}, nil
}

func (dc *DatabaseConnection) GetDB() *sqlx.DB {
	return dc.db
}

func (dc *DatabaseConnection) Close() error {
	if dc.db != nil {
		return dc.db.Close()
	}
	return nil
}

func (dc *DatabaseConnection) Health() error {
	return dc.db.Ping()
}
