use rubixwasm_std::errors::WasmError;
use rubixwasm_std::contract_fn;
use rubixwasm_std::call_transfer_ft_api;
use rubixwasm_std::helpers::TransferFt;
use serde::{Deserialize, Serialize};

/// Whitelist of allowed names that can initiate ytoken transfers
/// Only these names are authorized to perform transfers through this contract
pub const WHITELIST: &[&str] = &["rubix1", "rubix2"];

/// Input structure for ytoken transfer
///
/// This matches the original transfer_sample_ft structure:
/// - `name`: Must be in WHITELIST to authorize the transfer
/// - `ft_info`: Contains all transfer details (receiver, ft_count, comment, etc.)
#[derive(Serialize, Deserialize)]
pub struct TransferYTokenInput {
    pub name: String,
    pub ft_info: TransferFt,
}

/// Main contract function for transferring ytoken (fungible tokens)
///
/// **KEY CHANGE:** Returns full blockchain response JSON instead of just "success"
///
/// Input JSON format:
/// {
///   "name": "rubix1",
///   "ft_info": {
///     "receiver": "bafybmidi...",
///     "ft_count": 10,
///     "comment": "Reward transfer",
///     "ft_name": "ytoken",
///     "creatorDID": "bafybmic...",
///     "sender": "bafybmic..."
///   }
/// }
///
/// Output: Full blockchain response
/// {
///   "status": true,
///   "message": "FT Transfer finished successfully in 11.071221792s with trnxid db7e190a...",
///   "result": null
/// }
///
/// # Validations:
/// 1. Name must be in WHITELIST
/// 2. Name cannot be empty
/// 3. Receiver DID cannot be empty
/// 4. Token count must be greater than 0
///
/// # Errors:
/// Returns WasmError if:
/// - Name is not in whitelist
/// - Any validation fails
/// - Transfer API call fails
#[contract_fn]
pub fn transfer_ytoken(transfer_ytoken_input: TransferYTokenInput) -> Result<String, WasmError> {
    let input_name = transfer_ytoken_input.name;
    let ft_transfer_info = transfer_ytoken_input.ft_info;

    // Validation 1: Check if name is empty
    if input_name.is_empty() {
        return Err(WasmError::from("Name cannot be empty"));
    }

    // Validation 2: Check if name is in whitelist
    if !WHITELIST.contains(&input_name.as_str()) {
        return Err(WasmError::from(format!(
            "Name '{}' is not authorized to transfer ytokens. Allowed names: {:?}",
            &input_name, WHITELIST
        )));
    }

    // Validation 3: Check if receiver is empty
    if ft_transfer_info.receiver.is_empty() {
        return Err(WasmError::from("Receiver DID cannot be empty"));
    }

    // Validation 4: Check if token count is valid
    if ft_transfer_info.ft_count <= 0 {
        return Err(WasmError::from(format!(
            "Token count must be greater than 0, got: {}",
            ft_transfer_info.ft_count
        )));
    }

    // Call the transfer API using rubixwasm-std helper
    // ✨ KEY CHANGE: This now returns the FULL blockchain response
    // Previously: returned just "success"
    // Now: returns complete JSON with timing, transaction ID, detailed message, etc.
    match call_transfer_ft_api(ft_transfer_info) {
        Ok(response) => {
            // Return the complete blockchain response as-is
            // Example response:
            // {"status":true, "message":"FT Transfer finished successfully in 11.071221792s with trnxid db7e190a...", "result":null}
            Ok(response)
        }
        Err(e) => {
            // Return error from the API call
            Err(e)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_whitelist_contains_rubix1() {
        assert!(WHITELIST.contains(&"rubix1"));
    }

    #[test]
    fn test_whitelist_contains_rubix2() {
        assert!(WHITELIST.contains(&"rubix2"));
    }

    #[test]
    fn test_whitelist_does_not_contain_invalid() {
        assert!(!WHITELIST.contains(&"invalid_name"));
    }
}
