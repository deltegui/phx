package persistence

import (
	"log"

	"github.com/jmoiron/sqlx"
)

type Configuration struct {
	Driver     string `configName:"dbdriver"`
	Connection string `configName:"dbconn"`
}

func Connect(config Configuration) *sqlx.DB {
	db := connect(config)
	checkConnection(db)
	return db
}

func connect(config Configuration) *sqlx.DB {
	db, err := sqlx.Open(config.Driver, config.Connection)
	if err != nil {
		log.Fatalln("Error creating connection to database: ", err)
	}
	log.Printf("Connected to '%s', using %s\n", config.Connection, config.Driver)
	return db
}

func checkConnection(conn *sqlx.DB) {
	if err := conn.Ping(); err != nil {
		log.Fatalln("Error checking connection to database: ", err)
	}
}
