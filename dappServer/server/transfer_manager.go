package server

import (
	"dapp-server/database"
	"fmt"
	"sync"
	"time"
)

// CallbackResponse represents the result from ftDappHandler callback
type CallbackResponse struct {
	Success      bool        `json:"success"`
	Message      string      `json:"message"`
	Data         interface{} `json:"data"`
	Error        string      `json:"error,omitempty"`
	BlockId      string      `json:"block_id"`
	ContractData string      `json:"contract_data,omitempty"`
}

// PendingRequest holds the channel for a request waiting for callback
type PendingRequest struct {
	TransactionID string
	ResponseChan  chan CallbackResponse
	CreatedAt     time.Time
}

// TransferManager manages both persistent status (DB) and pending channels (in-memory)
type TransferManager struct {
	// Temporary storage for pending requests: blockId -> channel
	pendingByBlockId map[string]*PendingRequest
	pendingMu        sync.RWMutex
}

var (
	transferManager     *TransferManager
	transferManagerOnce sync.Once
)

// GetTransferManager returns the singleton instance
func GetTransferManager() *TransferManager {
	transferManagerOnce.Do(func() {
		transferManager = &TransferManager{
			pendingByBlockId: make(map[string]*PendingRequest),
		}
		// Start cleanup goroutine
		go transferManager.cleanupStaleRequests()
	})
	return transferManager
}

// CreateTransfer creates a new transfer status in DB and returns the status
func (m *TransferManager) CreateTransfer(
	transactionID string,
	blockId string,
	contractHash string,
	activityIDs []string,
	userDID string,
	adminDID string,
	rewardPoints int,
) (*database.TransferStatus, error) {

	status := &database.TransferStatus{
		RequestID:    transactionID,
		BlockId:      blockId,
		ActivityIDs:  activityIDs,
		UserDID:      userDID,
		AdminDID:     adminDID,
		RewardPoints: rewardPoints,
		Status:       "pending",
		Message:      "Transfer initiated, waiting for blockchain confirmation",
		ContractHash: contractHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save to database
	err := database.CreateTransferStatus(status)
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer status: %w", err)
	}

	return status, nil
}

// RegisterPendingRequest creates a response channel for a blockId
func (m *TransferManager) RegisterPendingRequest(transactionID string, blockId string) chan CallbackResponse {
	m.pendingMu.Lock()
	defer m.pendingMu.Unlock()

	// Create response channel
	responseChan := make(chan CallbackResponse, 1)
	m.pendingByBlockId[blockId] = &PendingRequest{
		TransactionID: transactionID,
		ResponseChan:  responseChan,
		CreatedAt:     time.Now(),
	}

	fmt.Printf("Registered pending request: transactionID=%s, blockId=%s\n", transactionID, blockId)
	return responseChan
}

// SendCallbackResponse sends a callback response to a pending request by blockId
func (m *TransferManager) SendCallbackResponse(blockId string, response CallbackResponse) bool {
	m.pendingMu.Lock()
	defer m.pendingMu.Unlock()

	if req, exists := m.pendingByBlockId[blockId]; exists {
		fmt.Printf("Found pending request for blockId: %s, transactionID: %s\n", blockId, req.TransactionID)

		// Update persistent status in DB
		updates := map[string]interface{}{
			"message": response.Message,
		}
		if response.Success {
			updates["status"] = "success"
		} else {
			updates["status"] = "failed"
			updates["error_details"] = response.Error
		}

		err := database.UpdateTransferStatus(req.TransactionID, updates)
		if err != nil {
			fmt.Printf("Failed to update transfer status in DB: %v\n", err)
		}

		// Send to channel if still waiting
		select {
		case req.ResponseChan <- response:
			close(req.ResponseChan)
			delete(m.pendingByBlockId, blockId)
			fmt.Printf("Successfully sent callback response for blockId: %s\n", blockId)
			return true
		default:
			// Channel closed or full
			delete(m.pendingByBlockId, blockId)
			fmt.Printf("Failed to send callback response (channel closed/full) for blockId: %s\n", blockId)
			return false
		}
	}

	// Even if no pending request, update status in DB by blockId
	fmt.Printf("No pending request found for blockId: %s, updating DB anyway\n", blockId)
	m.updateStatusByBlockId(blockId, response)
	return false
}

// updateStatusByBlockId updates status when we only have blockId (fallback for late callbacks)
func (m *TransferManager) updateStatusByBlockId(blockId string, response CallbackResponse) {
	status, err := database.GetTransferStatusByBlockId(blockId)
	if err != nil {
		fmt.Printf("Failed to find transfer by blockId %s: %v\n", blockId, err)
		return
	}

	updates := map[string]interface{}{
		"message": response.Message,
	}
	if response.Success {
		updates["status"] = "success"
	} else {
		updates["status"] = "failed"
		updates["error_details"] = response.Error
	}

	err = database.UpdateTransferStatus(status.RequestID, updates)
	if err != nil {
		fmt.Printf("Failed to update transfer status: %v\n", err)
	} else {
		fmt.Printf("Updated transfer status for transactionID: %s\n", status.RequestID)
	}
}

// MarkTimeout marks a transfer as timed out and cleans up pending request
func (m *TransferManager) MarkTimeout(transactionID string, blockId string) error {
	// Update in database
	err := database.UpdateTransferStatus(transactionID, map[string]interface{}{
		"status":  "timeout",
		"message": "Transfer confirmation timed out (blockchain may still be processing)",
	})
	if err != nil {
		return fmt.Errorf("failed to mark timeout in DB: %w", err)
	}

	// Clean up pending request
	m.pendingMu.Lock()
	defer m.pendingMu.Unlock()

	if req, exists := m.pendingByBlockId[blockId]; exists {
		close(req.ResponseChan)
		delete(m.pendingByBlockId, blockId)
		fmt.Printf("Cleaned up timed out request: transactionID=%s, blockId=%s\n", transactionID, blockId)
	}

	return nil
}

// cleanupStaleRequests removes stale pending requests (timeout after 10 minutes)
func (m *TransferManager) cleanupStaleRequests() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.pendingMu.Lock()
		now := time.Now()
		for blockId, req := range m.pendingByBlockId {
			if now.Sub(req.CreatedAt) > 10*time.Minute {
				close(req.ResponseChan)
				delete(m.pendingByBlockId, blockId)
				fmt.Printf("Cleaned up stale pending request: transactionID=%s, blockId=%s\n", req.TransactionID, blockId)
			}
		}
		m.pendingMu.Unlock()
	}
}

// GetPendingCount returns the number of pending requests (for debugging)
func (m *TransferManager) GetPendingCount() int {
	m.pendingMu.RLock()
	defer m.pendingMu.RUnlock()
	return len(m.pendingByBlockId)
}
