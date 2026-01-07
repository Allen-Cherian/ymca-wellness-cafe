package server

import (
	rubix_interaction "dapp-server/rubix-interaction"
	"encoding/json"
	"fmt"
)

// ExtractLatestBlockId fetches smart contract data and extracts the latest BlockId
// Note: GetSmartContractData already passes "latest: true", so it returns only the latest block
func ExtractLatestBlockId(contractHash string, nodeURL string) (string, error) {
	// Fetch smart contract data (returns latest block only)
	// Note: Caller should ensure sufficient delay after SignatureResponse
	smartContractData := rubix_interaction.GetSmartContractData(contractHash, nodeURL)
	if smartContractData == nil {
		return "", fmt.Errorf("failed to fetch smart contract data")
	}

	// Parse the response
	var dataReply SmartContractDataReply
	if err := json.Unmarshal(smartContractData, &dataReply); err != nil {
		return "", fmt.Errorf("failed to unmarshal smart contract data: %w", err)
	}

	// Check if we got any blocks
	if len(dataReply.SCTDataReply) == 0 {
		return "", fmt.Errorf("no blocks found in smart contract data")
	}

	// Get the latest block (since latest: true, this should be the most recent)
	latestBlock := dataReply.SCTDataReply[len(dataReply.SCTDataReply)-1]
	if latestBlock.BlockId == "" {
		return "", fmt.Errorf("block ID is empty")
	}

	fmt.Printf("Successfully extracted BlockId: %s\n", latestBlock.BlockId)
	return latestBlock.BlockId, nil
}
