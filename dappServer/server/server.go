package server

import (
	"dapp-server/config"
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

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	wasmbridge "github.com/rubixchain/rubix-wasm/go-wasm-bridge"
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
	ActivityID string `json:"activity_id"`
	UserDID    string `json:"user_did"`
	AdminDID   string `json:"admin_did"`
}

type Activity struct {
	ActivityID   string `json:"activity_id"`
	BlockHash    string `json:"block_hash"`
	RewardPoints int    `json:"reward_points"`
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
	router.POST("/api/admin/add", APIAddAdmin)
	router.POST("/api/callback/add-admin", APIAddAdminCallBackTrigger)

	// router.GET("/request-status", getRequestStatusHandler)

	// Start the server on port 9000
	router.Run(":9000")
}
func APITransferReward(c *gin.Context) {
	fmt.Println("APITransferReward triggered")
	var req TransferRewardRequest
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
	filePath := config.GetEnvConfig().ActivityUpdatePath
	rewardPoints, err := GetRewardPoints(filePath, req.ActivityID)
	if err != nil {
		fmt.Println("Failed to get reward points")
		return
	}
	// contractMsg := fmt.Sprintf(`{"activity_id":"%s","reward_points":%d,"user_did":%s,"admin_did":%s}`, req.ActivityID, rewardPoints, req.UserDID, req.AdminDID)
	contractMsg := fmt.Sprintf(`{"transfer_sample_ft":{"name": "rubix1", "ft_info": {"comment":"Transfer of reward via contract","ft_count":%f,"ft_name":"ytoken","sender": "%s","creatorDID": "%s", "receiver": "%s"}}}`, float64(rewardPoints), req.AdminDID, req.AdminDID, req.UserDID)
	fmt.Println("The contract message is:", contractMsg)
	transferContractHash := config.GetEnvConfig().TransferContract //Loading the smart contract hash from config
	if transferContractHash == "" {
		fmt.Println("transferContractHash is not set in the config")
		return
	}
	smartContractResponse, err := rubix_interaction.ExecuteSmartContract(url, transferContractHash, req.AdminDID, contractMsg)

	if err != nil {
		fmt.Println("failed to execute smart contract:", err)
		return
	}
	fmt.Println("Smart contract response:", smartContractResponse)
	response := rubix_interaction.SignatureResponse(url, smartContractResponse)
	if err != nil {
		fmt.Println("failed to send signature response:", err)
		return
	}
	fmt.Println("Signature response sent successfully")
	var data interface{}
	if response == nil {
		data = gin.H{
			"rewards awarded": float64(rewardPoints),
			"activity_id":     req.ActivityID,
		}
	} else {
		data = response
	}
	resultFinal := gin.H{
		"message": "Reward Transfer initated succesfully",
		"data":    data,
	}

	// Return a response
	c.JSON(http.StatusOK, resultFinal)

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
	response := rubix_interaction.SignatureResponse(url, smartContractResponse)
	if response != nil {
		fmt.Println("failed to send signature response:", response)
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
	registry.Register(rubix_interaction.NewWriteToJsonFile())
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

// Handler function for /callback/nft
func ftDappHandler(c *gin.Context) {
	var req ContractInputRequest
	fmt.Println("Handler trggered")
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
