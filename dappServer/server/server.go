package server

import (
	"dapp-server/config"
	"dapp-server/database"
	rubix_interaction "dapp-server/rubix-interaction"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"dapp-server/wasmbridge"
)

// /home/rubix/Rubix/adminNode
// const SMART_CONTRACT_HASH = "QmZdkRPESpodVMMpYaf6bvPQ2bMckjMzQKaoBaY7C9jjdD"
// const TRANSFER_CONTRACT_HASH = "QmQVbspit7vvT1NNLFDevGxYcEX1PCyU9yrtULXzvvc4wG"

type ContractInputRequest struct {
	Port              string `json:"port"`
	SmartContractHash string `json:"smart_contract_hash"` //port should also be added here, so that the api can understand which node.
}

type RubixResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

type SmartContractDataReply struct {
	RubixResponse
	SCTDataReply []SCTDataReply `json:"SCTDataReply"`
}

type SCTDataReply struct {
	BlockNo            uint64 `json:"BlockNo"`
	BlockId            string `json:"BlockId"`
	SmartContractData  string `json:"SmartContractData"`
	Epoch              uint64 `json:"Epoch"`
	InitiatorSignature string `json:"InitiatorSignature"`
	ExecutorDID        string `json:"ExecutorDID"`
	InitiatorSignData  string `json:"InitiatorSignData"`
}

type AddActivity struct {
	ActivityID   string `json:"activity_id"`
	RewardPoints int    `json:"reward_points"`
}

type AddActivityPayload struct {
	AddActivity AddActivity `json:"add_activity"`
}

type AddActivityRequest struct {
	ActivityID   string `json:"activity_id"`
	RewardPoints int    `json:"reward_points"`
	AdminDID     string `json:"admin_did"`
}

type TransferRewardRequest struct {
	ActivityID []string `json:"activity_id"`
	UserDID    string   `json:"user_did"`
	AdminDID   string   `json:"admin_did"`
}

type Activity struct {
	ActivityID   string `json:"activity_id"`
	BlockHash    string `json:"block_hash"`
	RewardPoints int    `json:"reward_points"`
}
type requestId struct {
	RequestID string `json:"request_id"`
}

type FinalResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func BootupServer() {
	gin.SetMode(gin.ReleaseMode) //
	log.Println("Current Gin Mode:", gin.Mode())

	// Initialize a Gin router
	router := gin.Default()
	log.Println("Current Gin Mode:", gin.Mode())

	// config := GetConfig()

	log.SetFlags(log.LstdFlags)

	// Configure CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders: []string{"Content-Length"},
	}))

	// nftDappCallbackHandler := config.ContractsInfo["nft"].CallBackUrl
	// ftDappCallbackHandler := config.ContractsInfo["ft"].CallBackUrl

	// Define endpoints
	// router.POST(nftDappCallbackHandler, nftDappHandler) // NFT
	router.POST("/api/call-back-trigger", ftDappHandler) // FT
	// router.POST("/api/trigger-contract-2", ftContract2Handler)
	router.POST("/api/deploy-contract", APIDeployContract)
	router.POST("/api/execute-contract", APIExecuteContract)
	router.POST("/api/activity/add", APIAddActivity)
	router.POST("/api/callback/trigger", APICallBackTrigger)
	router.POST("/api/rewards/transfer", APITransferReward)
	router.GET("/api/rewards/status/:transactionID", APIGetTransferStatus)
	router.GET("/api/queue/metrics", APIGetQueueMetrics)
	router.POST("/api/admin/add", APIAddAdmin)
	router.POST("/api/callback/add-admin", APIAddAdminCallBackTrigger)

	// router.GET("/request-status", getRequestStatusHandler)

	// Initialize the queue worker at startup
	GetTransferQueue()

	// Start the server on port 9000
	fmt.Println("🚀 Starting server on port 9000...")
	err := router.Run(":9000")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
