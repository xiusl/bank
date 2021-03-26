package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/xiusl/bank/api"
	db "github.com/xiusl/bank/db/sqlc"
	"github.com/xiusl/bank/util"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
		return
	}
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	store := db.NewStore(conn)
	server := api.NewServer(store)

	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("connot start server:", err)
	}
}
