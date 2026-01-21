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
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)
}
