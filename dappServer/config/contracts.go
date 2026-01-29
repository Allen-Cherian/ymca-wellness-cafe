package config

import (
	"fmt"
	"log"
	"sync"

	"github.com/BurntSushi/toml"
)

// AdminContracts holds all smart contract hashes for a single admin
type AdminContracts struct {
	DID                 string `toml:"did"`
	AddActivityContract string `toml:"add_activity_contract"`
	TransferContract    string `toml:"transfer_contract"`
	AddAdminContract    string `toml:"add_admin_contract"`
}

// ContractsConfig holds all admin contract configurations
type ContractsConfig struct {
	Admin1  AdminContracts `toml:"admin1"`
	Admin2  AdminContracts `toml:"admin2"`
	Admin3  AdminContracts `toml:"admin3"`
	Admin4  AdminContracts `toml:"admin4"`
	Admin5  AdminContracts `toml:"admin5"`
	Admin6  AdminContracts `toml:"admin6"`
	Admin7  AdminContracts `toml:"admin7"`
	Admin8  AdminContracts `toml:"admin8"`
	Admin9  AdminContracts `toml:"admin9"`
	Admin10 AdminContracts `toml:"admin10"`
}

var (
	contractsInstance *ContractsConfig
	contractsOnce     sync.Once
)

// LoadContractsConfig loads the contracts configuration file (Singleton)
func LoadContractsConfig(filepath string) error {
	var loadErr error
	contractsOnce.Do(func() {
		contractsInstance = &ContractsConfig{}
		if _, err := toml.DecodeFile(filepath, contractsInstance); err != nil {
			loadErr = fmt.Errorf("error loading contracts config file: %w", err)
			return
		}
		log.Println("Contracts configuration loaded successfully")
	})
	return loadErr
}

// GetContractsConfig returns the global contracts configuration instance
func GetContractsConfig() *ContractsConfig {
	if contractsInstance == nil {
		log.Fatal("Contracts config not loaded. Call LoadContractsConfig() first.")
	}
	return contractsInstance
}

// GetContractForAdmin retrieves the contract hash for a specific admin DID and contract type
func GetContractForAdmin(adminDID string, contractType string) (string, error) {
	if contractsInstance == nil {
		return "", fmt.Errorf("contracts config not loaded. Call LoadContractsConfig() first")
	}

	// Collect all admins into a slice for easy iteration
	allAdmins := []AdminContracts{
		contractsInstance.Admin1,
		contractsInstance.Admin2,
		contractsInstance.Admin3,
		contractsInstance.Admin4,
		contractsInstance.Admin5,
		contractsInstance.Admin6,
		contractsInstance.Admin7,
		contractsInstance.Admin8,
		contractsInstance.Admin9,
		contractsInstance.Admin10,
	}

	// Find the admin with matching DID
	for _, admin := range allAdmins {
		if admin.DID == adminDID {
			// Return the appropriate contract based on type
			switch contractType {
			case "add_activity":
				if admin.AddActivityContract == "" {
					return "", fmt.Errorf("add_activity contract not configured for admin DID: %s", adminDID)
				}
				return admin.AddActivityContract, nil
			case "transfer":
				if admin.TransferContract == "" {
					return "", fmt.Errorf("transfer contract not configured for admin DID: %s", adminDID)
				}
				return admin.TransferContract, nil
			case "add_admin":
				if admin.AddAdminContract == "" {
					return "", fmt.Errorf("add_admin contract not configured for admin DID: %s", adminDID)
				}
				return admin.AddAdminContract, nil
			default:
				return "", fmt.Errorf("unknown contract type: %s", contractType)
			}
		}
	}

	return "", fmt.Errorf("no contracts found for admin DID: %s", adminDID)
}

