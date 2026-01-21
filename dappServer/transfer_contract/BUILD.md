# Building and Deploying transfer_contract

## Prerequisites

1. **Install Rust** (if not already installed):
   ```bash
   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
   ```

2. **Add WASM target**:
   ```bash
   rustup target add wasm32-unknown-unknown
   ```

## Build Steps

### 1. Build the WASM Contract

```bash
cd transfer_contract
cargo build --release --target wasm32-unknown-unknown
```

### 2. Verify Build Output

The compiled WASM file will be at:
```
target/wasm32-unknown-unknown/release/transfer_contract.wasm
```

Check the file size (should be optimized):
```bash
ls -lh target/wasm32-unknown-unknown/release/transfer_contract.wasm
```

### 3. Deploy to Rubix Node

**Deploy the contract:**
```bash
curl -X POST http://localhost:20050/api/deploy-smart-contract \
  -H "Content-Type: multipart/form-data" \
  -F "smart_contract_token=@target/wasm32-unknown-unknown/release/transfer_contract.wasm" \
  -F "raw_code_path=@src/lib.rs" \
  -F "schema_file_path=@state.json" \
  -F "did=bafybmicjpbgevsegrj3hbdyeilprk5mdbtdrkiec7hgmi3kdrcfhexvk6q" \
  -F "binaryCodePath=transfer_contract.wasm" \
  -F "rawCodePath=lib.rs" \
  -F "schemaFilePath=state.json"
```

**Response will contain:**
```json
{
  "status": true,
  "message": "Smart contract deployed successfully",
  "result": {
    "smart_contract_hash": "Qm..."
  }
}
```

### 4. Update Environment Configuration

Copy the `smart_contract_hash` from the deployment response and update your `.env` file:

```bash
TRANSFER_CONTRACT=Qm...  # New hash from deployment
```

### 5. Restart the Server

```bash
cd ../
./dapp-server
```

## Testing the New Contract

### Test Transfer Request

**IMPORTANT:** The contract now uses the same input structure as the original `transfer_sample_ft`:
- `name`: Must be in WHITELIST ("rubix1" or "rubix2")
- `ft_info`: Contains transfer details

```bash
curl -X POST http://localhost:9000/api/rewards/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": ["test_activity"],
    "admin_did": "bafybmicjpbgevsegrj3hbdyeilprk5mdbtdrkiec7hgmi3kdrcfhexvk6q",
    "user_did": "bafybmidihrblwdyauw7q7sw5b4dch77zjuiqqyqtwxcs2j3nnfftxjkbeq"
  }'
```

**Note:** The Go server will format the input correctly for the WASM contract.

### Check Status

Use the `request_id` from the response:
```bash
curl http://localhost:9000/api/rewards/status/<request_id>
```

### Verify Full Response

Check server logs for the enhanced response:
```
Response Body in signature response : {"status":true,"message":"FT Transfer finished successfully in 11.071221792s with trnxid db7e190a...","result":null}
----------- FT Execution Result: {"status":true,"message":"FT Transfer finished successfully..."}
```

**Before (old contract):**
```
----------- FT Execution Result: success
```

**After (new contract with transfer_ytoken):**
```
----------- FT Execution Result: {"status":true,"message":"FT Transfer finished successfully in 11.071221792s with trnxid db7e190a..."}
```

## Rebuild After Changes

If you make changes to the contract:

```bash
cd transfer_contract
cargo build --release --target wasm32-unknown-unknown
```

Then re-deploy following steps 3-5 above.

## Troubleshooting

### Build Errors

If you get dependency errors:
```bash
cargo update
cargo clean
cargo build --release --target wasm32-unknown-unknown
```

### Contract Deployment Fails

- Verify the Rubix node is running: `curl http://localhost:20050/api/status`
- Check the DID exists and has permissions
- Verify file paths are correct

### Contract Not Executing

- Check the contract hash in `.env` matches the deployed hash
- Restart the server after updating `.env`
- Check server logs for contract loading errors

## File Structure

