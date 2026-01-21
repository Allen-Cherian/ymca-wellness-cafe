package main

import (
	"dapp-server/config"
	"dapp-server/database"
	"dapp-server/server"
	"fmt"
	"log"
)

const CONFIG_PATH = ".config/config.toml"
func main() {
	// Initialize database with PostgreSQL
	fmt.Println("Initializing PostgreSQL database...")
	connStr := config.GetPostgresConnectionString()
	err := database.InitDB(connStr)
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
