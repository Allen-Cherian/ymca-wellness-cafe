# Local Setup Guide - Queue System Testing

This guide will help you set up and test the queue system on your local machine.

---

## 📋 Prerequisites

### 1. **Rubix Blockchain Nodes** (Required)

You need at least **2 Rubix nodes** running locally:
- **Admin Node**: For sending transactions
- **User Node**: For receiving rewards

**Check if nodes are running:**
```bash
# Check if Rubix nodes are running
ps aux | grep rubix

# Test node connectivity
curl http://localhost:20010/api/node-status  # Admin node
curl http://localhost:20012/api/node-status  # User node
```

**If nodes are NOT running:**
```bash
# Start your Rubix nodes (adjust paths as needed)
# Example:
cd ~/rubix-nodes/admin-node
./rubix run -p 20010 -s &

cd ~/rubix-nodes/user-node
./rubix run -p 20012 -s &
```

---

## 🔧 Step-by-Step Setup

### **Step 1: Create Configuration Directory**

```bash
cd /Users/allen/Professional/ymca-wellness-cafe/dappServer
mkdir -p .config
```

---

### **Step 2: Create `config.toml`**

This file maps DIDs (Decentralized Identifiers) to node ports.

**Create file:**
```bash
cat > .config/config.toml << 'EOF'
# Rubix Node Configuration
# Map each DID to its node port and path

[nodes.admin]
name = "admin"
port = "20010"
did = "bafybmie5va2cw6rlo47xxgbzor37bliaefxl3gwnl2p7g4xtspg4dpcoku"
path = "/Users/allen/rubix-nodes/admin-node"

[nodes.user]
name = "user"
port = "20012"
did = "bafybmihptfnp6bsszwwrxdfp7o7uwi2qfkz6ng5k3zqcnnm23qrbvacqse"
path = "/Users/allen/rubix-nodes/user-node"

# Add more nodes as needed
# [nodes.user2]
# name = "user2"
# port = "20014"
# did = "bafybmi..."
# path = "/Users/allen/rubix-nodes/user2-node"
EOF
```

**⚠️ IMPORTANT:** Replace the DIDs with your actual node DIDs!

**How to get your node DIDs:**
```bash
# For admin node
curl http://localhost:20010/api/get-did

# For user node
curl http://localhost:20012/api/get-did
```

---

### **Step 3: Create `.env` File**

This file stores smart contract hashes.

**Create file:**
```bash
cat > .config/.env << 'EOF'
# Smart Contract Hashes
# These are the deployed contract hashes on your blockchain

# Activity contract (for adding activities)
ADD_ACTIVITY_CONTRACT=QmYourActivityContractHash

# Transfer contract (for transferring rewards)
TRANSFER_CONTRACT=QmYourTransferContractHash

# Admin contract (for adding admins)
ADD_ADMIN_CONTRACT=QmYourAdminContractHash

# Update paths (for WASM contract callbacks)
ACTIVITY_UPDATE_PATH=/path/to/activity/updates
ADD_ADMIN_PATH=/path/to/admin/updates
EOF
```

**⚠️ IMPORTANT:** You need to deploy smart contracts first!

---

### **Step 4: Deploy Smart Contracts (If Not Already Done)**

Your contracts are already compiled in the `contracts/` directory.

**Deploy the transfer contract:**
```bash
curl -X POST http://localhost:9000/api/deploy-contract \
  -H "Content-Type: application/json" \
  -d '{
    "wasm_path": "./contracts/activity_contract.wasm",
    "lib_path": "./activity_contract/src/lib.rs",
    "deployer_did": "bafybmie5va2cw6rlo47xxgbzor37bliaefxl3gwnl2p7g4xtspg4dpcoku",
    "state_path": "./activity_contract/state.json"
  }'
```

**Save the returned contract hash** and add it to `.env` as `TRANSFER_CONTRACT`

---

### **Step 5: Start the DApp Server**

```bash
cd /Users/allen/Professional/ymca-wellness-cafe/dappServer

# Start the server
./dapp-server
```

**Expected output:**
```
Initializing database...
Database initialized successfully
🚀 Transfer queue worker started (sequential processing)
[GIN-debug] Listening and serving HTTP on :9000
```

**If you see errors:**
- `Error loading config file` → Check `.config/config.toml` exists
- `Error loading .env file` → Check `.config/.env` exists
- `Failed to initialize database` → Check write permissions

