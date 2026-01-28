package rubix_interaction

// DeploymentResult represents the result of a contract deployment
type DeploymentResult struct {
	ContractHash string
	Success      bool
	Message      string
}

// DeploymentStage represents a stage in the deployment process
type DeploymentStage int

const (
	StageBuild DeploymentStage = iota
	StageGenerate
	StageDeploy
)

// StageCallback is a function that gets called when a stage begins
type StageCallback func(stage DeploymentStage)

// ExecutionResult represents the result of a contract execution
type ExecutionResult struct {
	Success bool
	Message string
	ContractResult string
}

type SmartContractResult struct {
	Id          string `json:"id"`
	Mode        int    `json:"mode"`
	Hash        string `json:"string"`
	OnlyPrivKey bool   `json:"only_priv_key"`
}

// SmartContractAPIResponseV1 represents the standard API response structure
type SmartContractAPIResponseV1 struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

// SmartContractAPIResponseV1 represents the standard API response structure
type SmartContractAPIResponseV2 struct {
	Status  bool                `json:"status"`
	Message string              `json:"message"`
	Result  SmartContractResult `json:"result"`
}

type SmartContractBlock struct {
	BlockNo           string `json:"BlockNo"`
	BlockId           string `json:"BlockId"`
	SmartContractData string `json:"SmartContractData"`
}

// ContractExecuteResponse represents the blockchain's response with both transaction_id and block_id
// This is returned by the new signature-response API that provides both values together
type ContractExecuteResponse struct {
	TransactionId string `json:"transaction_id"`
	BlockId       string `json:"block_id"`
}

// SmartContractAPIResponseV3 represents the new API response structure with structured result
// Use this with SignatureResponseV2() to get both transaction_id and block_id in one call
type SmartContractAPIResponseV3 struct {
	Status  bool                    `json:"status"`
	Message string                  `json:"message"`
	Result  ContractExecuteResponse `json:"result"`
}