```
transfer_contract/
├── Cargo.toml              # Rust project configuration
├── src/
│   └── lib.rs              # Main contract code with transfer_ytoken function
├── state.json              # Contract state schema (empty for stateless)
├── BUILD.md                # This file
└── target/                 # Build output (generated)
    └── wasm32-unknown-unknown/
        └── release/
            └── transfer_contract.wasm  # Compiled WASM binary
```

## What Changed from Old Contract

### OLD CONTRACT (returned "success"):
```rust
#[contract_fn]
pub fn transfer_sample_ft(transfer_sample_ft_req: TransferSampleFTReq) -> Result<String, WasmError> {
    let input_name = transfer_sample_ft_req.name;

    if !WHITELIST.contains(&input_name.as_str()) {
        return Err(WasmError::from(format!("name {} is not allowed", &input_name)));
    }

    let ft_transfer_info = transfer_sample_ft_req.ft_info;

    match call_transfer_ft_api(ft_transfer_info) {
        Ok(resp) => Ok(resp),  // ✅ Actually already returned full response!
        Err(e) => Err(e)
    }
}
```

### NEW CONTRACT (same logic, renamed function + enhanced validations):
```rust
#[contract_fn]
pub fn transfer_ytoken(transfer_ytoken_input: TransferYTokenInput) -> Result<String, WasmError> {
    let input_name = transfer_ytoken_input.name;

    // Additional validation: check empty name
    if input_name.is_empty() {
        return Err(WasmError::from("Name cannot be empty"));
    }

    // Whitelist check (same as original)
    if !WHITELIST.contains(&input_name.as_str()) {
        return Err(WasmError::from(format!("Name '{}' is not authorized", &input_name)));
    }

    let ft_transfer_info = transfer_ytoken_input.ft_info;

    // Additional validation: check empty receiver
    if ft_transfer_info.receiver.is_empty() {
        return Err(WasmError::from("Receiver DID cannot be empty"));
    }

    // Additional validation: check token count
    if ft_transfer_info.token_count <= 0.0 {
        return Err(WasmError::from("Token count must be greater than 0"));
    }

    // Same API call as original
    match call_transfer_ft_api(ft_transfer_info) {
        Ok(response) => Ok(response),  // ✅ Returns complete JSON
        Err(e) => Err(e)
    }
}
```

### Key Differences:
1. ✅ **Function renamed**: `transfer_sample_ft` → `transfer_ytoken`
2. ✅ **Input struct renamed**: `TransferSampleFTReq` → `TransferYTokenInput`
3. ✅ **Additional validations**: Empty name, empty receiver, token count > 0
4. ✅ **Better error messages**: More descriptive validation errors
5. ✅ **Same structure**: Uses `rubixwasm-std`, `call_transfer_ft_api`, `TransferFt`, `WHITELIST`

## Next Steps

After successful deployment and testing:

1. ✅ Verify full response appears in logs
2. ✅ Check database `message` field has detailed information
3. ✅ Test multiple concurrent transfers
4. ✅ Monitor for any errors
5. ✅ Update client applications to use detailed messages if needed

## Complete Flow (After This Change)

```
1. Client → POST /api/rewards/transfer
2. Server → Queue request (immediate response with request_id)
3. Worker → Picks up job from queue
4. Worker → Calls Rubix ExecuteSmartContract API
5. Rubix → Processes blockchain transaction (~11 seconds)
6. Rubix → Triggers callback to server
7. Server → ftDappHandler loads WASM contract (NEW contract)
8. WASM → Executes transfer_ytoken function
9. WASM → Calls host_do_transfer_ft_api_call
10. Host → Calls SignatureResponse API
11. Rubix → Returns full JSON: {"status":true, "message":"FT Transfer finished successfully in 11.071221792s with trnxid ...", "result":null}
12. Host → Returns full JSON to WASM (not "success") ✅
13. WASM → transfer_ytoken returns full JSON to ftDappHandler ✨ NEW!
14. ftDappHandler → Parses complete response
15. ftDappHandler → Updates database with detailed message
16. Client → Polls status endpoint and gets full details
```

**The chain is now complete!** 🎉
