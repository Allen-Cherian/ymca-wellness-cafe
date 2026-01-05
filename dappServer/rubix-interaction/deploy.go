package rubix_interaction

import (
	"bytes"
	"dapp-server/config"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const CONFIG_PATH = ".config/config.toml"

// Deploy handles the contract deployment process
func Deploy(wasmPath string, libPath string, deployerDid string, statePath string, nodeName string) (*DeploymentResult, error) {
	// Load config to get API URL
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	node := cfg.Nodes[nodeName]
	url := fmt.Sprintf("http://localhost:%s", node.Port)
	contractHash, err := generateSmartContract(url, deployerDid, wasmPath, libPath, statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate smart contract: %w", err)
	}

	requestID, err := deploySmartContract(url, contractHash, deployerDid)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy smart contract: %w", err)
	}

	// Call signature-response API
	_, err2 := SignatureResponse(url, requestID)
	if err2 != nil {
		return nil, fmt.Errorf("failed to process signature response: %w", err)
	}
	// RegisterCallBackUrl(contractHash, "8080", "api/call-back-trigger", "20002")
	RegisterCallBackUrl(contractHash, "8080", "api/trigger-contract-2", "20003") //This call back url and port should be accepted as a param
	return &DeploymentResult{
		ContractHash: contractHash,
		Success:      true,
		Message:      "Contract deployed successfully",
	}, nil
}

func generateSmartContract(baseURL, deployerDid, wasmPath, libPath, statePath string) (string, error) {
	// Create a buffer to store the multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add the deployerDid field
	if err := writer.WriteField("did", deployerDid); err != nil {
		return "", fmt.Errorf("failed to add did field: %w", err)
	}

	// Add the WASM file
	wasmFile, err := os.Open(wasmPath)
	if err != nil {
		return "", fmt.Errorf("failed to open WASM file: %w", err)
	}
	defer wasmFile.Close()
	wasmPart, err := writer.CreateFormFile("binaryCodePath", filepath.Base(wasmPath))
	if err != nil {
		return "", fmt.Errorf("failed to create WASM form file: %w", err)
	}
	if _, err := io.Copy(wasmPart, wasmFile); err != nil {
		return "", fmt.Errorf("failed to copy WASM file: %w", err)
	}

	// Add the lib.rs file
	libFile, err := os.Open(libPath)
	if err != nil {
		return "", fmt.Errorf("failed to open lib.rs file: %w", err)
	}
	defer libFile.Close()
	libPart, err := writer.CreateFormFile("rawCodePath", filepath.Base(libPath))
	if err != nil {
		return "", fmt.Errorf("failed to create lib.rs form file: %w", err)
	}
	if _, err := io.Copy(libPart, libFile); err != nil {
		return "", fmt.Errorf("failed to copy lib.rs file: %w", err)
	}

	// Add the state.json file
	stateFile, err := os.Open(statePath)
	if err != nil {
		return "", fmt.Errorf("failed to open state.json file: %w", err)
	}
	defer stateFile.Close()
	statePart, err := writer.CreateFormFile("schemaFilePath", filepath.Base(statePath))
	if err != nil {
		return "", fmt.Errorf("failed to create state.json form file: %w", err)
	}
	if _, err := io.Copy(statePart, stateFile); err != nil {
		return "", fmt.Errorf("failed to copy state.json file: %w", err)
	}

	// Close the multipart writer
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the request
	url := fmt.Sprintf("%s/api/generate-smart-contract", baseURL)
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "multipart/form-data")

	// Send the request
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
	var apiResp SmartContractAPIResponseV1
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response status
	if !apiResp.Status {
		return "", fmt.Errorf("%s", apiResp.Message)
	}

	return apiResp.Result, nil
}

func deploySmartContract(baseURL, contractHash, deployerDid string) (string, error) {
	// Create request body
	requestBody := struct {
		Comment            string  `json:"comment"`
		DeployerAddr       string  `json:"deployerAddr"`
		QuorumType         int     `json:"quorumType"`
		RbtAmount          float64 `json:"rbtAmount"`
		SmartContractToken string  `json:"smartContractToken"`
	}{
		Comment:            "Contract deployment",
		DeployerAddr:       deployerDid,
		QuorumType:         2,
		RbtAmount:          0.001,
		SmartContractToken: contractHash,
	}

	// Marshal request body
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	requestURL, err := url.JoinPath(baseURL, "/api/deploy-smart-contract")
	if err != nil {
		return "", fmt.Errorf("deploy: unable to form request URL")
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
		return "", fmt.Errorf("%s", apiResp.Message)
	}

	return apiResp.Result.Id, nil
}

func SignatureResponse(baseURL, requestID string) (*SmartContractAPIResponseV1, error) {
	// Create request body
	requestBody := struct {
		Id       string `json:"id"`
		Mode     int    `json:"mode"`
		Password string `json:"password"`
	}{
		Id:       requestID,
		Mode:     0,
		Password: "mypassword",
	}

	// Marshal request body
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request URL
	requestURL, err := url.JoinPath(baseURL, "/api/signature-response")
	if err != nil {
		return nil, fmt.Errorf("signature response: unable to form request URL")
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("signature request: failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("signature request: failed to read response: %w", err)
	}

	// Parse response
	var apiResp SmartContractAPIResponseV1
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("signature request: failed to parse response: %w", err)
	}

	// Check response status
	if !apiResp.Status {
		return &apiResp, fmt.Errorf(apiResp.Message)
	}

	return &apiResp, nil
}
