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

// TransferQueue manages sequential processing of transfer requests
type TransferQueue struct {
	queue chan *TransferJob
	mu    sync.RWMutex
}

var (
	transferQueue     *TransferQueue
	transferQueueOnce sync.Once
)

// GetTransferQueue returns the singleton queue instance
func GetTransferQueue() *TransferQueue {
	transferQueueOnce.Do(func() {
		transferQueue = &TransferQueue{
			queue: make(chan *TransferJob, 1000), // Buffer for 1000 jobs
		}
		// Start the sequential worker
		go transferQueue.worker()
	})
	return transferQueue
}

// Enqueue adds a transfer job to the queue
func (q *TransferQueue) Enqueue(requestID string, req TransferRewardRequest) error {
	job := &TransferJob{
		RequestID: requestID,
		Request:   req,
		QueuedAt:  time.Now(),
	}

	// Try to add to queue (non-blocking)
	select {
	case q.queue <- job:
		fmt.Printf("📥 Queued transfer: request_id=%s, queue_size=%d\n", requestID, len(q.queue))
		return nil
	default:
		// Queue is full
		return fmt.Errorf("queue is full (capacity: 1000), please retry later")
	}
}

// GetQueueSize returns the current queue size
func (q *TransferQueue) GetQueueSize() int {
	return len(q.queue)
}

// worker processes jobs sequentially (SINGLE GOROUTINE)
func (q *TransferQueue) worker() {
	fmt.Println("🚀 Transfer queue worker started (sequential processing)")

	for job := range q.queue {
		startTime := time.Now()
		waitTime := time.Since(job.QueuedAt)

		fmt.Printf("\n═══════════════════════════════════════════════════════════\n")
		fmt.Printf("⚙️  Processing: request_id=%s\n", job.RequestID)
		fmt.Printf("⏱️  Waited in queue: %v\n", waitTime)
		fmt.Printf("📊 Queue size: %d\n", len(q.queue))
		fmt.Printf("═══════════════════════════════════════════════════════════\n")

		// Update status to "processing"
		now := time.Now()
		err := database.UpdateTransferStatus(job.RequestID, map[string]interface{}{
			"status":     "processing",
			"message":    "Smart contract execution in progress",
			"started_at": now,
		})
		if err != nil {
			fmt.Printf("❌ Failed to update status to processing: %v\n", err)
		}

		// Process the transfer (this is where the actual work happens)
		q.processTransfer(job)

		processingTime := time.Since(startTime)
		fmt.Printf("\n✅ Completed: request_id=%s\n", job.RequestID)
		fmt.Printf("⏱️  Processing time: %v\n", processingTime)
		fmt.Printf("📊 Queue size: %d\n\n", len(q.queue))
	}
}