---

## 🧪 Testing the Queue System

### **Test 1: Simple Health Check**

```bash
# Check if server is running
curl http://localhost:9000/api/queue/metrics
```

**Expected response:**
```json
{
  "status": true,
  "data": {
    "queue_size": 0,
    "estimated_wait_sec": 0,
    "capacity": 1000,
    "available_slots": 1000
  }
}
```

---

### **Test 2: Submit a Transfer Request**

**⚠️ Replace DIDs with your actual values!**

```bash
curl -X POST http://localhost:9000/api/rewards/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": ["morning_yoga", "evening_run"],
    "user_did": "bafybmihptfnp6bsszwwrxdfp7o7uwi2qfkz6ng5k3zqcnnm23qrbvacqse",
    "admin_did": "bafybmie5va2cw6rlo47xxgbzor37bliaefxl3gwnl2p7g4xtspg4dpcoku"
  }'
```

**Expected response (< 100ms):**
```json
{
  "status": "queued",
  "message": "Transfer request queued for processing",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "rewards_to_award": 2,
    "activity_ids": ["morning_yoga", "evening_run"],
    "user_did": "bafybmi...",
    "admin_did": "bafybmi..."
  },
  "queue_info": {
    "position": 1,
    "estimated_wait_sec": 8
  },
  "check_status_url": "/api/rewards/status/550e8400-...",
  "note": "Use the check_status_url to poll for transfer completion"
}
```

**✅ Success!** You got an immediate response with a `request_id`.

---

### **Test 3: Check Transfer Status**

**Copy the `request_id` from the previous response:**

```bash
curl http://localhost:9000/api/rewards/status/550e8400-e29b-41d4-a716-446655440000
```

**Possible responses:**

**While queued:**
```json
{
  "status": true,
  "data": {
    "request_id": "550e8400-...",
    "status": "queued",
    "message": "Transfer request queued for processing",
    "queued_at": "2026-01-19T10:30:00Z"
  }
}
```

**While processing:**
```json
{
  "status": true,
  "data": {
    "request_id": "550e8400-...",
    "status": "processing",
    "message": "Smart contract execution in progress",
    "started_at": "2026-01-19T10:30:05Z"
  }
}
```

**When completed:**
```json
{
  "status": true,
  "data": {
    "request_id": "550e8400-...",
    "blockchain_tx_id": "bafkreiabcd...",
    "block_id": "block_abc123",
    "status": "success",
    "message": "FT Transferred Successfully",
    "activity_ids": ["morning_yoga", "evening_run"],
    "reward_points": 2,
    "queued_at": "2026-01-19T10:30:00Z",
    "started_at": "2026-01-19T10:30:05Z",
    "completed_at": "2026-01-19T10:30:15Z"
  }
}
```

---

### **Test 4: Monitor Server Logs**

In the terminal where the server is running, you should see:

```
═══════════════════════════════════════════════════════════
⚙️  Processing: request_id=550e8400-...
⏱️  Waited in queue: 100ms
📊 Queue size: 0
═══════════════════════════════════════════════════════════
🔗 Node URL: http://localhost:20010
📤 Step 1: Executing smart contract...
✅ Smart contract executed: blockchain_request_id=...
✍️  Step 2: Signing transaction...
✅ Transaction signed: blockchain_tx_id=...
🔍 Step 3: Extracting block ID...
✅ Block ID extracted: block_abc123
💾 Step 4: Updating database with blockchain IDs...
✅ Database updated with blockchain IDs
📞 Step 5: Registering for callback...
✅ Registered for callback, waiting...
⏳ Step 6: Waiting for callback (timeout: 3 minutes)...
🎉 Callback received: success=true
✅ Transfer SUCCEEDED: request_id=550e8400-...
✅ Completed: request_id=550e8400-...
⏱️  Processing time: 10.5s
```

---

### **Test 5: Load Test (10 Concurrent Requests)**

```bash
for i in {1..10}; do
  curl -X POST http://localhost:9000/api/rewards/transfer \
    -H "Content-Type: application/json" \
    -d "{
      \"activity_id\": [\"activity_$i\"],
      \"user_did\": \"bafybmihptfnp6bsszwwrxdfp7o7uwi2qfkz6ng5k3zqcnnm23qrbvacqse\",
      \"admin_did\": \"bafybmie5va2cw6rlo47xxgbzor37bliaefxl3gwnl2p7g4xtspg4dpcoku\"
    }" &
done
wait
```

