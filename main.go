package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/xiusl/bank/api"
	db "github.com/xiusl/bank/db/sqlc"
)

const (
	dbDriver      = "postgres"
	dbSource      = "postgresql://root:like@localhost:5432/bank?sslmode=disable"
	serverAddress = "0.0.0.0:8086"
)

func main() {
	conn, err := sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	store := db.NewStore(conn)
	server := api.NewServer(store)

	err = server.Start(serverAddress)
	if err != nil {
		log.Fatal("connot start server:", err)
	}
}
