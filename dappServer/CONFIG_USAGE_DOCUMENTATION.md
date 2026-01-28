# Configuration Files Usage Documentation

This document explains where and how `config.toml` and `.env` files are used throughout the dApp server.

---

## Configuration Files Location

```
dappServer/
├── .config/
│   ├── config.toml    # Node configuration
│   └── .env          # Environment variables (contracts, paths, DB)
```

---

## 1. config.toml - Node Configuration

### Purpose
Stores information about Rubix blockchain nodes that the dApp server needs to interact with.

### File Location
`.config/config.toml`

### Structure
```toml
[nodes.node1]
name = "node2"
port = "20002"
did = "bafybmicjpbgevsegrj3hbdyeilprk5mdbtdrkiec7hgmi3kdrcfhexvk6q"
path = "/Users/allen/Professional/rubix-scripts/rubix-testnet-scripts/rubix-workspace/rubixgoplatform/mac"
```

### Data Schema
```go
type Node struct {
    Name string  // Node identifier
    Port string  // HTTP port for Rubix node API
    DID  string  // Decentralized Identifier (blockchain address)
    Path string  // File system path to node's data directory
}

type Config struct {
    Nodes map[string]Node  // Key: "node1", "node2", etc.
}
```

### Loading
**File:** `config/config.go:32`
```go
func LoadConfig(filepath string) {
    once.Do(func() {
        instance = &Config{}
        toml.DecodeFile(filepath, instance)
    })
}
```

**Initialization:** `main.go:24`
```go
const CONFIG_PATH = ".config/config.toml"
config.LoadConfig(CONFIG_PATH)
```

---

## 2. .env - Environment Variables

### Purpose
Stores smart contract hashes, file paths, and database configuration.

### File Location
`.config/.env`

### Content
```bash
# Smart Contract Hashes (deployed on Rubix blockchain)
ADD_ACTIVITY_CONTRACT="QmeM75ukNH2837ZZq2poY7ztBdh11ShGNhpDP8FdfkqGpe"
TRANSFER_CONTRACT="Qmb8zejwY1XjXPxmGKgCiWVVt9xgENtwDnJkEkf5SEzerB"
ADD_ADMIN_CONTRACT="QmUeuMj5Lngz2QSa1TP5vZUs9CqvHtqvFAxidLjLe5Dftd"

# File Paths for local data storage
ADD_ADMIN_PATH="/Users/allen/Professional/ymca-wellness-cafe/dappServer/addAdmin_git.json"
ACTIVITY_UPDATE_PATH="/Users/allen/Professional/ymca-wellness-cafe/dappServer/test_git.json"

# Database Configuration (PostgreSQL)
# NOTE: Currently not used in fix/timeout branch (using SQLite instead)
DB_HOST=localhost
DB_PORT=5432
DB_USER=allen
DB_PASSWORD=
DB_NAME=dapp_server
DB_SSL_MODE=disable
```

### Data Schema
```go
type EnvConfig struct {
    AddActivityContract string  // Contract hash for adding activities
    AddAdminContract    string  // Contract hash for adding admins
    TransferContract    string  // Contract hash for FT transfers
    ActivityUpdatePath  string  // Local file path for activity updates
    AdminUpdatePath     string  // Local file path for admin updates
}
```

### Loading
**File:** `config/config.go:109`
```go
func LoadEnvConfig() *EnvConfig {
    envOnce.Do(func() {
        // Load from .config/.env
        godotenv.Load(".config/.env")

        envInstance = &EnvConfig{
            AddActivityContract: os.Getenv("ADD_ACTIVITY_CONTRACT"),
            TransferContract:    os.Getenv("TRANSFER_CONTRACT"),
            AddAdminContract:    os.Getenv("ADD_ADMIN_CONTRACT"),
            ActivityUpdatePath:  os.Getenv("ACTIVITY_UPDATE_PATH"),
            AdminUpdatePath:     os.Getenv("ADD_ADMIN_PATH"),
        }
    })
    return envInstance
}
```

**Initialization:** `main.go:25`
```go
config.LoadEnvConfig()
```

---

## Usage Throughout the Application

### 🔵 config.toml Usage (Nodes)

