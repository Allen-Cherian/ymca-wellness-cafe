package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
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
	FTTransferTxID string     `json:"ft_transfer_txid"` // Transaction ID from FT transfer callback
	QueuedAt       time.Time  `json:"queued_at"`        // When request was received
	StartedAt      *time.Time `json:"started_at"`       // When worker started processing
	CompletedAt    *time.Time `json:"completed_at"`     // When processing finished
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// InitDB initializes the PostgreSQL database
func InitDB(connStr string) error {
	var err error
	fmt.Printf("🔍 [DEBUG] Connection string: %s\n", connStr)
	db, err = sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings for better concurrency
	db.SetMaxOpenConns(100)                // Maximum number of open connections (increased for multi-admin parallel processing)
	db.SetMaxIdleConns(25)                 // Maximum number of idle connections (keep more ready)
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum lifetime of a connection

	// Test connection
	fmt.Println("🔍 [DEBUG] Testing connection with Ping...")
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	fmt.Println("🔍 [DEBUG] Ping successful!")

	// Create table if not exists
	fmt.Println("🔍 [DEBUG] Creating tables...")
	if err = createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	fmt.Println("✅ PostgreSQL database initialized successfully")
	fmt.Printf("📊 Connection pool: max_open=%d, max_idle=%d\n", 100, 25)
	return nil
}