func APITransferReward(c *gin.Context) {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("🎯 APITransferReward triggered (QUEUE MODE)")
	fmt.Println("═══════════════════════════════════════════════════════════")

	var req TransferRewardRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Printf("❌ Error reading request body: %s\n", err)
		return
	}

	fmt.Printf("📝 Request: user=%s, admin=%s, activities=%v\n", req.UserDID, req.AdminDID, req.ActivityID)

	// ═══════════════════════════════════════════════════════════
	// Step 1: Input Validation
	// ═══════════════════════════════════════════════════════════

	// Validate UserDID
	if req.UserDID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": "user_did is required and cannot be empty",
		})
		fmt.Println("❌ Validation failed: user_did is empty")
		return
	}

	// Validate AdminDID
	if req.AdminDID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": "admin_did is required and cannot be empty",
		})
		fmt.Println("❌ Validation failed: admin_did is empty")
		return
	}

	// Validate ActivityID array
	if len(req.ActivityID) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": "activity_id is required and must contain at least one activity",
		})
		fmt.Println("❌ Validation failed: activity_id is empty")
		return
	}

	// Validate ActivityID elements (no empty strings)
	for i, activityID := range req.ActivityID {
		if activityID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Validation failed",
				"message": fmt.Sprintf("activity_id[%d] cannot be empty", i),
			})
			fmt.Printf("❌ Validation failed: activity_id[%d] is empty\n", i)
			return
		}
	}

	// Basic DID format validation (DIDs should start with "bafyb")
	if !strings.HasPrefix(req.UserDID, "bafyb") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": "user_did has invalid format (should start with 'bafyb')",
		})
		fmt.Printf("❌ Validation failed: user_did has invalid format: %s\n", req.UserDID)
		return
	}

	if !strings.HasPrefix(req.AdminDID, "bafyb") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": "admin_did has invalid format (should start with 'bafyb')",
		})
		fmt.Printf("❌ Validation failed: admin_did has invalid format: %s\n", req.AdminDID)
		return
	}

	// Validate AdminDID exists in config
	cfg, err := config.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Configuration error",
			"message": "Failed to load server configuration",
		})
		fmt.Printf("❌ Failed to load config: %v\n", err)
		return
	}

	_, adminExists := config.GetPortByDid(cfg, req.AdminDID)
	if !adminExists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": "admin_did not found in configured nodes",
			"details": "The specified admin_did is not registered in the system",
		})
		fmt.Printf("❌ Validation failed: admin_did not found in config: %s\n", req.AdminDID)
		return
	}

	fmt.Println("✅ Input validation passed")

	// ═══════════════════════════════════════════════════════════
	// Step 2: Generate UUID immediately
	// ═══════════════════════════════════════════════════════════
	requestID := uuid.New().String()
	fmt.Printf("🆔 Generated request_id: %s\n", requestID)

	// ═══════════════════════════════════════════════════════════
	// Step 3: Validate configuration
	// ═══════════════════════════════════════════════════════════
	transferContractHash := config.GetEnvConfig().TransferContract
	if transferContractHash == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Transfer contract hash not configured",
		})
		fmt.Println("❌ transferContractHash is not set in the config")
		return
	}

	// ═══════════════════════════════════════════════════════════
	// Step 4: Create database record immediately with status "queued"
	// ═══════════════════════════════════════════════════════════
	rewardPoints := len(req.ActivityID)
	now := time.Now()

	err = database.CreateTransferStatus(&database.TransferStatus{
		RequestID:      requestID,
		BlockchainTxID: "", // Will be filled by worker
		BlockId:        "", // Will be filled by worker
		ActivityIDs:    req.ActivityID,
		UserDID:        req.UserDID,
		AdminDID:       req.AdminDID,
		RewardPoints:   rewardPoints,
		Status:         "queued",
		Message:        "Transfer request queued for processing",
		ContractHash:   transferContractHash,
		QueuedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create transfer record",
			"details": err.Error(),
		})
		fmt.Printf("❌ Failed to create transfer record: %v\n", err)
		return
	}

	fmt.Printf("✅ Database record created: status=queued\n")

	// ═══════════════════════════════════════════════════════════
	// Step 5: Add to queue
	// ═══════════════════════════════════════════════════════════
	queue := GetTransferQueue()
	err = queue.Enqueue(requestID, req)

	if err != nil {
		// Queue is full - update database and return error
		database.UpdateTransferStatus(requestID, map[string]interface{}{
			"status":  "rejected",
			"message": "System at capacity, please retry later",
		})

		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "rejected",
			"message": "System at capacity, please retry in a few minutes",
			"error":   err.Error(),
		})
		fmt.Printf("❌ Queue is full: %v\n", err)
		return
	}

	// ═══════════════════════════════════════════════════════════
	// Step 6: Return immediately with request_id
	// ═══════════════════════════════════════════════════════════
	queueSize := queue.GetQueueSize()

	requestIdtobeSent := requestId{
		RequestID: requestID,
	}

	response := FinalResponse{
		Status:  true,
		Message: "Transfer request queued for processing",
		Result:  requestIdtobeSent,
	}
	// response := gin.H{
	// 	"status":  true,
	// 	"message": "Transfer request queued for processing",
	// 	"result":  requestIdtobeSent,
	// }

	fmt.Printf("🔍 [DEBUG] Response payload: %+v\n", response)

	c.JSON(http.StatusAccepted, response)

	fmt.Printf("✅ Response sent: request_id=%s, queue_position=%d\n", requestID, queueSize)
	fmt.Println("═══════════════════════════════════════════════════════════")
}

