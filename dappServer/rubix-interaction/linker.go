package rubix_interaction

import (
	"dapp-server/config"
	"dapp-server/wasmbridge"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bytecodealliance/wasmtime-go"
)

type Activity struct {
	ActivityID   string `json:"activity_id"`
	BlockHash    string `json:"block_hash"`
	RewardPoints int    `json:"reward_points"`
}

type AddAdmin struct {
	AdminDID string `json:"admin_did"`
}

type WriteToJsonFile struct {
	allocFunc *wasmtime.Func
	memory    *wasmtime.Memory
}

func NewWriteToJsonFile() *WriteToJsonFile {
	return &WriteToJsonFile{}
}

func (h *WriteToJsonFile) Name() string {
	return "write_to_json_file"
}

func (h *WriteToJsonFile) FuncType() *wasmtime.FuncType {
	return wasmtime.NewFuncType(
		[]*wasmtime.ValType{
			wasmtime.NewValType(wasmtime.KindI32), // data_ptr
			wasmtime.NewValType(wasmtime.KindI32), // data_len
			wasmtime.NewValType(wasmtime.KindI32), // file_path_ptr
			wasmtime.NewValType(wasmtime.KindI32), // file_path_len
		},
		[]*wasmtime.ValType{wasmtime.NewValType(wasmtime.KindI32)}, // return i32
	)
}

func (h *WriteToJsonFile) Initialize(allocFunc, deallocFunc *wasmtime.Func, memory *wasmtime.Memory, nodeAddress string, quorumType int) {
	h.allocFunc = allocFunc
	h.memory = memory
}

func (h *WriteToJsonFile) Callback() wasmbridge.HostFunctionCallBack {
	return h.callback
}

func (h *WriteToJsonFile) callback(
	caller *wasmtime.Caller,
	args []wasmtime.Val,
) ([]wasmtime.Val, *wasmtime.Trap) {
	// Extract input arguments
	inputArgs, outputArgs := wasmbridge.HostFunctionParamExtraction(args, true, true)

	// Extract data and file path from WASM memory
	dataBytes, memory, err := wasmbridge.ExtractDataFromWASM(caller, inputArgs) // Extract data
	if err != nil {
		fmt.Println("Failed to extract data from WASM", err)
		return wasmbridge.HandleError(err.Error())
	}

	// filePathBytes, _, err := wasmbridge.ExtractDataFromWASM(caller, inputArgs) // Extract file path
	// if err != nil {
	// 	fmt.Println("Failed to extract file path from WASM", err)
	// 	return wasmbridge.HandleError(err.Error())
	// }
	h.memory = memory

	// data := string(dataBytes)
	// filePath := string(filePathBytes)

	// Parse the data into JSON (if necessary) and write it to a file
	var rawData map[string]interface{}
	if err := json.Unmarshal(dataBytes, &rawData); err != nil {
		fmt.Printf("Failed to parse incoming JSON: %v\n", err)
		return wasmbridge.HandleError(err.Error())
	}

	var jsonData interface{}
	var filePath string

	// Step 2: Dynamically identify the type and determine the file path
	if _, ok := rawData["activity_id"]; ok {
		var activity Activity
		if err := json.Unmarshal(dataBytes, &activity); err != nil {
			fmt.Printf("Failed to unmarshal as Activity: %v\n", err)
			return wasmbridge.HandleError(err.Error())
		}
		jsonData = activity
		filePath = config.GetEnvConfig().ActivityUpdatePath // File for Activity data
	} else if _, ok := rawData["admin_did"]; ok {
		var addAdmin AddAdmin
		if err := json.Unmarshal(dataBytes, &addAdmin); err != nil {
			fmt.Printf("Failed to unmarshal as AddAdmin: %v\n", err)
			return wasmbridge.HandleError(err.Error())
		}
		jsonData = addAdmin
		fmt.Println("The AddAdmin data is :", addAdmin)
		fmt.Println("The jsonData data is :", jsonData)
		filePath = config.GetEnvConfig().AdminUpdatePath // File for AddAdmin data
		fmt.Println("The file path is :", filePath)
	} else {
		fmt.Println("Unknown data structure")
		return wasmbridge.HandleError(err.Error())
	}

	// var jsonData interface{}
	if err := json.Unmarshal(dataBytes, &jsonData); err != nil {
		fmt.Printf("Failed to parse JSON data: %v\n", err)
		return wasmbridge.HandleError("Invalid JSON data")
	}

	// filePath := "C:/Users/allen/Working-repo/ymca/ymca-wellness-cafe-project/dappServer/test.json"
	// filePath := config.GetEnvConfig().ActivityUpdatePath
	// Step 1: Read the existing file content
	existingContent, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) { // Ignore error if file doesn't exist
		fmt.Printf("Failed to read existing file: %v\n", err)
		return wasmbridge.HandleError(err.Error())
	}

	var existingData []interface{}
	if len(existingContent) > 0 {
		if err := json.Unmarshal(existingContent, &existingData); err != nil {
			fmt.Printf("Failed to parse existing JSON data: %v\n", err)
			return wasmbridge.HandleError("Invalid existing JSON data")
		}
	} else {
		existingData = []interface{}{}
	}

	// Step 2: Append new data to existing data
	existingData = append(existingData, jsonData)

	// Step 3: Write the updated data back to the file
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err) // Handle the error appropriately in production code
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty-print JSON
	if err := encoder.Encode(existingData); err != nil {
		fmt.Printf("Failed to write JSON data to file: %v\n", err)
		return wasmbridge.HandleError(err.Error())
	}
	response := fmt.Sprintf("Succesfully wrote data to DB")
	err = wasmbridge.UpdateDataToWASM(caller, h.allocFunc, response, outputArgs)
	if err != nil {
		fmt.Println("Failed to update data to WASM", err)
		return wasmbridge.HandleError(err.Error())
	}

	fmt.Printf("Successfully wrote data to JSON file: %s\n", filePath)
	return wasmbridge.HandleOk() // Return success
}