// createTables creates the necessary database tables
func createTables() error {
	// Create table
	tableSQL := `
	CREATE TABLE IF NOT EXISTS transfer_status (
		request_id VARCHAR(255) PRIMARY KEY,
		blockchain_tx_id VARCHAR(255),
		block_id VARCHAR(255),
		activity_ids TEXT NOT NULL,
		user_did VARCHAR(255) NOT NULL,
		admin_did VARCHAR(255) NOT NULL,
		reward_points INTEGER NOT NULL,
		status VARCHAR(50) NOT NULL,
		message TEXT,
		contract_hash VARCHAR(255) NOT NULL,
		error_details TEXT,
		ft_transfer_txid VARCHAR(255),
		queued_at TIMESTAMP NOT NULL,
		started_at TIMESTAMP,
		completed_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`

	fmt.Println("🔍 [DEBUG] Executing CREATE TABLE statement...")
	result, err := db.Exec(tableSQL)
	if err != nil {
		fmt.Printf("🔍 [DEBUG] CREATE TABLE failed: %v\n", err)
		return fmt.Errorf("failed to create table: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("🔍 [DEBUG] CREATE TABLE succeeded, rows affected: %d\n", rowsAffected)

	// Create indexes (each statement separately for PostgreSQL compatibility)
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_blockchain_tx_id ON transfer_status(blockchain_tx_id)`,
		`CREATE INDEX IF NOT EXISTS idx_block_id ON transfer_status(block_id)`,
		`CREATE INDEX IF NOT EXISTS idx_status ON transfer_status(status)`,
		`CREATE INDEX IF NOT EXISTS idx_queued_at ON transfer_status(queued_at)`,
		`CREATE INDEX IF NOT EXISTS idx_admin_did ON transfer_status(admin_did)`,
		`CREATE INDEX IF NOT EXISTS idx_ft_transfer_txid ON transfer_status(ft_transfer_txid)`,
	}

	fmt.Printf("🔍 [DEBUG] Creating %d indexes...\n", len(indexes))
	for i, indexSQL := range indexes {
		result, err := db.Exec(indexSQL)
		if err != nil {
			fmt.Printf("🔍 [DEBUG] CREATE INDEX %d failed: %v\n", i, err)
			return fmt.Errorf("failed to create index: %w", err)
		}
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("🔍 [DEBUG] CREATE INDEX %d succeeded, rows affected: %d\n", i, rowsAffected)
	}

	// Verify table was created by querying it
	fmt.Println("🔍 [DEBUG] Verifying table creation...")

	// First, check which database we're connected to
	var currentDB string
	db.QueryRow("SELECT current_database()").Scan(&currentDB)
	fmt.Printf("🔍 [DEBUG] Connected to database: %s\n", currentDB)

	// List ALL tables we can see
	rows, _ := db.Query("SELECT tablename FROM pg_tables WHERE schemaname='public'")
	var allTables []string
	for rows.Next() {
		var t string
		rows.Scan(&t)
		allTables = append(allTables, t)
	}
	rows.Close()
	fmt.Printf("🔍 [DEBUG] All tables visible to app: %v\n", allTables)

	// Now check for our specific table
	var tableName string
	verifyErr := db.QueryRow("SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename='transfer_status'").Scan(&tableName)
	if verifyErr == sql.ErrNoRows {
		fmt.Println("❌ [DEBUG] ERROR: Table transfer_status NOT FOUND in pg_tables!")
		return fmt.Errorf("table creation verification failed: table not found")
	} else if verifyErr != nil {
		fmt.Printf("❌ [DEBUG] ERROR querying pg_tables: %v\n", verifyErr)
		return fmt.Errorf("table verification query failed: %w", verifyErr)
	}
	fmt.Printf("✅ [DEBUG] Verified: Table '%s' exists in database\n", tableName)

	fmt.Println("✅ Database tables and indexes created successfully")
	return nil
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
			reward_points, status, message, contract_hash, error_details, ft_transfer_txid,
			queued_at, started_at, completed_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	fmt.Printf("🔍 [DEBUG] CreateTransferStatus called for request_id=%s\n", status.RequestID)
	fmt.Printf("🔍 [DEBUG] Database pointer: %v\n", db)

	result, err := db.Exec(
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
		status.FTTransferTxID,
		status.QueuedAt,
		status.StartedAt,
		status.CompletedAt,
		status.CreatedAt,
		status.UpdatedAt,
	)

	if err != nil {
		fmt.Printf("🔍 [DEBUG] Exec FAILED: %v\n", err)
		return fmt.Errorf("failed to create transfer status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("🔍 [DEBUG] Exec succeeded, rows affected: %d\n", rowsAffected)

	return nil
}

// GetTransferStatus retrieves a transfer status by request ID
func GetTransferStatus(requestID string) (*TransferStatus, error) {
	query := `
		SELECT request_id, blockchain_tx_id, block_id, activity_ids, user_did, admin_did,
		       reward_points, status, message, contract_hash, error_details, ft_transfer_txid,
		       queued_at, started_at, completed_at, created_at, updated_at
		FROM transfer_status
		WHERE request_id = $1
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
		&status.FTTransferTxID,
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
		WHERE block_id = $1
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
	// Build dynamic update query with PostgreSQL numbered placeholders
	query := "UPDATE transfer_status SET updated_at = $1"
	args := []interface{}{time.Now()}
	paramCount := 1

	if blockchainTxID, ok := updates["blockchain_tx_id"]; ok {
		paramCount++
		query += fmt.Sprintf(", blockchain_tx_id = $%d", paramCount)
		args = append(args, blockchainTxID)
	}
	if blockId, ok := updates["block_id"]; ok {
		paramCount++
		query += fmt.Sprintf(", block_id = $%d", paramCount)
		args = append(args, blockId)
	}
	if status, ok := updates["status"]; ok {
		paramCount++
		query += fmt.Sprintf(", status = $%d", paramCount)
		args = append(args, status)
	}
	if message, ok := updates["message"]; ok {
		paramCount++
		query += fmt.Sprintf(", message = $%d", paramCount)
		args = append(args, message)
	}
	if errorDetails, ok := updates["error_details"]; ok {
		paramCount++
		query += fmt.Sprintf(", error_details = $%d", paramCount)
		args = append(args, errorDetails)
	}
	if ftTransferTxID, ok := updates["ft_transfer_txid"]; ok {
		paramCount++
		query += fmt.Sprintf(", ft_transfer_txid = $%d", paramCount)
		args = append(args, ftTransferTxID)
	}
	if startedAt, ok := updates["started_at"]; ok {
		paramCount++
		query += fmt.Sprintf(", started_at = $%d", paramCount)
		args = append(args, startedAt)
	}
	if completedAt, ok := updates["completed_at"]; ok {
		paramCount++
		query += fmt.Sprintf(", completed_at = $%d", paramCount)
		args = append(args, completedAt)
	}

	paramCount++
	query += fmt.Sprintf(" WHERE request_id = $%d", paramCount)
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
