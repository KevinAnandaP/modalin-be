package main

import (
	"log"

	"modalin-be/pkg/config"
	"modalin-be/pkg/database"
)

func main() {
	config.LoadConfig()
	if err := config.AppConfig.Validate(); err != nil {
		log.Fatal(err)
	}
	database.ConnectDB()
	if err := database.MigrateDB(); err != nil {
		log.Fatal(err)
	}
}
