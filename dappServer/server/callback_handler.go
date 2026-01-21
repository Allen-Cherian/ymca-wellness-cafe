package server

import (
	rubix_interaction "dapp-server/rubix-interaction"
	"dapp-server/wasmbridge"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// http://localhost:9000/api/callback/add-admin
type AddAdmin struct {
	AdminDID string `json:"admin_did"`
}

type Payload struct {
	AddAdmin AddAdmin `json:"add_admin"`
}

func APIAddAdminCallBackTrigger(c *gin.Context) {
	var req ContractInputRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Printf("Error reading response body: %s\n", err)
		return
	}
	fmt.Println("The request body is:", req)
	url := fmt.Sprintf("http://localhost:%s", req.Port)
	fmt.Println("The url is :", url)

	// // config := GetConfig()
	smartContractHash := req.SmartContractHash
	fmt.Println("Received Smart Contract hash: ", smartContractHash)

	smartContractTokenData := rubix_interaction.GetSmartContractData(smartContractHash, url) //config.NodeAddress)
	if smartContractTokenData == nil {
		fmt.Println("Unable to fetch latest smart contract data")
		return
	}

	fmt.Println("Smart Contract Token Data :", string(smartContractTokenData))

	var dataReply SmartContractDataReply

	if err := json.Unmarshal(smartContractTokenData, &dataReply); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Data reply in APICallBackTrigger", dataReply)
	smartContractData := dataReply.SCTDataReply
	var relevantBlock *SCTDataReply

	// var blockId string
	var blockNo uint64
	for _, data := range smartContractData {
		relevantBlock = &data // Assuming you want the last block
		fmt.Println("The relevant block is :", relevantBlock)
		// blockId = data.BlockId
		blockNo = data.BlockNo
	}
	if blockNo == 0 {
		fmt.Println("The block number is zero which is the genesis block")
		return
	}
	// var payload Payload
	// err = json.Unmarshal([]byte(relevantBlock.SmartContractData), &payload)
	// if err != nil {
	// 	fmt.Println("Error unmarshaling JSON:", err)
	// 	return
	// }
	fmt.Println("Smart Contract Data:", relevantBlock.SmartContractData)
	registry := wasmbridge.NewHostFunctionRegistry()

	// Create your custom host function
	// TODO: Re-enable after fixing WriteToJsonFile if needed for activity/admin contracts
	registry.Register(rubix_interaction.NewWriteToJsonFile())
	hostFunction := registry.GetHostFunctions()
	fmt.Println("Host function is :", hostFunction)
	wasmPath, err := getWasmContractPath(smartContractHash, req.Port)
	if err != nil {
		fmt.Println("Failed to get wasm path")
	}
	fmt.Println("The wasm path is :", wasmPath)
	wasmModule, err := wasmbridge.NewWasmModule(
		wasmPath,
		registry,
		// wasmbridge.WithRubixNodeAddress("http://localhost:20002"), //config.NodeAddress),
		// wasmbridge.WithQuorumType(2),
	)
	if err != nil {
		log.Printf("Failed to initialize WASM module: %v", err)
		return
	}
	// contractInput := fmt.Sprintf(`{"add_activity": {"activity_id":"%s","reward_points":%d,"block_hash":"%s"}}`, parsedData.ActivityID, parsedData.RewardPoints, relevantBlock.BlockId)
	// fmt.Println("The contract input is :", contractInput)
	fmt.Println("The smart contract data is :", relevantBlock.SmartContractData)
	result, err := wasmModule.CallFunction(relevantBlock.SmartContractData)
	if err != nil {
		log.Printf("Failed to call WASM function: %v", err)
		return
	}
	fmt.Println("The result is :", result)
}