// APIGetTransferStatus retrieves the status of a reward transfer by transaction ID
func APIGetTransferStatus(c *gin.Context) {
	transactionID := c.Param("transactionID")
	fmt.Printf("APIGetTransferStatus called for transaction: %s\n", transactionID)

	if transactionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Transaction ID is required",
		})
		return
	}

	// Retrieve status from database
	status, err := database.GetTransferStatus(transactionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  false,
			"message": "Transfer not found",
			"error":   err.Error(),
		})
		fmt.Printf("Transfer not found: %v\n", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   status,
	})
}

// APIGetQueueMetrics returns queue statistics
func APIGetQueueMetrics(c *gin.Context) {
	queue := GetTransferQueue()
	queueSize := queue.GetQueueSize()

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data": gin.H{
			"queue_size":         queueSize,
			"estimated_wait_sec": queueSize * 8,
			"capacity":           1000,
			"available_slots":    1000 - queueSize,
		},
	})
}

func APIAddActivity(c *gin.Context) {
	fmt.Println("APIAddActivity triggered")
	var req AddActivityRequest
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
	nodePort, exists := config.GetPortByDid(cfg, req.AdminDID)
	if !exists {
		fmt.Println("failed to get node port: not found")
		return
	}
	fmt.Println("The node port is:", nodePort)
	url := fmt.Sprintf("http://localhost:%s", nodePort)
	fmt.Println("The url is :", url)
	contractMsg := fmt.Sprintf(`{"add_activity": {"activity_id":"%s","reward_points":%d}}`, req.ActivityID, req.RewardPoints)
	fmt.Println("The contract message is:", contractMsg)
	smartContractHash := config.GetEnvConfig().AddActivityContract //Loading the smart contract hash from config
	if smartContractHash == "" {
		fmt.Println("Smart contract hash is not set in the config")
		return
	}
	smartContractResponse, err := rubix_interaction.ExecuteSmartContract(url, smartContractHash, req.AdminDID, contractMsg)
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
	addActivityContractHash := config.GetEnvConfig().AddActivityContract //Loading the smart contract hash from config
	if addActivityContractHash == "" {
		fmt.Println("addActivityContractHash is not set in the config")
		return
	}
	block := rubix_interaction.GetSmartContractData(addActivityContractHash, url) //config.NodeAddress)
	if block == nil {
		fmt.Println("Unable to fetch latest smart contract data")
		return
	}
	resultFinal := gin.H{
		"message": "Activity added to smart contract tokenchain",
		"data":    string(block),
	}

	// Return a response
	c.JSON(http.StatusOK, resultFinal)

}

