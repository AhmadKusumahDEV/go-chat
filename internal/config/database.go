package config

// func InitDb()

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func NewDB(url string) (*sql.DB, error) {

	dbpostgres, errdb := sql.Open("postgres", url)

	if errdb != nil {
		fmt.Println(errdb)
	}

	dbpostgres.SetMaxIdleConns(5)
	dbpostgres.SetMaxOpenConns(20)
	dbpostgres.SetConnMaxLifetime(60 * time.Minute)
	dbpostgres.SetConnMaxIdleTime(10 * time.Minute)

	return dbpostgres, errdb
}
