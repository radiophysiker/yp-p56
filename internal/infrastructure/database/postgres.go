package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// DatabaseConnection представляет подключение к базе данных
type DatabaseConnection struct {
	db *sqlx.DB
	// txMgr TransactionManager
}

// NewPostgreSQLConnection создает новое подключение к PostgreSQL
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
		// txMgr: NewPostgreSQLTransactionManager(db),
	}, nil
}

// GetDB возвращает объект базы данных для использования в репозиториях
func (dc *DatabaseConnection) GetDB() *sqlx.DB {
	return dc.db
}

// GetTransactionManager возвращает менеджер транзакций
// func (dc *DatabaseConnection) GetTransactionManager() TransactionManager {
// 	return dc.txMgr
// }

// Close закрывает подключение к базе данных
func (dc *DatabaseConnection) Close() error {
	if dc.db != nil {
		return dc.db.Close()
	}
	return nil
}

// Health проверяет состояние подключения
func (dc *DatabaseConnection) Health() error {
	return dc.db.Ping()
}