func APICallBackTrigger(c *gin.Context) {
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
	var payload AddActivityPayload
	err = json.Unmarshal([]byte(relevantBlock.SmartContractData), &payload)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}
	registry := wasmbridge.NewHostFunctionRegistry()

	// Create your custom host function
	// TODO: Fix WriteToJsonFile to work with local wasmbridge (utils dependency issue)
	// registry.Register(rubix_interaction.NewWriteToJsonFile())
	hostFunction := registry.GetHostFunctions()
	fmt.Println("Host function is :", hostFunction)
	wasmPath, err := getWasmContractPath(smartContractHash, req.Port)
	if err != nil {
		fmt.Println("Failed to get wasm path")
	}
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
	contractInput := fmt.Sprintf(`{"add_activity": {"activity_id":"%s","reward_points":%d,"block_hash":"%s"}}`, payload.AddActivity.ActivityID, payload.AddActivity.RewardPoints, relevantBlock.BlockId)
	fmt.Println("The contract input is :", contractInput)
	result, err := wasmModule.CallFunction(contractInput)
	if err != nil {
		log.Printf("Failed to call WASM function: %v", err)
		return
	}
	fmt.Println("The result is :", result)
}

// Function to read BlockId from a JSON file
func getBlockIDFromJSONFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var data struct {
		BlockID      string `json:"block_id"`
		ActivityID   string `json:"activity_id"`
		RewardPoints int    `json:"reward_points"`
	}

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		return "", err
	}

	return data.BlockID, nil
}

// func getSCTDataAfterBlockID(sctDataReplies []SCTDataReply, blockID string) []SCTDataReply {
// 	var result []SCTDataReply
// 	found := false

// 	for _, data := range sctDataReplies {
// 		if found {
// 			result = append(result, data)
// 		} else if data.BlockId == blockID {
// 			found = true
// 		}
// 	}

// 	return result
// }

func getNextSCTDataAfterBlockID(sctDataReplies []SCTDataReply, blockID string) *SCTDataReply {
	for i, data := range sctDataReplies {
		if data.BlockId == blockID && i+1 < len(sctDataReplies) {
			return &sctDataReplies[i+1] // Return the next entry
		}
	}
	return nil // Return nil if no matching block or no next entry
}

// extractTransactionID extracts the transaction ID from FT transfer message
// Message format: "FT Transfer finished successfully in 11.071221792s with trnxid db7e190a5fdb1cc9109d00287cb8e2c8d1457cecf4129fbf6b49fdaf8036b2b2"
func extractTransactionID(message string) string {
	// Find "trnxid " in the message
	const trnxidPrefix = "trnxid "
	idx := strings.Index(message, trnxidPrefix)
	if idx == -1 {
		return "" // trnxid not found
	}

	// Extract everything after "trnxid "
	txID := message[idx+len(trnxidPrefix):]

	// Take only the transaction ID (stops at space or end of string)
	if spaceIdx := strings.Index(txID, " "); spaceIdx != -1 {
		txID = txID[:spaceIdx]
	}

	return strings.TrimSpace(txID)
}

