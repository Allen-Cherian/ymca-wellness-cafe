package main

import (
	"dapp-server/config"
	"dapp-server/database"
	"dapp-server/server"
	"fmt"
	"log"
)

const CONFIG_PATH = ".config/config.toml"
const CONTRACTS_PATH = ".config/contracts.toml"
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
	fmt.Println("Loading configurations...")
	config.LoadConfig(CONFIG_PATH)
	config.LoadContractsConfig(CONTRACTS_PATH)
	config.LoadEnvConfig()

	// Validate admin configuration
	fmt.Println("Validating admin configurations...")
	if err := config.ValidateAdminConfig(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Start server
	fmt.Println("Starting server...")
	server.BootupServer()
}
