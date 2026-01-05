package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"dapp-server/config"
	rubix "dapp-server/rubix-interaction"

	"github.com/gin-gonic/gin"
)

// Need to check whether all the params here are needed
type ExecuteRequest struct {
	ContractHash  string `json:"contract_hash"`
	ExecutorDid   string `json:"executor_did"`
	ContractInput string `json:"contract_input"`
}

type DeployRequest struct {
	WasmPath    string `json:"wasm_path"`
	LibPath     string `json:"lib_path"`
	DeployerDid string `json:"deployer_did"`
	StatePath   string `json:"state_path"`
}

func APIExecuteContract(c *gin.Context) {
	var req ExecuteRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Printf("Error reading response body: %s\n", err)
		return
	}
	// Load config to get API URL
	cfg, err := config.GetConfig()
	if err != nil {
		return
	}
	nodeName, exist := config.GetNodeNameByDid(cfg, req.ExecutorDid)
	if !exist {
		fmt.Println("Failed to fetch node name from config")
	}
	fmt.Println("The node name is :", nodeName)
	result, err := rubix.Execute(req.ContractHash, req.ExecutorDid, req.ContractInput, nodeName)
	if err != nil {
		fmt.Println("Failed to execute Contract err :", err)
	}
	fmt.Println("The result returned : ", result)
	port, exist := config.GetPortByNodeName(cfg, nodeName)
	if !exist {
		fmt.Println("Failed to fetch port from config")
	}

	url := fmt.Sprintf("http://localhost:%s", port)
	fmt.Println("The url is :", url)
	// Call signature-response API
	_, err = rubix.SignatureResponse(url, result.ContractResult)
	if err != nil {
		fmt.Println("Failed to send signature response:", err)
		return
	}
	// if err := rubix.SignatureResponse(url, result.ContractResult); err != nil {
	// 	return
	// }

	resultFinal := gin.H{
		"message": "DApp executed successfully",
		"data":    result,
	}

	// Return a response
	c.JSON(http.StatusOK, resultFinal)
}

func APIDeployContract(c *gin.Context) {
	var req DeployRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Printf("Error reading response body: %s\n", err)
		return
	}
	// Load config to get API URL
	cfg, err := config.GetConfig()
	if err != nil {
		return
	}
	nodeName, exist := config.GetNodeNameByDid(cfg, req.DeployerDid)
	if !exist {
		fmt.Println("Failed to fetch node name from config")
	}
	result, err := rubix.Deploy(req.WasmPath, req.LibPath, req.DeployerDid, req.StatePath, nodeName)
	if err != nil {
		fmt.Println("Failed to deploy contract err :", err)
	}
	fmt.Println("The result returned : ", result)
	resultFinal := gin.H{
		"message": "Contract Deployed Successfully",
		"data":    result,
	}
	c.JSON(http.StatusOK, resultFinal)
}
