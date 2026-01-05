package main

import (
	"dapp-server/config"
	"dapp-server/database"
	"dapp-server/server"
	"fmt"
	"log"
)

const CONFIG_PATH = ".config/config.toml"
const DB_PATH = "./transfer_status.db"

func main() {
	// Initialize database
	fmt.Println("Initializing database...")
	err := database.InitDB(DB_PATH)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()

	// Load configuration
	config.LoadConfig(CONFIG_PATH)
	config.LoadEnvConfig()

	// Start server
	server.BootupServer()
}
