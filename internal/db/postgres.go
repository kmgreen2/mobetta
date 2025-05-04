package db

import (
	"database/sql"
	_ "github.com/lib/pq"
	"log"
)

type PostgresDb struct {
	db      *sql.DB
	connStr string
}

func NewPostgresDb(connStr string) *PostgresDb {
	return &PostgresDb{connStr: connStr}
}

func (pg *PostgresDb) Open() error {
	var err error
	pg.db, err = sql.Open("postgres", pg.connStr)
	if err != nil {
		return err
	}

	_, err = pg.db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (pg *PostgresDb) Close() {
	pg.db.Close()
}

func (pg *PostgresDb) Exec(query string, args ...interface{}) (sql.Result, error) {
	result, err := pg.db.Exec(query, args...)
	return result, err
}

func (pg *PostgresDb) Query(query string, args ...interface{}) (*sql.Rows, error) {
	result, err := pg.db.Query(query, args...)
	return result, err
}