The node configuration is used to:
1. **Map DID to Port** - Find which node to contact for a given admin/user
2. **Map DID to Node Name** - Get node identifier
3. **Map Port to Path** - Locate WASM contract files on filesystem
4. **Map Port to Node Name** - Identify which node is calling back

#### Lookup Functions

**File:** `config/config.go`

| Function | Purpose | Used By |
|----------|---------|---------|
| `GetNodeNameByPort(port)` | Port → Node Name | WASM path resolution, callbacks |
| `GetPathByPort(port)` | Port → File Path | WASM contract loading |
| `GetNodeNameByDid(did)` | DID → Node Name | Contract execution |
| `GetPortByNodeName(name)` | Name → Port | API URL construction |
| `GetPortByDid(did)` | DID → Port | Most common: find node for admin |

---

### Usage in API Endpoints

#### 1️⃣ POST /api/rewards/transfer
**File:** `server/server.go:131`
```go
cfg, err := config.GetConfig()
nodePort, exists := config.GetPortByDid(cfg, req.AdminDID)
url := fmt.Sprintf("http://localhost:%s", nodePort)
```
**Purpose:** Find admin's node port to execute transfer contract

**Also uses .env:**
```go
transferContractHash := config.GetEnvConfig().TransferContract
```

---

#### 2️⃣ POST /api/activity/add
**File:** `server/server.go:335`
```go
cfg, err := config.GetConfig()
nodePort, exists := config.GetPortByDid(cfg, req.AdminDID)
url := fmt.Sprintf("http://localhost:%s", nodePort)
```
**Purpose:** Find admin's node to add activity to blockchain

**Also uses .env:**
```go
smartContractHash := config.GetEnvConfig().AddActivityContract
```

---

#### 3️⃣ POST /api/admin/add
**File:** `server/handler.go:28`
```go
cfg, err := config.GetConfig()
nodePort, exists := config.GetPortByDid(cfg, req.ExistingAdminDID)
url := fmt.Sprintf("http://localhost:%s", nodePort)
```
**Purpose:** Find existing admin's node to add new admin

**Also uses .env:**
```go
smartContractHash := config.GetEnvConfig().AddAdminContract
```

---

#### 4️⃣ POST /api/execute-contract
**File:** `server/api.go:38`
```go
cfg, err := config.GetConfig()
nodeName, exist := config.GetNodeNameByDid(cfg, req.ExecutorDid)
port, exist := config.GetPortByNodeName(cfg, nodeName)
url := fmt.Sprintf("http://localhost:%s", port)
```
**Purpose:** Find executor's node to run contract

---

#### 5️⃣ POST /api/deploy-contract
**File:** `server/api.go:87`
```go
cfg, err := config.GetConfig()
nodeName, exist := config.GetNodeNameByDid(cfg, req.DeployerDid)
```
**Purpose:** Find deployer's node name for contract deployment

---

### Usage in Callbacks & WASM Loading

#### 6️⃣ getWasmContractPath()
**File:** `server/server.go:792`
```go
cfg, err := config.GetConfig()
path, exists := config.GetPathByPort(cfg, port)
nodeName, exists := config.GetNodeNameByPort(cfg, port)

// Construct path to WASM contract
contractDir := filepath.Join(path, nodeName, "SmartContract", contractHash)
```
**Purpose:** Locate WASM contract files on filesystem for execution

**Example Path:**
```
/Users/allen/.../rubixgoplatform/mac/node2/SmartContract/QmXyz.../contract.wasm
```

---

#### 7️⃣ WriteToJsonFile (Host Function)
**File:** `rubix-interaction/linker.go:101`
```go
switch functionName {
case "add_activity":
    filePath = config.GetEnvConfig().ActivityUpdatePath
case "add_admin":
    filePath = config.GetEnvConfig().AdminUpdatePath
}
```
**Purpose:** WASM contracts write results to local JSON files

**Files:**
- `test_git.json` - Activity data
- `addAdmin_git.json` - Admin data

---

### Usage in Rubix Interaction Layer

#### 8️⃣ Deploy Contract
**File:** `rubix-interaction/deploy.go:22`
```go
cfg, err := config.GetConfig()
nodeURL, exists := cfg.Nodes[nodeName]
apiURL := fmt.Sprintf("http://localhost:%s/api/deploy-smart-contract", nodeURL.Port)
```
**Purpose:** Get node URL for contract deployment

