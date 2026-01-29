package server

import (
	"dapp-server/config"
	rubix_interaction "dapp-server/rubix-interaction"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AddAdminRequest struct {
	NewAdminDID      string `json:"new_admin_did"`
	ExistingAdminDID string `json:"existing_admin_did"`
}

func APIAddAdmin(c *gin.Context) {
	fmt.Println("APIAddAdmin triggered")
	var req AddAdminRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Printf("Error reading response body: %s\n", err)
		return
	}
	fmt.Println("The request body is:", req)
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println("failed to load config: %w", err)
	}
	nodePort, exists := config.GetPortByDid(cfg, req.ExistingAdminDID)
	if !exists {
		fmt.Println("failed to get node port: not found")
		return
	}
	fmt.Println("The node port is:", nodePort)
	url := fmt.Sprintf("http://localhost:%s", nodePort)
	fmt.Println("The url is :", url)
	contractMsg := fmt.Sprintf(`{"add_admin": {"admin_did":"%s"}}`, req.NewAdminDID)
	fmt.Println("The contract message is:", contractMsg)

	// Try to get admin-specific contract first, fallback to global config
	smartContractHash, err := config.GetContractForAdmin(req.ExistingAdminDID, "add_admin")
	if err != nil {
		// Fallback to global contract from environment config
		fmt.Printf("⚠️  Using fallback add_admin contract: %v\n", err)
		smartContractHash = config.GetEnvConfig().AddAdminContract
		if smartContractHash == "" {
			fmt.Println("❌ Smart contract hash is not set in the config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Add admin contract not configured"})
			return
		}
	} else {
		fmt.Printf("✅ Using admin-specific add_admin contract: %s\n", smartContractHash)
	}
	smartContractResponse, err := rubix_interaction.ExecuteSmartContract(url, smartContractHash, req.ExistingAdminDID, contractMsg)
	if err != nil {
		fmt.Println("failed to execute smart contract:", err)
		return
	}
	fmt.Println("Smart contract response:", smartContractResponse)
	_, err = rubix_interaction.SignatureResponse(url, smartContractResponse)
	if err != nil {
		fmt.Println("failed to send signature response:", err)
		return
	}
	fmt.Println("Signature response sent successfully")

	// Use the same contract hash from above (already validated)
	block := rubix_interaction.GetSmartContractData(smartContractHash, url) //config.NodeAddress)
	if block == nil {
		fmt.Println("Unable to fetch latest smart contract data")
		return
	}
	resultFinal := gin.H{
		"message": "Admin added to smart contract tokenchain",
		"data":    string(block),
	}

	// Return a response
	c.JSON(http.StatusOK, resultFinal)

}
