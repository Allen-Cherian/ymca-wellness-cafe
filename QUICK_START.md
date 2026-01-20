# 🚀 Quick Start Guide - 5 Minutes Setup

Follow these steps to get the queue system running locally in ~5 minutes.

---

## Step 1: Check Prerequisites (30 seconds)

```bash
# Check if Rubix nodes are running
ps aux | grep rubix

# If not running, start them:
# cd ~/your-rubix-node-path && ./rubix run -p 20010 -s &
```

---

## Step 2: Create Configuration Files (2 minutes)

```bash
cd /Users/allen/Professional/ymca-wellness-cafe/dappServer

# Create config directory
mkdir -p .config

# Copy example files
cp .config/config.toml.example .config/config.toml
cp .config/.env.example .config/.env
```

**Edit `.config/config.toml`:**
```bash
nano .config/config.toml
```

**Update these lines with your actual values:**
```toml
[nodes.admin]
port = "20010"                    # Your admin node port
did = "YOUR_ACTUAL_ADMIN_DID"     # ⚠️ CHANGE THIS

[nodes.user]
port = "20012"                    # Your user node port
did = "YOUR_ACTUAL_USER_DID"      # ⚠️ CHANGE THIS
```

**Get your DIDs:**
```bash
curl http://localhost:20010/api/get-did  # Admin DID
curl http://localhost:20012/api/get-did  # User DID
```

**Edit `.config/.env`:**
```bash
nano .config/.env
```

**Update this line:**
```bash
TRANSFER_CONTRACT=YOUR_DEPLOYED_CONTRACT_HASH  # ⚠️ CHANGE THIS
```

**If you don't have a contract hash yet, you'll need to deploy one (see LOCAL_SETUP_GUIDE.md)**

---

## Step 3: Start the Server (10 seconds)

```bash
./dapp-server
```

**You should see:**
```
Initializing database...
Database initialized successfully
🚀 Transfer queue worker started (sequential processing)
[GIN-debug] Listening and serving HTTP on :9000
```

---

## Step 4: Test It! (1 minute)

**Test 1: Health Check**
```bash
curl http://localhost:9000/api/queue/metrics
```

**Expected:**
```json
{"status": true, "data": {"queue_size": 0, ...}}
```

**Test 2: Submit a Transfer (⚠️ Use YOUR DIDs)**
```bash
curl -X POST http://localhost:9000/api/rewards/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": ["test_activity"],
    "user_did": "YOUR_USER_DID",
    "admin_did": "YOUR_ADMIN_DID"
  }'
```

**Expected (< 100ms response):**
```json
{
  "status": "queued",
  "request_id": "550e8400-...",
  "check_status_url": "/api/rewards/status/550e8400-..."
}
```

**Test 3: Check Status**
```bash
# Copy the request_id from above
curl http://localhost:9000/api/rewards/status/550e8400-e29b-41d4-a716-446655440000
```

---

## ✅ Success Criteria

- [ ] Server starts without errors
- [ ] Queue metrics endpoint returns data
- [ ] Transfer request returns immediately with `request_id`
- [ ] Status endpoint shows transfer status
- [ ] Server logs show sequential processing

---

## 🐛 Quick Troubleshooting

**Error: "Error loading config file"**
```bash
# Check if file exists
ls -la .config/config.toml

# If not, copy from example
cp .config/config.toml.example .config/config.toml
```

**Error: "Node port not found for admin DID"**
```bash
# Your DID doesn't match config.toml
# Get the correct DID:
curl http://localhost:20010/api/get-did

# Update config.toml with this DID
nano .config/config.toml
```

**Error: "Transfer contract hash not configured"**
```bash
# You need to deploy a contract first
# See LOCAL_SETUP_GUIDE.md for deployment instructions
```

**Error: "Failed to execute smart contract"**
```bash
# Check if Rubix node is running
curl http://localhost:20010/api/node-status

# If not, start it
cd ~/your-rubix-node && ./rubix run -p 20010 -s
```

---

## 📖 Next Steps

- **Full setup guide**: See `LOCAL_SETUP_GUIDE.md`
- **Queue implementation details**: See `QUEUE_IMPLEMENTATION_GUIDE.md`
- **Load testing**: See examples in `LOCAL_SETUP_GUIDE.md`

---

## 🎯 Minimal Test Without Blockchain

If you just want to test the queue mechanism without blockchain:

**1. Modify `transfer_queue.go` to simulate success:**
```go
// In processTransfer(), comment out lines 142-240 and add:
time.Sleep(2 * time.Second)  // Simulate processing
database.UpdateTransferStatus(requestID, map[string]interface{}{
    "status": "success",
    "message": "Simulated success",
    "completed_at": time.Now(),
})
```

**2. Start server and test:**
```bash
./dapp-server

# Test with multiple requests
for i in {1..10}; do
  curl -X POST http://localhost:9000/api/rewards/transfer \
    -H "Content-Type: application/json" \
    -d "{\"activity_id\": [\"test_$i\"], \"user_did\": \"test\", \"admin_did\": \"test\"}" &
done
```

All 10 should return immediately, then process sequentially!

---

**You're ready to go! 🚀**
