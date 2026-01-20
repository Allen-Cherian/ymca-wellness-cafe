package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// TransferStatus represents a reward transfer record
type TransferStatus struct {
	RequestID      string     `json:"request_id"`       // Application UUID (generated immediately)
	BlockchainTxID string     `json:"blockchain_tx_id"` // Blockchain transaction ID (filled after execution)
	BlockId        string     `json:"block_id"`         // Blockchain block ID
	ActivityIDs    []string   `json:"activity_ids"`
	UserDID        string     `json:"user_did"`
	AdminDID       string     `json:"admin_did"`
	RewardPoints   int        `json:"reward_points"`
	Status         string     `json:"status"` // "queued", "processing", "pending", "success", "failed", "timeout"
	Message        string     `json:"message"`
	ContractHash   string     `json:"contract_hash"`
	ErrorDetails   string     `json:"error_details"`
	QueuedAt       time.Time  `json:"queued_at"`     // When request was received
	StartedAt      *time.Time `json:"started_at"`    // When worker started processing
	CompletedAt    *time.Time `json:"completed_at"`  // When processing finished
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// InitDB initializes the SQLite database
func InitDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create table if not exists
	if err = createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	fmt.Println("Database initialized successfully")
	return nil
}

// createTables creates the necessary database tables
func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS transfer_status (
		request_id TEXT PRIMARY KEY,
		blockchain_tx_id TEXT,
		block_id TEXT,
		activity_ids TEXT NOT NULL,
		user_did TEXT NOT NULL,
		admin_did TEXT NOT NULL,
		reward_points INTEGER NOT NULL,
		status TEXT NOT NULL,
		message TEXT,
		contract_hash TEXT NOT NULL,
		error_details TEXT,
		queued_at DATETIME NOT NULL,
		started_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_blockchain_tx_id ON transfer_status(blockchain_tx_id);
	CREATE INDEX IF NOT EXISTS idx_block_id ON transfer_status(block_id);
	CREATE INDEX IF NOT EXISTS idx_status ON transfer_status(status);
	CREATE INDEX IF NOT EXISTS idx_queued_at ON transfer_status(queued_at);
	CREATE INDEX IF NOT EXISTS idx_admin_did ON transfer_status(admin_did);
	`

	_, err := db.Exec(schema)
	return err
}

// CreateTransferStatus creates a new transfer status record
func CreateTransferStatus(status *TransferStatus) error {
	// Convert activity IDs to JSON
	activityIDsJSON, err := json.Marshal(status.ActivityIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal activity IDs: %w", err)
	}

	query := `
		INSERT INTO transfer_status (
			request_id, blockchain_tx_id, block_id, activity_ids, user_did, admin_did,
			reward_points, status, message, contract_hash, error_details,
			queued_at, started_at, completed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(
		query,
		status.RequestID,
		status.BlockchainTxID,
		status.BlockId,
		string(activityIDsJSON),
		status.UserDID,
		status.AdminDID,
		status.RewardPoints,
		status.Status,
		status.Message,
		status.ContractHash,
		status.ErrorDetails,
		status.QueuedAt,
		status.StartedAt,
		status.CompletedAt,
		status.CreatedAt,
		status.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create transfer status: %w", err)
	}

	return nil
}

// GetTransferStatus retrieves a transfer status by request ID
func GetTransferStatus(requestID string) (*TransferStatus, error) {
	query := `
		SELECT request_id, blockchain_tx_id, block_id, activity_ids, user_did, admin_did,
		       reward_points, status, message, contract_hash, error_details,
		       queued_at, started_at, completed_at, created_at, updated_at
		FROM transfer_status
		WHERE request_id = ?
	`

	var status TransferStatus
	var activityIDsJSON string

	err := db.QueryRow(query, requestID).Scan(
		&status.RequestID,
		&status.BlockchainTxID,
		&status.BlockId,
		&activityIDsJSON,
		&status.UserDID,
		&status.AdminDID,
		&status.RewardPoints,
		&status.Status,
		&status.Message,
		&status.ContractHash,
		&status.ErrorDetails,
		&status.QueuedAt,
		&status.StartedAt,
		&status.CompletedAt,
		&status.CreatedAt,
		&status.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transfer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer status: %w", err)
	}

	// Unmarshal activity IDs
	if err := json.Unmarshal([]byte(activityIDsJSON), &status.ActivityIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal activity IDs: %w", err)
	}

	return &status, nil
}

// GetTransferStatusByBlockId retrieves a transfer status by block ID
func GetTransferStatusByBlockId(blockId string) (*TransferStatus, error) {
	query := `
		SELECT request_id, blockchain_tx_id, block_id, activity_ids, user_did, admin_did,
		       reward_points, status, message, contract_hash, error_details,
		       queued_at, started_at, completed_at, created_at, updated_at
		FROM transfer_status
		WHERE block_id = ?
	`

	var status TransferStatus
	var activityIDsJSON string

	err := db.QueryRow(query, blockId).Scan(
		&status.RequestID,
		&status.BlockchainTxID,
		&status.BlockId,
		&activityIDsJSON,
		&status.UserDID,
		&status.AdminDID,
		&status.RewardPoints,
		&status.Status,
		&status.Message,
		&status.ContractHash,
		&status.ErrorDetails,
		&status.QueuedAt,
		&status.StartedAt,
		&status.CompletedAt,
		&status.CreatedAt,
		&status.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transfer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer status: %w", err)
	}

	// Unmarshal activity IDs
	if err := json.Unmarshal([]byte(activityIDsJSON), &status.ActivityIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal activity IDs: %w", err)
	}

	return &status, nil
}

// UpdateTransferStatus updates an existing transfer status
func UpdateTransferStatus(requestID string, updates map[string]interface{}) error {
	// Build dynamic update query
	query := "UPDATE transfer_status SET updated_at = ?"
	args := []interface{}{time.Now()}

	if blockchainTxID, ok := updates["blockchain_tx_id"]; ok {
		query += ", blockchain_tx_id = ?"
		args = append(args, blockchainTxID)
	}
	if blockId, ok := updates["block_id"]; ok {
		query += ", block_id = ?"
		args = append(args, blockId)
	}
	if status, ok := updates["status"]; ok {
		query += ", status = ?"
		args = append(args, status)
	}
	if message, ok := updates["message"]; ok {
		query += ", message = ?"
		args = append(args, message)
	}
	if errorDetails, ok := updates["error_details"]; ok {
		query += ", error_details = ?"
		args = append(args, errorDetails)
	}
	if startedAt, ok := updates["started_at"]; ok {
		query += ", started_at = ?"
		args = append(args, startedAt)
	}
	if completedAt, ok := updates["completed_at"]; ok {
		query += ", completed_at = ?"
		args = append(args, completedAt)
	}

	query += " WHERE request_id = ?"
	args = append(args, requestID)

	result, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update transfer status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transfer not found")
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
