package config

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
)

// Struct to represent each node
type Node struct {
	Name string `toml:"name"`
	Port string `toml:"port"`
	DID  string `toml:"did"`
	Path string `toml:"path"` // Assuming Path is a field in the Node struct
}

// Struct to hold the configuration
type Config struct {
	Nodes map[string]Node `toml:"nodes"`
}

var (
	instance *Config
	once     sync.Once
)

// LoadConfig initializes the configuration (Singleton)
func LoadConfig(filepath string) {
	once.Do(func() {
		instance = &Config{}
		if _, err := toml.DecodeFile(filepath, instance); err != nil {
			log.Fatalf("Error loading config file: %v", err)
		}
	})
}

// GetConfig returns the global configuration instance
func GetConfig() (*Config, error) {
	if instance == nil {
		// log.Fatal("Config not loaded. Call LoadConfig() first.")
		return nil, fmt.Errorf("Config not loaded. Call LoadConfig() first")
	}
	return instance, nil
}

// GetNodeNameByPort searches for a node by its port and returns its name
func GetNodeNameByPort(config *Config, port string) (string, bool) {
	for _, node := range config.Nodes {
		if node.Port == port {
			return node.Name, true
		}
	}
	return "", false
}
func GetPathByPort(config *Config, port string) (string, bool) {
	for _, node := range config.Nodes {
		if node.Port == port {
			return node.Path, true
		}
	}
	return "", false
}

func GetNodeNameByDid(config *Config, did string) (string, bool) {
	for _, node := range config.Nodes {
		if node.DID == did {
			return node.Name, true
		}
	}
	return "", false
}

func GetPortByNodeName(config *Config, nodeName string) (string, bool) {
	for _, node := range config.Nodes {
		if node.Name == nodeName {
			return node.Port, true
		}
	}
	return "", false
}

func GetPortByDid(config *Config, did string) (string, bool) {
	for _, node := range config.Nodes {
		if node.DID == did {
			return node.Port, true
		}
	}
	return "", false
}

type EnvConfig struct {
	AddActivityContract string
	AddAdminContract    string
	TransferContract    string
	ActivityUpdatePath  string
	AdminUpdatePath     string
	// PostgreSQL connection parameters
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

var (
	envInstance *EnvConfig
	envOnce     sync.Once
)

// LoadConfig initializes the configuration
func LoadEnvConfig() *EnvConfig {
	envOnce.Do(func() {
		// Load the .env file
		err := godotenv.Load(".config/.env")
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}

		// Populate the config instance
		envInstance = &EnvConfig{
			AddActivityContract: os.Getenv("ADD_ACTIVITY_CONTRACT"),
			TransferContract:    os.Getenv("TRANSFER_CONTRACT"),
			AddAdminContract:    os.Getenv("ADD_ADMIN_CONTRACT"),
			ActivityUpdatePath:  os.Getenv("ACTIVITY_UPDATE_PATH"),
			AdminUpdatePath:     os.Getenv("ADD_ADMIN_PATH"),
			// PostgreSQL connection parameters
			DBHost:     getEnvOrDefault("DB_HOST", "localhost"),
			DBPort:     getEnvOrDefault("DB_PORT", "5432"),
			DBUser:     getEnvOrDefault("DB_USER", "postgres"),
			DBPassword: os.Getenv("DB_PASSWORD"),
			DBName:     getEnvOrDefault("DB_NAME", "dapp_server"),
			DBSSLMode:  getEnvOrDefault("DB_SSL_MODE", "disable"),
		}
	})
	return envInstance
}

func GetEnvConfig() *EnvConfig {
	if envInstance == nil {
		return LoadEnvConfig()
	}
	return envInstance
}

// getEnvOrDefault returns the environment variable value or a default value if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetPostgresConnectionString builds the PostgreSQL connection string
func GetPostgresConnectionString() string {
	cfg := GetEnvConfig()

	// Build connection string, omitting password if empty
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	// Only add password if it's not empty
	if cfg.DBPassword != "" {
		connStr = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBSSLMode,
		)
	}

	return connStr
}

// ═══════════════════════════════════════════════════════════
// DID Mapping Configuration
// ═══════════════════════════════════════════════════════════

// DIDMapping holds the mapping from incoming DIDs to replacement DIDs
type DIDMapping struct {
	Mapping map[string]string `toml:"mapping"`
}

var (
	didMappingInstance *DIDMapping
	didMappingOnce     sync.Once
)

// LoadDIDMapping loads the DID replacement mapping from file
func LoadDIDMapping(filepath string) {
	didMappingOnce.Do(func() {
		didMappingInstance = &DIDMapping{}
		if _, err := toml.DecodeFile(filepath, didMappingInstance); err != nil {
			log.Printf("⚠️  Warning: DID mapping file not found or invalid: %v", err)
			log.Printf("    Continuing without DID mapping (all DIDs will be used as-is)")
			// Initialize empty mapping if file doesn't exist
			didMappingInstance.Mapping = make(map[string]string)
		} else {
			log.Printf("✅ DID mapping loaded successfully with %d mappings", len(didMappingInstance.Mapping))
		}
	})
}

// GetDIDMapping returns the global DID mapping instance
func GetDIDMapping() *DIDMapping {
	if didMappingInstance == nil {
		LoadDIDMapping(".config/did_mapping.toml")
	}
	return didMappingInstance
}

// ResolveAdminDID returns the replacement DID if a mapping exists, otherwise returns the original DID
// This function is the main entry point for DID replacement logic
func ResolveAdminDID(incomingDID string) string {
	mapping := GetDIDMapping()
	if mapping == nil || len(mapping.Mapping) == 0 {
		return incomingDID
	}

	if replacementDID, exists := mapping.Mapping[incomingDID]; exists {
		log.Printf("🔄 DID Mapping Applied: %s → %s", incomingDID[:20]+"...", replacementDID[:20]+"...")
		return replacementDID
	}

	// No mapping found, return original DID
	return incomingDID
}

// ValidateDIDMapping ensures all DID mappings are valid:
// 1. Both incoming and replacement DIDs must exist in config.toml
// 2. Both DIDs must use the same port (same node)
func ValidateDIDMapping(cfg *Config, mapping *DIDMapping) error {
	if mapping == nil || len(mapping.Mapping) == 0 {
		log.Println("ℹ️  No DID mappings configured - skipping validation")
		return nil
	}

	log.Println("🔍 Validating DID mappings...")

	for incomingDID, replacementDID := range mapping.Mapping {
		// Check if incoming DID exists in config
		incomingPort, incomingExists := GetPortByDid(cfg, incomingDID)
		if !incomingExists {
			return fmt.Errorf("incoming DID not found in config.toml: %s", incomingDID)
		}

		// Check if replacement DID exists in config
		replacementPort, replacementExists := GetPortByDid(cfg, replacementDID)
		if !replacementExists {
			return fmt.Errorf("replacement DID not found in config.toml: %s", replacementDID)
		}

		// Ensure both DIDs use the same port
		if incomingPort != replacementPort {
			return fmt.Errorf("DID mapping port mismatch: incoming DID %s (port %s) → replacement DID %s (port %s) - both must use the same port",
				incomingDID[:20]+"...", incomingPort, replacementDID[:20]+"...", replacementPort)
		}

		log.Printf("   ✓ Valid mapping: %s → %s (port %s)", incomingDID[:20]+"...", replacementDID[:20]+"...", incomingPort)
	}

	log.Printf("✅ All %d DID mappings validated successfully", len(mapping.Mapping))
	return nil
}
