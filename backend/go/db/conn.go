/* Place: backend/go/db/conn.go */
package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/denisenkom/go-mssqldb" // mssql driver
)

// Connect opens a connection to SQL Server using the provided DSN.
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	// reasonable defaults
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// ping with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}
