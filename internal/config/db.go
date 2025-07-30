package config

import (
	"database/sql"
	"log"
	"sync"
	"time"
)

// DB is an interface to abstract database operations for mocking/testing.
type DB interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	Begin() (*sql.Tx, error)
	Close() error
}

// RealDB wraps *sql.DB to implement the DB interface.
type RealDB struct {
	conn *sql.DB
}

func (r *RealDB) Query(query string, args ...any) (*sql.Rows, error) {
	return r.conn.Query(query, args...)
}

func (r *RealDB) QueryRow(query string, args ...any) *sql.Row {
	return r.conn.QueryRow(query, args...)
}

func (r *RealDB) Exec(query string, args ...any) (sql.Result, error) {
	return r.conn.Exec(query, args...)
}

func (r *RealDB) Begin() (*sql.Tx, error) {
	return r.conn.Begin()
}

func (r *RealDB) Close() error {
	return r.conn.Close()
}

// Singleton instance
var (
	dbInstance DB
	once       sync.Once
)

// InitDB initializes the database connection once and wraps it with RealDB.
func InitDB(dataSourceName string) DB {
	once.Do(func() {
		conn, err := sql.Open("sqlite3", dataSourceName)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		conn.SetMaxOpenConns(10)
		conn.SetMaxIdleConns(5)
		conn.SetConnMaxLifetime(time.Hour)

		if err := conn.Ping(); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}

		log.Println("Database connection established.")
		dbInstance = &RealDB{conn: conn}
	})
	return dbInstance
}

// GetDB returns the singleton DB interface.
func GetDB() DB {
	if dbInstance == nil {
		log.Fatal("Database is not initialized. Call InitDB first.")
	}
	return dbInstance
}

// CloseDB closes the real database connection.
func CloseDB() {
	if dbInstance != nil {
		if err := dbInstance.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		} else {
			log.Println("Database connection closed.")
		}
	}
}
