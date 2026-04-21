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
const DID_MAPPING_PATH = ".config/did_mapping.toml"

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

	// Load DID mapping configuration
	fmt.Println("Loading DID mapping configuration...")
	config.LoadDIDMapping(DID_MAPPING_PATH)

	// Validate DID mapping (ensures both DIDs exist and use same port)
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("❌ Failed to get config: %v", err)
	}
	didMapping := config.GetDIDMapping()
	err = config.ValidateDIDMapping(cfg, didMapping)
	if err != nil {
		log.Fatalf("❌ DID Mapping validation failed: %v\n   Please check your .config/did_mapping.toml file", err)
	}

	// Validate admin configuration (optional - will warn if not configured)
	err = config.ValidateAdminConfig()
	if err != nil {
		log.Printf("⚠️  Warning: Admin configuration validation: %v", err)
	}

	// Start server
	server.BootupServer()
}