// processTransfer executes the actual blockchain operations
func (q *TransferQueue) processTransfer(job *TransferJob) {
	req := job.Request
	requestID := job.RequestID

	// ═══════════════════════════════════════════════════════════
	// Load configuration
	// ═══════════════════════════════════════════════════════════
	cfg, err := config.GetConfig()
	if err != nil {
		q.markFailed(requestID, "Failed to load config", err)
		return
	}

	nodePort, exists := config.GetPortByDid(cfg, req.AdminDID)
	if !exists {
		q.markFailed(requestID, "Node port not found for admin DID", nil)
		return
	}

	url := fmt.Sprintf("http://localhost:%s", nodePort)
	rewardPoints := len(req.ActivityID)

	contractMsg := fmt.Sprintf(`{"transfer_ytoken":{"name": "rubix1", "ft_info": {"comment":"Transfer of reward via contract","ft_count":%f,"ft_name":"ytoken","sender": "%s","creatorDID": "%s", "receiver": "%s"}}}`,
		float64(rewardPoints), req.AdminDID, req.AdminDID, req.UserDID)

	transferContractHash := config.GetEnvConfig().TransferContract
	if transferContractHash == "" {
		q.markFailed(requestID, "Transfer contract hash not configured", nil)
		return
	}

	fmt.Printf("🔗 Node URL: %s\n", url)
	fmt.Printf("📝 Contract message: %s\n", contractMsg)

	// ═══════════════════════════════════════════════════════════
	// Step 1: Execute smart contract
	// ═══════════════════════════════════════════════════════════
	fmt.Println("📤 Step 1: Executing smart contract...")
	blockchainRequestID, err := rubix_interaction.ExecuteSmartContract(url, transferContractHash, req.AdminDID, contractMsg)
	if err != nil {
		q.markFailed(requestID, "Failed to execute smart contract", err)
		return
	}
	fmt.Printf("✅ Smart contract executed: blockchain_request_id=%s\n", blockchainRequestID)

	// ═══════════════════════════════════════════════════════════
	// Step 2: Sign transaction (creates block on blockchain)
	// ═══════════════════════════════════════════════════════════
	fmt.Println("✍️  Step 2: Signing transaction...")
	signatureResponse, err := rubix_interaction.SignatureResponse(url, blockchainRequestID)
	if err != nil {
		q.markFailed(requestID, "Failed to sign transaction", err)
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
	go q.handleCallbackAsync(requestID, blockId, responseChan)

	// ✅ DONE - Worker can now pick next job immediately!
	fmt.Println("✅ Contract executed, moving to next job")
}

// handleCallbackAsync waits for callback or timeout in background (non-blocking)
func (q *TransferQueue) handleCallbackAsync(requestID string, blockId string, responseChan chan CallbackResponse) {
	manager := GetTransferManager()

	select {
	case callbackResult := <-responseChan:
		// Callback arrived!
		fmt.Printf("🎉 Callback received: request_id=%s, success=%v\n", requestID, callbackResult.Success)

		completedAt := time.Now()
		if callbackResult.Success {
			err := database.UpdateTransferStatus(requestID, map[string]interface{}{
				"status":          "success",
				"message":         callbackResult.Message,
				"ft_transfer_txid": callbackResult.FTTransferTxID,
				"completed_at":    completedAt,
			})
			if err != nil {
				fmt.Printf("❌ Failed to update status to success: %v\n", err)
			} else {
				fmt.Printf("✅ Transfer SUCCEEDED: request_id=%s\n", requestID)
				if callbackResult.FTTransferTxID != "" {
					fmt.Printf("📝 FT Transfer TxID: %s\n", callbackResult.FTTransferTxID)
				}
			}
		} else {
			err := database.UpdateTransferStatus(requestID, map[string]interface{}{
				"status":           "failed",
				"message":          callbackResult.Message,
				"error_details":    callbackResult.Error,
				"ft_transfer_txid": callbackResult.FTTransferTxID,
				"completed_at":     completedAt,
			})
			if err != nil {
				fmt.Printf("❌ Failed to update status to failed: %v\n", err)
			} else {
				fmt.Printf("❌ Transfer FAILED: request_id=%s, error=%s\n", requestID, callbackResult.Error)
			}
		}

	case <-time.After(3 * time.Minute):
		// Timeout - callback didn't arrive in time
		fmt.Printf("⏰ Timeout: request_id=%s - Callback did not arrive within 3 minutes\n", requestID)

		completedAt := time.Now()
		err := manager.MarkTimeout(requestID, blockId)
		if err != nil {
			fmt.Printf("⚠️  Failed to mark timeout: %v\n", err)
		}

		err = database.UpdateTransferStatus(requestID, map[string]interface{}{
			"status":       "timeout",
			"message":      "Transfer confirmation timed out (blockchain may still be processing)",
			"completed_at": completedAt,
		})
		if err != nil {
			fmt.Printf("❌ Failed to update status to timeout: %v\n", err)
		} else {
			fmt.Printf("⏰ Transfer TIMEOUT: request_id=%s\n", requestID)
		}
	}
}

// markFailed updates the database to mark a transfer as failed
func (q *TransferQueue) markFailed(requestID string, message string, err error) {
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
