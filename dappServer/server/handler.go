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
	smartContractHash := config.GetEnvConfig().AddAdminContract //Loading the smart contract hash from config
	if smartContractHash == "" {
		fmt.Println("Smart contract hash is not set in the config")
		return
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
	addAdminContractHash := config.GetEnvConfig().AddAdminContract //Loading the smart contract hash from config
	if addAdminContractHash == "" {
		fmt.Println("addAdminContractHash is not set in the config")
		return
	}
	block := rubix_interaction.GetSmartContractData(addAdminContractHash, url) //config.NodeAddress)
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
