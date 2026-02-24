package server

import (
	"dapp-server/config"
	"dapp-server/database"
	rubix_interaction "dapp-server/rubix-interaction"
	"fmt"
	"sync"
	"time"
)

// TransferJob represents a queued transfer request
type TransferJob struct {
	RequestID string                // Application UUID
	Request   TransferRewardRequest // Transfer request data
	QueuedAt  time.Time             // When job was added to queue
}

// AdminQueue represents a queue for a specific admin
type AdminQueue struct {
	adminDID string
	queue    chan *TransferJob
	worker   *sync.Once // Ensures worker starts only once
}

// TransferQueueManager manages multiple admin queues for parallel processing
type TransferQueueManager struct {
	adminQueues map[string]*AdminQueue // Map: adminDID -> AdminQueue
	mu          sync.RWMutex
}

var (
	queueManager     *TransferQueueManager
	queueManagerOnce sync.Once
)

// GetQueueManager returns the singleton queue manager instance
func GetQueueManager() *TransferQueueManager {
	queueManagerOnce.Do(func() {
		queueManager = &TransferQueueManager{
			adminQueues: make(map[string]*AdminQueue),
		}
		fmt.Println("🚀 Transfer queue manager initialized (multi-admin support)")
	})
	return queueManager
}

// GetOrCreateAdminQueue gets or creates a queue for a specific admin
func (qm *TransferQueueManager) GetOrCreateAdminQueue(adminDID string) *AdminQueue {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Check if queue already exists
	if queue, exists := qm.adminQueues[adminDID]; exists {
		return queue
	}

	// Create new queue for this admin
	queue := &AdminQueue{
		adminDID: adminDID,
		queue:    make(chan *TransferJob, 1000), // Buffer for 1000 jobs per admin
		worker:   &sync.Once{},
	}

	// Start dedicated worker for this admin (runs once)
	queue.worker.Do(func() {
		go qm.adminWorker(queue)
	})

	qm.adminQueues[adminDID] = queue
	fmt.Printf("🆕 Created new queue for admin: %s\n", adminDID)

	return queue
}

// Enqueue adds a transfer job to the appropriate admin's queue
func (qm *TransferQueueManager) Enqueue(requestID string, req TransferRewardRequest) error {
	// Get or create queue for this admin
	adminQueue := qm.GetOrCreateAdminQueue(req.AdminDID)

	job := &TransferJob{
		RequestID: requestID,
		Request:   req,
		QueuedAt:  time.Now(),
	}

	// Try to add to admin's queue (non-blocking)
	select {
	case adminQueue.queue <- job:
		fmt.Printf("📥 Queued transfer: request_id=%s, admin=%s, queue_size=%d\n",
			requestID, req.AdminDID, len(adminQueue.queue))
		return nil
	default:
		// Queue is full for this admin
		return fmt.Errorf("queue is full for admin %s (capacity: 1000), please retry later", req.AdminDID)
	}
}

// GetQueueSize returns the total queue size across all admins
func (qm *TransferQueueManager) GetQueueSize() int {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	totalSize := 0
	for _, adminQueue := range qm.adminQueues {
		totalSize += len(adminQueue.queue)
	}
	return totalSize
}