---

#### 9️⃣ Execute Contract
**File:** `rubix-interaction/execute.go:23`
```go
cfg, err := config.GetConfig()
nodeURL, exists := cfg.Nodes[nodeName]
apiURL := fmt.Sprintf("http://localhost:%s/api/execute-smart-contract", nodeURL.Port)
```
**Purpose:** Get node URL for contract execution

---

## 🟢 .env Usage (Contract Hashes & Paths)

### Smart Contract Hashes

These are IPFS/Rubix hashes of deployed smart contracts:

| Variable | Purpose | Used In |
|----------|---------|---------|
| `ADD_ACTIVITY_CONTRACT` | Add rewards for activities | `/api/activity/add` |
| `TRANSFER_CONTRACT` | Transfer FT tokens to users | `/api/rewards/transfer` |
| `ADD_ADMIN_CONTRACT` | Add new admins to system | `/api/admin/add` |

### File Paths

| Variable | Purpose | Used In |
|----------|---------|---------|
| `ACTIVITY_UPDATE_PATH` | Store activity execution results | WASM host function |
| `ADD_ADMIN_PATH` | Store admin addition results | WASM host function |

### Database Configuration

**NOTE:** In the current `fix/timeout` branch, these are **NOT USED**. The application uses SQLite instead.

```bash
DB_HOST=localhost       # Not used
DB_PORT=5432           # Not used
DB_USER=allen          # Not used
DB_PASSWORD=           # Not used
DB_NAME=dapp_server    # Not used
DB_SSL_MODE=disable    # Not used
```

**Current Database:** SQLite at `./transfer_status.db` (see `main.go:12`)

**Note:** The `production-changes` branch may use PostgreSQL. Check that branch's database implementation.

---

## Configuration Flow Diagram

```
Application Startup (main.go)
    ↓
1. config.LoadConfig(".config/config.toml")
    ├─ Loads node information
    └─ Singleton pattern (loaded once)
    ↓
2. config.LoadEnvConfig()
    ├─ Loads .env from ".config/.env"
    └─ Reads contract hashes, paths, DB config
    ↓
3. API Request Received
    ↓
4. Lookup Configuration
    ├─ GetConfig() → Get nodes
    ├─ GetEnvConfig() → Get contracts/paths
    └─ Resolve DID → Port → Node URL
    ↓
5. Execute Blockchain Operation
    ├─ Call Rubix node API
    └─ Execute smart contract
    ↓
6. Process Callback
    ├─ Lookup WASM path using config
    └─ Execute contract logic
```

---

## Configuration Access Patterns

### Pattern 1: DID → Node URL
```go
cfg, _ := config.GetConfig()
port, _ := config.GetPortByDid(cfg, adminDID)
url := fmt.Sprintf("http://localhost:%s", port)
```
**Used for:** Executing contracts on admin's node

---

### Pattern 2: Get Contract Hash
```go
contractHash := config.GetEnvConfig().TransferContract
```
**Used for:** Referencing deployed smart contracts

---

### Pattern 3: WASM Path Resolution
```go
cfg, _ := config.GetConfig()
path, _ := config.GetPathByPort(cfg, port)
nodeName, _ := config.GetNodeNameByPort(cfg, port)
wasmPath := filepath.Join(path, nodeName, "SmartContract", hash, "contract.wasm")
```
**Used for:** Loading WASM contract files for execution

---

## Singleton Pattern

Both configurations use singleton pattern to ensure they're loaded only once:

```go
var (
    instance *Config
    once     sync.Once
)

func LoadConfig(filepath string) {
    once.Do(func() {
        // Load only once
    })
}
```

**Benefits:**
- Thread-safe
- Loaded once at startup
- No redundant file I/O
- Consistent configuration across application

---

## Configuration Files Summary

### config.toml (Nodes)
- **What:** Rubix blockchain node connection details
- **Contains:** Node name, port, DID, filesystem path
- **Used for:**
  - Finding which node to contact for a DID
  - Locating WASM contracts on filesystem
  - Building API URLs for blockchain operations

### .env (Environment Variables)
- **What:** Application secrets and configuration
- **Contains:**
  - Smart contract hashes (3 contracts)
  - Local file paths (2 paths)
  - Database config (not used in fix/timeout)