**All 10 requests should return immediately with `status: "queued"`**

**Check queue:**
```bash
curl http://localhost:9000/api/queue/metrics
```

**You should see:**
```json
{
  "queue_size": 10,
  "estimated_wait_sec": 80
}
```

---

## 🐛 Troubleshooting

### **Issue 1: "Error loading config file"**

**Cause:** `config.toml` not found or invalid format

**Solution:**
```bash
# Check if file exists
ls -la .config/config.toml

# Check if it's valid TOML
cat .config/config.toml
```

---

### **Issue 2: "Node port not found for admin DID"**

**Cause:** DID in request doesn't match any DID in `config.toml`

**Solution:**
1. Get your actual DIDs:
   ```bash
   curl http://localhost:20010/api/get-did
   ```
2. Update `config.toml` with correct DIDs
3. Restart server

---

### **Issue 3: "Transfer contract hash not configured"**

**Cause:** `TRANSFER_CONTRACT` not set in `.env`

**Solution:**
1. Deploy the transfer contract (see Step 4 above)
2. Get the contract hash from the deployment response
3. Add to `.config/.env`:
   ```
   TRANSFER_CONTRACT=QmYourActualContractHash
   ```
4. Restart server

---

### **Issue 4: "Failed to execute smart contract"**

**Cause:** Rubix node not running or not accessible

**Solution:**
```bash
# Check if nodes are running
ps aux | grep rubix

# Check node connectivity
curl http://localhost:20010/api/node-status
curl http://localhost:20012/api/node-status

# If not running, start them
./start-rubix-nodes.sh  # Or your node startup script
```

---

### **Issue 5: All requests timeout**

**Cause:** Callback endpoint not reachable

**Solution:**
1. Check callback URL is registered:
   ```bash
   curl -X POST http://localhost:20010/api/register-callback-url \
     -H "Content-Type: application/json" \
     -d '{
       "CallBackURL": "http://localhost:9000/api/call-back-trigger",
       "SmartContractToken": "YOUR_CONTRACT_HASH"
     }'
   ```
2. Ensure firewall allows local connections
3. Check server logs for callback errors

---

## 📁 Directory Structure

```
dappServer/
├── .config/
│   ├── config.toml          # ← YOU NEED TO CREATE THIS
│   └── .env                 # ← YOU NEED TO CREATE THIS
├── contracts/
│   ├── activity_contract.wasm
│   └── second_contract.wasm
├── database/
│   └── db.go
├── server/
│   ├── server.go
│   ├── transfer_queue.go    # ← NEW (created by queue implementation)
│   └── transfer_manager.go
├── config/
│   └── config.go
├── main.go
├── transfer_status.db       # ← AUTO-CREATED on first run
└── dapp-server              # ← COMPILED BINARY
```

---

## ✅ Minimal Working Setup

**If you just want to test the queue mechanism (without real blockchain):**

You can create a **mock setup** for testing:

1. **Create config files with dummy data**
2. **Comment out blockchain calls** in `transfer_queue.go` (lines 142-180)
3. **Simulate success** by returning success immediately

This lets you test:
- ✅ Queue accepting requests
- ✅ Immediate API responses
- ✅ Status API
- ✅ Database operations
- ✅ Sequential processing

---

## 🎯 Verification Checklist

Before testing, ensure:

- [ ] Rubix nodes are running (`ps aux | grep rubix`)
- [ ] `.config/config.toml` exists with correct DIDs
- [ ] `.config/.env` exists with contract hashes
- [ ] Smart contracts are deployed
- [ ] Server compiles without errors (`go build`)
- [ ] Server starts successfully (`./dapp-server`)
- [ ] Queue metrics endpoint responds (`curl localhost:9000/api/queue/metrics`)

---

## 📞 Need Help?

**Common issues:**
1. **DIDs mismatch** → Most common issue, double-check DIDs
2. **Nodes not running** → Start Rubix nodes first
3. **Contracts not deployed** → Deploy contracts before testing
4. **Config files missing** → Create `.config/config.toml` and `.config/.env`

**Quick debug:**
```bash
# Check server status
curl http://localhost:9000/api/queue/metrics

# Check node status
curl http://localhost:20010/api/node-status

# View server logs
tail -f dapp-server.log  # If you're logging to file
```

---

Ready to test! Start with **Test 1** (health check) and work your way through. 🚀