// GetAllAdminDIDs returns a list of all configured admin DIDs
func GetAllAdminDIDs() []string {
	if contractsInstance == nil {
		return []string{}
	}

	allAdmins := []AdminContracts{
		contractsInstance.Admin1,
		contractsInstance.Admin2,
		contractsInstance.Admin3,
		contractsInstance.Admin4,
		contractsInstance.Admin5,
		contractsInstance.Admin6,
		contractsInstance.Admin7,
		contractsInstance.Admin8,
		contractsInstance.Admin9,
		contractsInstance.Admin10,
	}

	dids := []string{}
	for _, admin := range allAdmins {
		if admin.DID != "" {
			dids = append(dids, admin.DID)
		}
	}

	return dids
}

// ValidateAdminConfig validates that all admins have both node config and contract config
func ValidateAdminConfig() error {
	// Get main config
	cfg, err := GetConfig()
	if err != nil {
		return fmt.Errorf("failed to load node config: %w", err)
	}

	// Get contracts config
	if contractsInstance == nil {
		return fmt.Errorf("contracts config not loaded")
	}

	allAdmins := []AdminContracts{
		contractsInstance.Admin1,
		contractsInstance.Admin2,
		contractsInstance.Admin3,
		contractsInstance.Admin4,
		contractsInstance.Admin5,
		contractsInstance.Admin6,
		contractsInstance.Admin7,
		contractsInstance.Admin8,
		contractsInstance.Admin9,
		contractsInstance.Admin10,
	}

	validatedCount := 0

	// Validate each admin
	for i, admin := range allAdmins {
		if admin.DID == "" || admin.DID == "REPLACE_WITH_ADMIN1_DID" ||
		   admin.DID == "REPLACE_WITH_ADMIN2_DID" || admin.DID == "REPLACE_WITH_ADMIN3_DID" ||
		   admin.DID == "REPLACE_WITH_ADMIN4_DID" || admin.DID == "REPLACE_WITH_ADMIN5_DID" ||
		   admin.DID == "REPLACE_WITH_ADMIN6_DID" || admin.DID == "REPLACE_WITH_ADMIN7_DID" ||
		   admin.DID == "REPLACE_WITH_ADMIN8_DID" || admin.DID == "REPLACE_WITH_ADMIN9_DID" ||
		   admin.DID == "REPLACE_WITH_ADMIN10_DID" ||
		   len(admin.DID) < 10 || !isValidDID(admin.DID) {
			continue // Skip empty, placeholder, or invalid admin entries
		}

		adminNum := i + 1

		// Check if DID exists in node config
		port, exists := GetPortByDid(cfg, admin.DID)
		if !exists {
			return fmt.Errorf("admin%d: DID %s has contracts but no node configuration in config.toml", adminNum, admin.DID)
		}

		// Validate all three contracts are configured
		if admin.AddActivityContract == "" {
			return fmt.Errorf("admin%d: missing add_activity_contract (DID: %s, Port: %s)", adminNum, admin.DID, port)
		}
		if admin.TransferContract == "" {
			return fmt.Errorf("admin%d: missing transfer_contract (DID: %s, Port: %s)", adminNum, admin.DID, port)
		}
		if admin.AddAdminContract == "" {
			return fmt.Errorf("admin%d: missing add_admin_contract (DID: %s, Port: %s)", adminNum, admin.DID, port)
		}

		nodeName, _ := GetNodeNameByDid(cfg, admin.DID)
		fmt.Printf("✅ admin%d validated: %s (port %s)\n", adminNum, nodeName, port)
		validatedCount++
	}

	if validatedCount == 0 {
		log.Println("⚠️  WARNING: No valid admin configurations found in contracts.toml")
		log.Println("⚠️  Please update contracts.toml with actual admin DIDs and contract hashes")
		log.Println("⚠️  System will use fallback contract from environment config")
		return nil // Don't fail, just warn
	}

	fmt.Printf("✅ Total admins validated: %d\n", validatedCount)
	return nil
}

// isValidDID checks if a DID looks valid (basic check)
func isValidDID(did string) bool {
	// Basic validation: should start with "bafyb" for Rubix DIDs
	return len(did) > 20 && (did[:5] == "bafyb" || did[:2] == "Qm")
}