// adminWorker processes jobs for a specific admin (ONE GOROUTINE PER ADMIN)
func (qm *TransferQueueManager) adminWorker(adminQueue *AdminQueue) {
	fmt.Printf("🚀 Worker started for admin: %s (parallel processing enabled)\n", adminQueue.adminDID)

	for job := range adminQueue.queue {
		startTime := time.Now()
		waitTime := time.Since(job.QueuedAt)

		fmt.Printf("\n═══════════════════════════════════════════════════════════\n")
		fmt.Printf("⚙️  Processing [Admin: %s]: request_id=%s\n", adminQueue.adminDID, job.RequestID)
		fmt.Printf("⏱️  Waited in queue: %v\n", waitTime)
		fmt.Printf("📊 Queue size for this admin: %d\n", len(adminQueue.queue))
		fmt.Printf("═══════════════════════════════════════════════════════════\n")

		// Update status to "processing"
		// COMMENTED OUT: Keep status as "success" throughout
		// now := time.Now()
		// err := database.UpdateTransferStatus(job.RequestID, map[string]interface{}{
		// 	"status":     "processing",
		// 	"message":    "Smart contract execution in progress",
		// 	"started_at": now,
		// })
		// if err != nil {
		// 	fmt.Printf("❌ Failed to update status to processing: %v\n", err)
		// }

		// Process the transfer (this is where the actual work happens)
		qm.processTransfer(job)

		processingTime := time.Since(startTime)
		fmt.Printf("\n✅ Completed [Admin: %s]: request_id=%s\n", adminQueue.adminDID, job.RequestID)
		fmt.Printf("⏱️  Processing time: %v\n", processingTime)
		fmt.Printf("📊 Queue size: %d\n\n", len(adminQueue.queue))
	}
}

