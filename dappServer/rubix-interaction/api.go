package rubix_interaction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func GetSmartContractData(token string, address string) []byte {
	data := map[string]interface{}{
		"token":  token,
		"latest": true,
	}
	bodyJSON, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return nil
	}
	url := address + "/api/get-smart-contract-token-chain-data"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{
		Timeout: 30 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return nil
	}

	fmt.Println("Response Status:", resp.Status)
	data2, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %s\n", err)
		return nil
	}
	// Process the data as needed
	fmt.Println("Response Body in get smart contract data :", string(data2))

	return data2

}

func RegisterCallBackUrl(smartContractTokenHash string, urlPort string, endPoint string, nodePort string) {
	callBackUrl := fmt.Sprintf("http://localhost:%s/%s", urlPort, endPoint)
	data := map[string]interface{}{
		"CallBackURL":        callBackUrl,
		"SmartContractToken": smartContractTokenHash,
	}
	bodyJSON, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}
	url := fmt.Sprintf("http://localhost:%s/api/register-callback-url", nodePort)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return
	}
	fmt.Println("Response Status:", resp.Status)
	data2, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %s\n", err)
		return
	}
	fmt.Println("Response Body in register callback url :", string(data2))
}
