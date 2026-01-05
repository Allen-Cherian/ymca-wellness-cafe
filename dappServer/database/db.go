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
	RequestID    string    `json:"request_id"`
	BlockId      string    `json:"block_id"`
	ActivityIDs  []string  `json:"activity_ids"`
	UserDID      string    `json:"user_did"`
	AdminDID     string    `json:"admin_did"`
	RewardPoints int       `json:"reward_points"`
	Status       string    `json:"status"` // "pending", "success", "failed", "timeout"
	Message      string    `json:"message"`
	ContractHash string    `json:"contract_hash"`
	ErrorDetails string    `json:"error_details"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
		block_id TEXT,
		activity_ids TEXT NOT NULL,
		user_did TEXT NOT NULL,
		admin_did TEXT NOT NULL,
		reward_points INTEGER NOT NULL,
		status TEXT NOT NULL,
		message TEXT,
		contract_hash TEXT NOT NULL,
		error_details TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_block_id ON transfer_status(block_id);
	CREATE INDEX IF NOT EXISTS idx_status ON transfer_status(status);
	CREATE INDEX IF NOT EXISTS idx_created_at ON transfer_status(created_at);
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
			request_id, block_id, activity_ids, user_did, admin_did,
			reward_points, status, message, contract_hash, error_details,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(
		query,
		status.RequestID,
		status.BlockId,
		string(activityIDsJSON),
		status.UserDID,
		status.AdminDID,
		status.RewardPoints,
		status.Status,
		status.Message,
		status.ContractHash,
		status.ErrorDetails,
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
		SELECT request_id, block_id, activity_ids, user_did, admin_did,
		       reward_points, status, message, contract_hash, error_details,
		       created_at, updated_at
		FROM transfer_status
		WHERE request_id = ?
	`

	var status TransferStatus
	var activityIDsJSON string

	err := db.QueryRow(query, requestID).Scan(
		&status.RequestID,
		&status.BlockId,
		&activityIDsJSON,
		&status.UserDID,
		&status.AdminDID,
		&status.RewardPoints,
		&status.Status,
		&status.Message,
		&status.ContractHash,
		&status.ErrorDetails,
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
		SELECT request_id, block_id, activity_ids, user_did, admin_did,
		       reward_points, status, message, contract_hash, error_details,
		       created_at, updated_at
		FROM transfer_status
		WHERE block_id = ?
	`

	var status TransferStatus
	var activityIDsJSON string

	err := db.QueryRow(query, blockId).Scan(
		&status.RequestID,
		&status.BlockId,
		&activityIDsJSON,
		&status.UserDID,
		&status.AdminDID,
		&status.RewardPoints,
		&status.Status,
		&status.Message,
		&status.ContractHash,
		&status.ErrorDetails,
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
