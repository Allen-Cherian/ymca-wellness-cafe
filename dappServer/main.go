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

func main() {
	// Initialize database with PostgreSQL
	fmt.Println("Initializing PostgreSQL database...")
	connStr := config.GetPostgresConnectionString()
	err := database.InitDB(connStr)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()

	// Load main configuration
	config.LoadConfig(CONFIG_PATH)
	config.LoadEnvConfig()

	// Load contracts configuration
	fmt.Println("Loading contracts configuration...")
	err = config.LoadContractsConfig(CONTRACTS_PATH)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to load contracts config: %v", err)
		log.Println("⚠️  System will use fallback contract from environment config")
	}

	// Validate admin configuration (optional - will warn if not configured)
	err = config.ValidateAdminConfig()
	if err != nil {
		log.Printf("⚠️  Warning: Admin configuration validation: %v", err)
	}

	// Start server
	server.BootupServer()
}