// Handler function for /callback/nft
func ftDappHandler(c *gin.Context) {
	var req ContractInputRequest
	fmt.Println("==========================================")
	fmt.Println("🔔 ftDappHandler TRIGGERED - Callback received!")
	fmt.Println("==========================================")
	fmt.Printf("Timestamp: %s\n", time.Now().Format(time.RFC3339))

	// Add delay to give API time to complete setup (transactionID, DB creation, registration)
	fmt.Println("⏳ Waiting 5 seconds for API setup to complete...")
	time.Sleep(5 * time.Second)
	fmt.Println("✅ Delay complete, processing callback...")
	// cfg, err := config.GetConfig()
	// if err != nil {
	// 	fmt.Println("failed to load config: %w", err)
	// }
	// config.GetNodeNameByPort(cfg, req.Port)
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Printf("Error reading response body: %s\n", err)
		return
	}
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
	fmt.Println("Data reply in runDappHandler", dataReply)
	smartContractData := dataReply.SCTDataReply
	var relevantData string
	for _, reply := range smartContractData {
		fmt.Println("SmartContractData:", reply.SmartContractData)
		relevantData = reply.SmartContractData
	}
	fmt.Println("The relevant data is :", relevantData)
	var inputMap map[string]interface{}
	err1 := json.Unmarshal([]byte(relevantData), &inputMap)
	if err1 != nil {
		fmt.Println("Error unmarshalling input map:", err1)
		return
	}
	if len(inputMap) != 1 {
		return
	}

	var funcName string
	var inputStruct interface{}
	for key, value := range inputMap {
		funcName = key
		inputStruct = value
	}
	fmt.Println("The function name extracted =", funcName)
	fmt.Println("The inputStruct Value :", inputStruct)

	hostFnRegistry := wasmbridge.NewHostFunctionRegistry()
	wasmPath, err := getWasmContractPath(smartContractHash, req.Port)
	if err != nil {
		fmt.Println("Failed to get wasm path")
	}
	// Initialize the WASM module

	wasmModule, err := wasmbridge.NewWasmModule(
		wasmPath,
		hostFnRegistry,
		wasmbridge.WithRubixNodeAddress(url), //config.NodeAddress),
		wasmbridge.WithQuorumType(2),
	)
	if err != nil {
		log.Printf("Failed to initialize WASM module: %v", err)
		return
	}

	executionResult, errExecuteContract := executeAndGetContractResult(wasmModule, relevantData)
	fmt.Println("----------- FT Execution Result: ", executionResult)
	if errExecuteContract != nil {
		fmt.Println("The executionResult is ", executionResult)
		return
	}

	var response RubixResponse

	// Convert JSON string to struct
	if executionResult == "success" {
		response = RubixResponse{Status: true, Message: "FT Transferred Succesfully"}
	} else {
		err = json.Unmarshal([]byte(executionResult), &response)
		if err != nil {
			log.Printf("Error parsing JSON: %v", err)
			return
		}
	}

	// Extract transaction ID from message
	// Message format: "FT Transfer finished successfully in 11.071221792s with trnxid db7e190a5fdb1cc9109d00287cb8e2c8d1457cecf4129fbf6b49fdaf8036b2b2"
	ftTransferTxID := extractTransactionID(response.Message)
	if ftTransferTxID != "" {
		fmt.Printf("📝 Extracted FT Transfer TxID: %s\n", ftTransferTxID)
	}

	// Extract BlockId from the latest block for callback correlation
	var latestBlockId string
	if len(smartContractData) > 0 {
		latestBlockId = smartContractData[len(smartContractData)-1].BlockId
		fmt.Println("==========================================")
		fmt.Printf("🔑 ftDappHandler: Extracted BlockId: %s\n", latestBlockId)
		fmt.Printf("📊 Total blocks in data: %d\n", len(smartContractData))
		fmt.Println("==========================================")
	} else {
		fmt.Println("⚠️  ftDappHandler: No blocks found in smart contract data!")
	}

	// Signal the waiting APITransferReward through the TransferManager
	if latestBlockId != "" {
		fmt.Println("==========================================")
		fmt.Printf("⏰ [%s] Attempting to send callback response for BlockId: %s\n", time.Now().Format("15:04:05.000"), latestBlockId)
		fmt.Println("==========================================")

		manager := GetTransferManager()
		callbackResponse := CallbackResponse{
			Success:        response.Status,
			Message:        response.Message,
			Data:           response.Result,
			BlockId:        latestBlockId,
			ContractData:   relevantData,
			FTTransferTxID: ftTransferTxID, // Store the extracted transaction ID
		}

		if !response.Status {
			callbackResponse.Error = fmt.Sprintf("Contract execution failed: %v", response.Message)
		}

		// This will update DB and notify waiting channel (if any)
		success := manager.SendCallbackResponse(latestBlockId, callbackResponse)
		fmt.Println("==========================================")
		if success {
			fmt.Printf("✅ [%s] ftDappHandler: Successfully notified pending request for BlockId: %s\n", time.Now().Format("15:04:05.000"), latestBlockId)
		} else {
			fmt.Printf("⚠️  [%s] ftDappHandler: No pending request found for BlockId: %s (checking DB fallback)\n", time.Now().Format("15:04:05.000"), latestBlockId)
		}
		fmt.Println("==========================================")
	}

	resultFinal := gin.H{
		"message": "DApp executed successfully",
		"data":    response,
	}

	// Return a response
	c.JSON(http.StatusOK, resultFinal)
}

