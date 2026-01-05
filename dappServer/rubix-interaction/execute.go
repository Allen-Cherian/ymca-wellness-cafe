package rubix_interaction

import (
	"bytes"
	"dapp-server/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Execute handles the contract execution process
func Execute(
	contractHash string, executorDid string,
	contractInput string, nodeName string,
) (*ExecutionResult, error) {
	// Load config to get API URL
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	port, exists := config.GetPortByNodeName(cfg, nodeName)
	if !exists {
		fmt.Println("failed to find the node in config")
	}
	fmt.Println("port :", port)
	url := fmt.Sprintf("http://localhost:%s", port)
	fmt.Println("The url is :", url)
	requestID, err := ExecuteSmartContract(url, contractHash, executorDid, contractInput)
	if err != nil {
		return nil, fmt.Errorf("failed to execute smart contract: %w", err)
	}

	contractResponse, err := SignatureResponse(url, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to process signature response: %w", err)
	}
	// Call signature-response API
	// if err := SignatureResponse(url, requestID); err != nil {
	// 	return nil, fmt.Errorf("failed to process signature response: %w", err)
	// }

	return &ExecutionResult{
		ContractResult: contractResponse.Result,
		Success:        true,
		Message:        contractResponse.Message,
	}, nil
}

func ExecuteSmartContract(baseURL, contractHash, executorDid, contractMsg string) (string, error) {
	// Create request body
	requestBody := struct {
		Comment            string `json:"comment"`
		ExecutorAddr       string `json:"executorAddr"`
		QuorumType         int    `json:"quorumType"`
		SmartContractData  string `json:"smartContractData"`
		SmartContractToken string `json:"smartContractToken"`
	}{
		Comment:            "Contract execution",
		ExecutorAddr:       executorDid,
		QuorumType:         2,
		SmartContractData:  contractMsg,
		SmartContractToken: contractHash,
	}

	// Marshal request body
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	requestURL, err := url.JoinPath(baseURL, "/api/execute-smart-contract")
	if err != nil {
		return "", fmt.Errorf("execute: unable to form request URL")
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var apiResp SmartContractAPIResponseV2
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response status
	if !apiResp.Status {
		return "", fmt.Errorf(apiResp.Message)
	}

	return apiResp.Result.Id, nil
}

func getSmartContractChainBlocks(baseURL string, contractHash string, onlyLatest bool) ([]*SmartContractBlock, error) {
	// Create request body
	requestBody := struct {
		Latest bool   `json:"latest"`
		Token  string `json:"token"`
	}{
		Latest: onlyLatest,
		Token:  contractHash,
	}

	// Marshal request body
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	requestURL, err := url.JoinPath(baseURL, "/api/get-smart-contract-token-chain-data")
	if err != nil {
		return nil, fmt.Errorf("execute: unable to form request URL")
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var apiResp struct {
		Status              bool                  `json:"status"`
		Message             string                `json:"message"`
		SmartContractBlocks []*SmartContractBlock `json:"SCDataReply"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response status
	if !apiResp.Status {
		return nil, fmt.Errorf(apiResp.Message)
	}

	if len(apiResp.SmartContractBlocks) == 0 {
		return nil, fmt.Errorf("unable to fetch blocks for smart contract token : %v", contractHash)
	}

	return apiResp.SmartContractBlocks, nil
}

func getWasmContractPath(contractHash string) (string, error) {
	currentWorkingDir, err := os.Getwd()
	fmt.Println("The current working Directory is : ", currentWorkingDir)
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	// Here this path should be dynamic
	contractDir := filepath.Join(currentWorkingDir, "rubix-nodes/node2/SmartContract", contractHash)

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