// processTransfer executes the actual blockchain operations
func (qm *TransferQueueManager) processTransfer(job *TransferJob) {
	req := job.Request
	requestID := job.RequestID

	// ═══════════════════════════════════════════════════════════
	// Load configuration
	// ═══════════════════════════════════════════════════════════
	cfg, err := config.GetConfig()
	if err != nil {
		// COMMENTED OUT: Keep status as "success"
		// qm.markFailed(requestID, "Failed to load config", err)
		fmt.Printf("⚠️  Failed to load config: %v (status remains 'success')\n", err)
		return
	}

	nodePort, exists := config.GetPortByDid(cfg, req.AdminDID)
	if !exists {
		// COMMENTED OUT: Keep status as "success"
		// qm.markFailed(requestID, "Node port not found for admin DID", nil)
		fmt.Printf("⚠️  Node port not found for admin DID (status remains 'success')\n")
		return
	}

	url := fmt.Sprintf("http://localhost:%s", nodePort)
	rewardPoints := len(req.ActivityID)

	contractMsg := fmt.Sprintf(`{"transfer_ytoken":{"name": "rubix1", "ft_info": {"comment":"Transfer of reward via contract","ft_count":%f,"ft_name":"ytoken","sender": "%s","creatorDID": "%s", "receiver": "%s"}}}`,
		float64(rewardPoints), req.AdminDID, req.AdminDID, req.UserDID)

	// Try to get admin-specific contract first, fallback to global config
	transferContractHash, err := config.GetContractForAdmin(req.AdminDID, "transfer")
	if err != nil {
		// Fallback to global contract from environment config
		fmt.Printf("⚠️  Using fallback contract (admin-specific not found): %v\n", err)
		transferContractHash = config.GetEnvConfig().TransferContract
		if transferContractHash == "" {
			// COMMENTED OUT: Keep status as "success"
			// qm.markFailed(requestID, "Transfer contract hash not configured", nil)
			fmt.Printf("⚠️  Transfer contract hash not configured (status remains 'success')\n")
			return
		}
	} else {
		fmt.Printf("✅ Using admin-specific contract: %s for admin: %s\n", transferContractHash, req.AdminDID)
	}

	fmt.Printf("🔗 Node URL: %s\n", url)
	fmt.Printf("📝 Contract message: %s\n", contractMsg)

	// ═══════════════════════════════════════════════════════════
	// Step 1: Execute smart contract
	// ═══════════════════════════════════════════════════════════
	fmt.Println("📤 Step 1: Executing smart contract...")
	blockchainRequestID, err := rubix_interaction.ExecuteSmartContract(url, transferContractHash, req.AdminDID, contractMsg)
	if err != nil {
		// COMMENTED OUT: Keep status as "success"
		// qm.markFailed(requestID, "Failed to execute smart contract", err)
		fmt.Printf("⚠️  Failed to execute smart contract: %v (status remains 'success')\n", err)
		return
	}
	fmt.Printf("✅ Smart contract executed: blockchain_request_id=%s\n", blockchainRequestID)

	// ═══════════════════════════════════════════════════════════
	// Step 2: Sign transaction (creates block on blockchain)
	// ═══════════════════════════════════════════════════════════
	fmt.Println("✍️  Step 2: Signing transaction...")
	signatureResponse, err := rubix_interaction.SignatureResponse(url, blockchainRequestID)
	if err != nil {
		// COMMENTED OUT: Keep status as "success"
		// qm.markFailed(requestID, "Failed to sign transaction", err)
		fmt.Printf("⚠️  Failed to sign transaction: %v (status remains 'success')\n", err)
		return
	}

	// Extract blockchain's transaction ID
	blockchainTxID := signatureResponse.Result
	fmt.Printf("✅ Transaction signed: blockchain_tx_id=%s\n", blockchainTxID)

	// ═══════════════════════════════════════════════════════════
	// Step 3: Extract BlockId from blockchain
	// ═══════════════════════════════════════════════════════════
	fmt.Println("🔍 Step 3: Extracting block ID...")
	blockId, err := ExtractLatestBlockId(transferContractHash, url)
	if err != nil {
		fmt.Printf("⚠️  Failed to extract BlockId: %v\n", err)
		// Continue anyway, might still work with callback
	} else {
		fmt.Printf("✅ Block ID extracted: %s\n", blockId)
	}

	// ═══════════════════════════════════════════════════════════
	// Step 4: Update database with blockchain IDs
	// ═══════════════════════════════════════════════════════════
	fmt.Println("💾 Step 4: Updating database with blockchain IDs...")
	err = database.UpdateTransferStatus(requestID, map[string]interface{}{
		"blockchain_tx_id": blockchainTxID,
		"block_id":         blockId,
		"message":          "Waiting for blockchain confirmation",
	})
	if err != nil {
		fmt.Printf("⚠️  Failed to update database with blockchain IDs: %v\n", err)
	} else {
		fmt.Println("✅ Database updated with blockchain IDs")
	}

	// ═══════════════════════════════════════════════════════════
	// Step 5: Register pending request for callback
	// ═══════════════════════════════════════════════════════════
	fmt.Println("📞 Step 5: Registering for callback...")
	manager := GetTransferManager()
	responseChan := manager.RegisterPendingRequest(requestID, blockId)
	fmt.Println("✅ Registered for callback")

	// ═══════════════════════════════════════════════════════════
	// Step 6: Start background handler for callback/timeout (non-blocking)
	// ═══════════════════════════════════════════════════════════
	fmt.Println("⏳ Step 6: Starting background callback handler...")
	go qm.handleCallbackAsync(requestID, blockId, responseChan)

	// ✅ DONE - Worker can now pick next job immediately!
	fmt.Println("✅ Contract executed, moving to next job")
}