// Handler function for /callback/nft
func ftContract2Handler(c *gin.Context) {
	var req ContractInputRequest
	fmt.Println("Handler trggered")
	// cfg, err := config.GetConfig()
	// if err != nil {
	// 	fmt.Println("failed to load config: %w", err)
	// }

	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Printf("Error reading response body: %s\n", err)
		return
	}
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
	fmt.Println("Data reply in runDappHandler", dataReply)
	smartContractData := dataReply.SCTDataReply
	var relevantData string
	for _, reply := range smartContractData {
		fmt.Println("SmartContractData:", reply.SmartContractData)
		relevantData = reply.SmartContractData
	}
	var inputMap map[string]interface{}
	err1 := json.Unmarshal([]byte(relevantData), &inputMap)
	if err1 != nil {
		return
	}
	if len(inputMap) != 1 {
		return
	}

	var funcName string
	var inputStruct interface{}
	for key, value := range inputMap {
		funcName = key
		inputStruct = value
	}
	fmt.Println("The function name extracted =", funcName)
	fmt.Println("The inputStruct Value :", inputStruct)

	hostFnRegistry := wasmbridge.NewHostFunctionRegistry()
	wasmPath, err := getWasmContractPath(smartContractHash, req.Port)
	if err != nil {
		fmt.Println("Failed to get wasm path")
	}
	// Initialize the WASM module

	wasmModule, err := wasmbridge.NewWasmModule(
		wasmPath,
		hostFnRegistry,
		wasmbridge.WithRubixNodeAddress(url), //config.NodeAddress),
		wasmbridge.WithQuorumType(2),
	)
	if err != nil {
		log.Printf("Failed to initialize WASM module: %v", err)
		return
	}

	executionResult, errExecuteContract := executeAndGetContractResult(wasmModule, relevantData)
	fmt.Println("----------- FT Execution Result: ", executionResult)
	if errExecuteContract != nil {
		fmt.Println("The executionResult is ", executionResult)
		return
	}

	var response RubixResponse

	// Convert JSON string to struct
	if executionResult == "success" {
		response = RubixResponse{Status: true, Message: "FT Transferred Succesfully"}
	} else {
		err = json.Unmarshal([]byte(executionResult), &response)
		if err != nil {
			log.Printf("Error parsing JSON: %v", err)
			return
		}
	}

	resultFinal := gin.H{
		"message": "DApp executed successfully",
		"data":    response,
	}

	// Return a response
	c.JSON(http.StatusOK, resultFinal)
}

func getWasmContractPath(contractHash, port string) (string, error) {
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	fmt.Println("The current working Directory is:", currentWorkingDir)
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println("Failed to get config file")
	}
	path, exists := config.GetPathByPort(cfg, port)
	if !exists {
		fmt.Println("Failed to get path by port")
		return "", fmt.Errorf("failed to get path by port: %s", port)
	}
	nodeName, exists := config.GetNodeNameByPort(cfg, port)
	if !exists {
		fmt.Println("Failed to get node name associated with the port", port)
	}
	// Construct the path in a cleaner way
	contractDir := filepath.Join(path, nodeName, "SmartContract", contractHash)
	fmt.Println("The contract directory is:", contractDir)

	entries, err := os.ReadDir(contractDir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".wasm") {
			return filepath.Join(contractDir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("no wasm contract found in directory: %v", contractDir)
}

func executeAndGetContractResult(wasmModule *wasmbridge.WasmModule, contractInput string) (string, error) {
	// Call the function
	contractResult, err := wasmModule.CallFunction(contractInput)
	if err != nil {
		return "", fmt.Errorf("function call failed: %v", err)
	}

	return contractResult, nil
}

// GetRewardPoints takes a JSON file path and an activity ID and returns the reward points for that activity.
func GetRewardPoints(filePath string, activityID string) (int, error) {
	// Read the JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	// Parse the JSON into a slice of Activity structs
	var activities []Activity
	err = json.Unmarshal(data, &activities)
	if err != nil {
		return 0, err
	}

	// Search for the activity ID and return the reward points
	for _, activity := range activities {
		if activity.ActivityID == activityID {
			return activity.RewardPoints, nil
		}
	}

	return 0, errors.New("activity ID not found")
}
