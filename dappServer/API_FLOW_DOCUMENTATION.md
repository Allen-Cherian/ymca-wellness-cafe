# API Flow Documentation - fix/timeout Branch

This document describes the flow of each API endpoint implemented in the dApp server.

---

## Table of Contents

1. [POST /api/deploy-contract](#1-post-apideploy-contract)
2. [POST /api/execute-contract](#2-post-apiexecute-contract)
3. [POST /api/activity/add](#3-post-apiactivityadd)
4. [POST /api/callback/trigger](#4-post-apicallbacktrigger)
5. [POST /api/rewards/transfer](#5-post-apirewardstransfer)
6. [GET /api/rewards/status/:transactionID](#6-get-apirewardsstatustransactionid)
7. [POST /api/admin/add](#7-post-apiadminadd)
8. [POST /api/callback/add-admin](#8-post-apicallbackadd-admin)
9. [POST /api/call-back-trigger](#9-post-apicall-back-trigger)

---

## 1. POST /api/deploy-contract

**Purpose:** Deploy a WASM smart contract to the Rubix blockchain

**File:** `server/api.go:78`

### Request Body
```json
{
  "wasm_path": "/path/to/contract.wasm",
  "lib_path": "/path/to/library",
  "deployer_did": "bafyb...",
  "state_path": "/path/to/state"
}
```

### Flow
1. **Parse Request** - Extract deployment parameters
2. **Load Config** - Get configuration to find node details
3. **Get Node Name** - Find node name associated with deployer DID
4. **Call Rubix Deploy** - `rubix.Deploy()` to deploy contract to blockchain
5. **Return Response** - Send success message with contract hash

### Response
```json
{
  "message": "Contract Deployed Successfully",
  "data": {
    // Deployment result from Rubix
  }
}
```

---

## 2. POST /api/execute-contract

**Purpose:** Execute a deployed smart contract

**File:** `server/api.go:28`

### Request Body
```json
{
  "contract_hash": "Qm...",
  "executor_did": "bafyb...",
  "contract_input": "{\"function_name\": {...}}"
}
```

### Flow
1. **Parse Request** - Extract contract execution parameters
2. **Load Config** - Get node configuration
3. **Get Node Name** - Find node name from executor DID
4. **Execute Contract** - `rubix.Execute()` to run contract
5. **Get Node Port** - Find port for signature API call
6. **Sign Response** - `rubix.SignatureResponse()` to sign the transaction
7. **Return Result** - Send execution result

### Response
```json
{
  "message": "DApp executed successfully",
  "data": {
    // Execution result
  }
}
```

---

## 3. POST /api/activity/add

**Purpose:** Add a new activity with reward points to the smart contract

**File:** `server/server.go:325`

### Request Body
```json
{
  "activity_id": "activity_123",
  "reward_points": 10,
  "admin_did": "bafyb..."
}
```

### Flow
1. **Parse Request** - Extract activity details
2. **Load Config** - Get server configuration
3. **Get Admin Node Port** - Find port associated with admin DID
4. **Build URL** - Construct node API URL
5. **Build Contract Message** - Format: `{"add_activity": {"activity_id":"...", "reward_points":N}}`
6. **Get Contract Hash** - Load AddActivityContract hash from config
7. **Execute Smart Contract** - Call `rubix_interaction.ExecuteSmartContract()`
8. **Sign Transaction** - Call `rubix_interaction.SignatureResponse()`
9. **Fetch Latest Block** - Get smart contract data to confirm addition
10. **Return Response** - Send success with block data

### Response
```json
{
  "message": "Activity added to smart contract tokenchain",
  "data": "..." // Block data
}
```

---

## 4. POST /api/callback/trigger

**Purpose:** Manual callback trigger to process activity data from smart contract

**File:** `server/server.go:386`

### Request Body
```json
{
  "port": "20000",
  "smart_contract_hash": "Qm..."
}
```

### Flow
1. **Parse Request** - Get port and contract hash
2. **Build URL** - Construct node URL from port
3. **Fetch Smart Contract Data** - Get latest block data
4. **Parse Block Data** - Extract smart contract token data
5. **Find Relevant Block** - Get the latest non-genesis block
6. **Parse Activity Payload** - Extract activity details from block
7. **Initialize WASM Module** - Create WASM bridge with host functions
8. **Build Contract Input** - Format with activity_id, reward_points, and block_hash
9. **Execute WASM Function** - Call contract function with input
10. **Return Result** - Send execution result

### Response
Implicit - Processes callback and logs result

---

## 5. POST /api/rewards/transfer

**Purpose:** Transfer reward tokens to a user (SYNCHRONOUS with 1-hour timeout)

**File:** `server/server.go:121`

### Request Body
```json
{
  "activity_id": ["act1", "act2", "act3"],
  "user_did": "bafyb...",
  "admin_did": "bafyb..."
}
```

### Flow

#### Phase 1: Validation & Execution (0-2s)
1. **Parse Request** - Extract transfer details
2. **Load Config** - Get server configuration
3. **Get Admin Node Port** - Find port from admin DID
4. **Calculate Reward Points** - Count activities = reward amount
5. **Build Contract Message** - Format transfer_sample_ft message
6. **Get Transfer Contract Hash** - Load from config
7. **Execute Smart Contract** - Returns requestID
8. **Sign Transaction** - Blockchain creates block and triggers callback (5s delay)
9. **Extract Transaction ID** - Get blockchain transaction ID from signature response

#### Phase 2: Parallel Processing (0-5s)
10. **Register Pending Request** - Store transactionID in TransferManager with response channel
11. **Background Goroutine Starts** (runs in parallel):
    - Fetch BlockId from smart contract data
    - Create database record with status "pending"
    - Update TransferManager mapping: transactionID → actual BlockId

#### Phase 3: Wait for Callback (1-3600s)
12. **Wait on Channel** - Wait up to 1 hour for callback response
13. **Callback Arrives** (from ftDappHandler):
    - Callback has 1s delay
    - Callback sends response through channel
    - Updates database status to "success" or "failed"
14. **OR Timeout** (after 1 hour):
    - Mark as "timeout" in database
    - Return timeout response to client

### Response (Success)
```json
{
  "status": "success",
  "message": "Reward transfer completed successfully",
  "transaction_id": "...",
  "block_id": "...",
  "data": {
    "rewards_awarded": 3.0,
    "activity_ids": ["act1", "act2", "act3"],
    "user_did": "bafyb...",
    "admin_did": "bafyb..."
  }
}
```

### Response (Timeout)
```json
{
  "status": "timeout",
  "message": "Transfer initiated but confirmation timed out. Check status later using transaction_id.",
  "transaction_id": "...",
  "data": {...},
  "note": "Use GET /api/rewards/status/... to check transfer status"
}
```

### Key Components

#### TransferManager
- **Purpose:** Coordinates between API and callback handler
- **Storage:** In-memory map of blockId → response channel
- **Functions:**
  - `RegisterPendingRequest()` - Creates channel for waiting API request
  - `SendCallbackResponse()` - Callback sends response through channel
  - `UpdatePendingRequestBlockId()` - Updates mapping when BlockId is fetched
  - `MarkTimeout()` - Handles timeout cleanup
  - `cleanupStaleRequests()` - Background cleanup of stale requests (>90 min)

---

## 6. GET /api/rewards/status/:transactionID

**Purpose:** Check the status of a reward transfer

**File:** `server/server.go:295`

### Request
```
GET /api/rewards/status/550e8400-e29b-41d4-a716-446655440000
```

### Flow
1. **Extract Transaction ID** - Get from URL parameter
2. **Validate Input** - Ensure transaction ID is present
3. **Query Database** - `database.GetTransferStatus(transactionID)`
4. **Return Status** - Send transfer status details

### Response
```json
{
  "status": true,
  "data": {
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "blockchain_tx_id": "db7e190a5fdb1cc9...",
    "block_id": "QmXyz...",
    "activity_ids": ["act1", "act2", "act3"],
    "user_did": "bafyb...",
    "admin_did": "bafyb...",
    "reward_points": 3,
    "status": "success",
    "message": "Reward transfer completed successfully",
    "created_at": "2026-01-27T10:00:00Z",
    "updated_at": "2026-01-27T10:00:15Z"
  }
}
```

---

## 7. POST /api/admin/add

**Purpose:** Add a new admin to the system via smart contract

**File:** `server/handler.go:18`

### Request Body
```json
{
  "new_admin_did": "bafyb...",
  "existing_admin_did": "bafyb..."
}
```

### Flow
1. **Parse Request** - Extract admin DIDs
2. **Load Config** - Get server configuration
3. **Get Existing Admin Port** - Find port for existing admin DID
4. **Build URL** - Construct node API URL
5. **Build Contract Message** - Format: `{"add_admin": {"admin_did":"..."}}`
6. **Get Contract Hash** - Load AddAdminContract from config
7. **Execute Smart Contract** - Call with add_admin message
8. **Sign Transaction** - Confirm transaction with signature
9. **Fetch Latest Block** - Get confirmation data
10. **Return Response** - Send success message

### Response
```json
{
  "message": "Admin added to smart contract tokenchain",
  "data": "..." // Block data
}
```

---

## 8. POST /api/callback/add-admin

**Purpose:** Callback handler for add admin smart contract execution

**File:** `server/callback_handler.go:23`

### Request Body
```json
{
  "port": "20000",
  "smart_contract_hash": "Qm..."
}
```

### Flow
1. **Parse Request** - Get port and contract hash
2. **Build URL** - Construct node URL
3. **Fetch Smart Contract Data** - Get latest block
4. **Parse Block Data** - Extract contract execution results
5. **Find Relevant Block** - Get latest non-genesis block
6. **Initialize WASM Module** - Create WASM bridge with custom host functions
7. **Execute WASM Function** - Process the smart contract data
8. **Log Result** - Output execution result

### Response
Implicit - Processes callback internally

---

## 9. POST /api/call-back-trigger

**Purpose:** Main callback handler for FT (Fungible Token) transfers triggered by blockchain

**File:** `server/server.go:520`

### Request Body
```json
{
  "port": "20000",
  "smart_contract_hash": "Qm..."
}
```

### Flow

#### Timing
- **Delay:** 1 second wait at start (gives API time to register pending request)
- **Called By:** Rubix blockchain after SignatureResponse
- **Purpose:** Process transfer and notify waiting API request

#### Steps
1. **Wait 1 Second** - Ensure API has registered pending request
2. **Parse Request** - Get port and contract hash
3. **Build URL** - Construct node API URL
4. **Fetch Smart Contract Data** - Get latest block from blockchain
5. **Parse Block Data** - Extract smart contract execution results
6. **Extract Relevant Data** - Get the latest block's SmartContractData
7. **Parse Function Name** - Identify which function was called
8. **Initialize WASM Module** - Create bridge with node address and quorum
9. **Execute Contract** - Run WASM function with contract data
10. **Parse Response** - Extract success/failure status and message
11. **Extract BlockId** - Get BlockId from latest block
12. **Notify TransferManager**:
    - Build CallbackResponse object
    - Call `manager.SendCallbackResponse(blockId, response)`
    - Updates database status
    - Sends response through channel to waiting API request
13. **Return Result** - Send execution result

### Callback Response Format
```json
{
  "success": true,
  "message": "FT Transferred Successfully",
  "data": {...},
  "block_id": "QmXyz...",
  "contract_data": "{...}"
}
```

### Response to Blockchain
```json
{
  "message": "DApp executed successfully",
  "data": {
    "status": true,
    "message": "FT Transferred Successfully",
    "result": {...}
  }
}
```

---

## Architecture Overview

### Synchronous Transfer Flow (fix/timeout)

```
Client Request → APITransferReward
     ↓
1. Execute Smart Contract (requestID returned)
     ↓
2. Sign Transaction (blockchain creates block)
     ↓
3. Register Pending Request (transactionID → channel)
     ↓                           ↓
     ↓                    Background Goroutine:
     ↓                    - Fetch BlockId
     ↓                    - Create DB record
     ↓                    - Update mapping
     ↓
4. Wait on Channel (1 hour timeout)
     ↓
     ├── Callback arrives (1-10s) → Success Response
     │        ↑
     │        │
     │   ftDappHandler (triggered by blockchain)
     │   - Waits 1s
     │   - Processes transfer
     │   - Sends response through channel
     │
     └── OR Timeout (1 hour) → Timeout Response
```

### Key Features
- **Synchronous API** - Client waits for blockchain confirmation
- **1-hour timeout** - Returns timeout response if callback doesn't arrive
- **Channel-based coordination** - Uses Go channels for callback communication
- **Database persistence** - All transfers stored in PostgreSQL
- **Background processing** - BlockId fetching doesn't block callback
- **Fallback mechanism** - Database update even if channel is closed

---

## Configuration Requirements

### Environment Variables (config)
```
AddActivityContract    = "Qm..." # Contract hash for adding activities
TransferContract       = "Qm..." # Contract hash for transfers
AddAdminContract       = "Qm..." # Contract hash for adding admins
```

### Node Configuration (nodes.toml)
```toml
[[nodes]]
name = "admin1"
did = "bafyb..."
port = "20000"
path = "/path/to/rubix/nodes"
```

---

## Error Handling

### Common Errors

1. **Invalid DID Format** - Returns 400 if DID doesn't start with "bafyb"
2. **Admin Not Found** - Returns 400 if admin DID not in config
3. **Empty Activity List** - Returns 400 if no activities provided
4. **Contract Hash Missing** - Returns 500 if contract hash not configured
5. **Transfer Timeout** - Returns 202 with timeout status after 1 hour
6. **Database Error** - Returns 500 if DB operations fail

### Status Codes
- `200 OK` - Successful operation
- `202 Accepted` - Transfer initiated but timed out (check status later)
- `400 Bad Request` - Invalid input
- `404 Not Found` - Transfer not found
- `500 Internal Server Error` - Server/blockchain error

---

## Testing Recommendations

### 1. Test Activity Addition
```bash
curl -X POST http://localhost:9000/api/activity/add \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": "yoga_class_001",
    "reward_points": 5,
    "admin_did": "bafyb..."
  }'
```

### 2. Test Reward Transfer
```bash
curl -X POST http://localhost:9000/api/rewards/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": ["yoga_class_001", "gym_session_002"],
    "user_did": "bafyb...",
    "admin_did": "bafyb..."
  }'
```

### 3. Check Transfer Status
```bash
curl http://localhost:9000/api/rewards/status/550e8400-e29b-41d4-a716-446655440000
```

### 4. Test Admin Addition
```bash
curl -X POST http://localhost:9000/api/admin/add \
  -H "Content-Type: application/json" \
  -d '{
    "new_admin_did": "bafyb...",
    "existing_admin_did": "bafyb..."
  }'
```

---

## Database Schema

### transfer_status table
```sql
CREATE TABLE transfer_status (
    request_id VARCHAR(36) PRIMARY KEY,
    blockchain_tx_id VARCHAR(255),
    block_id VARCHAR(255),
    activity_ids TEXT[],
    user_did VARCHAR(255),
    admin_did VARCHAR(255),
    reward_points INTEGER,
    status VARCHAR(50),
    message TEXT,
    error_details TEXT,
    contract_hash VARCHAR(255),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### Status Values
- `pending` - Transfer initiated, waiting for confirmation
- `success` - Transfer completed successfully
- `failed` - Transfer failed (contract execution error)
- `timeout` - Confirmation timeout (blockchain may still be processing)

---

## Monitoring & Debugging

### Log Messages to Watch For

**APITransferReward:**
- `🆔 Generated request_id: ...`
- `⚡ Registered pending request with temporary key`
- `⏳ Waiting for callback (timeout: 1 hour)...`
- `🎉 Received callback for transaction ...`

**ftDappHandler:**
- `🔔 ftDappHandler TRIGGERED - Callback received!`
- `⏳ Waiting 1 seconds for API setup to complete...`
- `🔑 ftDappHandler: Extracted BlockId: ...`
- `✅ ftDappHandler: Successfully notified pending request`

**TransferManager:**
- `Registered pending request: transactionID=..., blockId=...`
- `🔄 Updated pending request blockId mapping: ... → ...`
- `✅ Found pending request for blockId: ...`
- `Successfully sent callback response for blockId: ...`

### Health Checks
- Monitor pending request count: Check TransferManager logs
- Check database: Query transfer_status for stuck "pending" statuses
- Watch for stale requests: Cleanup runs every 2 minutes

---

## Branch Comparison: fix/timeout vs production-changes

### fix/timeout (Current)
- **Synchronous API** - Client waits for result
- **1-hour timeout** - Returns timeout response
- **TransferManager** - In-memory channel coordination
- **Response:** Success/timeout with full details

### production-changes
- **Asynchronous API** - Client gets immediate response
- **Queue-based** - Requests processed sequentially
- **Response:** Just request_id, status, message
- **Polling:** Client must poll /status endpoint

---

## Next Steps for Development

1. **Add Unit Tests** - Test each API endpoint individually
2. **Integration Tests** - Test full transfer flow with mock blockchain
3. **Load Testing** - Test concurrent transfers
4. **Metrics Collection** - Add Prometheus metrics for monitoring
5. **Rate Limiting** - Protect APIs from abuse
6. **API Documentation** - Generate Swagger/OpenAPI docs
7. **Webhook Support** - Allow clients to register callbacks instead of polling

---

**Generated:** 2026-01-28
**Branch:** fix/timeout
**Server Port:** 9000