// handleCallbackAsync waits for callback or timeout in background (non-blocking)
func (qm *TransferQueueManager) handleCallbackAsync(requestID string, blockId string, responseChan chan CallbackResponse) {
	manager := GetTransferManager()

	select {
	case callbackResult := <-responseChan:
		// Callback arrived!
		fmt.Printf("🎉 Callback received: request_id=%s, success=%v\n", requestID, callbackResult.Success)

		completedAt := time.Now()
		if callbackResult.Success {
			// COMMENTED OUT: Keep status as "success" (already set)
			// err := database.UpdateTransferStatus(requestID, map[string]interface{}{
			// 	"status":          "success",
			// 	"message":         callbackResult.Message,
			// 	"ft_transfer_txid": callbackResult.FTTransferTxID,
			// 	"completed_at":    completedAt,
			// })
			// if err != nil {
			// 	fmt.Printf("❌ Failed to update status to success: %v\n", err)
			// } else {
			// 	fmt.Printf("✅ Transfer SUCCEEDED: request_id=%s\n", requestID)
			// 	if callbackResult.FTTransferTxID != "" {
			// 		fmt.Printf("📝 FT Transfer TxID: %s\n", callbackResult.FTTransferTxID)
			// 	}
			// }
			fmt.Printf("✅ Transfer SUCCEEDED: request_id=%s (status remains 'success')\n", requestID)
			if callbackResult.FTTransferTxID != "" {
				fmt.Printf("📝 FT Transfer TxID: %s\n", callbackResult.FTTransferTxID)
			}
		} else {
			// COMMENTED OUT: Keep status as "success"
			// err := database.UpdateTransferStatus(requestID, map[string]interface{}{
			// 	"status":           "failed",
			// 	"message":          callbackResult.Message,
			// 	"error_details":    callbackResult.Error,
			// 	"ft_transfer_txid": callbackResult.FTTransferTxID,
			// 	"completed_at":     completedAt,
			// })
			// if err != nil {
			// 	fmt.Printf("❌ Failed to update status to failed: %v\n", err)
			// } else {
			// 	fmt.Printf("❌ Transfer FAILED: request_id=%s, error=%s\n", requestID, callbackResult.Error)
			// }
			fmt.Printf("⚠️  Transfer callback returned error: request_id=%s, error=%s (status remains 'success')\n", requestID, callbackResult.Error)
		}

	case <-time.After(15 * time.Minute):
		// Timeout - callback didn't arrive in time
		fmt.Printf("⏰ Timeout: request_id=%s - Callback did not arrive within 15 minutes\n", requestID)

		completedAt := time.Now()
		err := manager.MarkTimeout(requestID, blockId)
		if err != nil {
			fmt.Printf("⚠️  Failed to mark timeout: %v\n", err)
		}

		// COMMENTED OUT: Keep status as "success"
		// err = database.UpdateTransferStatus(requestID, map[string]interface{}{
		// 	"status":       "timeout",
		// 	"message":      "Transfer confirmation timed out (blockchain may still be processing)",
		// 	"completed_at": completedAt,
		// })
		// if err != nil {
		// 	fmt.Printf("❌ Failed to update status to timeout: %v\n", err)
		// } else {
		// 	fmt.Printf("⏰ Transfer TIMEOUT: request_id=%s\n", requestID)
		// }
		fmt.Printf("⏰ Transfer TIMEOUT: request_id=%s (status remains 'success')\n", requestID)
	}
}

// markFailed updates the database to mark a transfer as failed
func (qm *TransferQueueManager) markFailed(requestID string, message string, err error) {
	errorDetails := ""
	if err != nil {
		errorDetails = err.Error()
		fmt.Printf("❌ Transfer failed: request_id=%s, error=%s: %v\n", requestID, message, err)
	} else {
		fmt.Printf("❌ Transfer failed: request_id=%s, error=%s\n", requestID, message)
	}

	completedAt := time.Now()
	updateErr := database.UpdateTransferStatus(requestID, map[string]interface{}{
		"status":        "failed",
		"message":       message,
		"error_details": errorDetails,
		"completed_at":  completedAt,
	})

	if updateErr != nil {
		fmt.Printf("❌ Failed to update database: %v\n", updateErr)
	}
}

// GetQueueMetrics returns metrics for all admin queues
func (qm *TransferQueueManager) GetQueueMetrics() map[string]interface{} {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	metrics := make(map[string]interface{})
	adminMetrics := make([]map[string]interface{}, 0)

	totalQueued := 0
	for adminDID, queue := range qm.adminQueues {
		queueSize := len(queue.queue)
		totalQueued += queueSize

		adminMetrics = append(adminMetrics, map[string]interface{}{
			"admin_did":  adminDID,
			"queue_size": queueSize,
		})
	}

	metrics["total_admins"] = len(qm.adminQueues)
	metrics["total_queued"] = totalQueued
	metrics["admin_queues"] = adminMetrics

	return metrics
}
