package main

import (
	"database/sql"
	_ "embed"
	"log"

	_ "github.com/lib/pq"
)

//go:embed schema.sql
var schemaSQL []byte

func openDB(c *Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", c.DSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	return db, nil
}

func migrate(db *sql.DB) error {
	log.Println("running schema migration")
	_, err := db.Exec(string(schemaSQL))
	return err
}