- **Used for:**
  - Referencing deployed smart contracts
  - Writing WASM execution results
  - (Future) Database connection

---

## Example: Full Request Flow

**Request:** Transfer 3 rewards to a user

```
1. Client sends POST /api/rewards/transfer
   {
     "activity_id": ["act1", "act2", "act3"],
     "user_did": "bafyb...",
     "admin_did": "bafyb...AdminDID"
   }

2. Server looks up admin's node:
   config.GetConfig()
   config.GetPortByDid("bafyb...AdminDID") → "20002"
   url = "http://localhost:20002"

3. Server gets transfer contract:
   config.GetEnvConfig().TransferContract → "Qmb8zej..."

4. Server executes contract on admin's node:
   POST http://localhost:20002/api/execute-smart-contract
   {
     "contract_hash": "Qmb8zej...",
     "executor_did": "bafyb...AdminDID",
     "contract_input": "{\"transfer_sample_ft\": {...}}"
   }

5. Blockchain triggers callback to dApp server:
   POST http://localhost:9000/api/call-back-trigger
   {
     "port": "20002",
     "smart_contract_hash": "Qmb8zej..."
   }

6. Callback handler loads WASM:
   config.GetConfig()
   config.GetPathByPort("20002") → "/Users/allen/.../mac"
   config.GetNodeNameByPort("20002") → "node2"

   wasmPath = "/Users/allen/.../mac/node2/SmartContract/Qmb8zej.../contract.wasm"

7. Execute WASM and return result
```

---

## Environment-Specific Configuration

### Development
```toml
# .config/config.toml
[nodes.node1]
name = "node2"
port = "20002"
did = "bafyb..."
path = "/Users/allen/Professional/rubix-scripts/.../mac"
```

### Production (Example)
```toml
# .config/config.toml
[nodes.node1]
name = "prod-node-1"
port = "20000"
did = "bafyb...ProductionDID"
path = "/var/rubix/nodes"

[nodes.node2]
name = "prod-node-2"
port = "20001"
did = "bafyb...ProductionDID2"
path = "/var/rubix/nodes"
```

---

## Configuration Validation

### Missing config.toml
```
Error loading config file: no such file or directory
Fatal error at startup
```

### Missing .env
```
Error loading .env file: no such file or directory
Fatal error at startup
```

### Invalid DID in request
```
{
  "error": "Validation failed",
  "message": "admin_did not found in configured nodes"
}
```

### Missing contract hash
```
{
  "error": "Transfer contract hash not configured"
}
```

---

## Best Practices

### ✅ Do's
1. **Version control config.toml.example** - Keep template in git
2. **Never commit .env** - Contains secrets (contract hashes are public but paths are local)
3. **Validate at startup** - Ensure all required configs are present
4. **Use absolute paths** - Avoid relative paths in config files
5. **Document all variables** - Maintain this documentation

### ❌ Don'ts
1. **Don't hardcode values** - Always use config files
2. **Don't modify configs at runtime** - Read-only after startup
3. **Don't store secrets in code** - Use .env
4. **Don't skip validation** - Check DIDs exist before using

---

## Troubleshooting

### Issue: "Node port not found for admin DID"
**Cause:** DID in request not in config.toml
**Solution:** Add node to config.toml or use correct DID

### Issue: "Smart contract hash is not set"
**Cause:** Missing variable in .env
**Solution:** Add contract hash to .env

### Issue: "Failed to get wasm path"
**Cause:** Wrong port or path in config.toml
**Solution:** Verify node's path and port are correct

### Issue: "Unable to fetch latest smart contract data"
**Cause:** Node not running or wrong port
**Solution:** Start Rubix node, verify port in config.toml

---

## Future Improvements

1. **Configuration Validation:** Add validation at startup
2. **Environment-specific configs:** Support dev/staging/prod
3. **Hot reload:** Allow config updates without restart
4. **Configuration API:** Add endpoint to view/update config
5. **PostgreSQL Support:** Implement DB config usage
6. **Secrets Management:** Use vault for sensitive data
7. **Configuration UI:** Web interface for managing config

---

**Generated:** 2026-01-28
**Branch:** fix/timeout
**Config Version:** 1.0
