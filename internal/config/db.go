package config

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type DB interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Begin() (*sql.Tx, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	Close() error
}

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

func (r RealDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.conn.ExecContext(ctx, query, args...)
}

func (r *RealDB) Begin() (*sql.Tx, error) {
	return r.conn.Begin()
}

func (r RealDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return r.conn.BeginTx(ctx, opts)
}

func (r *RealDB) Close() error {
	return r.conn.Close()
}

var (
	dbInstance DB
	once       sync.Once
	initErr    error
)

// InitDB initializes the database connection once.
func InitDB(dataSourceName string) (DB, error) {
	once.Do(func() {
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			getEnv("POSTGRES_HOST", "localhost"),
			getEnv("POSTGRES_PORT", "5432"), getEnv("POSTGRES_USER", "myuser"),
			getEnv("POSTGRES_PASSWORD", "mypassword."),
			getEnv("POSTGRES_DB", "mydatabase"),
		)
		log.Println(connStr)
		conn, err := sql.Open("postgres", connStr)
		if err != nil {
			initErr = fmt.Errorf("[DB]Failed to connect to database: %v", err)
			return
		}

		conn.SetMaxOpenConns(10)
		conn.SetMaxIdleConns(5)
		conn.SetConnMaxLifetime(time.Hour)

		if err := conn.Ping(); err != nil {
			log.Fatalf("[DB]Failed to ping database: %v", err)
			initErr = fmt.Errorf("[DB]Failed to ping database: %v", err)
			return
		}

		log.Println("[DB]Database connection established.")
		dbInstance = &RealDB{conn: conn}
	})
	return dbInstance, initErr
}

// CloseDB closes the database connection.
func CloseDB() {
	if dbInstance != nil {
		if err := dbInstance.Close(); err != nil {
			log.Fatal("[DB]Error closing database:", err)
		} else {
			log.Println("[DB]Database connection closed.")
		}
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
